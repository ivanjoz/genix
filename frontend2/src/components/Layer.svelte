<script lang="ts">
  import { Env } from "$lib/security";
    import { Core } from "../core/store.svelte";
  
  let { children, css, title, titleCss, options }: {
    children: any, css?: string, title?: string, titleCss?: string,
    options?: {id: number, name: string}[]
  } = $props();

  let layerWidth = $derived.by(() => {
    return Env.sideLayerSize + "px"
  })

  console.log("Env.sideLayerSize",  Env.sideLayerSize)
</script>

{#if Core.showSideLayer > 0}
  <div class="_1 {css||""}" style="width: {layerWidth};">
    <div class="flex items-center justify-between">
      <div class="">{title}</div>
      <div class="items-center">
        <button class="bx-yellow" title="close"
          onclick={ev => {
            ev.stopPropagation()
            Core.showSideLayer = 0
          }}
        >
          <i class="icon-cancel"></i>
        </button>
      </div>
    </div>
    <div>{@render children()}</div>
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
    box-shadow: -3px 1px 5px #0000001a;
  }
</style>