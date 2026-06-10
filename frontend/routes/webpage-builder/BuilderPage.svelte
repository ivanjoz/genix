<script lang="ts">
import EcommerceBuilder from './builder/EcommerceBuilder.svelte';
import { storeExample } from './builder/store-example';
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
import type { SectionData } from '$ecommerce/renderer/section-types';
import { getPageContent, setCurrentPageID } from '$services/ecommerce/page-content.svelte';
import { editorStore } from '$ecommerce/stores/editor.svelte';
import '$domain/libs/fontello-prerender.css';
import Header from '$ecommerce/components/Header.svelte';
import Page from '$domain/Page.svelte';

  interface Props {
    // The PageID to load/save. 0 = the default Inicio page (resolved server-side).
    pageID?: number;
  }
  let { pageID = 0 }: Props = $props();

  let elements = $state<SectionData[]>([]);
  let values = $state<Record<string, string>>({});

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

    getPageContent(id).then((stored) => {
      if (token !== loadToken) return; // a newer load superseded this one
      elements = stored.sections.length > 0
        ? stored.sections
        : (id === 0 ? storeExample : []);
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
  <Header />
  <EcommerceBuilder
    bind:elements
    bind:values
    palette={defaultPalette}
    onUpdate={handleUpdate}
  />
</Page>
