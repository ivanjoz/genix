<script lang="ts">
	import { untrack } from "svelte";
	import { configureCacheByIDsRoute, getRecordWithCache, type IRecordRef } from "@genix/ui/cache";
	import { formatN } from "#libs/helpers.ts";
	import { Env } from "#core/env.ts";
	import { ecommerceCatalog } from "#ecommerce/services/ecommerce-catalog.svelte.ts";
	import {
		PRODUCT_DETAIL_ROUTE, productGalleryImages, type IProductDetail,
	} from "#ecommerce/services/ecommerce-catalog.ts";
	import { addProductoCant, cartUnitsLimitOf, productDetailLayer, ProductsSelectedMap } from "./store.svelte";

	// Unhiding a product in the admin must bring its detail back, so a hidden one (ss 0) is no tombstone.
	configureCacheByIDsRoute(PRODUCT_DETAIL_ROUTE, { disableTombstone: true });

	// Name, price, main image and stock come from the live catalog (instant paint); the detail
	// endpoint only adds the extra images and the HTML sheet, cached per product in IndexedDB.
	const catalogProduct = $derived(ecommerceCatalog.getProduct(productDetailLayer.productID));
	let productDetailReference = $state<IRecordRef<IProductDetail> | null>(null);
	let selectedImageName = $state("");

	$effect(() => {
		const openedProductID = productDetailLayer.productID;
		untrack(() => {
			selectedImageName = "";
			productDetailReference = openedProductID > 0
				? getRecordWithCache<IProductDetail>(PRODUCT_DETAIL_ROUTE, openedProductID)
				: null;
		});
	});

	// A detail with ss 0 is a product the store stopped selling: no extra images, no sheet.
	const productDetail = $derived.by(() => {
		const detailRecord = productDetailReference?.record;
		return detailRecord && detailRecord.ss > 0 ? detailRecord : null;
	});
	const galleryImages = $derived(productGalleryImages(Env.getCompanyID(), catalogProduct?.Image, productDetail));
	const shownImageName = $derived(selectedImageName || galleryImages[0]?.n || "");

	// The CDN keeps each product image in three sizes (backend/production/products.go): x2 = 360px
	// for the thumbnails, x4 = 570px for the large view.
	const productImageURL = (imageName: string, sizeLabel: "x2" | "x4") =>
		Env.makeCDNRoute("img-productos", `${imageName}-${sizeLabel}.avif`);

	// The large image whose file finished loading. Until it matches the shown one, the view paints the
	// already-downloaded thumbnail, blurred, under a spinner: a click changes the picture at once
	// instead of leaving the previous large image up while the new one downloads.
	let loadedLargeImageName = $state("");
	const isLargeImageLoaded = $derived(loadedLargeImageName === shownImageName);

	const finalPriceCents = $derived(catalogProduct?.FinalPrice || catalogProduct?.Price || 0);
	// The list price is struck through only when a discount actually lowers it.
	const hasDiscount = $derived((catalogProduct?.Price || 0) > finalPriceCents && finalPriceCents > 0);
	const cartQuantity = $derived(ProductsSelectedMap.get(productDetailLayer.productID)?.cant || 0);
	const cartUnitsLimit = $derived(cartUnitsLimitOf(productDetailLayer.productID));

	const closeLayer = () => {
		productDetailLayer.productID = 0;
	};

	const changeCartQuantity = (quantityChange: number) => {
		if (!catalogProduct) return;
		addProductoCant(catalogProduct, null, quantityChange);
	};

	const closeOnEscape = (keyboardEvent: KeyboardEvent) => {
		if (keyboardEvent.key === "Escape" && productDetailLayer.productID > 0) closeLayer();
	};
</script>

<svelte:window onkeydown={closeOnEscape} />

{#if catalogProduct}
	<!-- Backdrop: a click outside the panel closes it. -->
	<button
		type="button"
		class="fixed inset-0 z-[300] bg-black/40 border-0 cursor-default"
		onclick={closeLayer}
		aria-label="Cerrar detalle del producto"
	></button>
	<aside
		class="product-detail-panel fixed top-0 left-0 z-[301] h-full w-full md:w-[760px] bg-white flex flex-col"
		aria-label={catalogProduct.Name}
	>
		<header class="flex items-start gap-12 px-20 pt-16 pb-12 border-b border-gray-200">
			<h2 class="flex-1 text-[20px] md:text-[22px] font-semibold leading-snug m-0">{catalogProduct.Name}</h2>
			<button
				type="button"
				class="close-button shrink-0 h-36 w-36 rounded-full border-0 flex items-center justify-center"
				onclick={closeLayer}
				aria-label="Cerrar"
			>
				<i class="icon-[fa--close]"></i>
			</button>
		</header>

		<div class="flex-1 overflow-y-auto px-20 py-16">
			<!-- Gallery: thumbnails on the left (only with more than one image), the selected one large. -->
			<div class="flex gap-12">
				{#if galleryImages.length > 1}
					<div class="flex flex-col gap-8 shrink-0 max-h-[420px] overflow-y-auto">
						{#each galleryImages as galleryImage (galleryImage.n)}
							<button
								type="button"
								class="thumbnail-button shrink-0 h-64 w-64 md:h-80 md:w-80 p-4 rounded-[8px] bg-white overflow-hidden"
								class:selected={galleryImage.n === shownImageName}
								onclick={() => (selectedImageName = galleryImage.n)}
								title={galleryImage.d || undefined}
							>
								<img class="block h-full w-full object-contain" src={productImageURL(galleryImage.n, "x2")}
									alt={galleryImage.d || catalogProduct.Name} />
							</button>
						{/each}
					</div>
				{/if}
				<div class="flex-1 min-w-0">
					{#if shownImageName}
						<div class="relative w-full h-[70vw] md:h-[420px] overflow-hidden">
							{#if !isLargeImageLoaded}
								<img class="absolute inset-0 h-full w-full object-contain blur-[6px] scale-105"
									src={productImageURL(shownImageName, "x2")} alt="" />
								<span class="large-image-spinner"></span>
							{/if}
							<!-- Keyed: a new element per image, so the previous large file is never shown for
							     the new selection while it downloads. -->
							{#key shownImageName}
								<img
									class="absolute inset-0 h-full w-full object-contain"
									class:opacity-0={!isLargeImageLoaded}
									src={productImageURL(shownImageName, "x4")}
									alt={catalogProduct.Name}
									onload={() => (loadedLargeImageName = shownImageName)}
									onerror={() => (loadedLargeImageName = shownImageName)}
								/>
							{/key}
						</div>
					{:else}
						<div class="w-full h-[70vw] md:h-[420px] rounded-[8px] bg-gray-100"></div>
					{/if}
				</div>
			</div>

			<!-- Price and cart: "Agregar al carrito" until the product is in the cart, then -/qty/+ capped at stock. -->
			<div class="flex items-center flex-wrap gap-16 mt-20 pb-16 border-b border-gray-200">
				<div class="flex items-baseline gap-10">
					<span class="text-[24px] font-bold">s/. {formatN(finalPriceCents / 100, 2)}</span>
					{#if hasDiscount}
						<span class="text-[16px] line-through text-gray-500">
							s/. {formatN((catalogProduct.Price || 0) / 100, 2)}
						</span>
					{/if}
				</div>
				<div class="ml-auto">
					{#if cartUnitsLimit <= 0 && cartQuantity === 0}
						<span class="text-gray-500">Sin stock disponible</span>
					{:else if cartQuantity === 0}
						<button type="button" class="add-cart-button h-40 px-20 rounded-[20px] border-0 font-semibold"
							onclick={() => changeCartQuantity(1)}>
							Agregar al carrito <i class="icon--supermarket-cart"></i>
						</button>
					{:else}
						<div class="flex items-center gap-8">
							<button type="button" class="quantity-button h-36 w-36 rounded-full border-0 text-[18px]"
								onclick={() => changeCartQuantity(-1)} aria-label="Quitar uno">-</button>
							<span class="min-w-40 text-center text-[18px] font-semibold">{cartQuantity}</span>
							<button type="button" class="quantity-button h-36 w-36 rounded-full border-0 text-[18px]"
								onclick={() => changeCartQuantity(1)} disabled={cartQuantity >= cartUnitsLimit}
								title={cartQuantity >= cartUnitsLimit ? "No hay más stock disponible" : undefined}
								aria-label="Agregar uno">+</button>
						</div>
					{/if}
				</div>
			</div>

			<!-- The product sheet, written as HTML in the admin's product form (trusted tenant content).
			     Tailwind's preflight resets headings and lists, so they get their look back here. -->
			{#if productDetail?.ContentHTML}
				<div
					class="mt-16 leading-relaxed [&_p]:mb-10 [&_h1]:text-[22px] [&_h2]:text-[20px] [&_h3]:text-[18px]
						[&_h1]:font-bold [&_h2]:font-bold [&_h3]:font-semibold [&_:is(h1,h2,h3)]:mt-14 [&_:is(h1,h2,h3)]:mb-8
						[&_ul]:list-disc [&_ol]:list-decimal [&_:is(ul,ol)]:pl-24 [&_:is(ul,ol)]:mb-10 [&_a]:underline
						[&_img]:max-w-full [&_table]:w-full"
				>
					{@html productDetail.ContentHTML}
				</div>
			{:else if productDetailReference?.loading}
				<div class="mt-16 text-gray-500">Cargando ficha del producto...</div>
			{/if}
		</div>
	</aside>
{/if}

<style>
	.product-detail-panel {
		box-shadow: rgba(0, 0, 0, 0.25) 4px 0 24px;
		animation: product-detail-slide-in 0.25s cubic-bezier(0.22, 1, 0.36, 1);
	}

	/* Slides in from the left edge of the screen. */
	@keyframes product-detail-slide-in {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(0);
		}
	}

	.close-button {
		background-color: rgb(255, 233, 233);
		color: rgb(197, 71, 71);
	}
	.close-button:hover {
		background-color: rgb(228, 94, 94);
		color: white;
	}

	/* A border, not an outline: the scrolling thumbnail column clipped the outline at its edges. */
	.thumbnail-button {
		border: 2px solid #e3e5ea;
	}
	.thumbnail-button:hover,
	.thumbnail-button.selected {
		border-color: rgb(111, 82, 179);
	}

	/* Over the blurred thumbnail while the large image downloads. */
	.large-image-spinner {
		position: absolute;
		top: 50%;
		left: 50%;
		width: 44px;
		height: 44px;
		margin: -22px 0 0 -22px;
		border-radius: 50%;
		border: 4px solid rgba(255, 255, 255, 0.6);
		border-top-color: rgb(111, 82, 179);
		animation: large-image-spin 0.8s linear infinite;
	}

	@keyframes large-image-spin {
		to {
			transform: rotate(360deg);
		}
	}

	.add-cart-button,
	.quantity-button {
		background-color: rgb(228, 221, 248);
		color: rgb(100, 67, 160);
	}
	.add-cart-button:hover,
	.quantity-button:hover:not(:disabled) {
		background-color: rgb(111, 82, 179);
		color: white;
	}
	.quantity-button:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}
</style>
