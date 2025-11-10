<script lang="ts">
  import { Env } from "$lib/security";
  import { Core } from "../core/store.svelte";
  import OptionsStrip from "./micro/OptionsStrip.svelte";
  
  const { 
    children, css, title, titleCss, 
    type, options, selected, onSelect, onSave, onDelete, saveButtonName
  }: {
    children: any, css?: string, title?: string, titleCss?: string,
    options?: [number, string][],
    selected?: number, onSelect?: (e: any) => (void | undefined),
    onSave?: () => void
    onDelete?: () => void
    saveButtonName?: string
    type: "side" | "content"
  } = $props();

  const layerWidth = $derived.by(() => {
    return Env.sideLayerSize + "px"
  })

  const contentWidth = $derived.by(() => {
    return Core.showSideLayer > 0 
      ? `calc(var(--page-width) - ${layerWidth} - 8px)` 
      : undefined
  })

  console.log("Env.sideLayerSize",  Env.sideLayerSize)
</script>

{#if Core.showSideLayer > 0 && type == 'side'}
  <div class="_1 {css||""}" style="width: {layerWidth};">
    <div class="flex items-center justify-between">
      <div class={titleCss}>{title}</div>
      <div class="items-center">
        {#if onDelete}
          <button class="bx-red mr-8 lh-10" onclick={onDelete} aria-label="Eliminar">
            <i class="icon-trash"></i>
          </button>
        {/if}
        {#if onSave}
          <button class="bx-blue mr-8 lh-10" onclick={onSave} aria-label="Guardar">
            <i class="icon-floppy"></i>
            <span>{saveButtonName}</span>
          </button>
        {/if}
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
    {#if (options||[]).length > 0}
      <OptionsStrip options={options as [number, string][]} 
        selected={selected as number}
        onSelect={e => { onSelect?.(e) }}
      />
    {/if}
    <div>{@render children()}</div>
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
    box-shadow: -3px 1px 5px #0000001a;
    z-index: 101;
  }
</style>