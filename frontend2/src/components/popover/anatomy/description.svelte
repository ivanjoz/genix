<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverDescriptionProps extends PropsWithElement<'p'>, HTMLAttributes<'p'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverDescriptionProps = $props();

	const popover = PopoverRootContext.consume();

	const { element, children, class: className, ...rest } = $derived(props);

	const attributes = $derived(
		popover().getDescriptionProps?.() || {}
	);

	const combinedClass = $derived([
		'popover-description',
		className
	].filter(Boolean).join(' '));
</script>

{#if element}
	{@render element({ ...attributes, class: combinedClass, ...rest })}
{:else}
	<p class={combinedClass} {...attributes} {...rest}>
		{@render children?.()}
	</p>
{/if}
