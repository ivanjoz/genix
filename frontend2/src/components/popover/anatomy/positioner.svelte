<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverPositionerProps extends PropsWithElement<'div'>, HTMLAttributes<'div'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverPositionerProps = $props();

	const popover = PopoverRootContext.consume();
	let positionerElement: HTMLElement | null = $state(null);

	const { element, children, class: className, style: customStyle, ...rest } = $derived(props);

	const positionerProps = $derived(popover().getPositionerProps());
	const open = $derived(popover().open);

	const combinedClass = $derived([
		'popover-positioner',
		className
	].filter(Boolean).join(' '));

	// Update the positioner element in the popover context when it changes
	$effect(() => {
		if (positionerElement && popover().setPositionerElement) {
			popover().setPositionerElement(positionerElement);
		}
	});

	// Handle click outside to close
	$effect(() => {
		if (!open) return;

		const handleClickOutside = (event: MouseEvent) => {
			const target = event.target as Node;
			if (
				positionerElement &&
				!positionerElement.contains(target) &&
				popover().triggerElement &&
				!popover().triggerElement.contains(target)
			) {
				popover().onOpenChange({ open: false });
			}
		};

		// Add a small delay to avoid immediate closing when opening
		const timeoutId = setTimeout(() => {
			document.addEventListener('click', handleClickOutside);
		}, 0);

		return () => {
			clearTimeout(timeoutId);
			document.removeEventListener('click', handleClickOutside);
		};
	});

	// Calculate positioning dynamically
	const positionStyle = $derived(() => {
		if (!open) return '';
		
		const pos = popover().calculatePosition();
		const style = `position: fixed; top: ${pos.top}px; left: ${pos.left}px; z-index: 50;`;
		return customStyle ? `${style} ${customStyle}` : style;
	});
</script>

{#if open}
	{#if element}
		{@render element({ ...positionerProps, class: combinedClass, style: positionStyle(), ...rest })}
	{:else}
		<div bind:this={positionerElement} class={combinedClass} style={positionStyle()} {...positionerProps} {...rest}>
			{@render children?.()}
		</div>
	{/if}
{/if}
