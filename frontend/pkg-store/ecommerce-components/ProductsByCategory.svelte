<script lang="ts">
	import { getProductsByCategoryID, type IProducto } from '$services/services/productos.svelte';
	import ProductCard from "$store/components/ProductCard.svelte";

	export interface IProductsByCategory {
		css?: string;
		categoryID: number;
		limit?: number;
	}

	const { css = "", categoryID, limit = 8 }: IProductsByCategory = $props();

	let productos: IProducto[] = $state([]);

	// Load data when component mounts
	$effect(() => {
		console.log("obteniendo producto con categoría ID::", categoryID);

		getProductsByCategoryID(categoryID).then((productos_) => {
			productos = productos_
			if(limit && productos.length > limit){
				productos = productos.splice(0,limit)
			}
		});
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
