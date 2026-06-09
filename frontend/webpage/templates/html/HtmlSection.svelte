<script lang="ts" module>
	import type { SectionSchema } from '$ecommerce/renderer/section-types';

	export const schema: SectionSchema = {
		name: 'HTML Section',
		description: 'A section authored as raw HTML (with custom component tags), parsed to an AST and rendered.',
		category: 'text',
		content: [],
		css: ['container'],
	};
</script>

<script lang="ts">
	import type { SectionProps } from '$ecommerce/renderer/section-types';
	import AstRenderer from '$ecommerce/renderer/AstRenderer.svelte';

	// AST is the canonical model: parsed from the template HTML once at add/load time
	// (see editor.svelte addSection / EcommerceBuilder seed). Render it directly.
	const { ast = [], css = {} }: SectionProps = $props();
</script>

<div class={css.container || ''}>
	<AstRenderer nodes={ast} />
</div>

