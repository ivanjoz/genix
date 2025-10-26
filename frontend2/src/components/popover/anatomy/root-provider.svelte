<script lang="ts" module>
	import type { PropsWithChildren } from '../modules/props-with-children';
	import type { Props } from '@zag-js/popover';

	export interface PopoverRootProviderProps extends Props, PropsWithChildren {}
</script>

<script lang="ts">
	import { usePopover } from '../modules/provider.svelte';
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverRootProviderProps = $props();

	const { children, ...popoverProps } = $derived(props);

	const id = $props.id();
	const popover = usePopover(() => ({
		...popoverProps,
		id: id,
	}));

	PopoverRootContext.provide(() => popover());
</script>

{@render children?.()}
