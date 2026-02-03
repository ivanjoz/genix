<script lang="ts" module>
  import type { SectionSchema } from '../../../renderer/section-types';

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
  import type { StandardContent } from '../../../renderer/section-types';
  import ProductCard from '$store/components/ProductCard.svelte';

  interface Props {
    content: StandardContent;
    css: Record<string, string>;
  }

  let { content, css }: Props = $props();

  const containerClass = $derived(css.container || 'py-16 px-6 max-w-7xl mx-auto');
  const titleClass = $derived(css.title || 'text-3xl font-bold mb-10 text-center');
  const gridClass = $derived(css.grid || 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8');
  
  const products = $derived((content.productosIDs || []).slice(0, content.limit || 8));
</script>

<section class={containerClass}>
  {#if content.title}
    <h2 class={titleClass}>{content.title}</h2>
  {/if}

  <div class={gridClass}>
    {#each products as productoID}
      <ProductCard {productoID} />
    {/each}
  </div>
</section>
