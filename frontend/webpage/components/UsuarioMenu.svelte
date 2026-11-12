<script lang="ts">
  import { layerOpenedState } from "./store.svelte";
  import ButtonLayer from '#components/buttons/ButtonLayer.svelte';
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte';
  import Input from '#components/form/Input.svelte';
  import Modal from '#components/layers/Modal.svelte';
  import { useUI } from '@genix/ui';
  import { notifyFailure, notifySuccess, showLoading, hideLoading } from '@genix/ui/notify';
  import { formatTime } from '#libs/helpers.ts';
  import { unpackQuantityLine } from '#core/quantity.ts';
  import { useCitiesAPI } from '#services/services/cities.svelte.ts';
  import AddressForm from "#ecommerce/components/AddressForm.svelte";
  import s1 from "./styles.module.css"
  import {
    WebStore, loadWebStoreCompany, restoreWebCustomerSession, saveCustomerDraft, logoutWebCustomer,
    loginWebCustomer, requestWebCustomerCode, verifyWebCustomerCode, saveWebCustomerProfile,
    saveWebCustomerPassword, saveWebCustomerAddress, getWebCustomerOrders, downloadWebOrderPDF,
    type IWebOrder,
  } from "#ecommerce/services/web-store.svelte.ts";
  import {
    DELIVERY_PICKUP, FLAG_REQUIRE_CONTACT_NUMBER, formatSoles, hasCompanyFlag, validateAddress,
    validateCustomerDraft, webOrderStatusLabel, type IWebCustomerAddress,
  } from "#ecommerce/services/web-checkout.ts";

  export interface IProps {
    id?: number;
    isMobile?: boolean;
    css?: string;
  }

  const { id = 3, isMobile = false, css = "" }: IProps = $props();

  // Storefront-only modal id: the back-office ids (1, 2, ...) never mount here.
  const ADDRESS_MODAL_ID = 931;

  const ui = useUI();
  const cities = useCitiesAPI();

  let isOpen = $state(false);
  let selectedTab = $state(1);
  let accountSection = $state(1);

  // Login / Regístrate / password forms. `codeSentTo` is the email a 6-digit code went to.
  const forms = $state({
    loginEmail: "", password: "", code: "", codeSentTo: "",
    currentPassword: "", newPassword: "",
  });
  let addressForm = $state({ ID: 0, CityID: 0, Address: "", Reference: "" } as IWebCustomerAddress);
  let orders = $state([] as IWebOrder[]);
  let productNames = $state({} as Record<number, string>);
  let ordersLoaded = $state(false);

  const isLoggedIn = $derived(!!WebStore.customer);
  // Tab ids are shared by both strips: 1 Login / Mi Cuenta, 2 Regístrate / Mis Pedidos.
  const options = $derived(isLoggedIn
    ? [{ id: 1, name: "Mi Cuenta" }, { id: 2, name: "Mis Pedidos" }, { id: 3, name: "Mis Favoritos" }, { id: 4, name: "Config" }]
    : [{ id: 1, name: "Login" }, { id: 2, name: "Regístrate" }, { id: 3, name: "Mis Favoritos" }, { id: 4, name: "Config" }]);
  const accountSections = [
    { id: 1, name: "Mis Datos", icon: "icon-[fa--user]" },
    { id: 2, name: "Mis Direcciones", icon: "icon-[fa--map-marker]" },
    { id: 3, name: "Medios de Pago", icon: "icon-[fa--credit-card]" },
  ];

  $effect(() => {
    if (layerOpenedState.id === id) {
      isOpen = true;
      loadWebStoreCompany();
      restoreWebCustomerSession();
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

  // Mis Pedidos is read when its tab opens.
  $effect(() => {
    if (!(isLoggedIn && selectedTab === 2 && !ordersLoaded)) { return }
    ordersLoaded = true;
    getWebCustomerOrders()
      .then((result) => { orders = result.Orders || []; productNames = result.ProductNames || {} })
      .catch((error) => notifyFailure(error.message));
  });

  const cityName = (cityID: number): string => {
    const district = cities.citiesMap?.get(cityID);
    if (!district) { return "" }
    const province = cities.citiesMap.get(district.ParentID);
    return [district.Name, province?.Name].filter(Boolean).join(", ");
  };

  // runRequest wraps a backend call with the loading overlay and the error toast.
  const runRequest = async (request: () => Promise<unknown>): Promise<boolean> => {
    showLoading();
    try {
      await request();
      return true;
    } catch (error: any) {
      notifyFailure(error.message);
      return false;
    } finally {
      hideLoading();
    }
  };

  const onLoggedIn = () => {
    forms.password = ""; forms.code = ""; forms.codeSentTo = "";
    selectedTab = 1; accountSection = 1; ordersLoaded = false;
  };

  const onLoginWithPassword = async () => {
    if (await runRequest(() => loginWebCustomer(forms.loginEmail.trim(), forms.password))) { onLoggedIn() }
  };

  // sendCode mails the 6-digit code used by the passwordless login and by Regístrate.
  const sendCode = async (email: string) => {
    email = email.trim();
    if (await runRequest(() => requestWebCustomerCode(email))) { forms.codeSentTo = email; forms.code = "" }
  };

  const onRegister = () => {
    const problem = validateCustomerDraft(WebStore.draft, WebStore.company.Flags, false);
    if (problem) { notifyFailure(problem); return }
    saveCustomerDraft();
    sendCode(WebStore.draft.Email);
  };

  const onVerifyCode = async () => {
    if (!/^[0-9]{6}$/.test(forms.code.trim())) { notifyFailure("Ingrese el código de 6 dígitos."); return }
    if (await runRequest(() => verifyWebCustomerCode(forms.codeSentTo, forms.code.trim()))) { onLoggedIn() }
  };

  const onSaveProfile = async () => {
    const problem = validateCustomerDraft(WebStore.draft, WebStore.company.Flags, false);
    if (problem) { notifyFailure(problem); return }
    if (await runRequest(saveWebCustomerProfile)) { notifySuccess("Datos guardados.") }
  };

  const onSavePassword = async () => {
    if (await runRequest(() => saveWebCustomerPassword(forms.currentPassword, forms.newPassword))) {
      forms.currentPassword = ""; forms.newPassword = "";
      notifySuccess("Contraseña guardada.");
    }
  };

  const openAddressModal = (address?: IWebCustomerAddress) => {
    addressForm = address ? { ...address } : { ID: 0, CityID: 0, Address: "", Reference: "" };
    ui.openModal(ADDRESS_MODAL_ID);
  };

  const onSaveAddress = async (isDelete?: boolean) => {
    if (!isDelete) {
      const problem = validateAddress(addressForm);
      if (problem) { notifyFailure(problem); return }
    }
    const address = { ...addressForm, ss: isDelete ? 0 : 1 };
    if (await runRequest(() => saveWebCustomerAddress(address))) { ui.closeModal(ADDRESS_MODAL_ID) }
  };

  function handleLogout() {
    logoutWebCustomer();
    orders = []; ordersLoaded = false; selectedTab = 1;
  }
</script>

{#snippet primaryButton(label: string, onclick: () => void)}
  <button {onclick}
    class="bg-[#4c55d5] text-white py-10 px-24 rounded-lg ff-bold hover:bg-[#3b44b8] transition-all w-max cursor-pointer">
    {label}
  </button>
{/snippet}

{#snippet codeForm()}
  <p class="text-gray-700">Enviamos un código de 6 dígitos a <b>{forms.codeSentTo}</b>.</p>
  <Input label="Código" saveOn={forms} save="code" required={true} />
  <div class="flex gap-16 items-center">
    {@render primaryButton("Confirmar", onVerifyCode)}
    <button class="text-[#4c55d5] hover:underline cursor-pointer" onclick={() => sendCode(forms.codeSentTo)}>Reenviar código</button>
    <button class="text-gray-500 hover:underline cursor-pointer" onclick={() => { forms.codeSentTo = "" }}>Cambiar correo</button>
  </div>
{/snippet}

{#snippet profileFields(emailDisabled: boolean)}
  <div class="grid grid-cols-12 gap-12">
    <Input label="Nombres" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="FirstName" required={true} />
    <Input label="Apellidos" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="LastName" required={true} />
    <Input label="Correo Electrónico" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="Email"
      required={true} disabled={emailDisabled} />
    <Input label="Celular" css="col-span-12 md:col-span-6" saveOn={WebStore.draft} save="Phone"
      required={hasCompanyFlag(WebStore.company.Flags, FLAG_REQUIRE_CONTACT_NUMBER)} />
  </div>
{/snippet}

{#snippet guestContent()}
  {#if selectedTab === 1}
    <div class="flex flex-col gap-12 max-w-[440px]">
      {#if forms.codeSentTo}
        {@render codeForm()}
      {:else}
        <Input label="Correo Electrónico" saveOn={forms} save="loginEmail" required={true} />
        <Input label="Contraseña" type="password" saveOn={forms} save="password" />
        <div class="flex flex-wrap gap-16 items-center">
          {@render primaryButton("Ingresar", onLoginWithPassword)}
          <button class="text-[#4c55d5] hover:underline cursor-pointer" onclick={() => sendCode(forms.loginEmail)}>
            Ingresar con un código enviado a mi correo
          </button>
        </div>
      {/if}
    </div>
  {:else if selectedTab === 2}
    <div class="flex flex-col gap-12">
      {#if forms.codeSentTo}
        {@render codeForm()}
      {:else}
        {@render profileFields(false)}
        <p class="text-gray-500 text-[14px]">Te enviaremos un código para confirmar tu correo. Podrás crear una contraseña después.</p>
        {@render primaryButton("Registrarme", onRegister)}
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet accountContent()}
  <div class="flex gap-20">
    <div class="flex flex-col w-180 shrink-0 border-r border-gray-200 pr-12 gap-4">
      {#each accountSections as section (section.id)}
        <button onclick={() => { accountSection = section.id }}
          class={["flex items-center gap-10 px-12 py-10 rounded-lg text-left cursor-pointer transition-colors",
            accountSection === section.id ? "bg-[#f1f2ff] text-[#4c55d5] ff-bold" : "hover:bg-gray-50"].join(" ")}>
          <i class={section.icon}></i><span>{section.name}</span>
        </button>
      {/each}
      <button onclick={handleLogout}
        class="flex items-center gap-10 px-12 py-10 rounded-lg text-left cursor-pointer hover:bg-gray-50 mt-12 text-gray-600">
        <i class="icon-[fa--sign-out]"></i><span>Cerrar Sesión</span>
      </button>
    </div>
    <div class="grow min-w-0">
      {#if accountSection === 1}
        <div class="flex flex-col gap-12">
          {@render profileFields(true)}
          <Input label="DNI" css="w-full md:w-1/2" saveOn={WebStore.draft} save="DocumentNumber" />
          {@render primaryButton("Guardar", onSaveProfile)}
        </div>
      {:else if accountSection === 2}
        <div class="flex flex-col gap-10">
          {#each WebStore.addresses as address (address.ID)}
            <button onclick={() => openAddressModal(address)}
              class="flex items-center gap-12 p-12 rounded-[10px] border border-gray-200 hover:border-gray-400 text-left cursor-pointer">
              <i class="icon-[fa--map-marker] text-[22px] text-gray-500"></i>
              <div class="flex flex-col min-w-0">
                <span class="ff-semibold">{address.Address}</span>
                <span class="text-gray-500 text-[14px] truncate">
                  {[cityName(address.CityID), address.Reference].filter(Boolean).join(" · ")}
                </span>
              </div>
            </button>
          {:else}
            <div class="text-gray-500 py-8">Aún no tienes direcciones guardadas.</div>
          {/each}
          {@render primaryButton("Nueva dirección", () => openAddressModal())}
        </div>
      {:else}
        <div class="flex flex-col items-center justify-center py-40 text-gray-400">
          <i class="icon-[fa--credit-card] text-[48px] mb-8 opacity-30"></i>
          <p>Próximamente</p>
        </div>
      {/if}
    </div>
  </div>
{/snippet}

{#snippet ordersContent()}
  {#if orders.length === 0}
    <div class="flex flex-col items-center justify-center py-12 text-gray-400">
      <i class="icon--supermarket-cart text-[48px] mb-4 opacity-20"></i>
      <p>Aún no has realizado ningún pedido.</p>
    </div>
  {/if}
  <div class="flex flex-col gap-12">
    {#each orders as order (order.ID)}
      <div class="border border-gray-200 rounded-[10px] p-14">
        <div class="flex flex-wrap justify-between gap-8">
          <span class="ff-bold">Pedido N° {order.ID}</span>
          <span class="text-gray-500">{formatTime(order.Date, 'd-M-Y')}</span>
        </div>
        <div class="flex flex-wrap gap-x-16 text-[14px] text-gray-600 mt-4">
          <span class={order.ss === 0 ? "text-red-600" : order.ss === 1 ? "text-amber-600" : "text-green-700"}>
            {webOrderStatusLabel(order.ss)}
          </span>
          <span>{order.DeliveryType === DELIVERY_PICKUP ? "Recojo en tienda" : "Entrega a domicilio"}</span>
          <span class="ff-semibold">{formatSoles(order.TotalAmount)}</span>
        </div>
        <div class="mt-8 text-[14px]">
          {#each order.Lines || [] as line}
            <div>{unpackQuantityLine(line.Quantity).units} × {productNames[line.ProductID] || `Producto ${line.ProductID}`}</div>
          {/each}
        </div>
        {#if order.ss === 1 && order.DebtAmount > 0}
          <button class="mt-8 text-[#4c55d5] hover:underline cursor-pointer text-[14px]"
            onclick={() => downloadWebOrderPDF(order.ID).catch((error) => notifyFailure(error.message))}>
            <i class="icon-[fa--download] mr-4"></i>Orden de compra (PDF)
          </button>
        {/if}
      </div>
    {/each}
  </div>
{/snippet}

{#snippet menuContent()}
  <div class="p-16 flex flex-col min-h-300">
    <OptionsStrip
      {options}
      selected={selectedTab}
      keyId="id"
      keyName="name"
      onSelect={(opt: any) => selectedTab = opt.id}
      css="mb-12"
    />

    <div class="grow">
      {#if selectedTab === 3}
        <div class="fs18 ff-bold mb-12">Productos Favoritos</div>
        <div class="flex flex-col items-center justify-center py-12 text-gray-400">
          <i class="icon-[fa--heart] text-[48px] mb-4 opacity-20"></i>
          <p>No tienes productos en tu lista de deseos.</p>
        </div>
      {:else if selectedTab === 4}
        <div class="fs18 ff-bold mb-12">Cambiar Contraseña</div>
        {#if isLoggedIn}
          <div class="flex flex-col gap-12 max-w-[440px]">
            {#if WebStore.hasPassword}
              <Input label="Contraseña actual" type="password" saveOn={forms} save="currentPassword" required={true} />
            {/if}
            <Input label="Nueva contraseña" type="password" saveOn={forms} save="newPassword" required={true} />
            {@render primaryButton("Guardar contraseña", onSavePassword)}
          </div>
        {:else}
          <p class="text-gray-500">Inicia sesión para crear o cambiar tu contraseña.</p>
        {/if}
      {:else if !isLoggedIn}
        {@render guestContent()}
      {:else if selectedTab === 1}
        {@render accountContent()}
      {:else if selectedTab === 2}
        {@render ordersContent()}
      {/if}
    </div>
  </div>
{/snippet}

{#if !isMobile}
  <div class="hidden md:block w-120 relative h-full">
    <ButtonLayer bind:isOpen={isOpen}
      wrapperClass="w-full h-full"
      buttonClass="w-full h-full"
      edgeMargin={32}
      label="Opens the user account menu."
      useBig layerClass="w-768 max-w-[82vw] rounded-[11px]">
      {#snippet button(open)}
        <button class={[s1.bn1, "w-full", open ? s1.button_menu_top : ""].join(" ")}>
          <i class="icon-[fa--user]"></i>
          <span>Mi Usuario</span>
        </button>
      {/snippet}

      {@render menuContent()}
    </ButtonLayer>
  </div>
{:else if layerOpenedState.id === id}
  <div class={css}>
    <div class="bg-white rounded-[11px] shadow-lg overflow-hidden">
      {@render menuContent()}
    </div>
  </div>
{/if}

<Modal id={ADDRESS_MODAL_ID} title={addressForm.ID > 0 ? "Editar dirección" : "Nueva dirección"}
  size={7} bodyCss="px-16 py-14"
  onSave={() => onSaveAddress()}
  onDelete={addressForm.ID > 0 ? () => onSaveAddress(true) : undefined}
>
  <AddressForm address={addressForm} />
</Modal>
