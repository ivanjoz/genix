<script lang="ts">
  import Modal from '$components/layers/Modal.svelte'
  import Portal from '$components/misc/Portal.svelte'
  import CheckboxOptions from '$components/form/CheckboxOptions.svelte'
  import {
    TICKET_58MM, TICKET_WIDTH_OPTIONS, renderSaleTicket, saleDocumentNumber, saleSeriesID,
    type TicketContext,
  } from './sale_ticket'
  import type { SaleHistoryRow } from './sale_history.idb'

  let { id, row, context }: {
    id: number
    row?: SaleHistoryRow
    context: TicketContext
  } = $props()

  // Bound through saveOn/save because that is CheckboxOptions' only write path. A second click
  // on the active option clears it, so every read goes through the fallback below.
  const ticketWidthForm = $state({ columns: TICKET_58MM })
  const ticketColumns = $derived(ticketWidthForm.columns || TICKET_58MM)

  // Paper width against printed width: a 58 mm roll prints 48 mm of ink, an 80 mm roll 72 mm,
  // at the 1.5 mm character cell the font size below is chosen for.
  const paperWidthMm = $derived(ticketColumns === TICKET_58MM ? 58 : 80)
  const inkWidthMm = $derived(ticketColumns * 1.5)
  const sidePaddingMm = $derived((paperWidthMm - inkWidthMm) / 2)

  const ticketText = $derived(row ? renderSaleTicket(row, context, ticketColumns) : "")
  const documentNumber = $derived(row
    ? saleDocumentNumber(row.saleID, context.seriesByID.get(saleSeriesID(row.saleID)))
    : "")

  const printTicket = () => {
    console.log("[sale-ticket] printing", { saleID: row?.saleID, columns: ticketColumns })
    window.print()
  }
</script>

<svelte:head>
  <!-- Through @html because the page box has to change with the selected roll, and Svelte does
       not interpolate inside a real <style> tag. -->
  {@html `<style>@page { size: ${paperWidthMm}mm auto; margin: 0; }</style>`}
</svelte:head>

<Modal {id} title={documentNumber} size={2} bodyCss="px-16 py-12"
  saveIcon="icon-[fa--print]" saveButtonLabel="Print|Imprimir" onSave={printTicket}
>
  <div class="flex justify-center mb-12">
    <CheckboxOptions type="single" useButtonsSlim
      options={TICKET_WIDTH_OPTIONS}
      keyId="ID" keyName="Name"
      save="columns" saveOn={ticketWidthForm}
    />
  </div>

  <!-- The preview is the same string the printer gets, so it wraps identically; only the
       type size differs, because 2.5 mm of paper is unreadable on a screen. -->
  <div class="flex justify-center">
    <pre class="_paper text-[14px] px-12 py-10" style:width="{ticketColumns}ch">{ticketText}</pre>
  </div>
</Modal>

{#if row}
  <!-- The printed copy lives at body level and not inside the dialog: the modal body carries a
       `transform`, which would make this element's `position: fixed` resolve against the dialog
       instead of the page. -->
  <Portal>
    <pre class="_printCopy"
      style:--ticket-paper-width="{paperWidthMm}mm"
      style:--ticket-side-padding="{sidePaddingMm}mm"
    >{ticketText}</pre>
  </Portal>
{/if}

<style>
  ._paper {
    font-family: "Courier New", Courier, "DejaVu Sans Mono", monospace;
    /* The inline width is the character grid — `{columns}ch` — so the padding has to sit
       outside it, or the widest rows lose their last characters to the box model. */
    box-sizing: content-box;
    line-height: 1.35;
    white-space: pre;
    color: #1f2937;
    background-color: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 4px;
    box-shadow: rgba(0, 0, 0, 0.08) 0 2px 8px;
    overflow-x: auto;
  }

  ._printCopy {
    display: none;
  }

  @media print {
    /* Everything on the page gives way to the ticket. visibility rather than display, because
       an ancestor set to `display: none` would take this element down with it. */
    :global(body *) {
      visibility: hidden !important;
    }
    ._printCopy {
      display: block;
      visibility: visible !important;
      position: fixed;
      top: 0;
      left: 0;
      margin: 0;
      width: var(--ticket-paper-width);
      padding: 2mm 0 6mm var(--ticket-side-padding);
      font-family: "Courier New", Courier, "DejaVu Sans Mono", monospace;
      /* 1.5 mm per character at a 0.6em advance, which is what makes 32 columns fill 48 mm. */
      font-size: 2.5mm;
      line-height: 1.25;
      white-space: pre;
      color: #000000;
      background-color: #ffffff;
    }
  }
</style>
