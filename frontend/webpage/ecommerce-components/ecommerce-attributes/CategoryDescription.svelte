<script lang="ts">
	import { getProductEcommerceData, type ProductCatalog, type IProductCategory } from '$ecommerce/services/productos.svelte';

	export interface ICategoryDescription {
		css?: string;
		categoryIDs?: number[];
	}

	const { css = "", categoryIDs = [] }: ICategoryDescription = $props();

	let catalog = $state<ProductCatalog | null>(null);

	getProductEcommerceData().then((loaded) => { catalog = loaded; });

	// Reactive: resolves once the catalog (and its delta) is available.
	const categoria = $derived<IProductCategory | undefined>(
		catalog && categoryIDs.length > 0 ? catalog.getCategory(categoryIDs[0]) : undefined
	);
</script>

<div class={css}>
	{#if categoria}
		{categoria.Description}
	{:else}
		Loading...
	{/if}
</div>
