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
  if (status === 0) return tr('Pending (0)|Pendiente (0)')
  if (status === 1) return tr('Done (1)|Ejecutada (1)')
  if (status === 2) return tr('Abandoned (2)|Abandonada (2)')
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

const formatMessages = (row: ICronActionTableRow) => (row.messages || []).join(' · ')

const columns: ITableColumn<ICronActionTableRow>[] = [
  {
    header: "Time Slot|Franja Horaria",
    headerCss: "w-140",
    css: "px-6 nowrap",
    getValue: (row) => formatUnixMinutesFrame(row.UnixMinutesFrame),
    mobile: { order: 2, css: "col-span-14", labelLeft: "Franja:" },
  },
  {
    header: "A-ID",
    headerCss: "w-60",
    css: "text-center ff-mono",
    getValue: (row) => row.ActionID,
    mobile: { order: 1, css: "col-span-6 ff-bold", icon: "[fa--tag]" },
  },
  {
    header: "Action|Acción",
    highlight: true,
    css: "px-6",
    getValue: (row) => row.ActionName,
    mobile: { order: 3, css: "col-span-24", render: (row) => `<strong>${row.ActionName || ""}</strong>` },
  },
  {
    header: "Company|Empresa",
    headerCss: "w-96",
    css: "text-center ff-mono",
    getValue: (row) => row.CompanyID,
    mobile: { order: 5, css: "col-span-12", labelLeft: "Empresa:" },
  },
  {
    header: "Parameters|Parámetros",
    headerCss: "w-320",
    css: "px-6 nowrap",
    getValue: (row) => formatParams(row),
    mobile: { order: 7, css: "col-span-24", labelTop: "Parameters|Parámetros" },
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
    mobile: { order: 6, css: "col-span-12", labelLeft: "Invoc.:" },
  },
  {
    header: "Status",
    headerCss: "w-120",
    css: "text-center",
    getValue: (row) => getStatusLabel(row.ss||0),
    mobile: { order: 4, css: "col-span-12", labelLeft: "Estado:" },
  },
  {
    header: "Updated",
    headerCss: "w-160",
    css: "px-6 nowrap",
    getValue: (row) => formatUpdatedSunix(row.upd),
    mobile: { order: 8, css: "col-span-24", labelLeft: "Actualizado:" },
  },
  {
    // getValue and not render: messages carry handler errors and panic text, and render output
    // goes through {@html}.
    header: "Messages|Mensajes",
    css: "px-6",
    getValue: (row) => formatMessages(row),
    mobile: { order: 9, css: "col-span-24", labelTop: "Messages|Mensajes", if: (row) => (row.messages || []).length > 0 },
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
        formatMessages(row),
      ].join(" ").toLowerCase()}
    />
  </div>
</Page>
