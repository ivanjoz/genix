<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverTitleProps extends PropsWithElement<'h3'>, HTMLAttributes<'h3'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverTitleProps = $props();

	const popover = PopoverRootContext.consume();

	const { element, children, class: className, ...rest } = $derived(props);

	const attributes = $derived(
		popover().getTitleProps?.() || {}
	);

	const combinedClass = $derived([
		'popover-title',
		className
	].filter(Boolean).join(' '));
</script>

{#if element}
	{@render element({ ...attributes, class: combinedClass, ...rest })}
{:else}
	<h3 class={combinedClass} {...attributes} {...rest}>
		{@render children?.()}
	</h3>
{/if}
