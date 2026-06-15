/**
 * HTML-native COLOR attributes.
 *
 * Only color attributes are compiled to inline `style`, because a bare palette
 * index must become a CSS variable (`var(--color-N)`). Everything else
 * (spacing, sizing, alignment, etc.) is expressed with Tailwind classes,
 * compiled at runtime by the UnoCSS generator (`stores/uno-generator.ts`, driven
 * by the builder's live-css store) and at build time by Tailwind, so it stays out
 * of here.
 *
 *   - A bare integer 1..10 -> `var(--color-N)` (palette token, resolved by the
 *     cascade via `generatePaletteStyles`).
 *   - Anything else (`#fff`, `red`, `rgb(...)`) passes through unchanged.
 */

/** color attribute name -> the CSS properties it expands to */
const COLOR_ATTRS: Record<string, string[]> = {
	'background-color': ['background-color'],
	'color': ['color'],
	'border-color': ['border-color'],
};

const INT_RE = /^\d+$/;

function resolveColor(raw: string): string {
	const value = raw.trim();
	if (INT_RE.test(value)) {
		const n = parseInt(value, 10);
		if (n >= 1 && n <= 10) return `var(--color-${n})`;
	}
	return value;
}

export interface CompiledStyle {
	/** inline CSS string, e.g. "background-color: var(--color-9);" */
	style: string;
	/** attribute names consumed (so the parser does not re-emit them) */
	consumed: Set<string>;
}

/**
 * Compile recognised color attributes from a raw attribute map into an inline
 * `style` string. Unrecognised attributes are left for the caller.
 */
export function compileStyleAttributes(attribs: Record<string, string>): CompiledStyle {
	const decls: string[] = [];
	const consumed = new Set<string>();

	for (const [attr, raw] of Object.entries(attribs)) {
		const cssProps = COLOR_ATTRS[attr];
		if (!cssProps) continue;
		const value = resolveColor(raw);
		for (const prop of cssProps) decls.push(`${prop}: ${value};`);
		consumed.add(attr);
	}

	return { style: decls.join(' '), consumed };
}

export function isStyleAttribute(attr: string): boolean {
	return attr in COLOR_ATTRS;
}
