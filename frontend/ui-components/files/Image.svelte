<script lang="ts">
  import { Env } from '$core/env';
  import { getInMemoryImageBase64, isImageInFlight } from '$core/inMemoryImages.svelte';
  import { tr } from '$core/store.svelte';
  import s1 from '../components.module.css';

  interface ImageProps {
    src: string;
    folder?: string;
    size?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9;
    types?: string[];
    alt?: string;
    css?: string;
    imageCss?: string;
    onRemove?: () => void;
  }

  let {
    src,
    folder = '',
    size,
    types = [],
    alt = '',
    css = '',
    imageCss = '',
    onRemove,
  }: ImageProps = $props();

  // Optimistic images render from memory until their converted bytes reach the CDN.
  const liveBase64 = $derived(getInMemoryImageBase64(src));
  const isInFlight = $derived(isImageInFlight(src));

  const makeImageSrc = (format?: string): string => {
    if (liveBase64) return liveBase64;
    if (!src) return '';

    const isAbsolute = src.startsWith('https://') || src.startsWith('http://');
    let imageURL = isAbsolute
      ? src
      : Env.makeCDNRoute(folder, size ? `${src}-x${size}` : src);
    if (format) imageURL += `.${format}`;
    return imageURL;
  };
</script>

<div
  class="group relative overflow-hidden rounded-[10px] border border-gray-500 bg-white cursor-default {css}"
>
  {#if src}
    <picture class="contents">
      {#if !liveBase64 && types.includes('avif')}
        <source type="image/avif" srcset={makeImageSrc('avif')} />
      {/if}
      {#if !liveBase64 && types.includes('webp')}
        <source type="image/webp" srcset={makeImageSrc('webp')} />
      {/if}
      <img
        class="absolute inset-0 block w-full h-full object-cover {imageCss}"
        src={makeImageSrc(liveBase64 ? undefined : types[0])}
        {alt}
        loading="lazy"
      />
    </picture>
  {/if}

  {#if isInFlight}
    <div
      class="absolute top-9 right-9 z-4 w-20 h-20 pointer-events-none"
      title={tr('Processing image...|Procesando imagen...')}
    >
      <span class={s1.card_image_corner_loader_ring}></span>
    </div>
  {/if}

  {#if onRemove}
    <button
      class={s1.image_remove_button}
      type="button"
      aria-label={tr('Delete image|Eliminar imagen')}
      onclick={(event) => {
        event.stopPropagation();
        onRemove?.();
      }}
    >
      <i class="icon-cancel"></i>
    </button>
  {/if}
</div>
