<script lang="ts" module>
	import type { SectionSchema } from '$ecommerce/renderer/section-types';

	export const schema: SectionSchema = {
		name: 'HTML Section',
		description: 'A section authored as raw HTML (with custom component tags), parsed to an AST and rendered.',
		category: 'text',
		content: ['html'],
		css: ['container'],
	};
</script>

<script lang="ts">
	import type { StandardContent } from '$ecommerce/renderer/section-types';
	import { parseHTML } from '$ecommerce/html-ast/parse-html';
	import AstRenderer from '$ecommerce/renderer/AstRenderer.svelte';

	interface Props {
		content: StandardContent;
		css: Record<string, string>;
	}

	const { content, css }: Props = $props();

	// Prefer the stored AST (the editable model); fall back to parsing raw HTML.
	// Re-parses only when the HTML string changes (derived memoization).
	const ast = $derived(content.ast ?? parseHTML(content.html ?? ''));
</script>

<div class={css.container || ''}>
	<AstRenderer nodes={ast} />
</div>
