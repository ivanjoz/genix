<script lang="ts">
import Page from '$domain/Page.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import { formatTime } from '$libs/helpers';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import { tr } from '$core/store.svelte';
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
  if (status === 0) return tr('Pending (0)|Pendiente (0)')
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
    header: "Time Slot|Franja Horaria",
    headerCss: "w-140",
    css: "px-6 nowrap",
    getValue: (row) => formatUnixMinutesFrame(row.UnixMinutesFrame),
  },
  {
    header: "A-ID",
    headerCss: "w-60",
    css: "text-center ff-mono",
    getValue: (row) => row.ActionID,
  },
  {
    header: "Action|Acción",
    highlight: true,
    css: "px-6",
    getValue: (row) => row.ActionName,
  },
  {
    header: "Company|Empresa",
    headerCss: "w-96",
    css: "text-center ff-mono",
    getValue: (row) => row.CompanyID,
  },
  {
    header: "Parameters|Parámetros",
    headerCss: "w-320",
    css: "px-6 nowrap",
    getValue: (row) => formatParams(row),
  },
  {
    header: "Invocations|Invocaciones",
    headerCss: "w-240",
    css: "px-6",
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
    css: "text-center",
    getValue: (row) => getStatusLabel(row.ss||0),
  },
  {
    header: "Updated",
    headerCss: "w-160",
    css: "px-6 nowrap",
    getValue: (row) => formatUpdatedSunix(row.upd),
  },
]
</script>

<Page title="Cron Actions">
  <div class="h-full">
    <div class="flex items-center justify-between mb-6 gap-12" aria-label="Cron actions toolbar with filter and reload button">
      <FilterInput bind:value={filterText} css="w-320" />
      <Button color="blue" name="Reload|Recargar" icon="icon-[fa--refresh]" label="Reloads the cron actions list from the server."
        onClick={() => cronActionsService.fetchOnline()} />
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
