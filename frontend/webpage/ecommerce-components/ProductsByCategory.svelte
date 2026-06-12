<script lang="ts">
	import { getProductEcommerceData, type ProductCatalog } from '$ecommerce/services/productos.svelte';
	import ProductGrid from './ProductGrid.svelte';

	export interface IProductsByCategory {
		css?: string;
		categoryID: number;
		// Layout knobs forwarded to ProductGrid (see ProductGrid.svelte for semantics).
		maxWidth?: number;
		maxMargin?: number;
		rows?: number;
		rowsMobile?: number;
	}

	const { css = '', categoryID, maxWidth, maxMargin, rows, rowsMobile }: IProductsByCategory =
		$props();

	let catalog = $state<ProductCatalog | null>(null);

	getProductEcommerceData().then((loaded) => { catalog = loaded; });

	// Reactive: tracks the catalog state, so it fills in when the delta re-publishes.
	// The grid decides how many of these to actually show, so we hand it the full category list.
	const productos = $derived.by(() => {
		if (!catalog) return [];
		return catalog.getProductsByCategory(categoryID);
	});
</script>

<ProductGrid {css} products={productos} {maxWidth} {maxMargin} {rows} {rowsMobile} />
