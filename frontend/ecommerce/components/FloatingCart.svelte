<script lang="ts">
	import { formatN, parseSVG } from "$libs/helpers";
	import iconCartSvg from "$libs/assets/icon_cart.svg?raw";
	import iconCancelSvg from "$libs/assets/icon_cancel.svg?raw";
	import { layerOpenedState, ProductsSelectedMap } from "./store.svelte";
	import { Ecommerce } from "$ecommerce/stores/globals.svelte";
	import ProductCard from "$ecommerce/components/ProductCard.svelte";

	let isOpen = $state(false);

	const cartCount = $derived(ProductsSelectedMap.size);
	const totalCents = $derived(
		Array.from(ProductsSelectedMap.values()).reduce((acc, curr) => {
			const price = curr.producto.PrecioFinal || curr.producto.Precio || 0;
			return acc + price * curr.cant;
		}, 0),
	);

	const isCartMenuOpen = $derived(
		layerOpenedState.id === 1 || layerOpenedState.id === 2,
	);

	function toggle(ev: MouseEvent) {
		ev.stopPropagation();
		isOpen = !isOpen;
	}

	function goToPay(ev: MouseEvent) {
		ev.stopPropagation();
		Ecommerce.cartOption = 2;
		const isDesktop =
			typeof window !== "undefined" &&
			window.matchMedia("(min-width: 768px)").matches;
		layerOpenedState.id = isDesktop ? 1 : 2;
		isOpen = false;
	}
</script>

{#if !isCartMenuOpen}
	<div class="floating-cart-ref fx-c" class:is-open={isOpen}>
		<div class="floating-cart-ctn">
			<div class="floating-cart-content">
				<div class="floating-cart-list">
					{#each ProductsSelectedMap.values() as cartProducto (cartProducto.producto.ID)}
						<ProductCard
							mode="horizontal"
							productoID={cartProducto.producto.ID}
						/>
					{/each}
				</div>
				<button
					type="button"
					class="floating-cart-card fx-c"
					onclick={goToPay}
				>
					<div class="ff-bold-italic mr-3">Ir a Pagar -</div>
					<div class="ff-bold-italic">
						S/. {formatN(totalCents / 100, 2)}
					</div>
				</button>
			</div>
			<button
				type="button"
				class="floating-cart-btn fx-c"
				class:has-products={cartCount > 0}
				onclick={toggle}
				aria-label="Carrito flotante"
			>
				<img alt="" src={parseSVG(isOpen ? iconCancelSvg : iconCartSvg)} />
			</button>
		</div>
		{#if cartCount > 0 && !isOpen}
			<div class="floating-cart-count ff-semibold fx-c">{cartCount}</div>
		{/if}
	</div>
{/if}

<style>
	.floating-cart-ref {
		position: fixed;
		bottom: 24px;
		right: 24px;
		--size1: 5rem;
		--content-width: 30rem;
		height: var(--size1);
		width: var(--size1);
		z-index: 111;
	}

	.floating-cart-ref,
	.floating-cart-ref * {
		-webkit-tap-highlight-color: transparent;
		-webkit-user-select: none;
		user-select: none;
	}

	.floating-cart-ctn {
		position: absolute;
		bottom: 0;
		right: 0;
		height: 100%;
		width: 100%;
		background-color: white;
		border-radius: 8px;
		box-shadow:
			rgba(50, 50, 93, 0.25) 0px 8px 27px -4px,
			rgba(0, 0, 0, 0.3) 0px 6px 16px -8px;
		cursor: pointer;
		overflow: hidden;
		transition:
			width 0.28s ease,
			height 0.28s ease;
	}

	.floating-cart-ctn:hover {
		outline: 1px solid rgb(167, 125, 236);
	}

	.floating-cart-ref.is-open .floating-cart-ctn {
		height: 34rem;
		max-width: calc(100vw - 2rem);
		width: var(--content-width);
		outline: none;
	}

	.floating-cart-content {
		position: absolute;
		bottom: 0;
		right: 0;
		padding: 6px;
		height: 34rem;
		max-width: calc(100vw - 2rem);
		width: var(--content-width);
	}

	.floating-cart-list {
		overflow: auto;
		padding: 4px;
		height: calc(100% - 5rem);
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.floating-cart-btn {
		border-radius: 8px;
		height: var(--size1);
		width: var(--size1);
		padding: 0;
		position: absolute;
		bottom: 0;
		right: 0;
		background-color: white;
		border: none;
		cursor: pointer;
	}

	.floating-cart-ctn:hover .floating-cart-btn {
		background-color: rgb(246, 242, 255);
	}

	.floating-cart-btn > img {
		width: 70%;
	}

	.floating-cart-btn.has-products {
		padding-top: 10px;
	}

	.floating-cart-ref.is-open .floating-cart-btn {
		border-radius: 0 0 8px 0;
		padding: 1.6rem;
		background-color: rgb(255, 232, 232);
		outline: none;
	}

	.floating-cart-card {
		position: absolute;
		height: var(--size1);
		bottom: 0;
		left: 0;
		width: calc(100% - var(--size1));
		background-color: rgb(85, 75, 179);
		color: white;
		visibility: hidden;
		border-radius: 0 0 8px 8px;
		border: none;
		cursor: pointer;
		font-size: 1.25rem;
	}

	.floating-cart-ref.is-open .floating-cart-card {
		visibility: visible;
	}

	.floating-cart-card:hover {
		background-color: rgb(98, 87, 197);
	}

	.floating-cart-count {
		position: absolute;
		top: -8px;
		right: -8px;
		width: 2.4rem;
		height: 2.4rem;
		border-radius: 50%;
		background-color: #e75e5e;
		color: white;
		z-index: 112;
		pointer-events: none;
	}

	@media (max-width: 740px) {
		.floating-cart-ref {
			bottom: calc(1rem - 2px);
			right: calc(1rem - 2px);
			--size1: 4.5rem;
			--content-width: calc(100vw - 2rem + 4px);
		}
		.floating-cart-count {
			width: 2rem;
			height: 2rem;
			font-size: var(--fs2);
		}
		.floating-cart-card {
			font-size: 1.2rem;
		}
		.floating-cart-ref.is-open .floating-cart-ctn {
			height: 72vh;
		}
		.floating-cart-content {
			height: 72vh;
		}
	}
</style>
