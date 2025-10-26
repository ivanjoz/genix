<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverCloseTriggerProps extends PropsWithElement<'button'>, HTMLAttributes<'button'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverCloseTriggerProps = $props();

	const popover = PopoverRootContext.consume();

	const { element, children, class: className, ...rest } = $derived(props);

	const closeTriggerProps = $derived(popover().getCloseTriggerProps());

	const combinedClass = $derived([
		'popover-close-trigger',
		className
	].filter(Boolean).join(' '));
</script>

{#if element}
	{@render element({ ...closeTriggerProps, class: combinedClass, ...rest })}
{:else}
	<button class={combinedClass} {...closeTriggerProps} {...rest}>
		{@render children?.()}
	</button>
{/if}
