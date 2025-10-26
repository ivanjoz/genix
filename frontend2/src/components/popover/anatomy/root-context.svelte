<script lang="ts" module>
	import type { PropsWithChildren } from '../modules/props-with-children';

	export interface PopoverRootContextProps extends PropsWithChildren {}
</script>

<script lang="ts">
	import { PopoverRootContext } from '../modules/root-context';

	const props: PopoverRootContextProps = $props();

	const { children } = $derived(props);

	const popover = PopoverRootContext.consume();
</script>

{@render children?.()}
