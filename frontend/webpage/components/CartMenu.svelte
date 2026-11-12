<script lang="ts">
	export interface IProps {
		id: number;
		isMobile?: boolean;
		css?: string;
	}

	const { id = 0, isMobile = false, css = "" }: IProps = $props();

	import { layerOpenedState, ProductsSelectedMap } from "./store.svelte";
	import angleSvg from "#components/assets/angle.svg?raw";
	import { parseSVG } from '#libs/helpers.ts';
	import s1 from "./styles.module.css";
	import ArrowSteps from "#components/navigation/ArrowSteps.svelte";
	import Input from "#components/form/Input.svelte";
	import ProductCard from "#ecommerce/components/ProductCard.svelte";
	import AddressForm from "#ecommerce/components/AddressForm.svelte";
	import { useUI } from '@genix/ui';
	import { notifyFailure, showLoading, hideLoading } from '@genix/ui/notify';
	import { Ecommerce } from "#ecommerce/stores/globals.svelte.ts";
	import ButtonLayer from "#components/buttons/ButtonLayer.svelte";
	import { useCitiesAPI } from '#services/services/cities.svelte.ts';
	import CulqiCheckout from "./CulqiCheckout.svelte";
	import {
		WebStore, loadWebStoreCompany, restoreWebCustomerSession, saveCustomerDraft, identifyWebCheckout,
		verifyWebCustomerCode, getWebShippingAmount, postWebCheckout, downloadWebOrderPDF,
		type IWebCheckoutResult,
	} from "#ecommerce/services/web-store.svelte.ts";
	import {
		DELIVERY_HOME, DELIVERY_PICKUP, PAYMENT_CARD, PAYMENT_YAPE, PAYMENT_DEFERRED, NEW_ADDRESS_ID,
		FLAG_CASH_ON_DELIVERY, FLAG_REQUIRE_CONTACT_NUMBER, cartTotalCents, formatSoles, hasCompanyFlag,
		requiresDocumentNumber, validateCustomerDraft, validateDelivery, defaultAddressID,
		deferredPaymentLabel, buildWebCheckoutRequest, isOnlinePayment,
	} from "#ecommerce/services/web-checkout.ts";

	const ui = useUI();
	const cities = useCitiesAPI();

	let isOpen = $state(false);

	// The checkout choices of this visit to the cart.
	const checkout = $state({
		deliveryType: 0,
		pickupSiteID: 0,
		selectedAddressID: NEW_ADDRESS_ID,
		newAddress: { CityID: 0, Address: "", Reference: "" },
		// null = not resolved yet; -1 = the store does not deliver to that district.
		shippingAmount: null as number | null,
		paymentMethod: 0,
		// Set when identification mailed a code: the shopper types it before paying.
		codeRequired: false,
		code: { value: "" },
		// The online payment widget is shown once the shopper is identified.
		showCulqi: false,
		result: null as IWebCheckoutResult | null,
	});

	const cartLines = $derived(
		Array.from(ProductsSelectedMap.values()).map((cartProduct) => ({
			productID: cartProduct.producto.ID,
			quantity: cartProduct.cant,
			unitPriceCents: cartProduct.producto.FinalPrice || cartProduct.producto.Price || 0,
		})),
	);
	const productsCents = $derived(cartTotalCents(cartLines));
	const shippingCents = $derived(
		checkout.deliveryType === DELIVERY_HOME && (checkout.shippingAmount || 0) > 0 ? checkout.shippingAmount || 0 : 0,
	);
	const totalCents = $derived(productsCents + shippingCents);
	const companyFlags = $derived(WebStore.company.Flags);
	const needsDocumentNumber = $derived(requiresDocumentNumber(companyFlags, totalCents));

	// The district the order ships to: the chosen saved address, or the one being typed.
	const deliveryDistrictID = $derived.by(() => {
		if (checkout.deliveryType !== DELIVERY_HOME) { return 0 }
		if (checkout.selectedAddressID === NEW_ADDRESS_ID) { return checkout.newAddress.CityID || 0 }
		return WebStore.addresses.find((address) => address.ID === checkout.selectedAddressID)?.CityID || 0
	});

	const paymentOptions = $derived.by(() => {
		const options = [{ id: PAYMENT_CARD, name: "Tarjeta de crédito / débito", icon: "icon-[fa--credit-card]" }];
		if (WebStore.company.YapeEnabled) {
			options.push({ id: PAYMENT_YAPE, name: "Yape", icon: "icon-[fa--mobile]" });
		}
		if (hasCompanyFlag(companyFlags, FLAG_CASH_ON_DELIVERY)) {
			options.push({ id: PAYMENT_DEFERRED, name: deferredPaymentLabel(checkout.deliveryType), icon: "icon-[fa--money]" });
		}
		return options;
	});

	const cityName = (cityID: number): string => {
		const district = cities.citiesMap?.get(cityID);
		if (!district) { return "" }
		const province = cities.citiesMap.get(district.ParentID);
		return [district.Name, province?.Name].filter(Boolean).join(", ");
	};

	function toggleCartDiv() {
		layerOpenedState.id = layerOpenedState.id === id ? 0 : id;
	}

	$effect(() => {
		if (layerOpenedState.id === id) {
			isOpen = true;
			loadWebStoreCompany();
			restoreWebCustomerSession().then(() => {
				if (checkout.selectedAddressID === NEW_ADDRESS_ID) {
					checkout.selectedAddressID = defaultAddressID(WebStore.addresses);
				}
			});
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

	// The delivery rate follows the district: same fallback (district → province → department)
	// the backend charges.
	$effect(() => {
		const districtID = deliveryDistrictID;
		checkout.shippingAmount = null;
		if (districtID < 10000) { return }
		getWebShippingAmount(districtID)
			.then((amount) => { if (deliveryDistrictID === districtID) { checkout.shippingAmount = amount } })
			.catch(() => { if (deliveryDistrictID === districtID) { checkout.shippingAmount = -1 } });
	});

	// validateShippingStep: the contact and the destination, before the Pago step.
	const validateShippingStep = (): string => {
		const contactProblem = validateCustomerDraft(WebStore.draft, companyFlags, needsDocumentNumber);
		if (contactProblem) { return contactProblem }
		const deliveryProblem = validateDelivery(
			checkout.deliveryType, checkout.pickupSiteID, checkout.selectedAddressID, checkout.newAddress,
		);
		if (deliveryProblem) { return deliveryProblem }
		if (checkout.deliveryType === DELIVERY_HOME && checkout.shippingAmount === -1) {
			return "No hacemos envíos a este distrito.";
		}
		if (checkout.deliveryType === DELIVERY_HOME && checkout.shippingAmount === null) {
			return "Calculando el costo de envío, intente nuevamente.";
		}
		return "";
	};

	const goToStep = (step: number) => {
		if (step === 4 && !checkout.result) { return }
		if (step === 1 && checkout.result) { return }
		if (step >= 2 && ProductsSelectedMap.size === 0) {
			notifyFailure("El carrito está vacío.");
			return;
		}
		if (step >= 3) {
			const problem = validateShippingStep();
			if (problem) { notifyFailure(problem); return }
			saveCustomerDraft();
		}
		Ecommerce.cartOption = step;
	};

	// The contact typed in Datos Envío, kept apart: a session answer refills the draft from the
	// account and must not overwrite what this order ships to.
	let contactForOrder = $state(WebStore.draft);

	const placeOrder = async (culqiToken: string) => {
		showLoading();
		try {
			checkout.result = await postWebCheckout(buildWebCheckoutRequest({
				cartLines,
				draft: contactForOrder,
				deliveryType: checkout.deliveryType,
				pickupSiteID: checkout.pickupSiteID,
				selectedAddressID: checkout.selectedAddressID,
				newAddress: checkout.newAddress,
				paymentMethod: checkout.paymentMethod,
				culqiToken,
			}));
			checkout.showCulqi = false;
			ProductsSelectedMap.clear();
			Ecommerce.cartOption = 4;
		} catch (error: any) {
			// A declined card voids the order: the shopper may retry with another token.
			checkout.showCulqi = false;
			notifyFailure(error.message);
		} finally {
			hideLoading();
		}
	};

	// After the shopper is identified: online payments open Culqi, deferred ones place the order.
	const continueIdentified = async () => {
		checkout.codeRequired = false;
		if (isOnlinePayment(checkout.paymentMethod)) {
			checkout.showCulqi = true;
		} else {
			await placeOrder("");
		}
	};

	// onPay: identify the guest (an emailed code may be needed), then pay.
	const onPay = async () => {
		if (!checkout.paymentMethod) { notifyFailure("Seleccione un medio de pago."); return }
		const problem = validateShippingStep();
		if (problem) { notifyFailure(problem); return }
		contactForOrder = { ...WebStore.draft };
		if (WebStore.customer) { await continueIdentified(); return }

		showLoading();
		try {
			const result = await identifyWebCheckout(checkout.paymentMethod, []);
			if (result.CodeRequired) {
				checkout.codeRequired = true;
				checkout.code.value = "";
				return;
			}
		} catch (error: any) {
			notifyFailure(error.message);
			return;
		} finally {
			hideLoading();
		}
		await continueIdentified();
	};

	const onVerifyCode = async () => {
		const code = (checkout.code.value || "").trim();
		if (!/^[0-9]{6}$/.test(code)) { notifyFailure("Ingrese el código de 6 dígitos."); return }
		showLoading();
		try {
			await verifyWebCustomerCode(contactForOrder.Email, code);
		} catch (error: any) {
			notifyFailure(error.message);
			return;
		} finally {
			hideLoading();
		}
		await continueIdentified();
	};

	const resetCheckout = () => {
		checkout.result = null;
		checkout.paymentMethod = 0;
		checkout.showCulqi = false;
		checkout.codeRequired = false;
		checkout.newAddress = { CityID: 0, Address: "", Reference: "" };
		checkout.selectedAddressID = defaultAddressID(WebStore.addresses);
		Ecommerce.cartOption = 1;
		layerOpenedState.id = 0;
	};
</script>

{#snippet choiceCard(isSelected: boolean, icon: string, title: string, subtitle: string, onclick: () => void)}
	<button
		type="button"
		class={[
			"flex items-center gap-12 p-14 rounded-[10px] border-2 text-left cursor-pointer transition-colors w-full",
			isSelected ? "border-[#4c55d5] bg-[#f1f2ff]" : "border-gray-200 hover:border-gray-400",
		].join(" ")}
		{onclick}
	>
		<i class={`${icon} text-[26px] shrink-0 ${isSelected ? "text-[#4c55d5]" : "text-gray-500"}`}></i>
		<div class="flex flex-col min-w-0">
			<span class="ff-bold">{title}</span>
			{#if subtitle}<span class="text-gray-500 text-[14px] truncate">{subtitle}</span>{/if}
		</div>
	</button>
{/snippet}

{#snippet totals()}
	<div class="mt-8 mb-8 flex flex-wrap gap-x-24 gap-y-4 items-baseline">
		<span class="fs18 ff-bold">Total a pagar: {formatSoles(totalCents)}</span>
		{#if checkout.deliveryType === DELIVERY_HOME && shippingCents > 0}
			<span class="text-gray-500">Incluye envío: {formatSoles(shippingCents)}</span>
		{/if}
	</div>
{/snippet}

{#snippet cartContent(isMobileVersion: boolean)}
	<div class="p-4 md:p-12 flex flex-col h-[calc(100vh-120px)]">
		<ArrowSteps
			selected={Ecommerce.cartOption}
			columnsTemplate={!isMobileVersion && ui.state.deviceType === 3
				? "1fr 1fr 1fr 0.7fr"
				: ""}
			onSelect={(e) => goToStep(e.id)}
			options={[
				{ id: 1, name: "Carrito", icon: "icon--supermarket-cart" },
				{ id: 2, name: "Datos Envío", icon: "icon-[fa--file-text]" },
				{ id: 3, name: "Pago", icon: "icon-[fa--shield]" },
				{ id: 4, name: "Confirma ción", icon: "icon-[fa--check]" },
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
				{@render totals()}
				<div class="grid grid-cols-1 md:grid-cols-2 gap-12 pb-6">
					{#each ProductsSelectedMap.values() as cartProducto}
						<ProductCard mode="horizontal" productoID={cartProducto.producto.ID} />
					{/each}
				</div>
				{#if ProductsSelectedMap.size > 0}
					<div class="flex justify-end pb-12">
						<button class="bg-[#4c55d5] text-white px-24 py-10 rounded-lg ff-bold hover:bg-[#3b44b8] cursor-pointer"
							onclick={() => goToStep(2)}>Continuar</button>
					</div>
				{/if}
			{/if}
			{#if Ecommerce.cartOption === 2}
				{@render totals()}
				<div class="ff-bold mb-12">Datos de contacto</div>
				<div class="grid grid-cols-12 gap-12">
					<Input label="Nombres" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="FirstName" required={true} />
					<Input label="Apellidos" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="LastName" required={true} />
					<!-- A logged-in customer's email is the account: it is not edited here. -->
					<Input label="Correo Electrónico" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="Email"
						required={true} disabled={!!WebStore.customer} />
					<Input label="Celular" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="Phone"
						required={hasCompanyFlag(companyFlags, FLAG_REQUIRE_CONTACT_NUMBER)} />
					{#if needsDocumentNumber}
						<Input label="DNI (compras desde S/ 700)" css="col-span-12 md:col-span-6" saveOn={WebStore.draft}
							save="DocumentNumber" required={true} />
					{/if}
				</div>

				<div class="ff-bold mt-20 mb-12">Tipo de entrega</div>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-12">
					{@render choiceCard(checkout.deliveryType === DELIVERY_HOME, "icon-[fa--truck]", "Entrega a Domicilio",
						"Lo llevamos a tu dirección", () => { checkout.deliveryType = DELIVERY_HOME })}
					{@render choiceCard(checkout.deliveryType === DELIVERY_PICKUP, "icon-[fa--building]", "Recojo en Tienda",
						"Recoge tu pedido sin costo de envío", () => { checkout.deliveryType = DELIVERY_PICKUP })}
				</div>

				{#if checkout.deliveryType === DELIVERY_HOME}
					<div class="ff-bold mt-20 mb-12">Dirección de entrega</div>
					{#if WebStore.addresses.length > 0}
						<div class="grid grid-cols-1 md:grid-cols-2 gap-12 mb-12">
							{#each WebStore.addresses as address (address.ID)}
								{@render choiceCard(checkout.selectedAddressID === address.ID, "icon-[fa--map-marker]", address.Address,
									cityName(address.CityID), () => { checkout.selectedAddressID = address.ID })}
							{/each}
							{@render choiceCard(checkout.selectedAddressID === NEW_ADDRESS_ID, "icon-[fa--plus]", "Nueva dirección",
								"", () => { checkout.selectedAddressID = NEW_ADDRESS_ID })}
						</div>
					{/if}
					{#if checkout.selectedAddressID === NEW_ADDRESS_ID}
						<AddressForm address={checkout.newAddress} />
					{/if}
					<div class="mt-12 pb-12">
						{#if checkout.shippingAmount === -1}
							<span class="text-red-600">No hacemos envíos a este distrito.</span>
						{:else if checkout.shippingAmount !== null}
							<span class="text-gray-600">Costo de envío: <b>{formatSoles(checkout.shippingAmount)}</b></span>
						{/if}
					</div>
				{/if}

				{#if checkout.deliveryType === DELIVERY_PICKUP}
					<div class="ff-bold mt-20 mb-12">Tienda de recojo</div>
					{#if (WebStore.company.PickupSites || []).length === 0}
						<div class="text-gray-500">La tienda aún no tiene puntos de recojo.</div>
					{/if}
					<div class="grid grid-cols-1 md:grid-cols-2 gap-12 pb-12">
						{#each WebStore.company.PickupSites || [] as site (site.ID)}
							{@render choiceCard(checkout.pickupSiteID === site.ID, "icon-[fa--building]", site.Name,
								[site.Address, cityName(site.CityID)].filter(Boolean).join(" · "),
								() => { checkout.pickupSiteID = site.ID })}
						{/each}
					</div>
				{/if}

				<div class="flex justify-end py-12">
					<button class="bg-[#4c55d5] text-white px-24 py-10 rounded-lg ff-bold hover:bg-[#3b44b8] cursor-pointer"
						onclick={() => goToStep(3)}>Continuar al pago</button>
				</div>
			{/if}
			{#if Ecommerce.cartOption === 3}
				<div class="flex flex-col h-full">
					{@render totals()}
					{#if checkout.showCulqi}
						{#key checkout.paymentMethod}
							<CulqiCheckout
								amount={totalCents}
								email={contactForOrder.Email}
								useYape={checkout.paymentMethod === PAYMENT_YAPE}
								onSuccess={(tokenID) => placeOrder(tokenID)}
								onError={(error) => notifyFailure(error?.user_message || "No se pudo procesar el pago.")}
							/>
						{/key}
						<div class="py-4 text-center text-gray-500 text-[14px]">Pago seguro procesado por Culqi</div>
						<button class="text-[#4c55d5] hover:underline cursor-pointer self-center"
							onclick={() => { checkout.showCulqi = false }}>Elegir otro medio de pago</button>
					{:else if checkout.codeRequired}
						<div class="flex flex-col gap-12 max-w-[420px]">
							<p class="text-gray-700">
								Enviamos un código de 6 dígitos a <b>{contactForOrder.Email}</b>. Ingréselo para confirmar su pedido.
							</p>
							<Input label="Código" saveOn={checkout.code} save="value" required={true} />
							<div class="flex gap-12">
								<button class="bg-[#4c55d5] text-white px-24 py-10 rounded-lg ff-bold hover:bg-[#3b44b8] cursor-pointer"
									onclick={onVerifyCode}>Confirmar</button>
								<button class="text-[#4c55d5] hover:underline cursor-pointer" onclick={onPay}>Reenviar código</button>
							</div>
						</div>
					{:else}
						<div class="ff-bold mb-12">Medio de pago</div>
						<div class="grid grid-cols-1 md:grid-cols-2 gap-12">
							{#each paymentOptions as option (option.id)}
								{@render choiceCard(checkout.paymentMethod === option.id, option.icon, option.name, "",
									() => { checkout.paymentMethod = option.id })}
							{/each}
						</div>
						<div class="flex justify-end py-16">
							<button class="bg-[#4c55d5] text-white px-24 py-10 rounded-lg ff-bold hover:bg-[#3b44b8] cursor-pointer"
								onclick={onPay}>
								{checkout.paymentMethod === PAYMENT_DEFERRED ? "Confirmar pedido" : "Pagar " + formatSoles(totalCents)}
							</button>
						</div>
					{/if}
				</div>
			{/if}
			{#if Ecommerce.cartOption === 4 && checkout.result}
				{@const result = checkout.result}
				<div class="mt-12 text-center">
					<i class="icon-[fa--check] text-green-500 text-[64px] mb-4"></i>
					<div class="fs24 ff-bold mb-2">¡Gracias por tu compra!</div>
					<div class="ff-semibold mb-4">Pedido N° {result.OrderNumber} · {formatSoles(result.TotalAmount)}</div>
					<p class="text-gray-600 mb-8">
						{#if result.DocumentType === 2}
							Tu pedido quedó registrado. Paga al {checkout.deliveryType === DELIVERY_PICKUP ? "recogerlo en tienda" : "recibirlo"};
							tu orden de compra también te llegará por correo.
						{:else}
							Tu pago fue recibido. Recibirás un correo de confirmación a la brevedad.
						{/if}
					</p>
					<div class="flex flex-wrap justify-center gap-12">
						{#if result.DocumentType === 2}
							<button class="border-2 border-gray-800 px-24 py-10 rounded-lg ff-bold hover:bg-gray-100 cursor-pointer"
								onclick={() => downloadWebOrderPDF(result.OnlineSaleOrderID).catch((error) => notifyFailure(error.message))}>
								<i class="icon-[fa--download] mr-6"></i>Orden de compra (PDF)
							</button>
						{/if}
						<button
							class="bg-gray-800 text-white px-24 py-10 rounded-lg ff-bold hover:bg-black transition-all cursor-pointer"
							onclick={resetCheckout}
						>
							Volver a la tienda
						</button>
					</div>
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
			label="Opens the shopping cart panel."
			useBig
		>
			{#snippet button(open)}
				<button
					class={[s1.bn1, "w-full", open ? s1.button_menu_top : ""].join(" ")}
				>
					<i class="icon--supermarket-cart"></i>
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
