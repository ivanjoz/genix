<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverArrowProps extends PropsWithElement<'div'>, HTMLAttributes<'div'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverArrowProps = $props();

	const popover = PopoverRootContext.consume();

	const { element, children, class: className, ...rest } = $derived(props);

	const arrowProps = $derived(popover().getArrowProps());

	const combinedClass = $derived([
		'popover-arrow',
		className
	].filter(Boolean).join(' '));
</script>

{#if element}
	{@render element({ ...arrowProps, class: combinedClass, ...rest })}
{:else}
	<div class={combinedClass} {...arrowProps} {...rest}>
		{@render children?.()}
	</div>
{/if}
