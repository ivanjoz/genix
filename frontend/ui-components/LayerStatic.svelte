<script lang="ts">
  import { Core } from '$core/store.svelte'
  import MobileLayerVertical from '$components/MobileLayerVertical.svelte'

  interface Props {
    css?: string
    show?: boolean
    mobileLayerTitle?: string
    useMobileLayerVertical?: number
    children?: import('svelte').Snippet
  }

  let {
    css,
    show = false,
    mobileLayerTitle = '',
    useMobileLayerVertical,
    children,
  }: Props = $props()

  // Desktop keeps the static layer visible. Mobile turns it into an explicit off-canvas panel.
  const isOpen = $derived(Core.deviceType !== 3 || show)
  const useVerticalMobileLayer = $derived(
    Core.deviceType === 3 && typeof useMobileLayerVertical === 'number',
  )

  // Mobile keeps the drawer collapsed until the user expands it from the title bar.
  let mobileLayerIsOpen = $state(false)
</script>

{#if useVerticalMobileLayer}
  <MobileLayerVertical
    title={mobileLayerTitle}
    show={mobileLayerIsOpen}
    closedHeightPx={useMobileLayerVertical}
    onToggle={(nextState) => {
      mobileLayerIsOpen = nextState
    }}
  >
    {@render children?.()}
  </MobileLayerVertical>
{:else}
  <div
    class="layer-static {css || ''}"
    class:layer-static-open={isOpen}
    aria-hidden={!isOpen}
  >
    {@render children?.()}
  </div>
{/if}

<style>
  @media (max-width: 749px) {
    .layer-static {
      position: fixed;
      width: 100vw;
      max-width: 100vw;
      transform: translateX(100%);
      opacity: 0;
      visibility: hidden;
      pointer-events: none;
      transition:
        transform 0.3s ease,
        opacity 0.3s ease,
        visibility 0.3s ease;
      z-index: var(--layer-zindex);
    }

    .layer-static.layer-static-open {
      transform: translateX(0);
      opacity: 1;
      visibility: visible;
      pointer-events: auto;
    }
  }
</style>
