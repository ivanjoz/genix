<script lang="ts">
    import EcommerceBuilder from './EcommerceBuilder.svelte';
    import { storeExample } from './store-example';
    import type { ColorPalette, ComponentAST } from '../../pkg-store/renderer/renderer-types';
    import Header from '$store/components/Header.svelte';
    import Page from '$ui/Page.svelte';

    let elements = $state<ComponentAST[]>(storeExample);
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

    function handleUpdate(updated: ComponentAST | ComponentAST[]) {
        console.log('Store updated:', updated);
    }
</script>

<Page title="Builder" containerCss="p-0! w-[calc(100%-260px)]!" useTopMinimalMenu fixedFullHeight>
    <Header />
    <EcommerceBuilder 
        bind:elements 
        bind:values 
        palette={defaultPalette} 
        onUpdate={handleUpdate}
    />
</Page>
