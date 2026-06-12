/**
 * Helpers for editing a parsed HTML section in the builder.
 *
 * Nodes marked with `data-role` (e.g. "title", "content", "button") are the
 * editable parts. The editor walks the AST, collects these nodes, and binds
 * inputs to them. Because the AST lives inside the reactive editor store,
 * mutating a node's `text`/`attributes` re-renders the section in place.
 */
import type { ComponentAST } from '../renderer/renderer-types';
import type { EditorControl } from './coerce';
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

// --- Schema-driven editable props (builder editor groups) ----------------------
//
// Which props a component surfaces in the section editor — and the control to use —
// is declared per prop in component-schemas (`editor: 'category' | 'grid'`). The
// editor reads it from here so adding a component/prop is a schema change only.

/** Attribute names on a node's schema that use the given editor control, in declared order. */
export function editorPropNames(node: ComponentAST, control: EditorControl): string[] {
	const props = getComponentSchema(node.tagName)?.props;
	if (!props) return [];
	return Object.keys(props).filter((attr) => props[attr].editor === control);
}

/** Bilingual "EN|ES" label for an editor field, falling back to the attribute name. */
export function editorPropLabel(node: ComponentAST, attr: string): string {
	return getComponentSchema(node.tagName)?.props[attr]?.label ?? attr;
}

/** Collect every AST node whose schema declares at least one prop with the given editor control. */
export function collectEditableNodes(
	nodes: ComponentAST | ComponentAST[] | undefined,
	control: EditorControl,
): ComponentAST[] {
	const out: ComponentAST[] = [];
	const walk = (node: ComponentAST) => {
		if (editorPropNames(node, control).length) out.push(node);
		node.children?.forEach(walk);
	};
	if (Array.isArray(nodes)) nodes.forEach(walk);
	else if (nodes) walk(nodes);
	return out;
}

/** Read a numeric prop value; `number[]`-typed props (e.g. categoryIDs) expose their first entry. */
export function getNodeProp(node: ComponentAST, attr: string): number | undefined {
	const value = node.props?.[attr];
	return Array.isArray(value) ? value[0] : value;
}

/**
 * Write (or clear, when empty) a numeric prop, honoring its schema type so a `number[]` prop
 * stores the value as a single-element list. Mutation re-renders the live section.
 */
export function setNodeProp(node: ComponentAST, attr: string, value: number | undefined): void {
	if (!node.props) node.props = {};
	const isList = getComponentSchema(node.tagName)?.props[attr]?.type === 'number[]';
	if (value == null || Number.isNaN(value)) delete node.props[attr];
	else node.props[attr] = isList ? [value] : value;
}
