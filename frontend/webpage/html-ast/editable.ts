/**
 * Helpers for editing a parsed HTML section in the builder.
 *
 * Nodes marked with `data-role` (e.g. "title", "content", "button") are the
 * editable parts. The editor walks the AST, collects these nodes, and binds
 * inputs to them. Because the AST lives inside the reactive editor store,
 * mutating a node's `text`/`attributes` re-renders the section in place.
 */
import type { ComponentAST } from '../renderer/renderer-types';
import { TEXT_TAG } from './parse-html';
import { getComponentSchema } from './component-schemas';

export interface EditableNode {
	/** the role label, e.g. 'title' | 'content' | 'button' */
	role: string;
	/** reference to the live AST node (mutate to edit) */
	node: ComponentAST;
}

/** Walk the AST (depth-first) collecting every node that has a `role`. */
export function collectRoleNodes(
	nodes: ComponentAST | ComponentAST[] | undefined,
): EditableNode[] {
	const out: EditableNode[] = [];
	const walk = (node: ComponentAST) => {
		if (node.role) out.push({ role: node.role, node });
		node.children?.forEach(walk);
	};
	if (Array.isArray(nodes)) nodes.forEach(walk);
	else if (nodes) walk(nodes);
	return out;
}

/** A role node is link-like (has an editable href) if it is an anchor. */
export function isLinkNode(node: ComponentAST): boolean {
	return node.tagName === 'a';
}

/** A role node is image-like (edited via the ImageBlockEditor) if it is an ImageEffect. */
export function isImageNode(node: ComponentAST): boolean {
	return node.tagName === 'ImageEffect' || node.tagName === 'img';
}

// --- Recursive auto-editor helpers (AstEditor) ---------------------------------

/**
 * Opt out of editing for a node and its subtree. Authored as `data-noedit` on the
 * element; parse-html keeps unknown `data-*` in `node.attributes`.
 */
export function isNoEdit(node: ComponentAST): boolean {
	const a = node.attributes;
	if (a && ('data-noedit' in a || 'noedit' in a)) return true;
	// Custom components keep unknown attributes (incl. data-noedit) on `props`.
	const p = node.props;
	return !!p && ('data-noedit' in p || 'noedit' in p);
}

/** A standalone text run (mixed-content child) — edited as text only, no styling toolbar. */
export function isTextRun(node: ComponentAST): boolean {
	return node.tagName === TEXT_TAG;
}

/**
 * A leaf element that directly carries text (parse-html collapses it into `node.text`).
 * An empty string still counts: clearing the field must not drop the node from the editor.
 */
export function isTextLeaf(node: ComponentAST): boolean {
	return node.tagName !== TEXT_TAG && typeof node.text === 'string';
}

/**
 * Whether a component treats its direct children as navigable units — a carousel's
 * slides (`childrenAs: 'slides'`) or a tabbed layer's panels (`childrenAs: 'tabs'`).
 * The units are simply `node.children`; the editor pages through them via an OptionsStrip.
 */
export function childrenAsUnits(node: ComponentAST): boolean {
	return !!getComponentSchema(node.tagName)?.childrenAs;
}

/** Direct child nodes of a node (the slide/tab set, for a navigable container). */
export function unitChildren(node: ComponentAST): ComponentAST[] {
	return node.children ?? [];
}

/** Singular noun for one navigable child — "slide" or "tab" — for editor labels. */
export function unitNoun(node: ComponentAST): 'slide' | 'tab' {
	return getComponentSchema(node.tagName)?.childrenAs === 'tabs' ? 'tab' : 'slide';
}

/**
 * Label for the i-th navigable child. Tabs read their labels from the `options`
 * attribute ("A|B|C"); anything missing (and every slide) falls back to "Slide/Tab N".
 */
export function unitLabel(node: ComponentAST, i: number): string {
	if (unitNoun(node) === 'tab') {
		const raw = node.props?.options;
		if (typeof raw === 'string') {
			const labels = raw.split('|').map((s) => s.trim()).filter(Boolean);
			if (labels[i]) return labels[i];
		}
		return `Tab ${i + 1}`;
	}
	return `Slide ${i + 1}`;
}

const TAG_LABELS: Record<string, string> = {
	h1: 'Heading', h2: 'Heading', h3: 'Heading', h4: 'Heading', h5: 'Heading', h6: 'Heading',
	p: 'Text', span: 'Text', small: 'Text', strong: 'Text', em: 'Text', li: 'Text',
	a: 'Link', button: 'Button', img: 'Image', ImageEffect: 'Image',
};

/** Human label for an editor field: the explicit role wins, else a humanized tag name. */
export function humanizeLabel(node: ComponentAST): string {
	if (node.role) return node.role;
	return TAG_LABELS[node.tagName] ?? node.tagName;
}

/** Custom components whose rendered category is editable via the category selector. */
const CATEGORY_TAGS = new Set(['ProductsByCategory', 'CategoryDescription']);

/** Collect every category-bound custom-component node in the AST (ProductsByCategory / CategoryDescription). */
export function collectCategoryNodes(
	nodes: ComponentAST | ComponentAST[] | undefined,
): ComponentAST[] {
	const out: ComponentAST[] = [];
	const walk = (node: ComponentAST) => {
		if (CATEGORY_TAGS.has(node.tagName)) out.push(node);
		node.children?.forEach(walk);
	};
	if (Array.isArray(nodes)) nodes.forEach(walk);
	else if (nodes) walk(nodes);
	return out;
}

/** Read the category ID currently bound to a category node (CategoryDescription stores it as a list). */
export function getNodeCategoryID(node: ComponentAST): number | undefined {
	if (node.tagName === 'CategoryDescription') return node.props?.categoryIDs?.[0];
	return node.props?.categoryID;
}

/** Set the category ID on a category node, writing the prop shape each component expects (mutation re-renders). */
export function setNodeCategoryID(node: ComponentAST, id: number): void {
	if (!node.props) node.props = {};
	if (node.tagName === 'CategoryDescription') node.props.categoryIDs = [id];
	else node.props.categoryID = id;
}
