<script lang="ts">
	interface Props {
		children?: import('svelte').Snippet;
		target?: HTMLElement;
		zIndex?: number
	}
	
	let { children, target = undefined, zIndex }: Props = $props();
	
	let contentElement: HTMLDivElement | null = $state(null);
	let portalTarget: HTMLElement | null = $state(null);
	
	// Move the content element to the body when it's ready
	$effect(() => {
		if (!contentElement) return;
		
		const targetElement = target || document.body;
		portalTarget = targetElement;
		
		// Move the element to the target (body)
		targetElement.appendChild(contentElement);
		
		// Cleanup: move back or remove when component unmounts
		return () => {
			if (contentElement && contentElement.parentNode) {
				contentElement.parentNode.removeChild(contentElement);
			}
		};
	});
</script>

<!-- Render content normally, then move it to body via the effect above -->
<div bind:this={contentElement} class="_1 absolute top-0"
	style:z-index={zIndex ? zIndex : undefined}
>
	{@render children?.()}
</div>

<style>
	._1 { 
		z-index: var(--portal-zindex);
	}
</style>
