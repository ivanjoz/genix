<script lang="ts" module>

  export const schema: SectionSchema = {
    name: 'Product Grid Simple',
    description: 'A simple grid display for a list of products.',
    category: 'products',
    content: [
      'title',
      'productosIDs',
      'limit'
    ],
    css: ['container', 'title', 'grid']
  };
</script>

<script lang="ts">
  import ProductCard from '$store/components/ProductCard.svelte';
    import type { SectionSchema, StandardContent } from '$store/renderer/section-types';

  interface Props {
    content: StandardContent;
    css: Record<string, string>;
  }

  let { content, css }: Props = $props();

  const products = $derived((content.productosIDs || []).slice(0, content.limit || 8));
</script>

<section class="py-16 px-6 max-w-7xl mx-auto {css.container || ''}">
  {#if content.title}
    <h2 class="text-3xl font-bold mb-10 text-center {css.title || ''}">{content.title}</h2>
  {/if}

  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8 {css.grid || ''}">
    {#each products as productoID}
      <ProductCard {productoID} />
    {/each}
  </div>
</section>
