<script lang="ts">
import { browser } from '$app/environment';
import { onMount } from 'svelte';
import Input from '$components/Input.svelte';
import OptionsStrip from '$components/OptionsStrip.svelte';
import TableGrid from '$components/vTable/TableGrid.svelte';
import { accessHelper } from '$core/security';
import { Core } from '$core/store.svelte';
import { Env } from '$core/env';
import type { ICacheDebugRow } from '$libs/cache/cache-debug.types';
import type { TableGridColumn } from '$components/vTable/tableGridTypes';
import {
  listEnvironmentCacheRouteStats,
  makeDeltaCacheDatabaseName,
} from '$libs/cache/delta-cache.idb';
import { clearGroupCache, listGroupCacheStats } from '$libs/cache/group-cache.idb';
import { clearCacheByIDs } from '$libs/cache/cache-by-ids.svelte';
import { sendServiceMessage } from '$libs/sw-cache';
import pkg from 'notiflix'
const { Loading, Notify } = pkg;
import { postUsuarioPropio } from '$services/services/usuarios.svelte';
import type { IUsuario } from '$core/types/common';
import { HEADER_REQUEST_LOGS_MODAL_ID } from '$domain/HeaderRequestLogsModal.svelte';
import { formatN } from '$libs/helpers';

  const options = [
    { id: 1, name: "Usuario" }, { id: 2, name: "Config." }, { id: 3, name: "Data" }
  ]
  let selected = $state(1)
  let cacheRows: ICacheDebugRow[] = $state([])
  type IGroupedCacheRow = {
    baseRoute: string
    recordsCount: number
    sizeLabel: number
    sizeBytes: number
  }
  let cacheDataLoaded = $state(false)
  let cacheDataLoading = $state(false)
  let cacheDataClearing = $state(false)
  let viewportWidth = $state(browser ? window.innerWidth : 1280)

	function handleLogout() {
		// Clear session/tokens
		localStorage.clear();
		sessionStorage.clear();
		window.location.href = '/login';
	}

  let userInfo = $state(accessHelper.getUserInfo())
  $effect(() => {
    if(selected === 1){ userInfo = userInfo = accessHelper.getUserInfo()}
  })

  const getCurrentDeltaDatabaseName = () => {
    // Inspector reads the same scoped database used by cached services in the current company/environment.
    return makeDeltaCacheDatabaseName(Env.getEmpresaID(), Env.enviroment || 'main')
  }

  const sortCacheRows = (rows: ICacheDebugRow[]) => {
    return [...rows].sort((leftRow, rightRow) => {
      if (leftRow.baseRoute === rightRow.baseRoute) {
        if (leftRow.source === rightRow.source) {
          return leftRow.apiRoute.localeCompare(rightRow.apiRoute)
        }
        return leftRow.source.localeCompare(rightRow.source)
      }
      return leftRow.baseRoute.localeCompare(rightRow.baseRoute)
    })
  }

  const groupedCacheRows = $derived.by<IGroupedCacheRow[]>(() => {
    const groupedRowsByBaseRoute = new Map<string, {
      recordsCount: number
      sizeBytes: number
      hasKnownSize: boolean
    }>()

    for (const cacheRow of cacheRows) {
      const currentGroup = groupedRowsByBaseRoute.get(cacheRow.baseRoute) || {
        recordsCount: 0,
        sizeBytes: 0,
        hasKnownSize: false,
      }

      currentGroup.recordsCount += Number(cacheRow.recordsCount || 0)
      if (typeof cacheRow.sizeMB === 'number') {
        currentGroup.sizeBytes += Math.round(cacheRow.sizeMB * 1024 * 1024)
        currentGroup.hasKnownSize = true
      }
      groupedRowsByBaseRoute.set(cacheRow.baseRoute, currentGroup)
    }

    return [...groupedRowsByBaseRoute.entries()]
      .map(([baseRoute, aggregatedGroup]) => ({
        baseRoute,
        recordsCount: aggregatedGroup.recordsCount,
        sizeBytes: aggregatedGroup.sizeBytes,
        sizeLabel: aggregatedGroup.sizeBytes / (1024 * 1024),
      }))
      .sort((leftRow, rightRow) => leftRow.baseRoute.localeCompare(rightRow.baseRoute))
  })

  const cacheGridColumns: TableGridColumn<IGroupedCacheRow>[] = [
    {
      id: 'api',
      header: 'API',
      width: 'minmax(0, 1fr)',
      getValue: (cacheRow) => cacheRow.baseRoute,
      cellCss: 'px-6 text-[15px] text-slate-700',
      headerCss: 'px-6 py-6 text-[15px]',
    },
    {
      id: 'records',
      header: 'Regs.',
      width: '78px',
      align: 'right',
      getValue: (cacheRow) => formatN(cacheRow.recordsCount),
      cellCss: 'px-6 text-[15px] text-slate-600',
      headerCss: 'px-6 py-6 text-[15px]',
    },
    {
      id: 'size',
      header: 'Size',
      width: '66px',
      align: 'right',
      getValue: (cacheRow) => formatN(cacheRow.sizeLabel,2),
      cellCss: 'px-6 text-[15px] text-slate-600',
      headerCss: 'px-6 py-6 text-[15px]',
    },
  ]

  const getCacheGridRowID = (cacheRow: IGroupedCacheRow) => {
    return cacheRow.baseRoute
  }

  const cacheGridHeight = $derived(
    viewportWidth >= 749
      // The desktop settings layer is fixed to 460px height, so the grid must stay shorter than the remaining content area.
      ? '300px'
      : '48vh'
  )

  const loadCacheData = async (forceReload = false) => {
    if (!browser || cacheDataLoading || cacheDataClearing) { return }
    if (cacheDataLoaded && !forceReload) { return }

    cacheDataLoading = true
    console.debug('[HeaderConfig] Loading local cache inspector data.', {
      forceReload,
      enviroment: Env.enviroment,
      companyID: Env.getEmpresaID(),
    })

    try {
      // Both caches are read with metadata/index operations only; no full payload rebuild is needed here.
      const [deltaCacheRows, groupCacheRows] = await Promise.all([
        listEnvironmentCacheRouteStats(getCurrentDeltaDatabaseName()),
        listGroupCacheStats(),
      ])

      cacheRows = sortCacheRows([...deltaCacheRows, ...groupCacheRows])
      cacheDataLoaded = true
      console.debug('[HeaderConfig] Local cache inspector data loaded.', {
        deltaRoutes: deltaCacheRows.length,
        groupRoutes: groupCacheRows.length,
        totalRows: cacheRows.length,
      })
    } catch (error) {
      console.warn('[HeaderConfig] Failed to load local cache inspector data.', error)
      Notify.failure('No se pudo leer el cache local.')
    } finally {
      cacheDataLoading = false
    }
  }

  const clearLocalCache = async () => {
    if (!browser || cacheDataClearing) { return }

    cacheDataClearing = true
    Loading.standard('Eliminando cache local...')
    console.debug('[HeaderConfig] Clearing local cache.', {
      enviroment: Env.enviroment,
      companyID: Env.getEmpresaID(),
    })

    try {
      // The service worker owns the hot delta-cache memory, so cache clearing must happen there.
      const [deltaClearResponse, clearedIDsCache, deletedGroupRows] = await Promise.all([
        sendServiceMessage(26, {}),
        clearCacheByIDs(),
        clearGroupCache(),
      ])
      const deletedDeltaRoutes = Number(deltaClearResponse?.deletedRoutes || 0)

      cacheRows = []
      cacheDataLoaded = false
      console.debug('[HeaderConfig] Local cache cleared.', {
        deletedDeltaRoutes,
        clearedIDsCache,
        deletedGroupRows,
      })
      Notify.success(
        `Cache eliminado. Delta: ${deletedDeltaRoutes} rutas. Group: ${deletedGroupRows} grupos. IDs: ${clearedIDsCache.databaseName}.`
      )
    } catch (error) {
      console.warn('[HeaderConfig] Failed to clear local cache.', error)
      Notify.failure('No se pudo eliminar el cache local.')
    } finally {
      Loading.remove()
      cacheDataClearing = false
    }
  }

  $effect(() => {
    if (selected !== 3) { return }
    loadCacheData()
  })

  onMount(() => {
    if (!browser) { return }

    const syncViewportWidth = () => {
      viewportWidth = window.innerWidth
    }

    syncViewportWidth()
    window.addEventListener('resize', syncViewportWidth)
    return () => {
      window.removeEventListener('resize', syncViewportWidth)
    }
  })

  const saveUsuario = async () => {
    if(userInfo.Password && userInfo.Password !== userInfo.Password2){
      Notify.failure("Los password no coinciden.")
    }

    Loading.standard("Creando/Actualizando Usuario...")
    try {
      var result = await postUsuarioPropio(userInfo)
    } catch (error) {
      Notify.failure(error as string)
      Loading.remove()
      return
    }
    Loading.remove()
    accessHelper.setUserInfo(userInfo)
    console.log("usuario result::", result)
  }

</script>

<div class="flex items-center mb-6">
  <OptionsStrip options={options} keyId="id" keyName="name"
    selected={selected} onSelect={e => selected = e.id}
  />
</div>
{#if selected === 1}
  <div class="w-full flex mb-12 mt-[-2px]">
    <div class="mr-auto"></div>
    <button class="bx-blue mr-12" aria-label="Guardar Usuario"
      onclick={() => { saveUsuario() }}
    >
      <i class="icon-floppy"></i>
    </button>
    <button class="bx-orange" aria-label="Salir"
      onclick={handleLogout}
    >
      <i class="icon-logout-1"></i>
      <span>Salir</span>
    </button>
  </div>
  <div class="grid grid-cols-24 w-full gap-10">
    <Input label="Nombres" css="col-span-12"
      saveOn={userInfo} save="Nombres"
    />
    <Input label="Apellidos" css="col-span-12"
      saveOn={userInfo} save="Apellidos"
    />
    <Input label="Email" css="col-span-12"
      saveOn={userInfo} save="Email"
    />
    <Input label="Cargo" css="col-span-12"
      saveOn={userInfo} save="Cargo"
    />
    <Input label="Nº Documento" css="col-span-12"
      saveOn={userInfo} save="DocumentoNro"
    />
    <div class="col-span-24">
      <div class="ff-bold mb-[-4px] mt-2">Cambiar Password</div>
    </div>
    <Input label="Password" css="col-span-12"
      saveOn={userInfo} save="Password" type="password"
    />
    <Input label="Repetir Password" css="col-span-12"
      saveOn={userInfo} save="Password2" type="password"
    />
  </div>
{/if}
{#if selected === 2}
  <div class="w-full flex mb-12 mt-[-2px]">
    <div class="mr-auto"></div>
    <button class="bx-blue min-w-120 px-12" aria-label="Ver logs de requests"
      onclick={() => { 
	      // Close the global header dropdown first so the modal is the only visible overlay.
	      Core.closeHeaderSettings()
	      // Opening a globally mounted modal avoids losing it when the settings dropdown auto-closes.
	      Core.openModal(HEADER_REQUEST_LOGS_MODAL_ID)
      }}
    >
      <i class="icon-list"></i>
      <span>Reqs. Logs</span>
    </button>
  </div>
{/if}
{#if selected === 3}
  <div class="w-full flex items-center mb-12 mt-[-2px] gap-8">
    <div class="mr-auto text-[15px] text-slate-500">
      Cache local agrupado por ruta base.
    </div>
    <button class="bx-blue min-w-44 px-10 md:px-14" aria-label="Recargar cache local"
      disabled={cacheDataLoading || cacheDataClearing}
      onclick={() => { loadCacheData(true) }}
    >
      <i class="icon-arrows-cw"></i>
      <span class="hidden md:inline">Recargar</span>
    </button>
    <button class="bx-red min-w-44 px-10 md:px-14" aria-label="Eliminar cache local"
      disabled={cacheDataLoading || cacheDataClearing}
      onclick={() => { clearLocalCache() }}
    >
      <i class="icon-trash"></i>
      <span class="hidden md:inline">Eliminar cache</span>
    </button>
  </div>

  <TableGrid
    columns={cacheGridColumns}
    data={groupedCacheRows}
    height={cacheGridHeight}
    rowHeight={32}
    bufferSize={16}
    css="w-full"
    headerCss="bg-slate-50"
    rowCss="text-[15px]"
    emptyMessage={cacheDataLoading ? 'Leyendo cache local...' : 'No hay datos de cache para este entorno.'}
    getRowId={getCacheGridRowID}
  />
{/if}
