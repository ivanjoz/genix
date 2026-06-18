<script lang="ts" module>
		import type { SectionSchema } from './section-types';

	export const schema: SectionSchema = {
		name: 'HTML Section',
		description: 'A section authored as raw HTML (with custom component tags), parsed to an AST and rendered.',
		category: 'text',
		css: ['container'],
	};
</script>

<script lang="ts">
		import type { SectionProps } from './section-types';
		import AstRenderer from './AstRenderer.svelte';
		import ImageEffect from '../components/ImageEffect.svelte';
		import IconSprite from './IconSprite.svelte';

	// AST is the canonical model: parsed from the template HTML once at add/load time
	// (see editor.svelte addSection / EcommerceBuilder seed). Render it directly.
	// Padding and colors live on the AST's root element node (edited in-place by the
	// section editor); only the optional background image arrives via Attributes here.
	const { ast = [], css = {}, background, svgs }: SectionProps = $props();

	// Only paint the background-image layer when a source URL is actually set.
	const hasBackgroundImage = $derived(!!background?.src);
</script>

<!-- One hidden sprite per section; every Icon below references its symbols via `<use>`. -->
<IconSprite {svgs} />

{#if hasBackgroundImage}
	<!-- A positioned host so the image fills behind the content; content sits above via z-10. -->
	<div class={['relative isolate', css.container].filter(Boolean).join(' ')}>
		<!-- `fill` is forced last so a stray fill:false on the props can't disable it. -->
		<ImageEffect {...background} fill />
		<div class="relative z-10">
			<AstRenderer nodes={ast} />
		</div>
	</div>
{:else}
	<!-- No background image: render the AST directly (its root element is the container). -->
	{#if css.container}
		<div class={css.container}><AstRenderer nodes={ast} /></div>
	{:else}
		<AstRenderer nodes={ast} />
	{/if}
{/if}
