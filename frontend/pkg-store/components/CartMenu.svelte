<script lang="ts">
	export interface IProps {
		id: number;
		isMobile?: boolean;
		css?: string;
	}

	const { id = 0, isMobile = false, css = "" }: IProps = $props();

	import { layerOpenedState, ProductsSelectedMap } from "./store.svelte";
	import angleSvg from "$ui/assets/angle.svg?raw";
	import { parseSVG } from "$core/helpers";
	import s1 from "./styles.module.css";
	import ArrowSteps from "$components/ArrowSteps.svelte";
	import { Globals } from "$store/stores/globals.svelte";
	import Input from "$components/Input.svelte";
	import CiudadesSelector from "$store/components/CiudadesSelector.svelte";
	import ProductCardHorizonal from "$store/components/ProductCardHorizonal.svelte";
	import { Core } from "$core/store.svelte";
	import { Ecommerce } from "$store/stores/globals.svelte";

	import ButtonLayer from "$components/ButtonLayer.svelte";

	import { Env } from "$core/env";

	import CulqiCheckout from "./CulqiCheckout.svelte";

	let userForm = {} as any;

	let isOpen = $state(false);

	const total = $derived(
		Array.from(ProductsSelectedMap.values()).reduce((acc, curr) => {
			const price = curr.producto.PrecioFinal || curr.producto.Precio || 0;

			return acc + price * curr.cant;
		}, 0),
	);

	function toggleCartDiv() {
		layerOpenedState.id = layerOpenedState.id === id ? 0 : id;
	}

	$effect(() => {
		if (layerOpenedState.id === id) {
			isOpen = true;
			Env.loadEmpresaConfig();
		} else {
			isOpen = false;
		}
	});

	$effect(() => {
		if (isOpen) {
			if (layerOpenedState.id !== id) layerOpenedState.id = id;
		} else if (layerOpenedState.id === id) {
			layerOpenedState.id = 0;
		}
	});
</script>

{#snippet cartContent(isMobileVersion: boolean)}
	<div class="p-4 md:p-12 flex flex-col h-[calc(100vh-120px)]">
		<ArrowSteps
			selected={Ecommerce.cartOption}
			columnsTemplate={!isMobileVersion && Core.deviceType === 3
				? "1fr 1fr 1fr 0.7fr"
				: ""}
			onSelect={(e) => {
				Ecommerce.cartOption = e.id;
			}}
			options={[
				{ id: 1, name: "Carrito", icon: "icon-basket" },
				{ id: 2, name: "Datos Envío", icon: "icon-doc-inv-alt" },
				{ id: 3, name: "Pago", icon: "icon-shield" },
				{ id: 4, name: "Confirma ción", icon: "icon-ok" },
			]}
		>
			{#snippet optionRender(e)}
				<div class="flex items-center mt-1 ff-semibold">
					{#if !isMobileVersion}
						<i class={`text-[18px] ${e.icon} mr-2`}></i>
					{/if}
					<div class={["lh-11", !isMobileVersion ? "text-left mr-6" : "text-center"].join(" ")}>{e.name}</div>
				</div>
			{/snippet}
		</ArrowSteps>
		<div class="w-full px-4 overflow-auto grow">
			{#if Ecommerce.cartOption === 1}
				<div class="mt-8 mb-8 fs18 ff-bold">Total a pagar:</div>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-12 pb-6">
					{#each ProductsSelectedMap.values() as cartProducto}
						<ProductCardHorizonal productoID={cartProducto.producto.ID} />
					{/each}
				</div>
			{/if}
			{#if Ecommerce.cartOption === 2}
				<div class="mt-8 mb-8 fs18 ff-bold">Total a pagar:</div>
				<div class="grid grid-cols-12 gap-12">
					<Input
						label="Nombres"
						css="col-span-6"
						saveOn={userForm}
						save="nombres"
						required={true}
					/>
					<Input
						label="Apellidos"
						css="col-span-6"
						saveOn={userForm}
						save="apellidos"
						required={true}
					/>
					<Input
						label="Correo Electrónico"
						css="col-span-6"
						saveOn={userForm}
						save="email"
						required={true}
					/>
					<CiudadesSelector
						saveOn={userForm}
						save="ciudadID"
						css="col-span-6"
					/>
					<Input
						label="Dirección"
						css="col-span-6"
						saveOn={userForm}
						save="direccion"
						required={true}
					/>
					<Input
						label="Referencia"
						css="col-span-6"
						saveOn={userForm}
						save="referencia"
					/>
				</div>
			{/if}
			{#if Ecommerce.cartOption === 3}
				<div class="flex flex-col h-full">
					<div class="fs18 ff-bold mb-4 py-8">
						Total a pagar: S/ {total.toFixed(2)}
					</div>

					<CulqiCheckout
						amount={total * 100}
						email={userForm.email}
						onSuccess={() => {
							Ecommerce.cartOption = 4;
						}}
					/>

					<div class="py-4 text-center text-gray-500 text-sm">
						Pago seguro procesado por Culqi
					</div>
				</div>
			{/if}
			{#if Ecommerce.cartOption === 4}
				<div class="mt-12 text-center">
					<i class="icon-ok text-green-500 text-[64px] mb-4"></i>
					<div class="fs24 ff-bold mb-2">¡Gracias por tu compra!</div>
					<p class="text-gray-600 mb-8">
						Tu pedido ha sido procesado exitosamente. Recibirás un correo de
						confirmación a la brevedad.
					</p>
					<button
						class="bg-gray-800 text-white px-8 py-3 rounded-lg ff-bold hover:bg-black transition-all cursor-pointer"
						onclick={() => {
							ProductsSelectedMap.clear();
							Ecommerce.cartOption = 1;
							layerOpenedState.id = 0;
						}}
					>
						Volver a la tienda
					</button>
				</div>
			{/if}
		</div>
	</div>
{/snippet}

{#if !isMobile}
	<div class={css}>
		<ButtonLayer
			bind:isOpen
			wrapperClass="w-full h-full"
			buttonClass="w-full h-full"
			horizontalOffset={-200}
			edgeMargin={32}
			layerClass="w-768! max-w-[82vw]! rounded-[11px]!"
			useBig
		>
			{#snippet button(open)}
				<button
					class={[s1.bn1, "w-full", open ? s1.button_menu_top : ""].join(" ")}
				>
					<i class="icon1-basket"></i>
					<span>Carrito</span>
				</button>
			{/snippet}

			{@render cartContent(false)}
		</ButtonLayer>
	</div>
{:else if layerOpenedState.id === id}
	<div class={css}>
		<img class={"absolute h-20 _1 right-17"} alt="" src={parseSVG(angleSvg)} />
		<div class="_2 absolute p-12 flex flex-col">
			{@render cartContent(true)}
		</div>
	</div>
{/if}

<style>
	._1 {
		bottom: -10px;
		z-index: 121;
	}
	._2 {
		top: calc(100% + 6px);
		width: 100vw;
		max-width: 100vw;
		left: 0;
		padding: 10px 6px;
		height: calc(100vh - 120px);
		background-color: #fff;
		z-index: 120;
		box-shadow:
			#46466059 0 2px 18px -2px,
			#00000059 0 0 6px;
		border-radius: 11px;
	}
</style>
