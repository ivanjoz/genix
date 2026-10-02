<script lang="ts">
import DateInput from '#components/form/DateInput.svelte'
import Input from '#components/form/Input.svelte'
import SearchSelect from '#components/form/SearchSelect.svelte'
import T from '#components/misc/T.svelte'
import { tr } from '#core/store.svelte.ts'
import { formatN } from '#libs/helpers.ts'
import {
  PURCHASE_DOC_TYPES, purchaseDocTypeCreditsTax, purchaseDocumentTotal, splitPurchaseTotal,
  type IPurchaseDocument,
} from '#core/purchase-document.ts'

// The supplier's comprobante on a purchase order, an expense or a fixed asset — the Registro de
// Compras row. The three forms render this block; the backend validates it with one rule set
// (finance.NormalizePurchaseDocument).
interface Props {
  document: Partial<IPurchaseDocument>
  // Purchase orders carry their own currency on the document. Expenses and assets already ask
  // for it in their own form, and the document shares that column.
  showCurrency?: boolean
  // The amount the record says was paid; the split button proposes the amounts from it.
  expectedTotal?: number
  // The document is the bill for exactly that amount; flags the mismatch while editing.
  totalMustMatch?: boolean
  // Purchase orders: the total is the order's and is shown read-only. Only No Gravado and Otros
  // Cargos are typed; base and IGV (or, without crédito fiscal, No Gravado) are the remainder, so
  // the amounts always add up to the total and there is no split button.
  isTotalFixed?: boolean
  currencyOptions?: { id: number, name: string }[]
}

const {
  document, showCurrency = false, expectedTotal = 0, totalMustMatch = true, isTotalFixed = false,
  currencyOptions = [],
}: Props = $props()

const docTypeOptions = PURCHASE_DOC_TYPES.map(option => ({ id: option.id, name: tr(option.name) }))
const CURRENCY_USD = 2

const documentTotal = $derived(purchaseDocumentTotal(document))
const isTotalMismatch = $derived(
  totalMustMatch && !!document.DocType && expectedTotal > 0 && documentTotal !== expectedTotal,
)
const creditsTax = $derived(purchaseDocTypeCreditsTax(document.DocType || 0))

// Input re-reads its value only when saveOn is replaced or dependencyValue changes, so each split
// bumps this counter to refresh the amount fields that stay mounted.
let splitVersion = $state(0)

function splitExpectedTotal() {
  Object.assign(document, splitPurchaseTotal(document.DocType || 0, expectedTotal))
  splitVersion++
}

// With a fixed total the calculated amounts follow the typed ones and the total.
const fixedTotalAmounts = $derived(isTotalFixed && document.DocType
  ? splitPurchaseTotal(document.DocType, expectedTotal, document.UntaxedAmount || 0, document.OtherAmount || 0)
  : null)

$effect(() => {
  if (fixedTotalAmounts) { Object.assign(document, fixedTotalAmounts) }
})
</script>

{#snippet calculatedAmount(label: string, amount: number)}
  <Input label={label} saveOn={{ Amount: formatN(amount / 100, 2) }} save="Amount" disabled={true}
    css="col-span-12 md:col-span-6" inputCss="text-right" />
{/snippet}

<div class="grid grid-cols-24 items-start gap-x-10 gap-y-10" aria-label="Supplier comprobante">
  <SearchSelect label="Comprobante|Comprobante"
    saveOn={document}
    css="col-span-24 md:col-span-12"
    save="DocType"
    keyId="id"
    keyName="name"
    placeholder={tr("No comprobante|Sin comprobante")}
    options={docTypeOptions}
    onChange={() => { if (isTotalFixed) { splitExpectedTotal() } }}
  />
  {#if document.DocType}
    <Input label="Series|Serie" saveOn={document} save="DocSeries" css="col-span-8 md:col-span-6" />
    <Input label="Number|Número" saveOn={document} save="DocNumber" css="col-span-16 md:col-span-6"
      type="number" inputCss="text-right" />
    {#if showCurrency}
      <SearchSelect label="Currency|Moneda" saveOn={document} save="CurrencyType"
        css="col-span-12 md:col-span-6" keyId="id" keyName="name" options={currencyOptions} />
    {/if}
    {#if document.CurrencyType === CURRENCY_USD}
      <!-- RCE field 27: the SBS weighted-average selling rate of the issue date. -->
      <Input label="Exchange rate|Tipo de cambio" saveOn={document} save="ExchangeRate"
        css="col-span-12 md:col-span-6" type="number" baseDecimals={3} inputCss="text-right" />
    {/if}
    <DateInput label="Issue date|Fecha de Emisión" saveOn={document} save="DocIssueDate"
      css="col-span-24 md:col-span-6" />
    {#if creditsTax && isTotalFixed}
      {@render calculatedAmount("Taxable base|Base Imponible", document.TaxableAmount || 0)}
      {@render calculatedAmount("IGV", document.TaxAmount || 0)}
    {:else if creditsTax}
      <Input label="Taxable base|Base Imponible" saveOn={document} save="TaxableAmount"
        css="col-span-12 md:col-span-6" type="number" baseDecimals={2} inputCss="text-right"
        dependencyValue={splitVersion} />
      <Input label="IGV" saveOn={document} save="TaxAmount"
        css="col-span-12 md:col-span-6" type="number" baseDecimals={2} inputCss="text-right"
        dependencyValue={splitVersion} />
    {/if}
    {#if !creditsTax && isTotalFixed}
      {@render calculatedAmount("Untaxed|No Gravado", document.UntaxedAmount || 0)}
    {:else}
      <Input label="Untaxed|No Gravado" saveOn={document} save="UntaxedAmount"
        css="col-span-12 md:col-span-6" type="number" baseDecimals={2} inputCss="text-right"
        dependencyValue={splitVersion} />
    {/if}
    <Input label="Other charges|Otros Cargos" saveOn={document} save="OtherAmount"
      css="col-span-12 md:col-span-6" type="number" baseDecimals={2} inputCss="text-right"
      dependencyValue={splitVersion} />
    {#if isTotalFixed}
      <Input label="Total|Total" saveOn={{ Total: expectedTotal > 0 ? formatN(expectedTotal / 100, 2) : 0 }}
        save="Total" disabled={true} css="col-span-12 md:col-span-6" inputCss="text-right" />
    {:else}
    <div class="col-span-24 flex flex-wrap items-center gap-12 text-sm">
      <div>
        <T text="Document total|Total del comprobante" />:
        <span class="ff-mono font-semibold {isTotalMismatch ? 'text-red-600' : ''}">{formatN(documentTotal / 100, 2)}</span>
      </div>
      {#if isTotalMismatch}
        <span class="text-red-600">
          <T text="Must equal|Debe ser igual a" /> {formatN(expectedTotal / 100, 2)}
        </span>
      {/if}
      {#if expectedTotal > 0}
        <button type="button" class="bx-blue px-10 py-4" onclick={splitExpectedTotal}>
          <T text="Split the total|Desglosar el total" />
        </button>
      {/if}
    </div>
    {/if}
  {/if}
</div>
