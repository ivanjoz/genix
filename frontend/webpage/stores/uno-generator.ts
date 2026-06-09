import { createGenerator, type UnoGenerator } from '@unocss/core';
import presetWind4 from '@unocss/preset-wind4';
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
		generatorPromise = createGenerator({
			// Disable the reset preflight (the page already ships Tailwind's
			// preflight; a second reset would clash) but KEEP the theme layer, which
			// emits the :root CSS vars that utilities like text-4xl / max-w-3xl /
			// font-bold reference. UnoCSS namespaces these (--text-4xl-fontSize) so
			// they don't collide with build-Tailwind's (--text-4xl).
			presets: [presetWind4({ preflights: { reset: false } })],
			theme: {
				// Match routes/tailwind.css @theme exactly.
				spacing: { DEFAULT: '1px' },
				breakpoint: { md: '749px', lg: '1139px', xl: '1379px', '2xl': '1539px' },
				colors: PALETTE_COLORS,
			},
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
 * `section.Css`, the parsed HTML section AST (`section.Ast`, where agent classes
 * live), and any `Content.textLines`.
 */
export function collectTokens(sections: SectionData[]): Set<string> {
	const tokens = new Set<string>();

	for (const section of sections) {
		// Slot-based CSS (component sections + HTML section container).
		if (section.Css) {
			for (const val of Object.values(section.Css)) {
				if (typeof val === 'string') for (const c of val.split(/\s+/)) if (c) tokens.add(c);
			}
		}

		// HTML section AST — where agent-authored classes live.
		if (section.Ast) {
			for (const node of section.Ast) collectAstTokens(node, tokens);
		}

		// Standalone text lines on the flat content schema.
		const textLines = section.Content?.textLines;
		if (Array.isArray(textLines)) {
			for (const line of textLines) {
				if (line?.css) for (const c of line.css.split(/\s+/)) if (c) tokens.add(c);
			}
		}
	}

	return tokens;
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

	// Scope every runtime utility under a dedicated cascade layer (`ec-runtime`,
	// declared before `utilities` in routes/tailwind.css). Without this, the
	// runtime <style> is injected at the END of <head>, so any base utility it
	// re-emits (e.g. `grid`, `p-8`) would win a same-specificity tie against
	// build-time Tailwind's `md:*` variants purely by source order — collapsing
	// responsive layouts like ProductsByCategory's grid to their mobile form.
	// With the layer, when a class is emitted by BOTH engines the build copy
	// (correct base->variant ordering) governs, while runtime-only classes
	// (agent/editor authored, absent from source) still apply normally.
	return `@layer ec-runtime {\n${css}\n}`;
}
