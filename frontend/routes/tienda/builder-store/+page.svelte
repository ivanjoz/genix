<script lang="ts">
    import EcommerceBuilder from './EcommerceBuilder.svelte';
    import { storeExample } from './store-example';
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
import type { SectionData } from '$ecommerce/renderer/section-types';
    import "$domain/libs/fontello-prerender.css";
    import Header from '$ecommerce/components/Header.svelte';
    import Page from '$domain/Page.svelte';

    let elements = $state<SectionData[]>(storeExample);
    let values = $state<Record<string, string>>({});

    const defaultPalette: ColorPalette = {
        id: 'default',
        name: 'Default Palette',
        colors: [
            '#f8fafc', // 50
            '#f1f5f9', // 100
            '#e2e8f0', // 200
            '#cbd5e1', // 300
            '#94a3b8', // 400
            '#64748b', // 500
            '#475569', // 600
            '#334155', // 700
            '#1e293b', // 800
            '#0f172a'  // 900
        ]
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
