<script lang="ts">
	import { getProductEcommerceData, type ProductCatalog } from '$ecommerce/services/productos.svelte';
	import ProductCard from "$ecommerce/components/ProductCard.svelte";

	export interface IProductsByCategory {
		css?: string;
		categoryID: number;
		limit?: number;
	}

	const { css = "", categoryID, limit = 8 }: IProductsByCategory = $props();

	let catalog = $state<ProductCatalog | null>(null);

	getProductEcommerceData().then((loaded) => { catalog = loaded; });

	// Reactive: tracks the catalog state, so it fills in when the delta re-publishes.
	const productos = $derived.by(() => {
		if (!catalog) return [];
		const list = catalog.getProductsByCategory(categoryID);
		return limit && list.length > limit ? list.slice(0, limit) : list;
	});

</script>

<div class="w-full flex justify-center overflow-x-hidden pt-2">
	<div class={"grid grid-cols-2 gap-x-12 md:gap-x-20 md:flex md:flex-wrap md:justify-center max-w-1680 w100-p12 p-8 md:p-0"}
	>
		{#each productos as producto}
			<ProductCard css="w-full md:w-240" producto={producto} />
		{/each}
	</div>
</div>
