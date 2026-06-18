<script lang="ts">
	import { getProductEcommerceData, type IProduct, type ProductCatalog } from '$ecommerce/services/products.svelte';
	import ProductCard from '$ecommerce/components/ProductCard.svelte';

	export interface IProductGrid {
		css?: string;
		// Pre-fetched products to lay out (e.g. passed down by ProductsByCategory). When omitted the
		// grid self-fetches the catalog so it can be dropped straight into a template's AST.
		products?: IProduct[];
		// Self-fetch mode only: restrict to one category; absent shows every product.
		categoryID?: number;
		// Optional cap on the content area width (px). Without it the grid fills the whole parent.
		maxWidth?: number;
		// When the parent is wider than `maxWidth`, keep expanding the content area until the
		// leftover (parent − content) drops below `maxMargin` (px, total of both side margins).
		maxMargin?: number;
		// Row caps drive how many cards render: columns × rows.
		rows?: number;
		rowsMobile?: number;
		// Fixed card/column width in px (the unit the column count is derived from).
		cardWidth?: number;
	}

	const {
		css = '',
		products: providedProducts,
		categoryID,
		maxWidth,
		maxMargin,
		rows = 3,
		rowsMobile,
		cardWidth = 240,
	}: IProductGrid = $props();

	// Self-fetch only when no products were handed in. The catalog is memoized/reactive, so this is
	// cheap and re-fills when the delta publishes (same source ProductsByCategory uses).
	let catalog = $state<ProductCatalog | null>(null);
	$effect(() => {
		if (!providedProducts) getProductEcommerceData().then((loaded) => { catalog = loaded; });
	});

	// Hand the grid the full list (it caps how many to show): provided products win; otherwise the
	// self-fetched catalog, narrowed to `categoryID` when given.
	const products = $derived.by(() => {
		if (providedProducts) return providedProducts;
		if (!catalog) return [];
		return categoryID == null ? catalog.productos : catalog.getProductsByCategory(categoryID);
	});

	// Layout constants mirror the previous markup: 2-up phone grid below 740px,
	// 20px column gap on desktop, 12px on mobile. Row spacing comes from the card's own margin.
	const MOBILE_BREAKPOINT = 740;
	const DESKTOP_COLUMN_GAP = 20;
	// Width assumed during SSR/prerender (no DOM yet) so a sensible grid renders before hydration.
	const FALLBACK_DESKTOP_WIDTH = 1680;

	// `bind:clientWidth` tracks the parent width reactively (ResizeObserver under the hood).
	let parentWidth = $state(0);

	// Treat zero width (pre-mount) as desktop so prerender does not collapse to the phone layout.
	const isMobileLayout = $derived(parentWidth > 0 && parentWidth < MOBILE_BREAKPOINT);

	// Content area the cards occupy: clamp to maxWidth, then expand toward the parent edges
	// while the leftover margin still exceeds maxMargin.
	const contentWidth = $derived.by(() => {
		const availableWidth = parentWidth || maxWidth || FALLBACK_DESKTOP_WIDTH;
		if (!maxWidth) return availableWidth;
		if (maxMargin == null) return Math.min(availableWidth, maxWidth);
		return Math.min(availableWidth, Math.max(maxWidth, availableWidth - maxMargin));
	});

	// How many fixed-width cards fit: n cards span n·cardWidth + (n−1)·gap ≤ contentWidth.
	const columnCount = $derived.by(() => {
		if (isMobileLayout) return 2;
		const columnUnit = cardWidth + DESKTOP_COLUMN_GAP;
		return Math.max(1, Math.floor((contentWidth + DESKTOP_COLUMN_GAP) / columnUnit));
	});

	const visibleProductCount = $derived(
		isMobileLayout ? 2 * (rowsMobile ?? rows) : columnCount * rows,
	);

	const visibleProducts = $derived(products.slice(0, visibleProductCount));

	// Before the catalog resolves, `products` is empty — render that many skeleton cards so the
	// grid keeps its shape and the page doesn't jump when real products arrive.
	const placeholderCount = $derived(products.length === 0 ? visibleProductCount : 0);

	// Desktop: fixed-width columns, centered. Mobile: two fluid columns spanning the full width.
	const gridStyle = $derived(
		isMobileLayout
			? `grid-template-columns:repeat(2,minmax(0,1fr));`
			: `grid-template-columns:repeat(${columnCount},${cardWidth}px);column-gap:${DESKTOP_COLUMN_GAP}px`,
	);
</script>

<div bind:clientWidth={parentWidth}
	class="w-full flex justify-center pt-2 {css}"
>
	<div class="grid {isMobileLayout ? 'w-full gap-12' : 'justify-center'}" style={gridStyle}>
		{#each visibleProducts as producto (producto.ID)}
			<ProductCard css="w-full" producto={producto} />
		{/each}
		{#each { length: placeholderCount } as _, slotIndex (slotIndex)}
			<ProductCard css="w-full" usePlaceHolder />
		{/each}
	</div>
</div>
