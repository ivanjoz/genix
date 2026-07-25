<script lang="ts">
import { security } from '$libs/ui-runtime.svelte';
import { onDestroy, onMount, untrack } from "svelte";
import { Core } from '$core/store.svelte';
import { browser, Env, LocalStorage } from '$core/env';
import { checksumBase64_6 } from '@genix/ui/utilities';
import { useUI } from '@genix/ui';

  const headerMenuSelectedStorageKey = "headerMenuSelected"

  type TPageOption = {id: number, name: string}

  let { children, title, options, containerCss, useTopMinimalMenu, fixedFullHeight }: {
    children: any,
    title: string,
    containerCss?: string,
    useTopMinimalMenu?: boolean,
    fixedFullHeight?: boolean
    options?: TPageOption[]
  } = $props();
  const ui = useUI()

  const isLogged = $derived(security.checkIsLogin() === 2)
  let hasRestoredPageOptionSelection = $state(false)
  const pageOptions = $derived(options || [])

  const getRouteOptionHash = (pathname: string): string => checksumBase64_6(pathname || "/")
  
  const getSavedHeaderMenuSelections = (): string[] => {
    const rawSelections = LocalStorage.getItem(headerMenuSelectedStorageKey) || ""
    return rawSelections.split(",").filter(x => x)
  }

  const getInitialSelectedOptionID = (pathname: string, pageOptions: TPageOption[]): number => {
    const fallbackOptionID = pageOptions[0]?.id || 1
    const pathnameHash = getRouteOptionHash(pathname)
    // Resolve the current route selection from the compact "hash:id" localStorage payload.
    const persistedSelection = getSavedHeaderMenuSelections().find(x => x.startsWith(pathnameHash + ":")) || ""
    
    const persistedOptionID = Number(persistedSelection.split(":")[1] || 0)
    const hasValidPersistedOption = pageOptions.some((pageOption) => pageOption.id === persistedOptionID)

    return hasValidPersistedOption ? persistedOptionID : fallbackOptionID
  }

  const restorePageOptionSelection = () => {
    if (!browser || !isLogged) { return }

    ui.state.pageOptionSelected = pageOptions.length === 0
      ? 1
      : getInitialSelectedOptionID(Env.getPathname(), pageOptions)

    hasRestoredPageOptionSelection = true
  }

  // Restore the selected page option before the first render so inactive tab components never mount.
  restorePageOptionSelection()

  $effect(() => {
    untrack(() => {
      Env.useTopMinimalMenu = useTopMinimalMenu || false
      ui.state.useTopMinimalMenu = useTopMinimalMenu || false
      ui.state.pageTitle = title || ""
      ui.state.pageOptions = pageOptions
    })
  })

  $effect(() => {
    if (!browser || !isLogged || pageOptions.length === 0 || !hasRestoredPageOptionSelection) { return }

    if (!pageOptions.some((pageOption) => pageOption.id === ui.state.pageOptionSelected)) {
      untrack(() => {
        ui.state.pageOptionSelected = pageOptions[0].id
      })
      return
    }

    const pathnameHash = getRouteOptionHash(Env.getPathname())  
    const hashSelected = `${pathnameHash}:${ui.state.pageOptionSelected}`
    
    const selectionsSaved = getSavedHeaderMenuSelections()
    const idx = selectionsSaved.findIndex(x => x.startsWith(pathnameHash + ":"))
    idx === -1 ? selectionsSaved.push(hashSelected) : selectionsSaved[idx] = hashSelected

    LocalStorage.setItem(headerMenuSelectedStorageKey, selectionsSaved.join(","))
  })

  onMount(() => {
    if(!isLogged){
      Env.navigate("/login")
    }
  })

  onDestroy(() => {
    ui.openSideLayer(0)
    ui.closeAllModals()
  })

</script>

<div id="page-container" class="_1 px-10 pt-10 {containerCss}" class:useTopMinimalMenu={ui.state.useTopMinimalMenu}
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
