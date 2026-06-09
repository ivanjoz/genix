/**
 * parseHTML: raw HTML (with custom component tags) -> ComponentAST tree.
 *
 * Uses htmlparser2 with case preservation so `<ProductGrid>` stays distinct
 * from native `<div>`. Pure and isomorphic (browser builder + server/agent).
 *
 *   - `class`              -> node.css   (Tailwind tokens preserved)
 *   - style attributes     -> node.style (see style-attributes.ts)
 *   - custom-component tags -> node.props (coerced via component-schemas.ts)
 *   - native-tag leftovers  -> node.attributes (raw string passthrough)
 *   - text content          -> node.text (leaf) or `#text` child nodes (mixed)
 */
import { parseDocument } from 'htmlparser2';
import type { ComponentAST } from '../renderer/renderer-types';

/** Minimal structural view of the htmlparser2/domhandler node shape we use. */
interface DomNode {
	type: string;
	data?: string;
	name?: string;
	attribs?: Record<string, string>;
	children?: DomNode[];
}
import { compileStyleAttributes } from './style-attributes';
import { coerceProps } from './coerce';
import { getComponentSchema } from './component-schemas';

/** Marker tagName for standalone text nodes within mixed content. */
export const TEXT_TAG = '#text';

/** A custom component is any tag whose name starts with an uppercase letter. */
function isCustomComponent(tagName: string): boolean {
	return /^[A-Z]/.test(tagName);
}

function collapseWhitespace(text: string): string {
	return text.replace(/\s+/g, ' ');
}

function mapElement(el: DomNode): ComponentAST {
	const tagName = el.name ?? 'div';
	const attribs = el.attribs ?? {};
	const node: ComponentAST = { tagName };

	if (attribs.class) node.css = attribs.class;

	// Editable role marker for the builder. `data-role` is canonical (valid HTML);
	// plain `role` is also accepted. Consumed so it is not emitted to the DOM.
	const role = attribs['data-role'] ?? attribs.role;
	if (role) node.role = role.trim();

	const { style, consumed } = compileStyleAttributes(attribs);
	if (style) node.style = style;

	// Remaining attributes (not class, not role, not consumed as style)
	const rest: Record<string, string> = {};
	for (const [k, v] of Object.entries(attribs)) {
		if (k === 'class' || k === 'data-role' || k === 'role' || consumed.has(k)) continue;
		rest[k] = v;
	}

	if (isCustomComponent(tagName)) {
		const props = coerceProps(rest, getComponentSchema(tagName));
		if (Object.keys(props).length) node.props = props;
	} else if (Object.keys(rest).length) {
		node.attributes = rest;
	}

	const kids = mapChildren(el.children ?? []);
	const elementKids = kids.filter((k) => k.tagName !== TEXT_TAG);

	if (elementKids.length === 0 && kids.length > 0) {
		// Leaf with only text -> collapse into node.text
		node.text = kids.map((k) => k.text).join(' ').trim();
	} else if (kids.length > 0) {
		node.children = kids;
	}

	return node;
}

function mapChildren(children: DomNode[]): ComponentAST[] {
	const out: ComponentAST[] = [];
	for (const child of children) {
		if (child.type === 'text') {
			const text = collapseWhitespace(child.data ?? '');
			if (text.trim()) out.push({ tagName: TEXT_TAG, text: text.trim() });
		} else if (child.type === 'tag' || child.type === 'script' || child.type === 'style') {
			// scripts/styles are mapped as nodes but blocked at render time
			out.push(mapElement(child));
		}
	}
	return out;
}

/** Parse an HTML string into one or more top-level ComponentAST nodes. */
export function parseHTML(html: string): ComponentAST[] {
	if (!html || !html.trim()) return [];
	const doc = parseDocument(html, {
		lowerCaseTags: false,
		lowerCaseAttributeNames: false,
		recognizeSelfClosing: true,
	});
	return mapChildren(doc.children as unknown as DomNode[]);
}
