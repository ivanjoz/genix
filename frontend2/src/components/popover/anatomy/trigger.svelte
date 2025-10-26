<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverTriggerProps extends PropsWithElement<'button'>, HTMLAttributes<'button'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverTriggerProps = $props();

	const popover = PopoverRootContext.consume();
	let triggerElement: HTMLElement | null = $state(null);

	const { element, children, class: className, ...rest } = $derived(props);

	const triggerProps = $derived(popover().getTriggerProps());

	const combinedClass = $derived([
		'popover-trigger',
		className
	].filter(Boolean).join(' '));

	// Update the trigger element in the popover context when it changes
	$effect(() => {
		if (triggerElement && popover().setTriggerElement) {
			popover().setTriggerElement(triggerElement);
		}
	});
</script>

{#if element}
	{@render element({ ...triggerProps, class: combinedClass, ...rest })}
{:else}
	<button bind:this={triggerElement} class={combinedClass} {...triggerProps} {...rest}>
		{@render children?.()}
	</button>
{/if}
