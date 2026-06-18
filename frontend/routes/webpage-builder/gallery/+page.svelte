<script lang="ts">
  import ImageUploader from '$components/files/ImageUploader.svelte';
  import Image from '$components/files/Image.svelte';
  import FilterInput from '$components/form/FilterInput.svelte';
  import T from '$components/misc/T.svelte';
  import Page from '$domain/Page.svelte';
  import { ConfirmWarn, Loading, Notify } from '$libs/helpers';
  import { tr } from '$core/store.svelte';
  import { GalleryImagesService, type IGalleryImage } from '$services/webpage/gallery.svelte';

  const galleryImages = new GalleryImagesService(true);
  let filterText = $state('');

  // Gallery names are stored in Description; the generated image filename is not user-facing.
  const filteredImages = $derived(
    filterText
      ? galleryImages.records.filter((image) => image.Description.toLowerCase().includes(filterText))
      : galleryImages.records,
  );

  const removeImage = (image: IGalleryImage) => {
    ConfirmWarn(
      tr('DELETE IMAGE|ELIMINAR IMAGEN'),
      tr('Delete the selected gallery image?|¿Eliminar la imagen seleccionada?'),
      'SI',
      'NO',
      async () => {
        Loading.standard(tr('Deleting image...|Eliminando imagen...'));
        try {
          await galleryImages.remove(image.ImageID);
        } catch (error) {
          Notify.failure(tr('Could not delete the image.|No se pudo eliminar la imagen.'));
          console.error('[gallery] delete failed', { imageID: image.ImageID, error });
        } finally {
          Loading.remove();
        }
      },
    );
  };
</script>

<Page title="Gallery|Galería">
  <div class="mb-14 flex">
    <FilterInput
      bind:value={filterText}
      css="w-full max-w-220"
      icon="icon-[fa--search]"
      label="Search gallery images|Buscar imágenes de galería"
      placeholder="Search by name...|Buscar por nombre..."
    />
  </div>

  <div class="gallery-grid" aria-label="Website image gallery">
    <!-- Upload card converts locally; the endpoint persists only the x8 and x4 AVIF variants. -->
    <ImageUploader
      saveAPI="gallery-image"
      refreshRoutes={['gallery-images']}
      useConvertAvif={true}
      imageCounterConfig={4}
      convertResolutions={[8, 4]}
      clearOnUpload={true}
      types={['avif']}
      folder="img-galeria"
      size={4}
      cardCss="w-full h-190 p-4"
      processName={tr('Gallery image|Imagen de galería')}
      onUploaded={(image) => {
        // Render from ImageUploader's in-memory preview while the upload finishes.
        galleryImages.addSavedRecords({
          ID: image.id,
          ImageID: image.id,
          Image: image.name,
          Description: image.description || '',
          ss: 1,
          upd: 0,
        });
      }}
    />

    {#each filteredImages as image (image.ImageID)}
      <article class="gallery-card" title={image.Description || `Image ${image.ImageID}`}>
        <Image
          src={image.Image}
          types={['avif']}
          folder="img-galeria"
          size={4}
          alt={image.Description}
          css="gallery-card-image"
          onRemove={() => removeImage(image)}
        />
        <div class="gallery-card-footer">
          <p>{image.Description || tr('Gallery image|Imagen de galería')}</p>
        </div>
      </article>
    {/each}
  </div>

  {#if galleryImages.isReady > 0 && galleryImages.records.length === 0}
    <p class="p-20 text-center c-steel"><T text="Upload your first gallery image.|Sube tu primera imagen a la galería." /></p>
  {:else if filterText && filteredImages.length === 0}
    <p class="p-20 text-center c-steel"><T text="No images match that name.|No hay imágenes con ese nombre." /></p>
  {/if}
</Page>

<style>
  .gallery-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(210px, 240px));
    gap: 14px;
    align-items: start;
  }

  .gallery-card {
    overflow: hidden;
    border: 1px solid rgba(100, 116, 139, 0.22);
    border-radius: 10px;
    background: white;
    box-shadow: rgba(15, 23, 42, 0.08) 0 2px 5px;
    transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  }

  .gallery-card:hover {
    border-color: rgba(79, 70, 229, 0.28);
    box-shadow: rgba(15, 23, 42, 0.16) 0 8px 20px -6px;
    transform: translateY(-2px);
  }

  :global(.gallery-card-image) {
    width: 100%;
    aspect-ratio: 4 / 3;
    border: 0;
    border-radius: 0;
    background: #f8fafc;
  }

  .gallery-card-footer {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 38px;
    padding: 7px 9px;
    border-top: 1px solid rgba(100, 116, 139, 0.14);
  }

  .gallery-card-footer p {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    color: #475569;
    line-height: 18px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

</style>
