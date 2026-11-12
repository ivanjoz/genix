<script lang="ts">
	import { getEcommerceCatalog, type EcommerceCatalog } from '#ecommerce/services/ecommerce-catalog.svelte.ts';
	import type { IProductCategory } from '#ecommerce/services/ecommerce-catalog.ts';

	export interface ICategoryDescription {
		css?: string;
		categoryIDs?: number[];
	}

	const { css = "", categoryIDs = [] }: ICategoryDescription = $props();

	let catalog = $state<EcommerceCatalog | null>(null);

	getEcommerceCatalog().then((loaded) => { catalog = loaded; });

	// Reactive: resolves once the catalog (and its delta) is available.
	const categoria = $derived<IProductCategory | undefined>(
		catalog && categoryIDs.length > 0 ? catalog.getCategory(categoryIDs[0]) : undefined
	);
</script>

<div class={css}>
	{#if categoria}
		{categoria.Description}
	{/if}
</div>
