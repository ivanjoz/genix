<script lang="ts">
	import { getCategoryByID, type IProductCategory } from '$services/services/productos.svelte';

	export interface ICategoryDescription {
		css?: string;
		categoryIDs?: number[];
	}

	const { css = "", categoryIDs = [] }: ICategoryDescription = $props();

	let categoria: IProductCategory | undefined = $state(undefined);

	// Load data when component mounts
	$effect(() => {
		
		if (categoryIDs && categoryIDs.length > 0) {
			const categoryID = categoryIDs[0];
			console.log("obteniendo categoría con ID::", categoryID);

			getCategoryByID(categoryID).then((categoria_) => {
				categoria = categoria_;
				console.log("categoría obtenida::", categoria)
			});
		}
	});
</script>

<div class={css}>
	{#if categoria}
		{categoria.Description}
	{:else}
		Loading...
	{/if}
</div>
