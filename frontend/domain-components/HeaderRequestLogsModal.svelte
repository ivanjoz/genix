<script context="module" lang="ts">
// Keep the modal id next to the modal implementation so callers share one source of truth.
export const HEADER_REQUEST_LOGS_MODAL_ID = 9201
</script>

<script lang="ts">
import { browser } from '$app/environment';
import { onMount } from 'svelte';
import Modal from '$components/Modal.svelte';
import TableGrid from '$components/vTable/TableGrid.svelte';
import { Env } from '$core/env';
import { openModals } from '$core/store.svelte';
import { listRecentRequestLogRows, makeDeltaCacheDatabaseName } from '$libs/cache/delta-cache.idb';
import type { IRequestLogRow } from '$libs/cache/delta-cache.types';
import { formatN, formatTime } from '$libs/helpers';
import type { ITableColumn } from '$components/vTable/types';
import pkg from 'notiflix'

const { Notify } = pkg;

let requestLogRows = $state<IRequestLogRow[]>([])
let requestLogsLoaded = $state(false)
let requestLogsLoading = $state(false)
let viewportWidth = $state(browser ? window.innerWidth : 1280)

const getCurrentDeltaDatabaseName = () => {
  // The modal reads the same environment/company scoped database where the worker stores request logs.
  return makeDeltaCacheDatabaseName(Env.getEmpresaID(), Env.enviroment || 'main')
}

const requestLogsGridHeight = $derived(
  viewportWidth >= 749
    ? '440px'
    : '58vh'
)

const formatRequestParamsLabel = (requestLogRow: IRequestLogRow) => {
  const rawQueryParams = String(requestLogRow.qp || '').trim()
  if (!rawQueryParams) { return '-' }

  // Hide environment/company scope params because they add noise and are the same for most rows.
  const visibleParams = rawQueryParams
    .split('&')
    .map((queryParam) => queryParam.trim())
    .filter(Boolean)
    .filter((queryParam) => {
      const normalizedParamName = queryParam.split('=')[0]?.trim().toLowerCase() || ''
      return normalizedParamName !== 'empresa-id' && normalizedParamName !== 'company-id'
    })

  if (visibleParams.length === 0) { return '-' }
  return visibleParams.join(', ')
}

const requestLogColumns: ITableColumn<IRequestLogRow>[] = [
  {
    id: 'time',
    header: 'Hora',
    width: '76px',
    getValue: (requestLogRow) => formatTime(requestLogRow.id, 'h:n') as string,
    cellCss: 'ff-mono text-sm text-slate-700',
    headerCss: 'px-6 py-6 text-[15px]',
    mobile: { order: 2, css: "px-0 col-span-6" }
  },
  {
    id: 'route',
    header: 'Route',
    width: '340px',
    getValue: (requestLogRow) => requestLogRow.route || '',
    cellCss: 'ff-semibold text-sm text-slate-700',
    headerCss: 'px-6 py-6 text-[15px]',
    mobile: { order: 1, css: "px-0 col-span-18"  }
  },
  {
    id: 'params',
    header: 'Req. Params',
    width: 'minmax(0, 1fr)',
    getValue: (requestLogRow) => formatRequestParamsLabel(requestLogRow),
    cellCss: 'text-xs text-slate-600 overflow-hidden whitespace-nowrap text-ellipsis',
    headerCss: 'px-6 py-6 text-[15px]',
    mobile: { order: 3, css: "px-0 col-span-10 overflow-hidden whitespace-nowrap text-ellipsis" }
  },
  {
    id: 'server',
    header: 'Server',
    width: '110px',
    align: 'right',
    getValue: (requestLogRow) => {
      // Keep backend timings readable as base time plus extra work until the final server timestamp.
      const preSerializeMilliseconds = Number(requestLogRow.sPs || 0)
      const finalMilliseconds = Number(requestLogRow.sF || 0)
      const extraServerMilliseconds = Math.max(0, finalMilliseconds - preSerializeMilliseconds)
      return `${preSerializeMilliseconds} +${extraServerMilliseconds}`
    },
    cellCss: 'px-6 ff-mono text-sm text-slate-700',
    headerCss: 'px-6 py-6 text-[15px]',
    mobile: { order: 3, css: "px-0 overflow-hidden whitespace-nowrap col-span-7" }
  },
  {
    id: 'client',
    header: 'Client',
    width: '110px',
    align: 'right',
    getValue: (requestLogRow) => {
      return `${requestLogRow.req} +${requestLogRow.spc}`
    },
    mobile: { order: 3, css: "px-0 overflow-hidden whitespace-nowrap col-span-7", labelLeft: "Cl:" },
    cellCss: 'px-6 ff-mono text-sm text-slate-700',
    headerCss: 'px-6 py-6 text-[15px]',
  },
]

const getRequestLogRowID = (requestLogRow: IRequestLogRow) => {
  return requestLogRow.id
}

const loadRecentRequestLogs = async (forceReload = false) => {
  if (!browser || requestLogsLoading) { return }
  if (requestLogsLoaded && !forceReload) { return }

  requestLogsLoading = true
  console.debug('[HeaderRequestLogsModal] Loading request logs.', {
    forceReload,
    enviroment: Env.enviroment,
    companyID: Env.getEmpresaID(),
  })

  try {
    requestLogRows = await listRecentRequestLogRows(getCurrentDeltaDatabaseName(), 200)
    requestLogsLoaded = true
    console.debug('[HeaderRequestLogsModal] Request logs loaded.', {
      rows: requestLogRows.length,
    })
  } catch (error) {
    console.warn('[HeaderRequestLogsModal] Failed to load request logs.', error)
    Notify.failure('No se pudieron leer los request logs.')
  } finally {
    requestLogsLoading = false
  }
}

$effect(() => {
  if (!openModals.includes(HEADER_REQUEST_LOGS_MODAL_ID)) { return }
  loadRecentRequestLogs()
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
</script>

<Modal id={HEADER_REQUEST_LOGS_MODAL_ID} title="Reqs. Logs" size={9}
  bodyCss="px-10 py-10 md:px-14 md:py-12"
>
  <div class="w-full">
    <TableGrid useInnerMobilePadding={true}
      columns={requestLogColumns}
      data={requestLogRows}
      height={requestLogsGridHeight}
      rowHeight={32}
      bufferSize={18}
      css="w-full"
      headerCss="bg-slate-50"
      rowCss="text-[15px]"
      emptyMessage={requestLogsLoading ? 'Leyendo request logs...' : 'No hay request logs para este entorno.'}
      getRowId={getRequestLogRowID}
    />
  </div>
</Modal>
