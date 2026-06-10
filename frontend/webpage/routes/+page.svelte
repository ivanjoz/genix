<script lang="ts">
import { onMount } from 'svelte';
import MobileMenu from '$domain/MobileMenu.svelte';
import Header from '$ecommerce/components/Header.svelte';
import EcommerceRenderer from '$ecommerce/renderer/EcommerceRenderer.svelte';
import { getStoreWebpage } from '$services/ecommerce/page-content.svelte';
import type { SectionData } from '$ecommerce/renderer/section-types';
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';

  let { data } = $props();

  // The storefront renders the root/Inicio page content. It comes from +page.ts
  // load() so it's baked into the prerendered HTML (SEO); the onMount refresh below
  // then pulls the latest content for real users. EcommerceRenderer maps each
  // section to its registered component.
  let sections = $state<SectionData[]>(data?.sections ?? []);

  // SEO metatags for this page (from the same p-webpage call). Baked into the
  // prerendered <head> for crawlers; in dev (CSR) the load runs client-side.
  const seo = $derived<Record<string, string>>(data?.seo ?? {});

  // Runtime utility CSS for the runtime-authored Tailwind classes (slot CSS, HTML
  // section AST, text lines). These classes don't exist in source, so build-time
  // Tailwind can't cover them. The builder pre-generates this on save and stores
  // it per section, so the storefront injects it as-is — no UnoCSS at view time.
  let runtimeCss = $state(data?.css ?? '');

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
    // Refresh to the latest content after the prerendered paint (and recover if the
    // build-time fetch was empty). No hydration mismatch: the initial render uses
    // `data` (exactly what SSR rendered); this runs only afterwards.
    try {
      const stored = await getStoreWebpage();
      if (stored.sections.length > 0) {
        sections = stored.sections;
        runtimeCss = stored.css;
      }
    } catch (contentRefreshError) {
      console.error('[StorePage] content refresh failed', contentRefreshError);
    }
  });
</script>

<svelte:head>
  {#if seo.title}<title>{seo.title}</title>{/if}
  {#if seo.description}<meta name="description" content={seo.description} />{/if}
  {#if seo.keywords}<meta name="keywords" content={seo.keywords} />{/if}
  {#if seo.ogTitle}<meta property="og:title" content={seo.ogTitle} />{/if}
  {#if seo.ogDescription}<meta property="og:description" content={seo.ogDescription} />{/if}
  {#if seo.ogImage}<meta property="og:image" content={seo.ogImage} />{/if}
  {#if seo.favicon}<link rel="icon" href={seo.favicon} />{/if}
  {#if runtimeCss}
    {@html `<style id="store-runtime-css">${runtimeCss}</style>`}
  {/if}
</svelte:head>

<Header />
<MobileMenu />
<EcommerceRenderer elements={sections} palette={defaultPalette} />
