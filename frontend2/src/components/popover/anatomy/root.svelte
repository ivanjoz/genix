<script lang="ts" module>
	import type { PropsWithChildren } from '../modules/props-with-children';

	export interface PopoverRootProps extends PropsWithChildren {
		id?: string;
		open?: boolean;
		defaultOpen?: boolean;
		modal?: boolean;
		portalled?: boolean;
		autoFocus?: boolean;
		initialFocusEl?: (() => HTMLElement | null);
		closeOnInteractOutside?: boolean;
		closeOnEscape?: boolean;
		onOpenChange?: ((details: { open: boolean }) => void);
		positioning?: any;
	}
</script>

<script lang="ts">
	import { usePopover } from '../modules/provider.svelte';
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverRootProps = $props();

	const { children, id, ...popoverProps } = $derived(props);

	const popoverId = $derived(id || `popover-${Math.random().toString(36).substr(2, 9)}`);
	const popover = usePopover(() => ({
		...popoverProps,
		id: popoverId,
	}));

	PopoverRootContext.provide(() => popover());
</script>

{@render children?.()}
