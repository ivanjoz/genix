<script lang="ts">
	import { formatN } from "$libs/helpers";
	import {
		getRecordWithCache,
		type IMinimalRecord,
		type IRecordRef,
	} from "$libs/cache/cache-by-ids.svelte";
	import { untrack } from "svelte";
	import type { IProducto } from "$services/services/productos.svelte";
	import ImageHash from "$components/Imagehash.svelte";
	import { addProductoCant, ProductsSelectedMap } from "./store.svelte";

	type ProductCardMode = "vertical" | "horizontal";

	interface IProductoByIDRecord extends IMinimalRecord {
		Nombre?: string;
		Precio?: number;
		PrecioFinal?: number;
		Image?: { n?: string; d?: string };
		Images?: Array<{ n?: string; d?: string }>;
	}

	export interface IProductCard {
		css?: string;
		productoID?: number;
		producto?: IProducto;
		mode?: ProductCardMode;
		hideCloseButton?: boolean;
		useQuantityControls?: boolean;
	}

	const {
		css = "",
		productoID,
		producto,
		mode = "vertical",
		hideCloseButton = false,
		useQuantityControls = true,
	}: IProductCard = $props();

	let productRecordReference = $state<IRecordRef<IProductoByIDRecord> | null>(null);
	let lastResolvedProductID = $state(0);

	const resolvedProductID = $derived.by(() => productoID || producto?.ID || 0);
	const fetchedProductRecord = $derived(productRecordReference?.record || null);
	const resolvedProduct = $derived.by<IProducto>(() => {
		const fallbackProductRecord = producto || ({ ID: resolvedProductID } as IProducto);
		if (!fetchedProductRecord) return fallbackProductRecord;
		return {
			...fallbackProductRecord,
			...(fetchedProductRecord as Partial<IProducto>),
			ID: fetchedProductRecord.ID,
		} as IProducto;
	});
	const selectedProductQuantity = $derived.by(() => {
		return ProductsSelectedMap.get(resolvedProduct?.ID)?.cant || 0;
	});
	const resolvedProductPriceCents = $derived.by(() => {
		return resolvedProduct?.PrecioFinal || resolvedProduct?.Precio || 0;
	});
	const resolvedProductImageName = $derived.by(() => {
		return resolvedProduct?.Image?.n || resolvedProduct?.Images?.[0]?.n || "";
	});
	const isLoadingProductRecord = $derived(productRecordReference?.loading || false);

	$effect(() => {
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
	<div class={["horizontal-card", css].join(" ")} aria-busy={isLoadingProductRecord}>
		<div class="horizontal-image-wrapper">
			{#if resolvedProductImageName}
				<ImageHash
					css="horizontal-image"
					src={resolvedProductImageName}
					folder="img-productos"
				/>
			{:else}
				<div class="horizontal-image-placeholder"></div>
			{/if}
		</div>
		<div class="horizontal-body">
			<div class="horizontal-name" title={resolvedProduct.Nombre || `Producto #${resolvedProductID}`}>
				{resolvedProduct.Nombre || `Producto #${resolvedProductID}`}
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
		</div>
			{#if !hideCloseButton}
				<!-- Keep explicit remove action optional so search-card usage can hide destructive controls. -->
				<button class="remove-button" onclick={removeProductFromSelection} aria-label="Remover">
					<i class="icon-cancel"></i>
				</button>
			{/if}
		</div>
{:else}
	<div class="vertical-card-shell">
		<div class={["vertical-card", css].join(" ")} aria-busy={isLoadingProductRecord}>
			<ImageHash
				css="w-full h-[36vw] md:h-200"
				src={resolvedProductImageName}
				folder="img-productos"
			/>
			<div class="vertical-content pb-2">
				<div class="vertical-name mt-6 mb-4 min-h-26 md:min-h-32 fx-c">
					{resolvedProduct.Nombre || "???"}
				</div>
				<div class="px-4 ff-bold fs17">s/. {formatN(resolvedProductPriceCents / 100, 2)}</div>
				<div class="vertical-icon fx-c h-30 w-32">
					<i class="icon1-basket"></i>
				</div>
			</div>
			<button class="vertical-add-button" onclick={incrementSelectedQuantity} type="button">
				{#if selectedProductQuantity === 0}
					Agregar <i class="icon1-basket"></i>
				{/if}
				{#if selectedProductQuantity > 0}
					Agregar mas ({selectedProductQuantity}) <i class="icon1-basket"></i>
				{/if}
			</button>
		</div>
	</div>
{/if}

<style>
	.vertical-card {
		position: relative;
		background-color: white;
		box-shadow: rgba(54, 56, 67, 0.2) 0px 2px 8px 0px;
		min-height: 180px;
		padding: 6px;
		border-radius: 10px;
		margin-bottom: 24px;
	}

	.vertical-add-button {
		color: rgb(100, 67, 160);
		position: absolute;
		border-radius: 0 0 10px 10px;
		height: 36px;
		align-items: center;
		justify-content: center;
		display: none;
		width: 100%;
		bottom: 0;
		left: 0;
		user-select: none;
		border: none;
		background: transparent;
		font-family: inherit;
		font-size: inherit;
		padding: 0;
	}

	.vertical-content {
		position: relative;
		z-index: 10;
		width: 100%;
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

	@media (min-width: 739px) {
		.vertical-card-shell:hover .vertical-card {
			padding-bottom: 22px;
			margin-bottom: -10px;
			outline: 2px solid rgb(202, 173, 255);
		}

		.vertical-card-shell:hover .vertical-add-button {
			display: flex;
			background-color: rgb(228, 221, 248);
			cursor: pointer;
		}

		.vertical-card-shell:hover .vertical-content {
			background-color: white;
			margin-top: -14px;
			margin-bottom: 14px;
		}

		.vertical-add-button:hover,
		.vertical-card-shell .vertical-add-button:hover {
			background-color: rgb(111, 82, 179);
			height: 38px;
			padding-bottom: 2px;
			margin-bottom: -2px;
			margin-left: -2px;
			margin-right: -2px;
			width: calc(100% + 4px);
			color: white;
		}

		.vertical-card-shell:hover .vertical-icon {
			display: none;
		}
	}

	@media (max-width: 740px) {
		.vertical-card-shell .vertical-add-button {
			display: flex;
			background-color: rgb(228, 221, 248);
			cursor: pointer;
		}

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
