<script lang="ts">
  import { tick } from 'svelte';
import { parseSVG } from '$core/helpers';
import angleSvg from '$core/assets/angle.svg?raw';

  interface Props {
    /** Button text or content */
    buttonText?: string;
    useBig?: boolean;
    buttonClass?: string;
    layerClass?: string;
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
  let wrapperElement: HTMLElement | null = $state(null);
  let buttonElement: HTMLElement | null = $state(null);
  let layerElement: HTMLElement | null = $state(null);
  let position = $state({ top: 0, left: 0 });
  let placement = $state<'top' | 'bottom'>('bottom');
  let angleLeft = $state(20); // Position of the angle from the left of the layer

  // Find the nearest scrollable parent
  function getScrollParent(element: HTMLElement | null): HTMLElement {
    if (!element) return document.documentElement;

    let parent = element.parentElement;
    while (parent) {
      const { overflow, overflowY } = window.getComputedStyle(parent);
      const isScrollable = overflow.includes('auto') || overflow.includes('scroll') || overflowY.includes('auto') || overflowY.includes('scroll');
      if (isScrollable && parent.scrollHeight > parent.clientHeight) {
        return parent;
      }
      parent = parent.parentElement;
    }

    return document.documentElement;
  }

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

  // Calculate position of the layer below or above the button
  async function updatePosition() {
    await tick();
    if (!buttonElement || !layerElement) {
      return;
    }

    const buttonRect = buttonElement.getBoundingClientRect();
    const layerRect = layerElement.getBoundingClientRect();
    const scrollParent = getScrollParent(buttonElement);
    const parentRect = scrollParent === document.documentElement
      ? { top: 0, bottom: window.innerHeight, left: 0, right: window.innerWidth, width: window.innerWidth, height: window.innerHeight }
      : scrollParent.getBoundingClientRect();

    const isMobile = window.innerWidth <= 748;
    const offset = 8; // Distance from button

    // Determine vertical placement: prefer bottom if it fits, else use side with more space
    const spaceBelow = parentRect.bottom - buttonRect.bottom;
    const spaceAbove = buttonRect.top - parentRect.top;
    const requiredHeight = layerRect.height + offset;

    if (spaceBelow < requiredHeight && spaceAbove > spaceBelow) {
      placement = 'top';
    } else {
      placement = 'bottom';
    }

    let top = placement === 'bottom'
      ? buttonRect.bottom + offset
      : buttonRect.top - layerRect.height - offset;

    let left = isMobile ? 6 : buttonRect.left + horizontalOffset;

    // Desktop: position relative to button
    if (!isMobile) {
      // Check if layer would go off right edge of parent/viewport
      if (left + layerRect.width > parentRect.right - edgeMargin) {
        left = parentRect.right - layerRect.width - edgeMargin;
      }

      // Check if layer would go off left edge of parent/viewport
      if (left < parentRect.left + edgeMargin) {
        left = parentRect.left + edgeMargin;
      }
    }

    position = { top, left };

    // Calculate angle position to be centered below/above the button
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
      class:placement-top={placement === 'top'}
      class:placement-bottom={placement === 'bottom'}
    >
      <!-- Angle pointer -->
      <div class="button-layer-angle" style="left: {angleLeft}px;" class:use-big={useBig}>
        <img class="button-layer-angle-img" alt="" src={parseSVG(angleSvg)} class:use-big={useBig}/>
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

  .button-layer.placement-top {
    animation: slideUp 0.2s ease-out;
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

  .button-layer.placement-top .button-layer-angle {
    top: auto;
    bottom: -18px;
    transform: rotate(180deg);
  }

  .button-layer-angle.use-big {
    top: -22px;
	  height: 22px;
	  width: 28px;
  }

  .button-layer.placement-top .button-layer-angle.use-big {
    top: auto;
    bottom: -22px;
  }

  .button-layer-angle-img {
    width: 24px;
    height: 24px;
    filter: drop-shadow(0 -1px 1px rgba(0, 0, 0, 0.05));
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

  @keyframes slideUp {
    from {
      opacity: 0;
      transform: translateY(8px);
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
