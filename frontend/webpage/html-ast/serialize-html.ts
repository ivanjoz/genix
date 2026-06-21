/**
 * serializeAst: ComponentAST tree -> HTML string. The inverse of parseHTML,
 * used to hand a section's canonical AST to the agent as readable HTML with the
 * custom component tags PRESERVED (e.g. `<ProductGrid columns="3">`), unlike the
 * live-DOM snapshot which has already expanded them into divs.
 *
 * Field mapping (mirrors parse-html.ts in reverse):
 *   - node.css        -> class="..."
 *   - node.style      -> style="..."   (already-compiled inline CSS)
 *   - node.role       -> data-role="..."
 *   - node.attributes -> raw attribute passthrough (native tags)
 *   - node.props      -> attributes for custom components (tagName uppercase)
 *   - node.text       -> escaped text content (leaf)
 *   - node.children   -> recursively serialized children
 *   - '#text' nodes   -> escaped text only
 */
import type { ComponentAST } from '../renderer/renderer-types';
import { TEXT_TAG } from './parse-html';

/** Void elements never get a closing tag (and carry no children). */
const VOID_TAGS = new Set([
	'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
	'link', 'meta', 'param', 'source', 'track', 'wbr',
]);

/** Escape text destined for an element's body. */
function escapeText(value: string): string {
	return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

/** Escape a value going inside a double-quoted attribute. */
function escapeAttr(value: string): string {
	return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;');
}

/** Serialize one prop value back to an attribute string (arrays/objects -> JSON-ish). */
function propToAttrValue(value: unknown): string {
	if (Array.isArray(value)) return value.join(',');
	if (value !== null && typeof value === 'object') return JSON.stringify(value);
	return String(value);
}

/** Build the attribute list for a node, in a stable order for readability. */
function serializeAttributes(node: ComponentAST): string {
	const parts: string[] = [];
	if (node.css) parts.push(`class="${escapeAttr(node.css)}"`);
	if (node.style) parts.push(`style="${escapeAttr(node.style)}"`);
	if (node.role) parts.push(`data-role="${escapeAttr(node.role)}"`);
	// Custom-component props (uppercase tag) become attributes; native tags use
	// their raw attribute passthrough. A node only ever has one of the two.
	if (node.props) {
		for (const [key, value] of Object.entries(node.props)) {
			if (value === undefined || value === null) continue;
			// Bare boolean true -> valueless attribute (e.g. `<X flag>`).
			if (value === true) { parts.push(key); continue; }
			parts.push(`${key}="${escapeAttr(propToAttrValue(value))}"`);
		}
	}
	if (node.attributes) {
		for (const [key, value] of Object.entries(node.attributes)) {
			parts.push(`${key}="${escapeAttr(value)}"`);
		}
	}
	return parts.length ? ' ' + parts.join(' ') : '';
}

/** Serialize a single AST node to HTML. */
function serializeNode(node: ComponentAST): string {
	// Standalone text node within mixed content.
	if (node.tagName === TEXT_TAG) return escapeText(node.text ?? '');

	const attrs = serializeAttributes(node);
	const tag = node.tagName;

	if (VOID_TAGS.has(tag.toLowerCase())) return `<${tag}${attrs}>`;

	// Leaf collapsed to plain text, or children recursed.
	const inner = node.children?.length
		? node.children.map(serializeNode).join('')
		: escapeText(node.text ?? '');

	return `<${tag}${attrs}>${inner}</${tag}>`;
}

/** Serialize one or more top-level AST nodes into an HTML string. */
export function serializeAst(nodes: ComponentAST[]): string {
	return nodes.map(serializeNode).join('');
}
