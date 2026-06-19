<script lang="ts">
import { GET } from '$libs/http.svelte';
import FilterInput from '$components/form/FilterInput.svelte';
import Button from '$components/buttons/Button.svelte';
import { ImageAssetsService } from '$services/business/image-assets.svelte';
import { Loading, Notify } from '$libs/helpers';

// One text-search hit: the API returns only the image id and its GenixSearch
// weight; the thumbnail URL is resolved from the image-assets delta cache.
interface IImageSearchHit {
  id: number
  w: number
}

// Image asset cache holds CategoryID + categories, needed to build thumbnail URLs.
const imageAssets = new ImageAssetsService(true)

let query = $state("")
let hits = $state([] as IImageSearchHit[])
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
    // GET.image-asset-text-search returns the top ids + weights ordered by relevance.
    const result = await GET({
      route: `image-asset-text-search?q=${encodeURIComponent(trimmedQuery)}&limit=10`,
      errorMessage: "Error en la búsqueda de imágenes.",
    })
    hits = Array.isArray(result) ? result : []
    elapsedMs = Math.round(performance.now() - startedAt)
    lastQuery = trimmedQuery
  } finally {
    Loading.remove()
  }
}

// Resolve the cached image record to build its thumbnail URL; empty while the
// cache is still loading or when the asset is unknown.
const thumbnailURL = (imageID: number): string => {
  const record = imageAssets.get(imageID)
  return record ? imageAssets.getThumbnailURL(record) : ''
}
</script>

<div class="h-full">
  <div class="flex items-center gap-12 mb-8" aria-label="Stock image search tester toolbar">
    <FilterInput bind:value={query} css="w-320" placeholder="Buscar imagen por palabras…" />
    <Button color="blue" name="Buscar" icon="icon-[fa--search]"
      label="Runs the image asset text search against the GenixSearch index."
      onClick={runSearch} />
    {#if lastQuery}
      <span class="ml-auto fs14 text-slate-500">
        {hits.length} resultados para "{lastQuery}" · {elapsedMs} ms
      </span>
    {/if}
  </div>

  {#if hits.length > 0}
    <div class="flex flex-wrap gap-12" aria-label="Top image results">
      {#each hits as hit (hit.id)}
        <div class="flex flex-col items-center w-160 p-8 bd-1 br-8 bg-white">
          {#if thumbnailURL(hit.id)}
            <img src={thumbnailURL(hit.id)} alt={`Imagen ${hit.id}`}
              class="w-144 h-144 object-cover br-6" loading="lazy" />
          {:else}
            <div class="w-144 h-144 br-6 bg-slate-100 flex items-center justify-center text-slate-400 fs14">
              #{hit.id}
            </div>
          {/if}
          <span class="mt-6 fs14 text-slate-500 ff-mono">w: {hit.w}</span>
        </div>
      {/each}
    </div>
  {:else if lastQuery}
    <div class="p-16 text-center text-slate-500">No se encontraron imágenes.</div>
  {/if}
</div>
