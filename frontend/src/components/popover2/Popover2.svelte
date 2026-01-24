<script lang="ts">
	import { tick } from 'svelte';
	import Portal from './Portal.svelte';
	import { calculatePosition, type Placement } from './positioning';
  import { parseSVG } from '../../core/helpers';
	import angleSvg from "../../assets/angle.svg?raw"

	interface Props {
		/** The reference element to position relative to */
		referenceElement: HTMLElement | null;
		/** Whether the popover is visible */
		open?: boolean;
		/** Preferred placement (will auto-adjust based on space) */
		placement?: Placement;
		/** Offset distance from the reference element in pixels */
		offset?: number;
		/** Whether to constrain to viewport boundaries */
		fitViewport?: boolean;
		/** Custom class for the popover container */
		class?: string;
		/** Custom styles for the popover container */
		style?: string;
		/** Content to render inside the popover */
		children?: import('svelte').Snippet;
		/** Callback when position is calculated */
		onPositionUpdate?: (position: { top: number; left: number; placement: Placement }) => void;
	}
	
	let {
		referenceElement,
		open = false,
		placement = 'bottom',
		offset = 8,
		fitViewport = true,
		class: className = '',
		style: customStyle = '',
		children,
		onPositionUpdate
	}: Props = $props();
	
	let floatingElement: HTMLElement | null = $state(null);
	let position = $state({ top: 0, left: 0, placement: placement as Placement });
	
	// Update position when open changes or elements are ready
	$effect(() => {
		if (open && referenceElement && floatingElement) {
      console.log("referenceElement", referenceElement)
			updatePosition();
		}
	});
	
	// Recalculate position on scroll and resize using $effect
	$effect(() => {
		if (!open) return;
		
		const handleUpdate = () => {
			if (open && referenceElement && floatingElement) {
				updatePosition();
			}
		};
		
		window.addEventListener('scroll', handleUpdate, true);
		window.addEventListener('resize', handleUpdate);
		
		return () => {
			window.removeEventListener('scroll', handleUpdate, true);
			window.removeEventListener('resize', handleUpdate);
		};
	});
	
	async function updatePosition() {
		if (!referenceElement || !floatingElement) return;
		
		// Wait for next tick to ensure element is rendered
		await tick();
		
		const result = calculatePosition(referenceElement, floatingElement, {
			offset,
			preferredPlacement: placement,
			fitViewport,
		});
		
		position = result;
		
		if (onPositionUpdate) {
			onPositionUpdate(result);
		}
	}
	
	const computedStyle = $derived(() => {
		if (!open) return 'display: none;';
		
		return `
			position: absolute;
			top: ${position.top}px;
			left: ${position.left}px;
			z-index: 210;
			${customStyle}
		`.trim();
	});
</script>

{#if open}
	<Portal>
		<div bind:this={floatingElement}
			class="{className} ps-{position.placement}"
			style={computedStyle()}
		>
			{@render children?.()}
			<div class={[
					"absolute overflow-hidden h-18 flex justify-center w-full",
					position.placement === "bottom" && "top-[-18px]",
					position.placement === "top" && "bottom-[-18px] rotate-180",
				]}
			>
				<img class={`_5 w-24 h-24 top`} alt="" src={parseSVG(angleSvg)}/>
			</div>
		</div>
	</Portal>
{/if}

