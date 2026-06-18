<script lang="ts">
  import Modal from '$components/layers/Modal.svelte';
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
  import ImageUploader from '$components/files/ImageUploader.svelte';
  import FilterInput from '$components/form/FilterInput.svelte';
  import SearchSelect from '$components/form/SearchSelect.svelte';
  import { closeModal, openModals, tr } from '$core/store.svelte';
  import { GalleryImagesService } from '$services/webpage/gallery.svelte';
  import { ImageAssetsService } from '$services/business/image-assets.svelte';

  interface Props {
    /** Numeric id the host opens via openModal() to show this picker. */
    modalId: number;
    /** Receives the full-resolution CDN url of the chosen image. */
    onSelect: (url: string) => void;
  }
  let { modalId, onSelect }: Props = $props();

  // Two image sources: the user-uploaded gallery (with upload card) and the stock CDN catalog.
  const galleryImages = new GalleryImagesService();
  const stockImages = new ImageAssetsService();

  // Active source tab: 1 = user gallery, 2 = stock CDN images.
  let activeSource = $state(1);
  const sourceOptions = [[1, 'Gallery|Galería'], [2, 'Stock Images|Stock Imágenes']] as [number, string][];

  // Fetch each source once, lazily, the first time the modal opens (both are delta-cached).
  let hasLoaded = false;
  $effect(() => {
    if (!openModals.includes(modalId) || hasLoaded) { return; }
    hasLoaded = true;
    galleryImages.fetchCached().catch((error) => console.error('[image-picker] gallery load failed', error));
    stockImages.fetchCached().catch((error) => console.error('[image-picker] stock load failed', error));
  });

  // Gallery search: names live in Description; FilterInput already lowercases/trims the value.
  let gallerySearch = $state('');
  const filteredGallery = $derived(
    gallerySearch
      ? galleryImages.records.filter((image) => (image.Description || '').toLowerCase().includes(gallerySearch))
      : galleryImages.records
  );

  // Stock category filter: 0 = all categories (the prepended default option).
  let selectedCategoryID = $state(0);
  const categoryOptions = $derived([
    { ID: 0, Name: tr('All categories|Todas las categorías') },
    ...stockImages.categories,
  ]);

  // Stock thumbnails, filtered by category and bounded for the initial view.
  const stockThumbnails = $derived(stockImages.records
    .filter((record) => !selectedCategoryID || record.CategoryID === selectedCategoryID)
    .map((record) => ({ record, url: stockImages.getThumbnailURL(record) }))
    .filter((image) => image.url)
    .slice(0, 60)
  );

  // Apply the chosen url to the host node and dismiss the picker.
  const pickImage = (url: string) => {
    if (!url) { return; }
    onSelect(url);
    closeModal(modalId);
  };
</script>

<Modal id={modalId} title="Select image|Seleccionar imagen" size={6} onClose={() => closeModal(modalId)}>
  <!-- Tabs on the left; the per-source filter (search / category) sits on the right. -->
  <div class="mb-8 flex items-center gap-12 relative">
    <div class="min-w-240 relative shrink-0">
      <OptionsStrip options={sourceOptions} selected={activeSource} onSelect={(option) => (activeSource = option[0])} />
    </div>
    <div class="ml-8 relative">
      {#if activeSource === 1}
        <FilterInput bind:value={gallerySearch} css="w-240" icon="icon-[fa--search]" placeholder="Search by name...|Buscar por nombre..." />
      {:else}
        <SearchSelect css="w-280" 
          keyId="ID"
          keyName="Name"
          options={categoryOptions}
          selected={selectedCategoryID}
          onChange={(category) => (selectedCategoryID = category?.ID ?? 0)}
          placeholder="All categories|Todas las categorías"
        />
      {/if}
    </div>
  </div>

  {#if activeSource === 1}
    <div class="picker-grid p-4">
      <!-- Upload a new gallery image in place; once saved it appears in this same grid. -->
      <ImageUploader
        saveAPI="gallery-image"
        refreshRoutes={['gallery-images']}
        useConvertAvif={true}
        convertResolutions={[8, 4]}
        imageCounterConfig={4}
        clearOnUpload={true}
        types={['avif']}
        folder="img-galeria"
        size={4}
        cardCss="w-full aspect-[4/3]"
        processName={tr('Gallery image|Imagen de galería')}
        onUploaded={(image) => galleryImages.addSavedRecords({
          ID: image.id, ImageID: image.id, Image: image.name,
          Description: image.description || '', ss: 1, upd: 0,
        })}
      />
      {#each filteredGallery as image (image.ImageID)}
        <button type="button" class="picker-card" title={image.Description} onclick={() => pickImage(galleryImages.imageURL(image, 8))}>
          <img src={galleryImages.imageURL(image, 4)} alt={image.Description} loading="lazy" />
        </button>
      {/each}
    </div>
  {:else}
    <div class="picker-grid p-4">
      {#each stockThumbnails as image (image.record.ID)}
        <button type="button" class="picker-card" onclick={() => pickImage(stockImages.getImageURL(image.record))}>
          <img src={image.url} alt={`Image ${image.record.ID}`} loading="lazy" />
        </button>
      {/each}
    </div>
  {/if}
</Modal>

<style>
  .picker-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
    max-height: 62vh;
    overflow-y: auto;
  }

  .picker-card {
    aspect-ratio: 4 / 3;
    padding: 0;
    overflow: hidden;
    border: 1px solid rgba(100, 116, 139, 0.25);
    border-radius: 8px;
    background: #f1f5f9;
    cursor: pointer;
    transition: border-color 0.15s, transform 0.15s;
  }
  .picker-card:hover {
    border-color: #6366f1;
    transform: translateY(-2px);
  }
  .picker-card img {
    width: 100%;
    height: 100%;
    display: block;
    object-fit: cover;
  }
</style>
