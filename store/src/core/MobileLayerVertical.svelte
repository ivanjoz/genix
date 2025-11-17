<script lang="ts">
  import { Core, suscribeUrlFlag } from "./store.svelte";

  const { 
    id = false, title = "" 
  } = $props<{ id: number, title: string }>();
  let divContainer: HTMLDivElement

  const show = $derived(Core.openLayers.includes(id))
  
  $effect(() => {
    if(show){
      suscribeUrlFlag("mob-layer", () => { 
        Core.openLayers = Core.openLayers.filter(x => x !== id) 
      })
    }
  })

  let showInner = $state(show)

  $effect(() => {
    if(typeof show !== 'undefined' && typeof document !== 'undefined'){
      divContainer.style.setProperty("view-transition-name", "mobile-layer-vertical")
      setTimeout(() => {
        divContainer.style.setProperty("view-transition-name","")
      },400)

      document.startViewTransition(() => {
        showInner = show
      })
    }
  })

  let css = $derived.by(() => {
    let value = "_1";
    if (showInner) { value += " _2"; }
    return value;
  })
</script>

<div class={css + " top-10 left-0 fixed w-[100vw]"} aria-hidden={!showInner}
  bind:this={divContainer} id="mob-layer"
>
  <div class="flex items-center justify-between absolute w-full top-9 pl-20 pr-4">
    <div class="fs18 text-[white]">{title}</div>
    <button class="_4 fs20 border-none outline-none text-[white] bg-transparent"
      onclick={ev => {
        ev.stopPropagation()
        Core.openLayers = Core.openLayers.filter(x => x !== id)
      }}
    >
      <i class="icon-cancel"></i>
    </button>
  </div>
  <div class="_3 w-full absolute bottom-0">
    <slot />
  </div>
</div>

<style>
  ._1 {
    height: calc(100vh - 10px);
    background-color: #2c2d31;
    border-radius: 16px 16px 0 0;
    overflow: hidden;
    outline: 4px solid #ffffff69;
    opacity: 0;
    pointer-events: none;
    z-index: -1;
    transform: translateY(56vh);
  }

  ._2 {
    opacity: 1;
    pointer-events: auto;
    z-index: 221;
    transform: translateY(0);
  }

  ._3 {
    height: calc(100vh - 55px);
    background-color: #f4f4fa;
    border-radius: 16px 16px 0 0;
    outline: 2px solid #000000c2;
    padding: 8px;
  }

  /* View Transitions for Mobile Layer Vartical*/
  @keyframes slide-up {
    from {
      transform: translateY(56vh);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }

  @keyframes slide-down {
    from {
      transform: translateY(0);
      opacity: 1;
    }
    to {
      transform: translateY(56vh);
      opacity: 0;
    }
  }

  ::view-transition-new(mobile-layer-vertical) {
    animation: slide-up 360ms cubic-bezier(0.32, 0.72, 0, 1) forwards;
  }

  ::view-transition-old(mobile-layer-vertical) {
    animation: slide-down 380ms cubic-bezier(0.4, 0, 0.2, 1) forwards;
  }
</style>