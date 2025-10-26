<script lang="ts" module>
	import type { HTMLAttributes } from '../modules/html-attributes';
	import type { PropsWithElement } from '../modules/props-with-element';

	export interface PopoverContentProps extends PropsWithElement<'div'>, HTMLAttributes<'div'> {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverContentProps = $props();

	const popover = PopoverRootContext.consume();

	const { element, children, class: className, ...rest } = $derived(props);

	const contentProps = $derived(popover().getContentProps());
	const open = $derived(popover().open);

	const combinedClass = $derived([
		'popover-content',
		className
	].filter(Boolean).join(' '));
</script>

{#if open}
	{#if element}
		{@render element({ ...contentProps, class: combinedClass, ...rest })}
	{:else}
		<div class={combinedClass} {...contentProps} {...rest}>
			{@render children?.()}
		</div>
	{/if}
{/if}
