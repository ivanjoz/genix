<script lang="ts">
  // Per-page builder route: /webpage-builder/<PageID>. The PageID comes from the
  // pages list (pencil button) and selects which page's content to load/save.
  // The bare /webpage-builder route redirects here (see ../+page.ts).
  import { page } from '$app/state';
  import EcommerceBuilder from '../builder/EcommerceBuilder.svelte';
  import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
  import type { SectionData } from '$ecommerce/renderer/section-types';
  import { getPageContent, setCurrentPageID } from '$services/ecommerce/page-content.svelte';
  import { editorStore } from '../stores/editor.svelte';
  import '$domain/libs/fontello-prerender.css';
  import Header from '$ecommerce/components/Header.svelte';
  import Page from '$domain/Page.svelte';
  import T from '$components/misc/T.svelte';

  // params.pageID is a string; coerce to a number (0 falls back to the default page).
  const pageID = $derived(Number(page.params.pageID) || 0);

  let elements = $state<SectionData[]>([]);
  let values = $state<Record<string, string>>({});
  let loading = $state(false);

  // Load (re-load) the requested page whenever pageID changes — SvelteKit reuses
  // this component when navigating between /webpage-builder/<id> pages, so onMount
  // would fire only once. The editorStore is a singleton shared across builder
  // routes, so it must be reset before loading or it would keep showing the
  // previously opened page's sections (it only auto-loads when empty — see
  // EcommerceBuilder). A per-run token discards a stale response if pageID changes
  // again before the fetch resolves.
  let loadToken = 0;
  $effect(() => {
    const id = pageID;
    const token = ++loadToken;
    setCurrentPageID(id);
    editorStore.select(null);
    editorStore.sections = [];
    elements = [];
    loading = true;

    getPageContent(id).then((stored) => {
      if (token !== loadToken) return; // a newer load superseded this one
      elements = stored.sections;
      loading = false;
    });
  });

  const defaultPalette: ColorPalette = {
    id: 'default',
    name: 'Default Palette',
    colors: [
      '#f8fafc', '#f1f5f9', '#e2e8f0', '#cbd5e1', '#94a3b8',
      '#64748b', '#475569', '#334155', '#1e293b', '#0f172a',
    ],
  };

  function handleUpdate(updated: SectionData[]) {
    console.log('Store updated:', updated);
  }
</script>

<Page title="Builder" containerCss="p-0! w-[calc(100%-280px)]!" useTopMinimalMenu fixedFullHeight>
  <!-- The storefront chrome is desktop-styled and lives outside the canvas. In mobile
       preview it would sit, full-width, atop a 390px body — so hide it; the iframe shows
       the page body on its own. -->
  {#if editorStore.viewMode !== 'mobile'}
    <Header />
  {/if}
  {#if loading}
    <div class="builder-loader">
      <div class="loader-spinner"></div>
      <span><T text="Loading page content...|Obteniendo contenido de la página..." /></span>
    </div>
  {:else}
    <EcommerceBuilder
      bind:elements
      bind:values
      palette={defaultPalette}
      onUpdate={handleUpdate}
    />
  {/if}
</Page>

<style>
  .builder-loader {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    width: 100%;
    height: 100%;
    color: #475569;
    font-size: 15px;
  }
  .loader-spinner {
    width: 38px;
    height: 38px;
    border: 4px solid #e2e8f0;
    border-top-color: #64748b;
    border-radius: 50%;
    animation: builder-loader-spin 0.8s linear infinite;
  }
  @keyframes builder-loader-spin {
    to { transform: rotate(360deg); }
  }
</style>
