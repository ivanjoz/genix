<script lang="ts">
  import Button from '$components/buttons/Button.svelte'
  import T from '$components/misc/T.svelte'
  import { isSaleDelivered, isSalePaid } from './sale_history'
  import type { SaleHistoryRow } from './sale_history.idb'
  import {
    formatTicketAmount, formatTicketDateTime, saleDocumentNumber, saleDocumentTitle,
    saleSeriesID, type TicketContext,
  } from './sale_ticket'

  let { rows, isLoaded, context, onPrint }: {
    rows: SaleHistoryRow[]
    isLoaded: boolean
    context: TicketContext
    onPrint: (row: SaleHistoryRow) => void
  } = $props()

  const seriesOf = (row: SaleHistoryRow) => context.seriesByID.get(saleSeriesID(row.saleID))
</script>

<div class="h-full overflow-y-auto px-12 pb-12" aria-label="Local sale history of this till">
  {#if rows.length === 0}
    <div class="py-32 text-center text-sm text-gray-500">
      {#if isLoaded}
        <T text="No sales made from this device yet.|Aún no hay ventas hechas desde este equipo." />
      {:else}
        <T text="Loading history...|Cargando historial..." />
      {/if}
    </div>
  {/if}

  {#each rows as row (row.saleID)}
    {@const series = seriesOf(row)}
    {@const client = context.clientsByID.get(row.clientID)}
    <div class="_card mb-8 flex items-center gap-10 rounded-lg border border-gray-200 bg-white p-10">
      <div class="min-w-0 flex-1">
        <div class="flex items-baseline gap-8">
          <span class="ff-bold truncate text-[15px] text-gray-800">
            {saleDocumentNumber(row.saleID, series)}
          </span>
          <span class="shrink-0 text-[12px] text-gray-400">
            {formatTicketDateTime(row.savedAt)}
          </span>
        </div>

        <div class="truncate text-[13px] text-gray-500">
          {saleDocumentTitle(row.saleID, series)} · {client?.Name || "VARIOS"}
        </div>

        <div class="mt-4 flex items-center gap-6">
          {#if isSalePaid(row.status)}
            <span class="_chip ff-bold text-[11px] bg-green-50 text-green-700">
              <T text="PAID|PAGADO" />
            </span>
          {:else}
            <span class="_chip ff-bold text-[11px] bg-amber-50 text-amber-700">
              <T text="UNPAID|POR COBRAR" />
            </span>
          {/if}
          {#if isSaleDelivered(row.status)}
            <span class="_chip ff-bold text-[11px] bg-blue-50 text-blue-700">
              <T text="DELIVERED|ENTREGADO" />
            </span>
          {/if}
          <span class="text-[12px] text-gray-400">
            {row.lines.length}
            <T text={row.lines.length === 1 ? "item|producto" : "items|productos"} />
          </span>
        </div>
      </div>

      <div class="shrink-0 text-right">
        <div class="ff-bold text-[18px] leading-[1] text-blue-700">
          {formatTicketAmount(row.totalAmount)}
        </div>
      </div>

      <Button icon="icon-[fa--print]" color="blue" useCircle
        css="shrink-0"
        label="Opens the printable thermal ticket for this sale."
        onClick={() => onPrint(row)}
      />
    </div>
  {/each}
</div>

<style>
  ._card {
    transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }
  ._card:hover {
    border-color: #c7d2fe;
    box-shadow: rgba(79, 70, 229, 0.12) 0 2px 8px;
  }
  ._chip {
    padding: 1px 6px;
    border-radius: 4px;
    line-height: 1.4;
  }
</style>
