/**
 * Coercion of raw HTML attribute strings into typed component props,
 * driven by a per-component prop schema.
 */

export type PropType =
	| 'string'
	| 'number'
	| 'boolean'
	| 'number[]'
	| 'string[]'
	| 'json';

export interface PropSpec {
	type: PropType;
	/** target prop name on the component, if it differs from the attribute name */
	prop?: string;
	/** value used when the attribute is absent */
	default?: any;
	/** allowed values for `string` props */
	enum?: readonly string[];
}

/** schema for one custom component: attribute name -> spec */
export interface ComponentSchema {
	tagName: string;
	description?: string;
	props: Record<string, PropSpec>;
	/**
	 * How the builder editor treats this component's direct children.
	 * `'slides'` → each child is a slide in a carousel; `'tabs'` → each child is a
	 * panel selected by an OptionsStrip. In both cases each child is a navigable unit:
	 * the editor shows an OptionsStrip and edits one child at a time, synced with the
	 * live preview. Absent → children edit inline.
	 */
	childrenAs?: 'slides' | 'tabs';
}

function parseList(raw: string): string[] {
	const trimmed = raw.trim();
	const inner = trimmed.startsWith('[') && trimmed.endsWith(']')
		? trimmed.slice(1, -1)
		: trimmed;
	return inner
		.split(',')
		.map((s) => s.trim().replace(/^["']|["']$/g, ''))
		.filter((s) => s.length > 0);
}

/** Coerce a single raw attribute value to the requested type. */
export function coerceValue(raw: string, type: PropType): any {
	switch (type) {
		case 'number': {
			const n = Number(raw);
			return Number.isNaN(n) ? undefined : n;
		}
		case 'boolean':
			// `<X flag>` yields "" in htmlparser2 -> treat as true; "false"/"0" -> false
			return raw !== 'false' && raw !== '0';
		case 'number[]':
			return parseList(raw)
				.map((s) => Number(s))
				.filter((n) => !Number.isNaN(n));
		case 'string[]':
			return parseList(raw);
		case 'json':
			try {
				return JSON.parse(raw);
			} catch {
				return undefined;
			}
		case 'string':
		default:
			return raw;
	}
}

/**
 * Coerce a raw attribute map for a custom component using its schema.
 * Schema-typed attributes are coerced and renamed to their target prop;
 * unknown attributes pass through unchanged (lenient); defaults fill gaps.
 */
export function coerceProps(
	attribs: Record<string, string>,
	schema?: ComponentSchema,
): Record<string, any> {
	const out: Record<string, any> = {};

	for (const [attr, raw] of Object.entries(attribs)) {
		const spec = schema?.props[attr];
		if (spec) {
			const value = coerceValue(raw, spec.type);
			if (value !== undefined) out[spec.prop ?? attr] = value;
		} else {
			out[attr] = raw;
		}
	}

	if (schema) {
		for (const [attr, spec] of Object.entries(schema.props)) {
			const target = spec.prop ?? attr;
			if (spec.default !== undefined && out[target] === undefined) {
				out[target] = spec.default;
			}
		}
	}

	return out;
}
