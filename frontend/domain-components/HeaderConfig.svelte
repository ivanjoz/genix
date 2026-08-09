<script lang="ts">
import { browser } from '$app/environment';
import { onMount, untrack } from 'svelte';
import Input from '$components/form/Input.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import CheckboxOptions from '$components/form/CheckboxOptions.svelte';
import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
import TableGrid from '$components/vTable/TableGrid.svelte';
import { security } from '$libs/ui-runtime.svelte';
import { Core, setLanguaje, type ILanguaje } from '$core/store.svelte';
import { Env } from '$core/env';
import type { ICacheDebugRow } from '@genix/ui/cache';
import type { ITableColumn } from '$components/vTable/types';
import {
  listEnvironmentCacheRouteStats,
  makeDeltaCacheDatabaseName,
} from '@genix/ui/cache';
import { clearGroupCache, listGroupCacheStats } from '@genix/ui/cache';
import { clearCacheByIDs } from '@genix/ui/cache';
import { sendServiceMessage } from '@genix/ui/service-worker';
import pkg from 'notiflix'
const { Loading, Notify } = pkg;
import { postOwnUser } from '$services/services/users.svelte';
import type { IUser } from '$core/types/common';
import { HEADER_REQUEST_LOGS_MODAL_ID } from '$domain/HeaderRequestLogsModal.svelte';
import { formatN } from '$libs/helpers';
import { formatTime } from '$libs/helpers';
import { ChartCanvas, type ChartCanvasSeries } from '@genix/ui/charts';
import {
  getCreditUsage,
  type ICreditUsageResponse,
  type ICreditUsageScope,
} from '$services/services/credit-usage';
import {
  AgentModelsService,
  getSelectedAgentModelHash,
  setSelectedAgentModelHash,
} from '$core/agent/models.svelte';
import { useUI } from '@genix/ui';

  const options = [
    { id: 1, name: "Usuario" }, { id: 2, name: "Config." }, { id: 3, name: "Data" }
  ]
  // 1 = Spanish, 2 = English (see Core.languaje)
  const languajeOptions = [
    { id: 1, name: "Español" }, { id: 2, name: "English" }
  ]
  let selected = $state(1)
  const ui = useUI()
  const agentModelsService = new AgentModelsService()
  const canSelectAgentModel = Env.getCompanyID() === 1
  let agentModelForm = $state({ ModelHash: getSelectedAgentModelHash() })
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
  type CreditUsageScopeName = 'User' | 'Company'
  const creditUsageScopeOptions = [
    { ID: 'User' as CreditUsageScopeName, Name: 'Usuario' },
    { ID: 'Company' as CreditUsageScopeName, Name: 'Empresa' },
  ]
  let creditUsageScopeForm = $state<{ Scope: CreditUsageScopeName }>({ Scope: 'User' })
  let creditUsage = $state<ICreditUsageResponse | undefined>(undefined)
  let creditUsageLoading = $state(false)
  let creditUsageLoaded = $state(false)
  let creditUsageError = $state('')

	const loadCreditUsage = async (forceReload = false) => {
		if (!browser || creditUsageLoading || (creditUsageLoaded && !forceReload)) { return }
		creditUsageLoading = true
		creditUsageError = ''
		console.debug('[HeaderConfig] Loading 15-day credit usage.', { forceReload })
		try {
			creditUsage = await getCreditUsage()
			creditUsageLoaded = true
			console.debug('[HeaderConfig] Credit usage loaded.', {
				userDays: creditUsage?.User?.Days?.length || 0,
				companyDays: creditUsage?.Company?.Days?.length || 0,
			})
		} catch (error) {
			creditUsageError = 'No se pudo cargar el uso de créditos.'
			console.warn('[HeaderConfig] Failed to load credit usage.', error)
		} finally {
			creditUsageLoading = false
		}
	}

	$effect(() => {
		if (selected !== 2) { return }
		// Avoid subscribing the effect to loader state changed by the async request.
		untrack(() => { loadCreditUsage() })
	})

	const selectedCreditUsage = $derived<ICreditUsageScope | undefined>(
		creditUsage?.[creditUsageScopeForm.Scope]
	)
	const todayCreditUsage = $derived(
		selectedCreditUsage?.Days?.[selectedCreditUsage.Days.length - 1]
	)
	const cpuCreditHistory = $derived.by<ChartCanvasSeries[]>(() => [{
		type: 'bar',
		name: 'CPU',
		values: (selectedCreditUsage?.Days || []).map((usageDay) => usageDay.CPU),
		color: '#4874f5',
	}])
	const inferenceCreditHistory = $derived.by<ChartCanvasSeries[]>(() => [{
		type: 'bar',
		name: 'Inferencia',
		values: (selectedCreditUsage?.Days || []).map((usageDay) => usageDay.Inference),
		color: '#8b5cf6',
	}])
	const creditUsageDateLabels = $derived(
		(selectedCreditUsage?.Days || []).map((usageDay) => usageDay.Day)
	)
	const formatCreditUsageDay = (unixDay: string | number) => {
		// formatTime preserves the project's UnixDay convention while presenting the UTC bucket date.
		return String(formatTime(Number(unixDay), 'd-M') || '')
	}
	const creditUsagePercent = (usedCredits: number, limitCredits: number) => {
		if (limitCredits <= 0) { return 0 }
		return Math.min(100, Math.max(0, (usedCredits / limitCredits) * 100))
	}

	function handleLogout() {
		// Clear session/tokens
		localStorage.clear();
		sessionStorage.clear();
		window.location.href = '/welcome';
	}

  let userInfo = $state(security.getUserInfo())
  $effect(() => {
    if(selected === 1){ userInfo = security.getUserInfo()}
  })

  $effect(() => {
    if (agentModelsService.records.length === 0) { return }
    if (agentModelForm.ModelHash && agentModelsService.modelHashMap.has(agentModelForm.ModelHash)) { return }
    agentModelForm.ModelHash = agentModelsService.defaultHash
    setSelectedAgentModelHash(agentModelForm.ModelHash)
  })

  const getCurrentDeltaDatabaseName = () => {
    // Inspector reads the same scoped database used by cached services in the current company/environment.
    return makeDeltaCacheDatabaseName(Env.getCompanyID(), Env.enviroment || 'main')
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

  const cacheGridColumns: ITableColumn<IGroupedCacheRow>[] = [
    {
      id: 'api',
      header: 'API',
      width: 'minmax(0, 1fr)',
      getValue: (cacheRow) => cacheRow.baseRoute,
      css: 'px-6 text-[15px] text-slate-700',
      headerCss: 'px-6 py-6 text-[15px]',
    },
    {
      id: 'records',
      header: 'Regs.',
      width: '78px',
      align: 'right',
      getValue: (cacheRow) => formatN(cacheRow.recordsCount),
      css: 'px-6 text-[15px] text-slate-600',
      headerCss: 'px-6 py-6 text-[15px]',
    },
    {
      id: 'size',
      header: 'Size',
      width: '66px',
      align: 'right',
      getValue: (cacheRow) => formatN(cacheRow.sizeLabel,2),
      css: 'px-6 text-[15px] text-slate-600',
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
      companyID: Env.getCompanyID(),
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
      companyID: Env.getCompanyID(),
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
    // getUserInfo() is null until a session exists, so the form has nothing to save.
    if(!userInfo){ return }
    if(userInfo.Password && userInfo.Password !== userInfo.Password2){
      Notify.failure("Los password no coinciden.")
    }

    Loading.standard("Creando/Actualizando Usuario...")
    try {
      var result = await postOwnUser(userInfo)
    } catch (error) {
      Notify.failure(error as string)
      Loading.remove()
      return
    }
    Loading.remove()
    security.setUserInfo(userInfo)
    console.log("usuario result::", result)
  }

</script>

<div class="flex items-center mb-10">
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
      <i class="icon-[fa--floppy-o]"></i>
    </button>
    <button class="bx-orange" aria-label="Salir"
      onclick={handleLogout}
    >
      <i class="icon-[fa--sign-out]"></i>
      <span>Salir</span>
    </button>
  </div>
  <div class="grid grid-cols-24 w-full gap-10">
  {#if userInfo}
    <Input label="Nombres" css="col-span-12"
      saveOn={userInfo} save="FirstName"
    />
    <Input label="Apellidos" css="col-span-12"
      saveOn={userInfo} save="LastName"
    />
    <Input label="Email" css="col-span-12"
      saveOn={userInfo} save="Email"
    />
    <Input label="Cargo" css="col-span-12"
      saveOn={userInfo} save="JobTitle"
    />
    <Input label="Nº Documento" css="col-span-12"
      saveOn={userInfo} save="DocumentNumber"
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
  {/if}
  </div>
{/if}
{#if selected === 2}
  <div class="mb-10 rounded-[8px] border border-slate-200 bg-slate-50/70 p-8">
    <div class="mb-6 flex items-center gap-8">
      <div class="ff-semibold text-[14px] text-slate-700">Créditos · 15 días</div>
      <div class="ml-auto">
        <CheckboxOptions
          options={creditUsageScopeOptions}
          keyId="ID"
          keyName="Name"
          type="single"
          useButtonsSlim={true}
          saveOn={creditUsageScopeForm}
          save="Scope"
        />
      </div>
      <button
        class="flex h-32 w-32 shrink-0 items-center justify-center rounded-full border border-slate-300 bg-slate-100 text-[14px] text-slate-500 shadow-sm hover:bg-slate-200 disabled:opacity-50"
        aria-label="Actualizar uso de créditos"
        disabled={creditUsageLoading}
        onclick={() => { loadCreditUsage(true) }}
      >
        <i class={`icon-[fa--refresh] ${creditUsageLoading ? 'animate-spin' : ''}`}></i>
      </button>
    </div>

    {#if creditUsageError}
      <div class="flex h-116 items-center justify-center text-[14px] text-red-500">{creditUsageError}</div>
    {:else if !selectedCreditUsage || !todayCreditUsage}
      <div class="flex h-116 items-center justify-center text-[14px] text-slate-500">
        {creditUsageLoading ? 'Cargando uso de créditos…' : 'Sin datos de créditos.'}
      </div>
    {:else}
      <div class="mb-6 grid grid-cols-2 gap-10">
        <div>
          <div class="mb-3 flex items-center text-[12px] text-slate-600">
            <span class="ff-bold text-[14px] text-[#4874f5]">CPU hoy</span>
            <span class="ml-auto ff-mono">{formatN(todayCreditUsage.CPU)} / {formatN(selectedCreditUsage.CPU24hLimit)}</span>
          </div>
          <div class="h-5 overflow-hidden rounded-full bg-slate-200">
            <div class="h-full rounded-full bg-[#4874f5]"
              style={`width:${creditUsagePercent(todayCreditUsage.CPU, selectedCreditUsage.CPU24hLimit)}%`}
            ></div>
          </div>
        </div>
        <div>
          <div class="mb-3 flex items-center text-[12px] text-slate-600">
            <span class="ff-bold text-[14px] text-[#8b5cf6]">Inferencia hoy</span>
            <span class="ml-auto ff-mono">{formatN(todayCreditUsage.Inference)} / {formatN(selectedCreditUsage.Inference24hLimit)}</span>
          </div>
          <div class="h-5 overflow-hidden rounded-full bg-slate-200">
            <div class="h-full rounded-full bg-[#8b5cf6]"
              style={`width:${creditUsagePercent(todayCreditUsage.Inference, selectedCreditUsage.Inference24hLimit)}%`}
            ></div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-10">
        <div class="min-w-0">
          <ChartCanvas
            id={`credit-usage-cpu-${creditUsageScopeForm.Scope}`}
            data={cpuCreditHistory}
            dateLabels={creditUsageDateLabels}
            dateLabelFormatter={formatCreditUsageDay}
            dateLabelEvery={5}
            useHtmlRendered={true}
            showBottomBaseline={true}
            height={58}
          />
        </div>
        <div class="min-w-0">
          <ChartCanvas
            id={`credit-usage-inference-${creditUsageScopeForm.Scope}`}
            data={inferenceCreditHistory}
            dateLabels={creditUsageDateLabels}
            dateLabelFormatter={formatCreditUsageDay}
            dateLabelEvery={5}
            useHtmlRendered={true}
            showBottomBaseline={true}
            height={58}
          />
        </div>
      </div>
    {/if}
  </div>
  <div class="w-full mt-2">
    <div class="ff-semibold text-[15px] text-slate-600 mb-6">Idioma / Language</div>
    <CheckboxOptions
      options={languajeOptions}
      keyId="id"
      keyName="name"
      type="single"
      useButtons={true}
      saveOn={Core}
      save="languaje"
      onChange={(ids) => setLanguaje((Number(ids[0]) === 2 ? 2 : 1) as ILanguaje)}
    />
  </div>
  <div class="mt-14 flex w-full">
    {#if canSelectAgentModel}
      <SearchSelect
        label="Modelo"
        css="w-full max-w-[460px]"
        inputCss="h-32 text-[15px]"
        options={agentModelsService.records}
        keyId="Hash"
        keyName="ID"
        bind:saveOn={agentModelForm}
        save="ModelHash"
        notEmpty={true}
        showLoading={agentModelsService.isReady === 0}
        onChange={(modelOption) => setSelectedAgentModelHash(modelOption.Hash)}
      />
    {/if}
    <div class="mr-auto"></div>
    <button class="bx-blue min-w-120 px-12" aria-label="Ver logs de requests"
      onclick={() => {
        // Close the global header dropdown first so the modal is the only visible overlay.
        ui.state.headerSettingsOpen = false
        // Opening a globally mounted modal avoids losing it when the settings dropdown auto-closes.
        ui.openModal(HEADER_REQUEST_LOGS_MODAL_ID)
      }}
    >
      <i class="icon-[fa--list]"></i>
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
      <i class="icon-[fa--refresh]"></i>
      <span class="hidden md:inline">Recargar</span>
    </button>
    <button class="bx-red min-w-44 px-10 md:px-14" aria-label="Eliminar cache local"
      disabled={cacheDataLoading || cacheDataClearing}
      onclick={() => { clearLocalCache() }}
    >
      <i class="icon-[fa--trash]"></i>
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
