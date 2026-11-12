<script lang="ts">
import { security } from '#libs/ui-runtime.svelte.ts';
import { onDestroy, onMount, untrack } from "svelte";
import { Core } from '#core/store.svelte.ts';
import { browser, Env, LocalStorage } from '#core/env.ts';
import { checksumBase64_6 } from '@genix/ui/utilities';
import { useUI } from '@genix/ui';
import LoadingBar from '#components/misc/LoadingBar.svelte';
import { pageHeader } from '#domain/page-header.svelte.ts';
import type { Snippet } from 'svelte';

  const headerMenuSelectedStorageKey = "headerMenuSelected"

  type TPageOption = {id: number, name: string}

  // The header slot shows one thing: `header`, else the `options` (tabs, or the search box), else
  // the title with its back button and subtitle. So `showBackButton` / `subtitle` draw nothing on
  // a page with `options`.
  let {
    children, title, options, containerCss, useTopMinimalMenu, fixedFullHeight,
    header, isLoading = false, useMenuSearchBox = false, selectedOption, onSelectOption,
    showBackButton = false, onBack, subtitle, background, noPadding = false, hideHeader = false,
  }: {
    children: any,
    title: string,
    containerCss?: string,
    useTopMinimalMenu?: boolean,
    fixedFullHeight?: boolean
    options?: TPageOption[]
    // Replaces the header's title / tabs with the page's own content.
    header?: Snippet
    // While the first load of the data the page draws runs, a loader takes the place of the content,
    // so nothing shows as 0 or empty that is only waiting for its answer.
    isLoading?: boolean
    // Draws `options` as a search box in place of the title instead of tabs, so the list has no
    // length limit (a project picker...). The page owns the selection: `selectedOption` /
    // `onSelectOption`, not ui.state.pageOptionSelected.
    useMenuSearchBox?: boolean
    selectedOption?: number
    onSelectOption?: (optionID: number) => void
    showBackButton?: boolean
    // What the back button does; by default a browser "back" step.
    onBack?: () => void
    // A second line under the title.
    subtitle?: string
    // "white" paints the page on the surface color (a board of cards) instead of the gray background.
    background?: 'white'
    // Drops the page padding, so the content reaches the side menu and the header edge to edge.
    noPadding?: boolean
    // Hides the top bar on desktop and gives its height to the content (an embedded app, a preview).
    // Mobile keeps the bar: it holds the button that opens the side menu.
    hideHeader?: boolean
  } = $props();
  const ui = useUI()

  const isLogged = $derived(security.checkIsLogin() === 2)
  let hasRestoredPageOptionSelection = $state(false)
  // A search box's options are the page's, never header tabs.
  const pageOptions = $derived(useMenuSearchBox ? [] : (options || []))

  // What the page adds to the header around its title. Cleared only while it is still ours: the next
  // route's Page may have mounted first.
  $effect(() => {
    const headerContent = {
      content: header, subtitle, hiddenOnDesktop: hideHeader,
      onBack: showBackButton ? (onBack || (() => history.back())) : undefined,
      menuSearchBox: useMenuSearchBox ? { options: options || [], selectedOption, onSelectOption } : undefined,
    }
    untrack(() => { pageHeader.current = headerContent })
    return () => { if (pageHeader.current === headerContent) { pageHeader.current = {} } }
  })

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
      ui.state.pageOptions = pageOptions
    })
  })

  // The title follows the page's view (a list, then one of its records), so it is tracked.
  $effect(() => {
    const pageTitle = title || ""
    untrack(() => { ui.state.pageTitle = pageTitle })
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
      Env.navigate("/welcome")
    }
  })

  onDestroy(() => {
    ui.openSideLayer(0)
    ui.closeAllModals()
  })

</script>

<div id="page-container" class="_1 {noPadding ? '' : 'px-10 pt-10'} {background === 'white' ? 'bg-surface' : ''} {containerCss}" class:useTopMinimalMenu={ui.state.useTopMinimalMenu}
	class:fixed-full-height={fixedFullHeight} class:no-header={hideHeader}
>
  {#if Core.isLoading === 0 && isLogged}
    {#if isLoading}
      <div class="max-w-420 mx-auto pt-60" role="status" aria-label="Loading data">
        <LoadingBar label="Loading...|Cargando..." />
      </div>
    {:else}
      {@render children()}
    {/if}
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

  /* Same 751px breakpoint as AppHeader's `hidden-on-desktop`: the content takes the bar's height. */
  @media (min-width: 751px) {
    ._1.no-header {
      margin-top: 0;
      min-height: 100vh;
    }
    .fixed-full-height.no-header {
      height: 100vh;
    }
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
