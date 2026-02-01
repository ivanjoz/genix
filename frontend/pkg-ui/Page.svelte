<script lang="ts">
import { checkIsLogin } from '$core/lib/security';
  import { onDestroy, onMount } from "svelte";
import { closeAllModals, Core, openModal } from '$core/store.svelte';
import { Env } from '$core/env';

  let { children, sideLayerSize, title, options, containerCss, useTopMinimalMenu, fixedFullHeight }: {
    children: any, 
    sideLayerSize?: number, 
    title: string, 
    containerCss?: string,
    useTopMinimalMenu?: boolean,
    fixedFullHeight?: boolean
    options?: {id: number, name: string}[]
  } = $props();

  $effect(() => {
    Env.sideLayerSize = sideLayerSize || 0
    Env.useTopMinimalMenu = useTopMinimalMenu || false
    Core.useTopMinimalMenu = useTopMinimalMenu || false
  })
  const isLogged = $derived(checkIsLogin() === 2)
  $effect(() => {
    console.log("isLogged 22", isLogged)
  })

  onMount(() => {
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

<div class="_1 p-10 {containerCss}" class:useTopMinimalMenu={Core.useTopMinimalMenu}
	class:fixed-full-height={fixedFullHeight}
>
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
    position: relative;
  }

  ._1.useTopMinimalMenu {
    margin-left: 0;
    width: 100%;
  }
  
  .fixed-full-height {
	 	height: calc(100vh - var(--header-height));
	  overflow: auto;
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
