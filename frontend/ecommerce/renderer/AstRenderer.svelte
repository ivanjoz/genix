<script lang="ts" module>
	import type { ComponentAST, ColorPalette } from './renderer-types';

	/** Native HTML tags allowed for rendering (script/style intentionally excluded). */
	const NATIVE_TAGS = new Set([
		'div', 'section', 'article', 'aside', 'header', 'footer', 'main', 'nav',
		'span', 'p', 'a', 'button', 'img', 'ul', 'ol', 'li', 'figure', 'figcaption',
		'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'strong', 'em', 'small', 'br', 'hr',
		'table', 'thead', 'tbody', 'tr', 'td', 'th', 'label',
	]);

	/** Void tags must not receive children. */
	const VOID_TAGS = new Set(['img', 'br', 'hr', 'input']);

	export function isNativeTag(tagName: string): boolean {
		return NATIVE_TAGS.has(tagName);
	}
</script>

<script lang="ts">
	import { resolveTokens } from './token-resolver';
	import { astComponentRegistry } from '$ecommerce/html-ast/component-registry';
	import { TEXT_TAG } from '$ecommerce/html-ast/parse-html';

	interface Props {
		nodes: ComponentAST | ComponentAST[];
		palette?: ColorPalette;
		values?: Record<string, string>;
	}

	const { nodes, palette, values = {} }: Props = $props();

	const list = $derived(Array.isArray(nodes) ? nodes : [nodes]);
</script>

{#snippet text(value: string)}
	{#each value.split('\n') as line, i (i)}
		{#if i > 0}<br />{/if}{line}
	{/each}
{/snippet}

{#snippet renderNode(node: ComponentAST)}
	{#if node.tagName === TEXT_TAG}
		{@render text(node.text ?? '')}
	{:else if astComponentRegistry[node.tagName]}
		{@const Comp = astComponentRegistry[node.tagName]}
		{@const cls = resolveTokens(node.css, [], values, palette)}
		<!-- Custom components may wrap overlay markup (e.g. ImageEffect's caption);
		     pass nested AST nodes through as children. Components that ignore the
		     children snippet (ProductsByCategory, etc.) are unaffected. -->
		<Comp {...node.props} css={cls} style={node.style}>
			{#if node.children}
				{#each node.children as child, i (i)}
					{@render renderNode(child)}
				{/each}
			{/if}
		</Comp>
	{:else if isNativeTag(node.tagName)}
		{@const cls = resolveTokens(node.css, [], values, palette)}
		{#if VOID_TAGS.has(node.tagName)}
			<svelte:element this={node.tagName} class={cls} style={node.style} {...node.attributes} />
		{:else}
			<svelte:element this={node.tagName} class={cls} style={node.style} {...node.attributes}>
				{#if node.text}{@render text(node.text)}{/if}
				{#if node.children}
					{#each node.children as child, i (i)}
						{@render renderNode(child)}
					{/each}
				{/if}
			</svelte:element>
		{/if}
	{:else}
		<div class="bg-red-50 p-3 border border-red-200 text-red-600 my-2 rounded text-sm">
			<strong>Error:</strong> Unknown tag "<code>{node.tagName}</code>". Not a registered
			component or allowed native tag.
		</div>
	{/if}
{/snippet}

{#each list as node, i (i)}
	{@render renderNode(node)}
{/each}
