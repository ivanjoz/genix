<script lang="ts">
  import { tick } from 'svelte';
import { parseSVG } from '$core/helpers';
import angleSvg from '$core/assets/angle.svg?raw';

  interface Props {
    /** Button text or content */
    buttonText?: string;
    /** Custom class for the outer wrapper */
    wrapperClass?: string;
    /** Horizontal offset added to the calculated left position */
    horizontalOffset?: number;
    /** Minimum margin from the screen edges */
    edgeMargin?: number;
    /** Whether the layer is open */
    isOpen?: boolean;
    /** Whether the layer is open by default */
    defaultOpen?: boolean;
    /** Children content to render in the layer */
    children?: import('svelte').Snippet;
    /** Optional custom button snippet */
    button?: import('svelte').Snippet<[boolean]>;
    /** Callback when layer opens */
    onOpen?: () => void;
    /** Callback when layer closes */
    onClose?: () => void;
    /** css of the content */
    contentCss?: string;
    useBig?: boolean;
  }

  let {
    buttonText = 'Open',
    wrapperClass = '',
    buttonClass = '',
    layerClass = '',
    horizontalOffset = 0,
    edgeMargin = 10,
    isOpen = $bindable(false),
    defaultOpen = false,
    contentCss = '',
    children,
    button,
    onOpen,
    onClose,
    useBig
  }: Props = $props();

  // svelte-ignore state_referenced_locally
  if (defaultOpen) isOpen = true;
  let buttonElement: HTMLElement | null = $state(null);
  let layerElement: HTMLElement | null = $state(null);
  let position = $state({ top: 0, left: 0 });
  let angleLeft = $state(20); // Position of the angle from the left of the layer

  // Toggle the layer open/closed
  function toggleLayer() {
    isOpen = !isOpen;
    if (isOpen) {
      onOpen?.();
    } else {
      onClose?.();
    }
  }

  // Close the layer
  function closeLayer() {
    if (isOpen) {
      isOpen = false;
      onClose?.();
    }
  }

  // Calculate position of the layer below the button
  async function updatePosition() {
    if (!buttonElement || !layerElement) return;

    await tick();

    const buttonRect = buttonElement.getBoundingClientRect();
    const layerRect = layerElement.getBoundingClientRect();
    const isMobile = window.innerWidth <= 748;

    const offset = 8; // Distance from button
    let top = buttonRect.bottom + offset;
    let left = isMobile ? 6 : buttonRect.left + horizontalOffset;

    // Desktop: position relative to button
    if (!isMobile) {
      // Check if layer would go off right edge of viewport
      if (left + layerRect.width > window.innerWidth) {
        left = window.innerWidth - layerRect.width - edgeMargin;
      }

      // Check if layer would go off left edge of viewport
      if (left < edgeMargin) {
        left = edgeMargin;
      }
    }

    // Check if layer would go off bottom of viewport
    if (top + layerRect.height > window.innerHeight) {
      top = buttonRect.top - layerRect.height - offset;
    }

    position = { top, left };

    // Calculate angle position to be centered below the button
    const buttonCenter = buttonRect.left + (buttonRect.width / 2);

    if (isMobile) {
      // On mobile, center the angle relative to the button within the layer
      // Layer is at left = 6px, so calculate relative to that
      angleLeft = buttonCenter - left - 12; // 12 is half the angle width (24px / 2)
      // Ensure angle stays within layer bounds (layer width is calc(100vw - 12px))
      const layerWidth = window.innerWidth - 12;
      const minAngleLeft = 10;
      const maxAngleLeft = layerWidth - 24;
      if (angleLeft < minAngleLeft) angleLeft = minAngleLeft;
      if (angleLeft > maxAngleLeft) angleLeft = maxAngleLeft;
    } else {
      // Desktop: position relative to layer
      angleLeft = buttonCenter - left - 12;
      // Ensure angle stays within layer bounds
      const minAngleLeft = 10;
      const maxAngleLeft = layerRect.width - 24;
      if (angleLeft < minAngleLeft) angleLeft = minAngleLeft;
      if (angleLeft > maxAngleLeft) angleLeft = maxAngleLeft;
    }
  }

  // Update position when layer opens
  $effect(() => {
    if (isOpen && buttonElement && layerElement) {
      updatePosition();
    }
  });

  // Handle clicks outside the layer
  $effect(() => {
    if (!isOpen) return;

    function handleClickOutside(event: MouseEvent) {
      const target = event.target as Node;

      // Check if click is outside both button and layer
      if (
        buttonElement && !buttonElement.contains(target) &&
        layerElement && !layerElement.contains(target)
      ) {
        closeLayer();
      }
    }

    // Use mousedown for better UX (triggers before click)
    document.body.addEventListener('mousedown', handleClickOutside);

    return () => {
      document.body.removeEventListener('mousedown', handleClickOutside);
    };
  });

  // Recalculate position on scroll and resize
  $effect(() => {
    if (!isOpen) return;

    const handleUpdate = () => {
      if (isOpen && buttonElement && layerElement) {
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
</script>

<div class="button-layer-wrapper {wrapperClass}">
  {#if button}
    <div
      bind:this={buttonElement}
      onclick={toggleLayer}
      class="button-layer-trigger-wrapper {buttonClass}"
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && toggleLayer()}
    >
      {@render button(isOpen)}
    </div>
  {:else}
    <button
      bind:this={buttonElement}
      onclick={toggleLayer}
      class="button-layer-trigger {buttonClass}"
      type="button"
    >
      {buttonText}
    </button>
  {/if}

  {#if isOpen}
    <div bind:this={layerElement}
      class="button-layer min-w-200 {layerClass||'w-[calc(100vw-12px)]'}"
      style="top: {position.top}px; left: {position.left}px;"
      class:use-big={useBig}
    >
      <!-- Angle pointer -->
      <div class="button-layer-angle" style="left: {angleLeft}px;">
        <img class="button-layer-angle-img" alt="" src={parseSVG(angleSvg)}/>
      </div>

      <!-- Layer content -->
      <div class="button-layer-content-wrapper {contentCss}">
        {@render children?.()}
      </div>
    </div>
  {/if}
</div>

<style>
  .button-layer-wrapper {
    position: relative;
    display: inline-block;
  }

  .button-layer-trigger {
    padding: 8px 16px;
    background-color: #6d5dad;
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    transition: background-color 0.2s;
  }

  .button-layer-trigger:hover {
    background-color: #5a4a94;
  }

  .button-layer {
    position: fixed;
    z-index: 210;
    background-color: white;
    border-radius: 8px;
    box-shadow:
      0 4px 6px -1px rgba(0, 0, 0, 0.1),
      0 2px 4px -1px rgba(0, 0, 0, 0.06),
      0 0 0 1px rgba(0, 0, 0, 0.05);
    animation: slideDown 0.2s ease-out;
    overflow: visible;
  }

  .button-layer.use-big {
    box-shadow: #46466059 0 2px 18px -2px, #00000059 0 0 6px;
  }

  /* Mobile: fixed width and max-height */
  @media (max-width: 748px) {
    .button-layer {
      max-width: calc(100vw - 12px);
      max-height: calc(100vh - 100px);
      overflow: visible;
    }

    /* Make content scrollable while keeping angle visible */
    .button-layer-content-wrapper {
      overflow-y: auto;
      max-height: calc(100vh - 100px);
    }
  }

  .button-layer-angle {
    position: absolute;
    top: -18px;
    overflow: hidden;
    height: 18px;
    width: 24px;
    display: flex;
    justify-content: center;
    z-index: 211;
  }

  .button-layer-angle-img {
    width: 24px;
    height: 24px;
    filter: drop-shadow(0 -1px 1px rgba(0, 0, 0, 0.05));
  }

  .button-layer-angle-img.use-big {
    width: 28px;
    height: 28px;
  }

  .button-layer-angle.use-big {
	  height: 22px;
	  width: 28px;
  }

  @keyframes slideDown {
    from {
      opacity: 0;
      transform: translateY(-8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  :global(.dark) .button-layer {
    background-color: #2d2d3a;
    box-shadow:
      0 4px 6px -1px rgba(0, 0, 0, 0.3),
      0 2px 4px -1px rgba(0, 0, 0, 0.2),
      0 0 0 1px rgba(255, 255, 255, 0.1);
  }
</style>
