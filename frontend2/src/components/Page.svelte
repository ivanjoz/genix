<script lang="ts">
  import { Env } from "$lib/security";
    import { Core } from "../core/store.svelte";

  let { children, sideLayerSize }: {
    children: any, sideLayerSize?: number,
    options?: {id: number, name: string}[]
  } = $props();

  Env.sideLayerSize = sideLayerSize || 0
</script>

<div class="_1 p-8">
  {#if Core.isLoading === 0}
    {@render children()}
  {/if}
  {#if Core.isLoading > 0}
    <div class="p-16"><h2>Cargando...</h2></div>
  {/if}
</div>

<style>
  ._1 {
    margin-top: var(--header-height);
    margin-left: var(--menu-min-width);
    width: calc(100% - var(--menu-min-width));
    min-height: calc(100vh - var(--header-height) - 4px);
  }
</style>