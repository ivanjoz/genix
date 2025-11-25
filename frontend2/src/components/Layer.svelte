<script lang="ts">
  import { Env } from "$lib/security";
  import { tick } from "svelte";
  import { Core } from "../core/store.svelte";
  import OptionsStrip from "./micro/OptionsStrip.svelte";
  
  // svelte-ignore non_reactive_update
  let divLayer: HTMLDivElement
  // Animation duration in milliseconds - should match CSS animation duration
  const ANIMATION_DURATION = 350;

  const { 
    children, css, title, titleCss, contentCss, id,
    type, options, selected, onSelect, onSave, onDelete, onClose, saveButtonName
  }: {
    children: any, css?: string, title?: string, titleCss?: string,
    options?: [number, string, string[]?][], contentCss?: string
    selected?: number, onSelect?: (e: any) => (void | undefined),
    onSave?: () => void
    onDelete?: () => void
    onClose?: () => void
    saveButtonName?: string
    type: "side" | "content",
    id?: number
  } = $props();

  const layerWidth = $derived.by(() => {
    return Env.sideLayerSize + "px"
  })

  const contentWidth = $derived.by(() => {
    return Core.showSideLayer > 0 && Core.deviceType !== 3
      ? `calc(var(--page-width) - ${layerWidth} - 8px)` 
      : undefined
  })

  // Helper function to set layer with view transition on mobile
  const setSideLayerWithTransition = (layerId: number) => {
    if (Core.deviceType === 3 && document.startViewTransition) {
      // We need to wait for the next tick to ensure the element is available
      tick().then(() => {
        /*
        const targetDiv = document.querySelector(`[data-layer-id="${layerId}"]`) as HTMLDivElement;
        const currentDiv = divLayer;
        */
        // Set view-transition-name on both current and target elements
        if (divLayer) {
          divLayer.style.setProperty("view-transition-name", "mobile-side-layer");
        }
        /*
        if (targetDiv) {
          targetDiv.style.setProperty("view-transition-name", "mobile-side-layer");
        }
        */
        setTimeout(() => {
          if (divLayer) {
            divLayer.style.setProperty("view-transition-name", "");
          }
          /*
          if (targetDiv) {
            targetDiv.style.setProperty("view-transition-name", "");
          }
            */
        }, ANIMATION_DURATION);

        document.startViewTransition(() => {
          Core.showSideLayer = layerId;
        });
      });
    } else {
      Core.showSideLayer = layerId;
    }
  }

  // Helper function to close layer with view transition on mobile
  const closeLayer = () => {
    setSideLayerWithTransition(0);
  }

  // Set view-transition-name when layer is shown on mobile
  $effect(() => {
    if(type !== 'side' || Core.deviceType !== 3){ return }
    
    if (Core.showSideLayer === id && divLayer) {
      divLayer.style.setProperty("view-transition-name", "mobile-side-layer");
      
      setTimeout(() => {
        if (divLayer) {
          divLayer.style.setProperty("view-transition-name", "");
        }
      }, ANIMATION_DURATION);
    }
  })

  // Register the setSideLayer function in the Core store
  $effect(() => {
    Core.setSideLayer = setSideLayerWithTransition;
  })

  console.log("Env.sideLayerSize",  Env.sideLayerSize)
</script>

{#if Core.showSideLayer === id && type == 'side'}
  <div class="_1 flex flex-col {css||""}" bind:this={divLayer}
    data-layer-id={id}
    style="width: {layerWidth};"
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
            <i class="icon-floppy"></i>
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
        onSelect={e => { onSelect?.(e) }} css="mt-2"
      />
    {/if}
    <div class="_2 grow-1 {contentCss}">{@render children()}</div>
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
    overflow-y: auto;
    overflow-x: hidden;
    width: calc(100% + 6px);
    margin-right: -6px;
    padding-right: 6px;
  }

  @media (max-width: 749px) {
    ._2 {
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