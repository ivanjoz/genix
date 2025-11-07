<script lang="ts">
  import { Env } from "$lib/security";

  let { children, sideLayerSize }: {
    children: any, sideLayerSize?: number,
    options?: {id: number, name: string}[]
  } = $props();

  Env.sideLayerSize = sideLayerSize || 0
</script>

<div class="_1 p-8">
  {@render children()}
</div>

<style>
  ._1 {
    margin-top: var(--header-height);
    margin-left: var(--menu-min-width);
    width: calc(100% - var(--menu-min-width));
    min-height: calc(100vh - var(--header-height) - 4px);
  }
</style>