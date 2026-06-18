<script lang="ts">
	import { formatN } from "$libs/helpers";
	import {
		getRecordWithCache,
		type IMinimalRecord,
		type IRecordRef,
	} from "$libs/cache/cache-by-ids.svelte";
	import { untrack } from "svelte";
	import type { IProduct } from "$ecommerce/services/productos.svelte";
	import ImageHash from "$components/files/Imagehash.svelte";
	import { addProductoCant, ProductsSelectedMap } from "./store.svelte";

	type ProductCardMode = "vertical" | "horizontal";

	interface IProductoByIDRecord extends IMinimalRecord {
		Name?: string;
		Price?: number;
		FinalPrice?: number;
		Image?: { n?: string; d?: string };
		Images?: Array<{ n?: string; d?: string }>;
	}

	export interface IProductCard {
		css?: string;
		productoID?: number;
		producto?: IProduct;
		mode?: ProductCardMode;
		hideCloseButton?: boolean;
		useQuantityControls?: boolean;
		// When true, render a skeleton-loading card (Facebook-style shimmer strips) while there is
		// no product yet (ID 0) or the by-id record is still loading. Used by grids to fill slots.
		usePlaceHolder?: boolean;
	}

	const {
		css = "",
		productoID,
		producto,
		mode = "vertical",
		hideCloseButton = false,
		useQuantityControls = true,
		usePlaceHolder = false,
	}: IProductCard = $props();

	let productRecordReference = $state<IRecordRef<IProductoByIDRecord> | null>(null);
	let lastResolvedProductID = $state(0);

	const resolvedProductID = $derived.by(() => productoID || producto?.ID || 0);
	const fetchedProductRecord = $derived(productRecordReference?.record || null);
	const resolvedProduct = $derived.by<IProduct>(() => {
		const fallbackProductRecord = producto || ({ ID: resolvedProductID } as IProduct);
		if (!fetchedProductRecord) return fallbackProductRecord;
		return {
			...fallbackProductRecord,
			...(fetchedProductRecord as Partial<IProduct>),
			ID: fetchedProductRecord.ID,
		} as IProduct;
	});
	const selectedProductQuantity = $derived.by(() => {
		return ProductsSelectedMap.get(resolvedProduct?.ID)?.cant || 0;
	});
	const resolvedProductPriceCents = $derived.by(() => {
		return resolvedProduct?.FinalPrice || resolvedProduct?.Price || 0;
	});
	const resolvedProductImageName = $derived.by(() => {
		return resolvedProduct?.Image?.n || resolvedProduct?.Images?.[0]?.n || "";
	});

	// Fixed palette of 10 gradients used as a fallback "cover" when a product has no photo,
	// so the card shows colour instead of a broken-image icon.
	const PLACEHOLDER_GRADIENTS = [
		"linear-gradient(135deg, #c9cdd6 0%, #e3e6ec 100%)",
		"linear-gradient(135deg, #cdd3c8 0%, #e4e7df 100%)",
		"linear-gradient(135deg, #d3ccd6 0%, #e6e2ea 100%)",
		"linear-gradient(135deg, #c8d2d6 0%, #dfe6e9 100%)",
		"linear-gradient(135deg, #d6cdc8 0%, #eae3df 100%)",
		"linear-gradient(135deg, #ccd6d3 0%, #dfe9e6 100%)",
		"linear-gradient(135deg, #d6d0c8 0%, #ebe6dd 100%)",
		"linear-gradient(135deg, #cdd0d6 0%, #e2e4ea 100%)",
		"linear-gradient(135deg, #d4ccd0 0%, #e8e1e5 100%)",
		"linear-gradient(135deg, #c8d6d0 0%, #dde9e4 100%)",
	];
	// Pick deterministically by product ID; the ×7 spreads sequential IDs across the palette
	// (7 is coprime with 10) so neighbouring products don't share the same gradient.
	const productPlaceholderGradient = $derived(
		PLACEHOLDER_GRADIENTS[((resolvedProduct?.ID || 0) * 7) % PLACEHOLDER_GRADIENTS.length],
	);
	const isLoadingProductRecord = $derived(productRecordReference?.loading || false);
	// Show the shimmer skeleton while a placeholder card has nothing to render yet: either no
	// product at all (ID 0) or an ID whose record is still being fetched.
	const showLoadingSkeleton = $derived(
		usePlaceHolder && (resolvedProductID === 0 || isLoadingProductRecord),
	);

	$effect(() => {
		// When the caller already supplies the full product (in-memory search path), skip the by-id
		// fetch entirely — name, price and image are all present on `producto`.
		if (producto) {
			untrack(() => {
				productRecordReference = null;
				lastResolvedProductID = 0;
			});
			return;
		}
		// Resolve by ID through the shared cache-by-id pipeline instead of deprecated service call.
		const selectedProductID = resolvedProductID;
		if (!(selectedProductID > 0)) {
			untrack(() => {
				productRecordReference = null;
				lastResolvedProductID = 0;
			});
			return;
		}
		const alreadyLoadedThisID = untrack(() => lastResolvedProductID === selectedProductID);
		if (alreadyLoadedThisID) {
			return;
		}
		untrack(() => {
			lastResolvedProductID = selectedProductID;
			productRecordReference = getRecordWithCache<IProductoByIDRecord>(
				"p-productos-ids",
				selectedProductID,
			);
		});
	});

	const incrementSelectedQuantity = (clickEvent: MouseEvent) => {
		clickEvent.stopPropagation();
		if (!(resolvedProduct?.ID > 0)) return;
		addProductoCant(resolvedProduct, null, 1);
	};

	const decrementSelectedQuantity = (clickEvent: MouseEvent) => {
		clickEvent.stopPropagation();
		if (!(resolvedProduct?.ID > 0)) return;
		if (selectedProductQuantity <= 1) {
			addProductoCant(resolvedProduct, 0);
			return;
		}
		addProductoCant(resolvedProduct, null, -1);
	};

	const removeProductFromSelection = (clickEvent: MouseEvent) => {
		clickEvent.stopPropagation();
		if (!(resolvedProduct?.ID > 0)) return;
		addProductoCant(resolvedProduct, 0);
	};

	const updateSelectedQuantityFromInput = (changeEvent: Event) => {
		changeEvent.stopPropagation();
		if (!(resolvedProduct?.ID > 0)) return;

		const inputElement = changeEvent.target as HTMLInputElement;
		const parsedQuantity = Number.parseInt(inputElement.value, 10);
		if (!Number.isFinite(parsedQuantity) || parsedQuantity < 0) {
			inputElement.value = `${selectedProductQuantity}`;
			return;
		}
		addProductoCant(resolvedProduct, parsedQuantity);
	};
</script>

{#if mode === "horizontal"}
	<div class="horizontal-card {css}" aria-busy={isLoadingProductRecord || showLoadingSkeleton}>
		<div class="horizontal-image-wrapper">
			{#if showLoadingSkeleton}
				<div class="horizontal-image-placeholder skeleton-line"></div>
			{:else if resolvedProductImageName}
				<ImageHash
					css="horizontal-image"
					src={resolvedProductImageName}
					folder="img-productos"
				/>
			{:else}
				<div class="horizontal-image-placeholder" style:background={productPlaceholderGradient}></div>
			{/if}
		</div>
		<div class="horizontal-body">
			{#if showLoadingSkeleton}
				<div class="skeleton-line w-4/5"></div>
				<div class="skeleton-line w-2/5 mt-4"></div>
			{:else}
			<div class="horizontal-name" title={resolvedProduct.Name || `Producto #${resolvedProductID}`}>
				{resolvedProduct.Name || `Producto #${resolvedProductID}`}
			</div>
				<div class="horizontal-footer">
					{#if useQuantityControls}
						<div class="quantity-controls">
							<button class="quantity-button" onclick={decrementSelectedQuantity}>-</button>
							<input
								class="quantity-input"
								type="number"
								value={selectedProductQuantity}
								onchange={updateSelectedQuantityFromInput}
							/>
							<button class="quantity-button" onclick={incrementSelectedQuantity}>+</button>
						</div>
					{:else}
						<button class="horizontal-add-cart-button mb-2!" onclick={incrementSelectedQuantity}>
							{#if selectedProductQuantity > 0}
								+ Carrito ({selectedProductQuantity})
							{:else}
								+ Carrito
							{/if}
						</button>
					{/if}
					<div class="horizontal-price">
						<div>s/.</div>
						<div class="price-value">{formatN(resolvedProductPriceCents / 100, 2)}</div>
					</div>
				</div>
				{/if}
		</div>
			{#if !showLoadingSkeleton && !hideCloseButton}
				<!-- Keep explicit remove action optional so search-card usage can hide destructive controls. -->
				<button class="remove-button" onclick={removeProductFromSelection} aria-label="Remover">
					<i class="icon-[fa--close]"></i>
				</button>
			{/if}
		</div>
{:else}
	<div class="vertical-card-shell" class:loading={showLoadingSkeleton}>
		<div class="vertical-card {css}" aria-busy={isLoadingProductRecord || showLoadingSkeleton}>
			{#if resolvedProductImageName}
				<ImageHash
					css="w-full h-[36vw] md:h-200"
					src={resolvedProductImageName}
					folder="img-productos"
				/>
			{:else}
				<!-- No photo: deterministic gradient cover keyed off the product ID.
				     While loading it also hosts the centered spinner. -->
				<div
					class="w-full h-[36vw] md:h-200 relative"
					style:background={productPlaceholderGradient}
					style:border-radius="7px"
				>
					{#if showLoadingSkeleton}
						<span class="loader"></span>
					{/if}
				</div>
			{/if}
			<div class="vertical-content pb-2">
				{#if showLoadingSkeleton}
					<div class="vertical-name mt-10 mb-6 min-h-26 md:min-h-32 fx-c flex-col gap-8">
						<div class="skeleton-line w-4/5"></div>
						<div class="skeleton-line w-3/5"></div>
					</div>
					<div class="px-4 mb-4"><div class="skeleton-line skeleton-strong h-18 w-90"></div></div>
				{:else}
					<div class="vertical-name mt-6 mb-4 min-h-26 md:min-h-32 fx-c">
						{resolvedProduct.Name || "???"}
					</div>
					<div class="px-4 ff-bold fs17">s/. {formatN(resolvedProductPriceCents / 100, 2)}</div>
				{/if}
				<div class="vertical-icon fx-c h-30 w-32">
					<i class="icon--supermarket-cart"></i>
				</div>
			</div>
			{#if !showLoadingSkeleton}
				<button class="vertical-add-button" onclick={incrementSelectedQuantity} type="button">
					{#if selectedProductQuantity === 0}
						Agregar <i class="icon--supermarket-cart"></i>
					{/if}
					{#if selectedProductQuantity > 0}
						Agregar mas ({selectedProductQuantity}) <i class="icon--supermarket-cart"></i>
					{/if}
				</button>
			{/if}
		</div>
	</div>
{/if}

<style>
	/* Facebook-style loading strip: a grey bar with a light band sweeping across it. */
	.skeleton-line {
		height: 14px;
		border-radius: 6px;
		background-color: #e3e5ea;
		background-image: linear-gradient(
			90deg,
			rgba(255, 255, 255, 0) 0%,
			rgba(255, 255, 255, 0.65) 50%,
			rgba(255, 255, 255, 0) 100%
		);
		background-size: 200% 100%;
		background-repeat: no-repeat;
		background-position: -150% 0;
		animation: skeleton-sweep 1.3s ease-in-out infinite;
	}

	/* Darker base for the price strip, since the real price is bold. */
	.skeleton-strong {
		background-color: #c6c9d2;
	}

	/* Centered orbiting spinner shown over the cover while a card loads.
	   Two offset dots in muted gray-blue / gray-red orbit via the `spin` keyframe. */
	.loader {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%) rotateZ(45deg);
		perspective: 1000px;
		border-radius: 50%;
		width: 40px;
		height: 40px;
		color: #7e8aa6;
	}
	.loader:before,
	.loader:after {
		content: '';
		display: block;
		position: absolute;
		top: 0;
		left: 0;
		width: inherit;
		height: inherit;
		border-radius: 50%;
		transform: rotateX(70deg);
		animation: 1s spin linear infinite;
	}
	.loader:after {
		color: #b07f86;
		transform: rotateY(70deg);
		animation-delay: 0.4s;
	}

	@keyframes spin {
		0%,
		100% {
			box-shadow: 0.2em 0 0 0 currentcolor;
		}
		12% {
			box-shadow: 0.2em 0.2em 0 0 currentcolor;
		}
		25% {
			box-shadow: 0 0.2em 0 0 currentcolor;
		}
		37% {
			box-shadow: -0.2em 0.2em 0 0 currentcolor;
		}
		50% {
			box-shadow: -0.2em 0 0 0 currentcolor;
		}
		62% {
			box-shadow: -0.2em -0.2em 0 0 currentcolor;
		}
		75% {
			box-shadow: 0 -0.2em 0 0 currentcolor;
		}
		87% {
			box-shadow: 0.2em -0.2em 0 0 currentcolor;
		}
	}

	@keyframes skeleton-sweep {
		to {
			background-position: 250% 0;
		}
	}

	.vertical-card {
		position: relative;
		background-color: white;
		box-shadow: rgba(54, 56, 67, 0.2) 0px 2px 8px 0px;
		min-height: 180px;
		padding: 6px;
		border-radius: 10px;
		margin-bottom: 24px;
		/* Clip the add button while it is tucked below the card so it can slide up into view. */
		overflow: hidden;
		/* Outline never participates in layout, so the hover highlight cannot cause reflow. */
		outline: 2px solid transparent;
		transition: outline-color 0.2s ease;
	}

	.vertical-add-button {
		color: rgb(100, 67, 160);
		position: absolute;
		border-radius: 0 0 10px 10px;
		height: 36px;
		align-items: center;
		justify-content: center;
		display: flex;
		width: 100%;
		bottom: 0;
		left: 0;
		user-select: none;
		border: none;
		background: transparent;
		font-family: inherit;
		font-size: inherit;
		padding: 0;
		/* Start tucked below the card; slides up to its hover position via translateY. */
		transform: translateY(100%);
		will-change: transform;
		transition:
			transform 0.28s cubic-bezier(0.22, 1, 0.36, 1),
			height 0.15s ease,
			background-color 0.15s ease;
	}

	.vertical-content {
		position: relative;
		z-index: 10;
		width: 100%;
		/* Opaque so that when it slides up on hover it masks the bottom of the image,
		   freeing the bottom strip for the add button without growing the card. */
		background-color: white;
		transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
	}

	.vertical-name {
		text-align: center;
	}

	.vertical-icon {
		border-radius: 7px;
		background-color: rgb(228, 221, 248);
		color: rgb(100, 67, 160);
		flex-shrink: 0;
		position: absolute;
		bottom: -2px;
		right: -2px;
		outline: 2px solid rgba(255, 255, 255);
		user-select: none;
	}

	.horizontal-card {
		display: flex;
		position: relative;
		height: 110px;
		border-radius: 7px;
		background: #fff;
		box-shadow: rgba(50, 50, 105, 0.15) 0px 2px 5px 0px, rgba(0, 0, 0, 0.1) 0px 0px 2px 0px;
		overflow: hidden;
	}

	.horizontal-image-wrapper {
		padding: 8px;
		flex: 0 0 auto;
	}

	.horizontal-image {
		width: 100px;
		height: 100%;
	}

	.horizontal-image-placeholder {
		width: 100px;
		height: 100%;
		min-height: 84px;
		background: #f1f2f7;
		border-radius: 6px;
	}

	.horizontal-body {
		display: flex;
		flex-direction: column;
		width: 100%;
		padding: 6px 4px 4px 0;
	}

	.horizontal-name {
		font-size: 15px;
		line-height: 1.2;
		overflow: hidden;
		text-overflow: ellipsis;
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
	}

	.horizontal-footer {
		display: flex;
		align-items: center;
		margin-top: auto;
		gap: 10px;
	}

	.quantity-controls {
		display: flex;
		align-items: center;
	}

	.quantity-button {
		height: 28px;
		width: 28px;
		border-radius: 50%;
		border: none;
		background-color: #e5e3f4;
	}

	.quantity-button:hover {
		background-color: #5e51c0;
		color: white;
	}

	.quantity-input {
		height: 28px;
		width: 58px;
		text-align: center;
		outline: none;
	}

	.horizontal-add-cart-button {
		height: 28px;
		border: none;
		border-radius: 14px;
		padding: 0 12px;
		background-color: #e5e3f4;
		color: #2b2557;
		font-weight: 600;
	}

	.horizontal-add-cart-button:hover {
		background-color: #5e51c0;
		color: white;
	}

	.horizontal-price {
		display: flex;
		align-items: center;
		margin-left: auto;
		margin-right: 18px;
		font-size: 17px;
		gap: 4px;
	}

	.price-value {
		font-weight: bold;
	}

	.remove-button {
		position: absolute;
		top: -4px;
		right: -4px;
		height: 28px;
		width: 28px;
		border-radius: 50%;
		border: none;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: rgb(255, 233, 233);
		color: rgb(197, 71, 71);
	}

	.remove-button:hover {
		color: white;
		background-color: rgb(228, 94, 94);
	}

	/* While loading there is nothing to interact with; this also suppresses the hover lift/outline. */
	.vertical-card-shell.loading {
		pointer-events: none;
	}

	@media (min-width: 739px) {
		.vertical-card-shell:hover .vertical-card {
			outline-color: rgb(202, 173, 255);
			margin-bottom: 12px;
		}

		/* Slide the name/price block up (over the image) to free the bottom strip.
		   Using transform keeps the card height unchanged, so no neighbour reflow. */
		.vertical-card-shell:hover .vertical-content {
			transform: translateY(-30px);
			margin-top: 12px;
		}

		.vertical-card-shell:hover .vertical-add-button {
			transform: translateY(0);
			background-color: rgb(228, 221, 248);
			cursor: pointer;
		}

		.vertical-add-button:hover,
		.vertical-card-shell .vertical-add-button:hover {
			background-color: rgb(111, 82, 179);
			color: white;
		}

		.vertical-card-shell:hover .vertical-icon {
			display: none;
		}
	}

	@media (max-width: 740px) {
		.vertical-card-shell .vertical-add-button {
			transform: translateY(0);
			background-color: rgb(228, 221, 248);
			cursor: pointer;
		}

		/* No hover on touch: the button is always visible, so reserve the bottom strip
		   permanently instead of sliding the content up. All cards stay uniform height. */
		.vertical-card-shell .vertical-card {
			padding-bottom: 38px;
			border-bottom: 2px solid rgb(172, 153, 226);
			margin-bottom: 12px;
		}

		.vertical-icon {
			display: none;
		}

		.horizontal-card {
			height: 88px;
		}

		.horizontal-image,
		.horizontal-image-placeholder {
			width: 88px;
		}

		.horizontal-price {
			font-size: 15px;
			margin-right: 12px;
		}
	}
</style>
