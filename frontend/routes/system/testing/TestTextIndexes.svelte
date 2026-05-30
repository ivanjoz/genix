<script lang="ts">
import { GET } from '$libs/http.svelte';
import VTable from '$components/vTable/VTable.svelte';
import type { ITableColumn } from '$components/vTable/types';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import RecordByIDText from '$components/misc/RecordByIDText.svelte';
import { Loading, Notify } from '$libs/helpers';

// One text-search hit: only the product id and its GenixSearch weight come
// back from the API; the name is resolved per-row from the by-id cache.
interface ITextSearchHit {
  id: number
  w: number
}

let query = $state("")
let hits = $state([] as ITextSearchHit[])
let elapsedMs = $state(0)
let lastQuery = $state("")

const runSearch = async () => {
  const trimmedQuery = query.trim()
  if (trimmedQuery.length < 2) {
    Notify.failure("Ingresa al menos 2 caracteres para buscar.")
    return
  }

  Loading.standard("Buscando…")
  const startedAt = performance.now()
  try {
    // GET.product-text-search returns ids + weights ordered by relevance.
    const result = await GET({
      route: `product-text-search?q=${encodeURIComponent(trimmedQuery)}&limit=100`,
      errorMessage: "Error en la búsqueda de texto.",
    })
    hits = Array.isArray(result) ? result : []
    elapsedMs = Math.round(performance.now() - startedAt)
    lastQuery = trimmedQuery
  } finally {
    Loading.remove()
  }
}

const columns: ITableColumn<ITextSearchHit>[] = [
  {
    header: "Product ID",
    headerCss: "w-140",
    css: "text-center ff-mono",
    getValue: (row) => row.id,
  },
  {
    // The cellRenderer snippet fires only for columns with an `id`.
    header: "Nombre",
    id: "nombre",
    highlight: true,
    css: "px-6",
    getValue: (row) => String(row.id),
  },
  {
    header: "Weight",
    headerCss: "w-140",
    css: "text-center ff-mono",
    getValue: (row) => row.w,
  },
]
</script>

<div class="h-full">
  <div class="flex items-center gap-12 mb-8" aria-label="Text search tester toolbar">
    <FilterInput bind:value={query} css="w-320" placeholder="Buscar producto por nombre…" />
    <Button color="blue" name="Buscar" icon="icon-search"
      label="Runs the product text search against the GenixSearch index."
      onClick={runSearch} />
    {#if lastQuery}
      <span class="ml-auto fs14 text-slate-500">
        {hits.length} resultados para "{lastQuery}" · {elapsedMs} ms
      </span>
    {/if}
  </div>

  <VTable
    columns={columns}
    data={hits}
    css="w-full"
    maxHeight="calc(80vh - 13rem)"
  >
    {#snippet cellRenderer(record, column)}
      {#if column.id === 'nombre'}
        <RecordByIDText apiRoute="p-productos-ids" recordID={record.id} placeholder="-" />
      {/if}
    {/snippet}
  </VTable>
</div>
