<script lang="ts">
  import { tick } from 'svelte'
  import { Core } from "$core/store.svelte"
  import OptionsStrip from './micro/OptionsStrip.svelte'
    import { Env } from '$core/env.ts';

  // svelte-ignore non_reactive_update
  let divLayer: HTMLDivElement
  // Track previous showSideLayer value to detect changes
  let previousShowSideLayer = $state(Core.showSideLayer)
  // Track if we're in an opening transition
  let isInTransition = $state(false)

  let {
    children,
    css,
    title,
    titleCss,
    contentCss,
    id,
    type,
    options,
    selected = $bindable(),
    onSave,
    onDelete,
    onClose,
    saveButtonName,
    saveButtonIcon,
    contentOverflow,
  }: {
    children: any
    css?: string
    title?: string
    titleCss?: string
    options?: [number, string, string[]?][]
    contentCss?: string
    selected?: number
    // onSelect?: (e: any) => (void | undefined),
    onSave?: () => void
    onDelete?: () => void
    onClose?: () => void
    saveButtonName?: string
    saveButtonIcon?: string
    contentOverflow?: boolean
    type: 'side' | 'bottom' | 'content'
    id?: number
  } = $props()

  // Track visibility to prevent flash before transition starts
  let isVisible = $state(Core.deviceType !== 3 || type !== 'side')

  const layerWidth = $derived.by(() => {
    return Env.sideLayerSize ? Env.sideLayerSize + 'px' : ''
  })

  const contentWidth = $derived.by(() => {
    return Core.showSideLayer > 0 && Core.deviceType !== 3
      ? `calc(var(--page-width) - ${layerWidth || '0'} - 8px)`
      : undefined
  })

  // Helper function to close layer with view transition on mobile
  const closeLayer = () => {
    Core.openSideLayer(0)
  }

  // React to showSideLayer changes and handle transitions on mobile
  // ONLY handle opening transitions - closing is instant
  $effect(() => {
    // Desktop or non-side layers: visibility is instant
    if (type !== 'side' || Core.deviceType !== 3) {
      isVisible = Core.showSideLayer === id
      previousShowSideLayer = Core.showSideLayer
      return
    }

    // Detect if showSideLayer changed
    if (previousShowSideLayer !== Core.showSideLayer) {
      const isOpening = Core.showSideLayer === id

      // Only handle opening transitions
      if (isOpening && typeof document.startViewTransition === 'function') {
        // Clear any previous layer's transition name to prevent conflicts
        if (previousShowSideLayer !== 0) {
          const oldLayer = document.querySelector(
            `[data-layer-id="${previousShowSideLayer}"]`,
          )
          if (oldLayer) {
            ;(oldLayer as HTMLElement).style.setProperty(
              'view-transition-name',
              '',
            )
          }
        }

        // Start the view transition for opening
        const transition = document.startViewTransition(async () => {
          // Mark that we're in transition and make it visible ONLY now
          isInTransition = true
          isVisible = true
          // Wait for Svelte to update the DOM
          await tick()
        })

        // Clean up after animation completes
        transition.finished.finally(() => {
          isInTransition = false
        })
      } else if (isOpening) {
        // Fallback or non-mobile: just show it
        isVisible = true
      } else if (previousShowSideLayer === id) {
        // Closing: hide it
        isVisible = false
      }

      previousShowSideLayer = Core.showSideLayer
    }
  })

  $effect(() => {
    if (Core.showSideLayer || type) {
      console.log(
        'updated 122:',
        Core.showSideLayer,
        type,
        '| open:',
        Core.showSideLayer === id,
      )
    }
  })
</script>

{#if Core.showSideLayer === id && (type === 'side' || type === 'bottom')}
  <div
    class="flex flex-col w-800 {css || ''}"
    bind:this={divLayer}
    data-layer-id={id}
    class:_8={contentOverflow}
    class:_1={type === 'side'}
    class:_2={type === 'bottom'}
    class:in-transition={isInTransition}
    class:visible={isVisible || Core.deviceType !== 3 || type !== 'side'}
    style={layerWidth ? `width: ${layerWidth};` : ''}
  >
    <div class="flex items-center justify-between">
      <div class="overflow-hidden text-nowrap mr-8 {titleCss}">{title}</div>
      <div class="shrink-0 flex items-center mb-2">
        {#if onDelete}
          <button
            class="bx-red mr-10 lh-10"
            onclick={onDelete}
            aria-label="Eliminar"
          >
            <i class="icon-trash"></i>
          </button>
        {/if}
        {#if onSave}
          <button
            class="bx-blue mr-10 lh-10"
            onclick={onSave}
            aria-label="Guardar"
          >
            <i class={saveButtonIcon || 'icon-floppy'}></i>
            <span class="_11">{saveButtonName || 'Guardar'}</span>
          </button>
        {/if}
        <button
          class="bx-yellow"
          title="close"
          onclick={(ev) => {
            ev.stopPropagation()
            closeLayer()
            if (onClose) {
              if (Core.deviceType === 3) {
                setTimeout(() => {
                  onClose()
                }, 300)
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
    {#if (options || []).length > 0}
      <OptionsStrip
        options={options as [number, string][]}
        selected={selected as number}
        useMobileGrid={true}
        onSelect={(e) => {
          selected = e[0]
        }}
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
    box-shadow:
      -7px 0px 15px 0px #00000024,
      -3px 1px 5px 0px #00000017;
    z-index: var(--layer-zindex);
    max-width: 100vw;
    overflow: hidden;
    /* Prevent flash: hidden by default until transition starts or visible class applied */
    opacity: 0;
    visibility: hidden;
  }
  ._2 {
    position: fixed;
    bottom: 0;
    height: auto;
    width: 800px;
    right: 0;
    background-color: rgb(255, 255, 255);
    box-shadow:
      -7px 0px 15px 0px #00000024,
      -3px 1px 5px 0px #00000017;
    z-index: var(--layer-zindex);
    max-width: 100vw;
    overflow: hidden;
    /* Prevent flash */
    opacity: 0;
    visibility: hidden;
  }
  ._1.visible,
  ._2.visible {
    opacity: 1;
    visibility: visible;
  }
  ._1.in-transition,
  ._2.in-transition {
    opacity: 1;
    visibility: visible;
  }
  ._4 {
    overflow-y: auto;
    overflow-x: hidden;
    width: calc(100% + 10px);
    margin-right: -6px;
    padding-right: 6px;
    margin-left: -4px;
    padding-left: 4px;
  }
  ._1._8 ._4,
  ._1._8,
  ._2._8 ._4,
  ._2._8 {
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
      box-shadow:
        -7px 0px 15px 8px #00000030,
        -3px 1px 5px 2px #0000001c;
    }
    /* Set view-transition-name only when in transition */
    ._1.in-transition {
      view-transition-name: mobile-side-layer;
    }
    ._11 {
      display: none;
    }
  }

  /* View Transitions for Mobile Layer - Opening Only */
  @keyframes slide-in {
    from {
      transform: translateX(100%);
    }
    to {
      transform: translateX(0);
    }
  }

  /* Prevent default cross-fade */
  ::view-transition-image-pair(mobile-side-layer) {
    isolation: isolate;
  }

  /* Override default group animation */
  ::view-transition-group(mobile-side-layer) {
    animation-duration: 0.35s;
    animation-timing-function: ease-in-out;
  }

  /* When opening: NEW snapshot slides in from the right (off-screen) */
  ::view-transition-new(mobile-side-layer) {
    animation: slide-in 0.35s ease-in-out forwards;
    z-index: 2;
    opacity: 1;
  }

  /* Hide old snapshot completely when opening (no fade effect) */
  ::view-transition-old(mobile-side-layer) {
    opacity: 0 !important;
    animation: none !important;
    display: none !important;
  }
</style>
