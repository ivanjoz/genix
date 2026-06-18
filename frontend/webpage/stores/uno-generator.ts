import type { UnoGenerator } from '@unocss/core';
import type { SectionData } from '../renderer/section-types';
import type { ComponentAST } from '../renderer/renderer-types';

/**
 * On-demand Tailwind-compatible CSS generation for runtime-authored classes.
 *
 * Agents generate HTML with arbitrary Tailwind classes, and the WYSIWYG editor
 * mutates classes at runtime — neither exists in source, so build-time Tailwind
 * (`@tailwindcss/vite`) cannot cover them. We run the real UnoCSS engine on
 * demand (no global MutationObserver): collect the class strings we already
 * track in state, and ask the generator for their CSS.
 *
 * The theme MIRRORS the build `@theme` in routes/tailwind.css so a class renders
 * identically whether it came from source (build engine) or runtime (here):
 *   - `--spacing: 1px`  -> `p-4` = 4px
 *   - custom breakpoints md/lg/xl/2xl
 *   - palette colors `color-1..10` emit `var(--color-N)`, resolved live by the
 *     `--color-N` variables that `generatePaletteStyles` injects on the root, so
 *     palette swaps need no regeneration.
 */

const PALETTE_COLORS: Record<string, string> = {};
for (let i = 1; i <= 10; i++) PALETTE_COLORS[`color-${i}`] = `var(--color-${i})`;

let generatorPromise: Promise<UnoGenerator> | null = null;

function getGenerator(): Promise<UnoGenerator> {
	if (!generatorPromise) {
		console.debug('[live-css] Loading UnoCSS runtime');
		generatorPromise = Promise.all([
			import('@unocss/core'),
			import('@unocss/preset-wind4'),
		]).then(([{ createGenerator }, { default: presetWind4 }]) => {
			return createGenerator({
				// Keep runtime utilities aligned with the build-time Tailwind theme.
				presets: [presetWind4({ preflights: { reset: false } })],
				theme: {
					spacing: { DEFAULT: '1px' },
					breakpoint: { md: '749px', lg: '1139px', xl: '1379px', '2xl': '1539px' },
					colors: PALETTE_COLORS,
				},
			});
		});
	}
	return generatorPromise;
}

/** Push every class found on an AST subtree into `out`. */
function collectAstTokens(node: ComponentAST, out: Set<string>): void {
	if (node.css) for (const c of node.css.split(/\s+/)) if (c) out.add(c);
	if (node.children) for (const child of node.children) collectAstTokens(child, out);
}

/**
 * Collect the runtime class tokens for a set of sections: the slot-based
 * `section.Css` and the parsed HTML section AST (`section.Ast`, where agent
 * classes live).
 */
export function collectTokens(sections: SectionData[]): Set<string> {
	const tokens = new Set<string>();

	for (const section of sections) {
		// Slot-based CSS for the HTML section container.
		if (section.Css) {
			for (const val of Object.values(section.Css)) {
				if (typeof val === 'string') for (const c of val.split(/\s+/)) if (c) tokens.add(c);
			}
		}

		// HTML section AST — where agent-authored classes live.
		if (section.Ast) {
			for (const node of section.Ast) collectAstTokens(node, tokens);
		}
	}

	return tokens;
}

/**
 * Split a CSS string into its top-level `@media` blocks and everything else.
 *
 * Brace-counting walk over top-level items (comments, statement at-rules,
 * `selector{}` / `@media{}` / `@supports{}` / `@property{}` blocks). Only blocks
 * whose prelude starts with `@media` go to `media`; the rest (plain utilities,
 * `:root` theme vars, `@supports`/`@property` runtime-var init) go to `base`.
 */
function splitMediaRules(css: string): { base: string; media: string } {
	const base: string[] = [];
	const media: string[] = [];
	const n = css.length;
	let i = 0;

	while (i < n) {
		while (i < n && /\s/.test(css[i])) i++;
		if (i >= n) break;
		const start = i;

		// Comments: keep with base (harmless, preserves the `/* layer: ... */` notes).
		if (css[i] === '/' && css[i + 1] === '*') {
			const end = css.indexOf('*/', i + 2);
			i = end === -1 ? n : end + 2;
			base.push(css.slice(start, i));
			continue;
		}

		// Read the prelude up to the block open `{` or a statement terminator `;`.
		let j = i;
		while (j < n && css[j] !== '{' && css[j] !== ';') j++;
		if (j >= n) {
			base.push(css.slice(start));
			break;
		}
		if (css[j] === ';') {
			// Statement at-rule (e.g. `@import`/`@charset`) — no block.
			i = j + 1;
			base.push(css.slice(start, i));
			continue;
		}

		// Balanced `{...}` block (handles the rule nested inside `@media`).
		let depth = 0;
		let k = j;
		for (; k < n; k++) {
			if (css[k] === '{') depth++;
			else if (css[k] === '}' && --depth === 0) {
				k++;
				break;
			}
		}
		const chunk = css.slice(start, k);
		i = k;
		(css.slice(start, j).trim().startsWith('@media') ? media : base).push(chunk);
	}

	return { base: base.join('\n'), media: media.join('\n') };
}

/**
 * Wrap the split runtime CSS into its two cascade layers (declared in
 * routes/tailwind.css as `... ec-runtime, utilities, ec-runtime-media`):
 *
 *   - `ec-runtime` (BEFORE utilities) holds plain/base rules. When a base utility
 *     is re-emitted by BOTH engines (e.g. runtime `grid` vs build-time
 *     `md:grid-cols-3`), the build copy in `utilities` wins, so build-time
 *     responsive variants aren't collapsed to their mobile form by a runtime base
 *     duplicate (the original ProductsByCategory grid fix).
 *
 *   - `ec-runtime-media` (AFTER utilities) holds the `@media` (responsive) rules.
 *     Runtime-authored content owns responsive variants the storefront source
 *     never sees (e.g. `md:text-6xl`), while build-time may emit only the base
 *     (`text-4xl`, used elsewhere in source). Without this split the build-time
 *     base in `utilities` would override the runtime variant regardless of
 *     viewport; placing runtime media rules above `utilities` lets the responsive
 *     override win as authored.
 */
function wrapRuntimeLayers(base: string, media: string): string {
	let out = '';
	if (base.trim()) out += `@layer ec-runtime {\n${base}\n}`;
	if (media.trim()) out += `${out ? '\n' : ''}@layer ec-runtime-media {\n${media}\n}`;
	return out;
}

/**
 * Upgrade a stored runtime stylesheet to the two-layer format, idempotently.
 *
 * CSS persisted before the layer split is a single `@layer ec-runtime { … }`
 * block with the `@media` rules trapped inside it (below `utilities`), so any
 * runtime-only responsive variant loses to a build-time base utility. The
 * storefront serves stored CSS verbatim (no UnoCSS at view time), so we re-split
 * it here with the same pure string pass — no engine, cheap enough for prerender.
 * Already-split CSS (new saves) is returned untouched.
 */
export function normalizeRuntimeCss(css: string): string {
	if (!css || css.includes('@layer ec-runtime-media')) return css;
	// Unwrap the legacy single `@layer ec-runtime { … }` wrapper, if present.
	const wrapped = css.match(/^\s*@layer\s+ec-runtime\s*\{([\s\S]*)\}\s*$/);
	const { base, media } = splitMediaRules(wrapped ? wrapped[1] : css);
	return wrapRuntimeLayers(base, media) || css;
}

/**
 * Generate the utility CSS for the given tokens (plus the theme `:root` vars the
 * utilities reference — the reset preflight is disabled at the preset level).
 * The engine caches matched tokens internally, so repeated calls with
 * overlapping tokens are cheap.
 */
export async function generateCss(tokens: Set<string> | string[]): Promise<string> {
	const set = Array.isArray(tokens) ? new Set(tokens) : tokens;
	if (set.size === 0) return '';
	const uno = await getGenerator();
	const { css } = await uno.generate(set);
	if (!css) return '';

	const { base, media } = splitMediaRules(css);
	return wrapRuntimeLayers(base, media);
}
