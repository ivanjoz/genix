<script lang="ts">
  import { checkIsLogin } from "$core/security.ts";
  import { onDestroy, onMount } from "svelte";
  import { closeAllModals, Core, openModal } from "$core/store.svelte";
    import { Env } from "$core/env.ts";

  let { children, sideLayerSize, title, options }: {
    children: any, sideLayerSize?: number, title: string
    options?: {id: number, name: string}[]
  } = $props();

  Env.sideLayerSize = sideLayerSize || 0
  const isLogged = $derived(checkIsLogin() === 2)
  $effect(() => {
    console.log("isLogged 22", isLogged)
  })

  onMount(() => {
    console.log("isLogged 11", isLogged)
    if(!isLogged){
      Env.navigate("/login")
    } else {
      Core.pageTitle = title || ""
      Core.pageOptions = options || []
    }
  })

  onDestroy(() => {
    Core.openSideLayer(0)
    closeAllModals()
  })

</script>

<div class="_1 p-10">
  {#if Core.isLoading === 0 && isLogged}
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

  @media (max-width: 750px) {
    ._1 {
      margin-left: 0;
      width: 100%;
      max-width: 100vw;
      overflow-x: hidden;
    }
  }

</style>
