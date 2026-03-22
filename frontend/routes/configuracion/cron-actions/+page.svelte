<script lang="ts">
import Page from '$domain/Page.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { formatTime, throttle } from '$libs/helpers';
import {
  CronActionsService,
  type ICronActionTableRow,
} from './cron-actions.svelte';

const cronActionsService = new CronActionsService()
let filterText = $state("")

const formatUnixMinutesFrame = (unixMinutesFrame: number) => {
  // Scheduled frames are stored in 5-minute units, so convert them back to Unix seconds.
  return formatTime(unixMinutesFrame * 5 * 60, "M-d h:n") as string
}

const formatUpdatedSunix = (updatedSunix: number) => {
  return formatTime(updatedSunix, "M-d h:n") as string
}

const getStatusLabel = (status: number) => {
  // Status 0 is known as pending from the scheduler flow; other values stay explicit.
  if (status === 0) return 'Pendiente (0)'
  return String(status)
}

const formatParams = (row: ICronActionTableRow) => {
  const params = row.Params || {}
  const paramsEntries = [
    [1, params.p1],
    [2, params.p2],
    [3, params.p3],
    [4, params.p4],
    [5, params.p5],
    [6, params.p6],
  ]

  // Keep the column compact by showing only filled params on a single line.
  return paramsEntries
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([index, value]) => `[${index}]=${value}`)
    .join(', ')
}

const columns: ITableColumn<ICronActionTableRow>[] = [
  {
    header: "Franja Horaria",
    headerCss: "w-140",
    cellCss: "px-6 nowrap",
    getValue: (row) => formatUnixMinutesFrame(row.UnixMinutesFrame),
  },
  {
    header: "A-ID",
    headerCss: "w-60",
    cellCss: "text-center ff-mono",
    getValue: (row) => row.ActionID,
  },
  {
    header: "Acción",
    highlight: true,
    cellCss: "px-6",
    getValue: (row) => row.ActionName,
  },
  {
    header: "Empresa",
    headerCss: "w-96",
    cellCss: "text-center ff-mono",
    getValue: (row) => row.CompanyID,
  },
  {
    header: "Parámetros",
    headerCss: "w-320",
    cellCss: "px-6 nowrap",
    getValue: (row) => formatParams(row),
  },
  {
    header: "Invocaciones",
    headerCss: "w-240",
    cellCss: "px-6",
    getValue: (row) => `${row.InvocationCount}`,
    render: (row) => `
      <div class="leading-tight py-4">
        <div class="ff-semibold">${row.InvocationCount||0}</div>
        <div class="ff-mono fs14 text-slate-500">${row.ID}</div>
      </div>
    `,
  },
  {
    header: "Status",
    headerCss: "w-120",
    cellCss: "text-center",
    getValue: (row) => getStatusLabel(row.ss||0),
  },
  {
    header: "Updated",
    headerCss: "w-160",
    cellCss: "px-6 nowrap",
    getValue: (row) => formatUpdatedSunix(row.upd),
  },
]
</script>

<Page title="Cron Actions">
  <div class="h-full">
    <div class="flex items-center justify-between mb-6 gap-12">
      <div class="i-search w-320">
        <div><i class="icon-search"></i></div>
        <input class="w-full" autocomplete="off" type="text" onkeyup={(event) => {
          event.stopPropagation()
          throttle(() => {
            filterText = ((event.target as HTMLInputElement).value || "").toLowerCase().trim()
          }, 150)
        }}>
      </div>

      <button class="bx-blue" onclick={() => { cronActionsService.fetchOnline() }}>
        Recargar <i class="icon-rotate-right"></i>
      </button>
    </div>

    <VTable
      columns={columns}
      data={cronActionsService.rows}
      css="w-full"
      maxHeight="calc(80vh - 13rem)"
      filterText={filterText}
      getFilterContent={(row) => [
        row.ActionID,
        row.ActionName,
        row.CompanyID,
        formatParams(row),
        row.ID,
        row.InvocationCount,
        row.ss,
      ].join(" ").toLowerCase()}
    />
  </div>
</Page>
