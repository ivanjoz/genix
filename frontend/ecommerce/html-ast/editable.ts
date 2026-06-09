/**
 * Helpers for editing a parsed HTML section in the builder.
 *
 * Nodes marked with `data-role` (e.g. "title", "content", "button") are the
 * editable parts. The editor walks the AST, collects these nodes, and binds
 * inputs to them. Because the AST lives inside the reactive editor store,
 * mutating a node's `text`/`attributes` re-renders the section in place.
 */
import type { ComponentAST } from '../renderer/renderer-types';

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
	return node.tagName === 'ImageEffect';
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
