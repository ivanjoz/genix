<script lang="ts">
  import { onMount } from 'svelte';
  import T from '$components/misc/T.svelte';
  import { ImageAssetsService } from '$services/negocio/image-assets.svelte';

  const imageAssets = new ImageAssetsService();
  let isLoading = $state(true);
  let loadError = $state('');

  // Show a bounded initial catalog; search and pagination can expand this later.
  const galleryImages = $derived.by(() => imageAssets.records
    .toSorted((left, right) => left.ID - right.ID)
    .map((record) => ({ record, url: imageAssets.getThumbnailURL(record) }))
    .filter((image) => image.url)
    .slice(0, 24)
  );

  onMount(async () => {
    console.debug('[image-gallery] loading initial image assets');
    try {
      await imageAssets.fetchCached();
      console.debug(`[image-gallery] loaded thumbnails=${galleryImages.length}`);
    } catch (fetchError) {
      loadError = 'No se pudieron listar las imágenes.';
      console.error('[image-gallery] failed to load image assets', fetchError);
    } finally {
      isLoading = false;
    }
  });
</script>

<div class="gallery-tab">
  {#if isLoading}
    <p class="status-message"><T text="Listing images...|Listando Imágenes..." /></p>
  {:else if loadError}
    <p class="status-message error-message">{loadError}</p>
  {:else if galleryImages.length}
    <div class="images-grid" aria-label="Image asset gallery">
      {#each galleryImages as image (image.record.ID)}
        <div class="image-card" title={`${imageAssets.categoriesMap.get(image.record.CategoryID)?.Name || ''} #${image.record.ID}`}>
          <img
            src={image.url}
            alt={`Image ${image.record.ID}`}
            loading="lazy"
          />
        </div>
      {/each}
    </div>
  {:else}
    <p class="status-message"><T text="No images available.|No hay imágenes disponibles." /></p>
  {/if}
</div>

<style>
  .gallery-tab {
    width: 100%;
  }

  .images-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .image-card {
    aspect-ratio: 4 / 3;
    overflow: hidden;
    border: 1px solid #334155;
    border-radius: 8px;
    background: #1e293b;
  }

  .image-card img {
    width: 100%;
    height: 100%;
    display: block;
    object-fit: cover;
  }

  .status-message {
    padding: 32px 12px;
    margin: 0;
    color: #94a3b8;
    text-align: center;
  }

  .error-message {
    color: #fca5a5;
  }
</style>
