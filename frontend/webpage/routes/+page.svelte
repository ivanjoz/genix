<script lang="ts">
import { onMount } from 'svelte';
import MobileMenu from '$domain/MobileMenu.svelte';
import Header from '$ecommerce/components/Header.svelte';
import EcommerceRenderer from '$ecommerce/renderer/EcommerceRenderer.svelte';
import { getPageContent } from '$services/ecommerce/page-content.svelte';
import type { SectionData } from '$ecommerce/renderer/section-types';
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';

  // The storefront renders the content saved for the Inicio page (pageID 11) by
  // the builder. getPageContent loads those sections (Type/Ast/Content/Css/...);
  // EcommerceRenderer maps each to its registered section component.
  let sections = $state<SectionData[]>([]);

  // Runtime utility CSS for the runtime-authored Tailwind classes (slot CSS, HTML
  // section AST, text lines). These classes don't exist in source, so build-time
  // Tailwind can't cover them. The builder pre-generates this on save and stores
  // it per section, so the storefront injects it as-is — no UnoCSS at view time.
  let runtimeCss = $state('');

  // Matches the builder's default palette so `--color-N` vars resolve identically.
  const defaultPalette: ColorPalette = {
    id: 'default',
    name: 'Default Palette',
    colors: [
      '#f8fafc', '#f1f5f9', '#e2e8f0', '#cbd5e1', '#94a3b8',
      '#64748b', '#475569', '#334155', '#1e293b', '#0f172a'
    ]
  };

  onMount(async () => {
    const stored = await getPageContent();
    sections = stored.sections;
    runtimeCss = stored.css;
  });
</script>

<svelte:head>
  {#if runtimeCss}
    {@html `<style id="store-runtime-css">${runtimeCss}</style>`}
  {/if}
</svelte:head>

<Header />
<MobileMenu />
<EcommerceRenderer elements={sections} palette={defaultPalette} />
