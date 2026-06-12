<script lang="ts" module>
  export * from './section-types';
</script>

<script lang="ts">
  import type { SectionData } from './section-types';
  import HtmlSection from './HtmlSection.svelte';
  import { generatePaletteStyles } from './token-resolver';
  import type { ColorPalette } from './renderer-types';

  interface Props {
    elements: SectionData[];
    palette?: ColorPalette;
    isRoot?: boolean;
  }

  const {
    elements = [],
    palette,
    isRoot = true
  }: Props = $props();

  const paletteStyles = $derived(isRoot ? generatePaletteStyles(palette) : '');
</script>

<div class="ecommerce-render" style={paletteStyles}>
  {#each elements as element (element.id)}
    {#if element.Type === 'HtmlSection'}
      <HtmlSection
        ast={element.Ast}
        css={element.Css}
        {...element.Attributes}
      />
    {:else}
      <div class="bg-red-50 p-4 border border-red-200 text-red-600 my-4 mx-auto max-w-4xl rounded">
        <strong>Error:</strong> Unknown section type "<code>{element.Type}</code>".
        Only <code>HtmlSection</code> is supported.
      </div>
    {/if}
  {/each}
</div>
