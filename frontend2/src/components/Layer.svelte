<script lang="ts">
  import { Env } from "$lib/security";
  import { tick } from "svelte";
  import { Core } from "../core/store.svelte";
  import OptionsStrip from "./micro/OptionsStrip.svelte";
  
  // svelte-ignore non_reactive_update
  let divLayer: HTMLDivElement
  // Animation duration in milliseconds - should match CSS animation duration
  const ANIMATION_DURATION = 350;
  // Track previous showSideLayer value to detect changes
  let previousShowSideLayer = $state(Core.showSideLayer);

  let { 
    children, css, title, titleCss, contentCss, id,
    type, options, 
    selected = $bindable(), 
   /* onSelect, */ onSave, onDelete, onClose, 
    saveButtonName, saveButtonIcon, contentOverflow
  }: {
    children: any, css?: string, title?: string, titleCss?: string,
    options?: [number, string, string[]?][], contentCss?: string
    selected?: number, 
    // onSelect?: (e: any) => (void | undefined),
    onSave?: () => void
    onDelete?: () => void
    onClose?: () => void
    saveButtonName?: string
    saveButtonIcon?: string
    contentOverflow?: boolean
    type: "side" | "bottom" | "content",
    id?: number
  } = $props();

  const layerWidth = $derived.by(() => {
    return Env.sideLayerSize ? Env.sideLayerSize + "px" : ""
  })

  const contentWidth = $derived.by(() => {
    return Core.showSideLayer > 0 && Core.deviceType !== 3
      ? `calc(var(--page-width) - ${layerWidth||"0"} - 8px)` 
      : undefined
  })

  // Helper function to close layer with view transition on mobile
  const closeLayer = () => {
    Core.openSideLayer(0);
  }

  // React to showSideLayer changes and handle transitions on mobile
  $effect(() => {
    if (type !== 'side' || Core.deviceType !== 3) {
      previousShowSideLayer = Core.showSideLayer;
      return;
    }
    
    // Detect if showSideLayer changed
    if (previousShowSideLayer !== Core.showSideLayer) {
      const isOpening = Core.showSideLayer === id;
      const isClosing = previousShowSideLayer === id && Core.showSideLayer !== id;
      
      if ((isOpening || isClosing) && document.startViewTransition && divLayer) {
        // Set view-transition-name for the animation
        divLayer.style.setProperty("view-transition-name", "mobile-side-layer");
        
        // Start the view transition
        document.startViewTransition(() => {
          // The actual DOM update happens here
          tick();
        });
        
        // Clean up view-transition-name after animation completes
        setTimeout(() => {
          if (divLayer) {
            divLayer.style.setProperty("view-transition-name", "");
          }
        }, ANIMATION_DURATION);
      }
      
      previousShowSideLayer = Core.showSideLayer;
    }
  })

  $effect(() => {
    if(Core.showSideLayer || type){
      console.log("updated 122:", Core.showSideLayer, type,"| open:", Core.showSideLayer === id)
    }
  })

</script>

{#if Core.showSideLayer === id && (type === 'side'|| type === 'bottom')}
  <div class="flex flex-col w-800 {css||""}" bind:this={divLayer}
    data-layer-id={id}
    class:_8={contentOverflow}
    class:_1={type === 'side'}
    class:_2={type === 'bottom'}
    style={layerWidth ? `width: ${layerWidth};` : ""}
  >
    <div class="flex items-center justify-between">
      <div class="overflow-hidden text-nowrap mr-8 {titleCss}">{title}</div>
      <div class="shrink-0 flex items-center">
        {#if onDelete}
          <button class="bx-red mr-8 lh-10" onclick={onDelete} aria-label="Eliminar">
            <i class="icon-trash"></i>
          </button>
        {/if}
        {#if onSave}
          <button class="bx-blue mr-8 lh-10" onclick={onSave} aria-label="Guardar">
            <i class="{saveButtonIcon || 'icon-floppy'}"></i>
            <span>{saveButtonName || "Guardar"}</span>
          </button>
        {/if}
        <button class="bx-yellow" title="close"
          onclick={ev => {
            ev.stopPropagation()
            closeLayer()
            if(onClose){
              if(Core.deviceType === 3){
                setTimeout(() => { onClose() },300)
              } else {
                onClose()
              }
            }
          }}
        >
          <i class="icon-cancel"></i>
        </button>
      </div>
    </div>
    {#if (options||[]).length > 0}
      <OptionsStrip options={options as [number, string][]} 
        selected={selected as number} useMobileGrid={true}
        onSelect={e => { selected = e[0] }} 
        css="mt-2"
      />
    {/if}
    <div class="_4 grow-1 {contentCss}">{@render children()}</div>
  </div>
{/if}

{#if type == 'content'}
  <div class="w-page" style:width={contentWidth}>
    {@render children()}
  </div>
{/if}

<style>
  ._1 {
    position: fixed;
    top: var(--header-height);
    width: 800px;
    height: calc(100vh - var(--header-height));
    right: 0;
    background-color: rgb(255, 255, 255);
    box-shadow: -7px 0px 15px 0px #00000024, -3px 1px 5px 0px #00000017;
    z-index: 101;
    max-width: 100vw;
    overflow: hidden;
  }
  ._2 {
    position: fixed;
    bottom: 0;
    height: auto;
    width: 800px;
    right: 0;
    background-color: rgb(255, 255, 255);
    box-shadow: -7px 0px 15px 0px #00000024, -3px 1px 5px 0px #00000017;
    z-index: 101;
    max-width: 100vw;
    overflow: hidden;
  }
  ._4 {
    overflow-y: auto;
    overflow-x: hidden;
    width: calc(100% + 6px);
    margin-right: -6px;
    padding-right: 6px;
  }
  ._1._8 ._4, ._1._8, ._2._8 ._4, ._2._8 {
    overflow-y: visible;
    overflow-x: visible;
  }

  @media (max-width: 749px) {
    ._4 {
      width: calc(100% + 12px);
      margin-left: -6px;
      margin-right: -6px;
      padding: 0 6px;
    }
    ._1 {
      box-shadow: -7px 0px 15px 8px #00000030, -3px 1px 5px 2px #0000001c;
    }
  }

  /* View Transitions for Mobile Layer */
	@keyframes slide-out {
		from {
			transform: translateX(0);
			opacity: 1;
		}
		to {
			transform: translateX(100%);
			opacity: 1;
		}
	}

	@keyframes slide-in {
		from {
			transform: translateX(100%);
		}
		to {
			transform: translateX(0);
		}
	}

	/* When closing: OLD snapshot slides out to the left */
	::view-transition-old(mobile-side-layer) {
		animation: slide-out 0.35s ease-in-out forwards;
		animation-fill-mode: forwards;
	}

	/* When opening: NEW snapshot slides in from the left */
	::view-transition-new(mobile-side-layer) {
		animation: slide-in 0.35s ease-in-out;
	}

	/* Prevent the default fade out on OLD snapshot */
	::view-transition-image-pair(mobile-side-layer) {
		isolation: auto;
	}
</style>