<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverArrowTipProps extends PropsWithElement<'div'>, HTMLAttributes<'div'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverArrowTipProps = $props();

	const popover = PopoverRootContext.consume();

	const { element, children, class: className, ...rest } = $derived(props);

	const arrowTipProps = $derived(popover().getArrowTipProps());

	const combinedClass = $derived([
		'popover-arrow-tip',
		className
	].filter(Boolean).join(' '));
</script>

{#if element}
	{@render element({ ...arrowTipProps, class: combinedClass, ...rest })}
{:else}
	<div class={combinedClass} {...arrowTipProps} {...rest}>
		{@render children?.()}
	</div>
{/if}
