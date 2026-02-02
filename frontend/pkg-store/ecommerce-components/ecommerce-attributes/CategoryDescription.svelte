<script lang="ts">
	import { getCategoryByID, type IProductoCategoria } from '$services/services/productos.svelte';

	export interface ICategoryDescription {
		css?: string;
		categoriasIDs?: number[];
	}

	const { css = "", categoriasIDs = [] }: ICategoryDescription = $props();

	let categoria: IProductoCategoria | undefined = $state(undefined);

	// Load data when component mounts
	$effect(() => {
		
		if (categoriasIDs && categoriasIDs.length > 0) {
			const categoryID = categoriasIDs[0];
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
		{categoria.Descripcion}
	{:else}
		Loading...
	{/if}
</div>
