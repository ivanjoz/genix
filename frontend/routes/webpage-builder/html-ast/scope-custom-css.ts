import type { ComponentAST } from '$ecommerce/renderer/renderer-types';
import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Scope agent-authored custom CSS to page-unique minified classes.
 *
 * The agent returns raw CSS using semantic class names it also applies in the
 * HTML, e.g. `<div class="hero-gradient">` + `.hero-gradient { background: … }`.
 * We index every class that is BOTH defined in the CSS and used in the section
 * AST, and rename it to a page-global minified class `.x{n}` — rewriting the CSS
 * selectors AND the AST class tokens in lockstep. `@keyframes` names are
 * namespaced the same way (and their `animation` references rewritten).
 *
 * Safety: only class selectors that target at least one used class survive.
 * Global selectors (`body`, `*`, `#id`, bare tags) and unknown at-rules
 * (`@font-face`, `@import`, …) are dropped so the agent can only style via its
 * own classes. The id counter is global to the page (not tied to a section), so
 * names are stable across section reordering.
 */

/** Match a `.class` simple selector (CSS identifier). */
const CLASS_TOKEN_RE = /\.(-?[A-Za-z_][\w-]*)/g;
const KEYFRAMES_RE = /^@(?:-[a-z]+-)?keyframes\s+(.+)$/i;
const MEDIA_OR_SUPPORTS_RE = /^@(?:media|supports)\b/i;

interface CssBlock {
	prelude: string;
	inner: string;
}

/**
 * Split CSS into top-level brace blocks (`selector{…}`, `@media{…}`,
 * `@keyframes{…}`). Statement at-rules (`@import …;`) are skipped. Brace-counted
 * so nested blocks (a rule inside `@media`) stay intact in `inner`.
 */
function walkBlocks(css: string): CssBlock[] {
	const blocks: CssBlock[] = [];
	const n = css.length;
	let i = 0;
	while (i < n) {
		while (i < n && /\s/.test(css[i])) i++;
		if (i >= n) break;
		const start = i;
		let j = i;
		while (j < n && css[j] !== '{' && css[j] !== ';') j++;
		if (j >= n) break;
		if (css[j] === ';') {
			i = j + 1; // statement at-rule (@import/@charset) — drop
			continue;
		}
		let depth = 0;
		let k = j;
		for (; k < n; k++) {
			if (css[k] === '{') depth++;
			else if (css[k] === '}' && --depth === 0) {
				k++;
				break;
			}
		}
		blocks.push({ prelude: css.slice(start, j).trim(), inner: css.slice(j + 1, k - 1) });
		i = k;
	}
	return blocks;
}

/** Classes referenced by a selector + whether it targets anything non-class. */
function selectorInfo(selector: string): { classes: string[]; global: boolean } {
	const classes = [...selector.matchAll(CLASS_TOKEN_RE)].map((m) => m[1]);
	// Remove classes, attribute selectors, pseudos and combinators; anything left
	// (a bare tag, `*`, or an `#id`) means the selector reaches outside its classes.
	const rest = selector
		.replace(CLASS_TOKEN_RE, ' ')
		.replace(/\[[^\]]*\]/g, ' ')
		.replace(/::?[A-Za-z-]+(?:\([^)]*\))?/g, ' ')
		.replace(/[>+~,\s]/g, '');
	return { classes, global: rest.length > 0 };
}

function escapeRe(s: string): string {
	return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** Walk an AST subtree collecting every class token used in `node.css`. */
function collectUsedClasses(nodes: ComponentAST[] | undefined, out: Set<string>): void {
	if (!nodes) return;
	for (const node of nodes) {
		if (node.css) for (const c of node.css.split(/\s+/)) if (c) out.add(c);
		collectUsedClasses(node.children, out);
	}
}

/** Rewrite AST class tokens through the name→minified map, in place. */
function rewriteAstClasses(nodes: ComponentAST[] | undefined, classMap: Map<string, string>): void {
	if (!nodes) return;
	for (const node of nodes) {
		if (node.css && /\S/.test(node.css)) {
			node.css = node.css
				.split(/\s+/)
				.filter(Boolean)
				.map((tok) => classMap.get(tok) ?? tok)
				.join(' ');
		}
		rewriteAstClasses(node.children, classMap);
	}
}

/**
 * Returns a monotonic allocator for page-global custom-class ids: scans existing
 * `x{n}` names across all sections' AST + CustomCss and continues past the max,
 * so ids never collide and survive reload/reorder (no persisted counter).
 */
export function nextGlobalId(sections: SectionData[]): () => number {
	let max = 0;
	const scan = (s: string | undefined) => {
		if (!s) return;
		const re = /\bx(\d+)\b/g;
		let m: RegExpExecArray | null;
		while ((m = re.exec(s))) max = Math.max(max, Number(m[1]));
	};
	const walk = (nodes: ComponentAST[] | undefined) => {
		if (!nodes) return;
		for (const n of nodes) {
			scan(n.css);
			walk(n.children);
		}
	};
	for (const sec of sections) {
		scan(sec.CustomCss);
		walk(sec.Ast);
	}
	let n = max;
	return () => ++n;
}

/**
 * Scope one section's custom CSS. Mutates `ast` (renames class tokens) and
 * returns the rewritten CSS (empty string if nothing survives). `allocId`
 * supplies the next page-global id (see nextGlobalId).
 */
export function scopeCustomCss(cssText: string | undefined, ast: ComponentAST[], allocId: () => number): string {
	if (!cssText || !cssText.trim()) return '';

	const used = new Set<string>();
	collectUsedClasses(ast, used);

	const classMap = new Map<string, string>(); // original class -> x{n}
	const kfMap = new Map<string, string>(); // original @keyframes name -> x{n}
	const idFor = (map: Map<string, string>, name: string) => {
		let v = map.get(name);
		if (!v) {
			v = 'x' + allocId();
			map.set(name, v);
		}
		return v;
	};

	const scopeSelectorList = (prelude: string): string => {
		const kept: string[] = [];
		for (const raw of prelude.split(',')) {
			const sel = raw.trim();
			if (!sel) continue;
			const { classes, global } = selectorInfo(sel);
			// Drop global selectors and rules that don't target a used class.
			if (global || classes.length === 0) continue;
			if (!classes.some((c) => used.has(c))) continue;
			kept.push(sel.replace(CLASS_TOKEN_RE, (_m, name) => '.' + idFor(classMap, name)));
		}
		return kept.join(',');
	};

	const processBlocks = (css: string): string => {
		const out: string[] = [];
		for (const b of walkBlocks(css)) {
			const kf = b.prelude.match(KEYFRAMES_RE);
			if (kf) {
				out.push(`@keyframes ${idFor(kfMap, kf[1].trim())}{${b.inner}}`);
				continue;
			}
			if (MEDIA_OR_SUPPORTS_RE.test(b.prelude)) {
				const inner = processBlocks(b.inner);
				if (inner.trim()) out.push(`${b.prelude}{${inner}}`);
				continue;
			}
			if (b.prelude.startsWith('@')) continue; // drop @font-face etc.
			const sel = scopeSelectorList(b.prelude);
			if (sel) out.push(`${sel}{${b.inner.trim()}}`);
		}
		return out.join('');
	};

	let result = processBlocks(cssText);

	// Rewrite @keyframes references in `animation`/`animation-name` values. Done as
	// a final pass over the whole result so a rule that references a keyframe
	// declared later still gets rewritten.
	for (const [orig, scoped] of kfMap) {
		result = result.replace(new RegExp(`\\b${escapeRe(orig)}\\b`, 'g'), scoped);
	}

	rewriteAstClasses(ast, classMap);
	return result;
}
