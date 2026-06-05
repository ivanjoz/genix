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
