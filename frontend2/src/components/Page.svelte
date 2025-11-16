<script lang="ts">
  import { Env } from "$lib/security";
    import { onMount } from "svelte";
    import { Core } from "../core/store.svelte";

  let { children, sideLayerSize, title, options }: {
    children: any, sideLayerSize?: number, title: string
    options?: {id: number, name: string}[]
  } = $props();

  Env.sideLayerSize = sideLayerSize || 0

  onMount(() => {
    Core.pageTitle = title || ""
    Core.pageOptions = options || []
  })

</script>

<div class="_1 p-10">
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