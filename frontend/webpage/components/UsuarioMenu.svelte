<script lang="ts">
  import { layerOpenedState } from "./store.svelte";
  import ButtonLayer from '#components/buttons/ButtonLayer.svelte';
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte';
  import Input from '#components/form/Input.svelte';
  import Modal from '#components/layers/Modal.svelte';
  import CardsDetailLayer from '#components/layers/CardsDetailLayer.svelte';
  import WebOrderDetail from './WebOrderDetail.svelte';
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
    saveWebCustomerPassword, saveWebCustomerAddress, getWebCustomerOrders, type IWebOrder,
  } from "#ecommerce/services/web-store.svelte.ts";
  import {
    DELIVERY_PICKUP, FLAG_REQUIRE_CONTACT_NUMBER, formatSoles, formatWebOrderNumber, hasCompanyFlag,
    mostExpensiveOrderLines, validateAddress, validateCustomerDraft, webOrderPaymentLabel, webOrderProgress,
    type IWebCustomerAddress, type WebOrderProgressKey,
  } from "#ecommerce/services/web-checkout.ts";

  export interface IProps {
    id?: number;
    /** Rendered inside the phone menu (MobileMenu.svelte): its tiles pick the tab, so there is no
        tab strip and no layer of its own. */
    isMobile?: boolean;
    selectedTab?: number;
  }

  let { id = 3, isMobile = false, selectedTab = $bindable(1) }: IProps = $props();

  // Storefront-only modal ids: the back-office ids (1, 2, ...) never mount here. The phone menu's
  // instance mounts next to the (hidden) desktop one, so each needs its own.
  const addressModalID = $derived(isMobile ? 932 : 931);

  const ui = useUI();
  const cities = useCitiesAPI();

  let isOpen = $state(false);
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
  let ordersLoading = $state(false);

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

  // The desktop layer loads the store and the session when it opens; the phone menu mounts this
  // component only while one of its sections shows, so it loads them right away.
  $effect(() => {
    if (isMobile || layerOpenedState.id === id) {
      isOpen = !isMobile;
      loadWebStoreCompany();
      restoreWebCustomerSession();
    } else {
      isOpen = false;
    }
  });

  $effect(() => {
    if (isMobile) { return }
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
    ordersLoading = true;
    getWebCustomerOrders()
      .then((result) => { orders = result.Orders || []; productNames = result.ProductNames || {} })
      .catch((error) => notifyFailure(error.message))
      .finally(() => { ordersLoading = false });
  });

  // Mis Pedidos badge tint per progress step: neutral until the store acts, warm while it works on
  // the order, blue on the way, green once delivered.
  const progressBadgeClass: Record<WebOrderProgressKey, string> = {
    annulled: "bg-red-50 text-red-700 border-red-200",
    received: "bg-slate-100 text-slate-700 border-slate-200",
    preparing: "bg-amber-50 text-amber-700 border-amber-200",
    inTransit: "bg-blue-50 text-blue-700 border-blue-200",
    delivered: "bg-green-50 text-green-700 border-green-200",
  };

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
    ui.openModal(addressModalID);
  };

  const onSaveAddress = async (isDelete?: boolean) => {
    if (!isDelete) {
      const problem = validateAddress(addressForm);
      if (problem) { notifyFailure(problem); return }
    }
    const address = { ...addressForm, ss: isDelete ? 0 : 1 };
    if (await runRequest(() => saveWebCustomerAddress(address))) { ui.closeModal(addressModalID) }
  };

  function handleLogout() {
    logoutWebCustomer();
    orders = []; ordersLoaded = false; selectedTab = 1;
  }
</script>

{#snippet primaryButton(label: string, onclick: () => void, icon?: string)}
  <button {onclick}
    class="flex items-center gap-8 bg-[#4c55d5] text-white py-10 px-24 rounded-lg ff-bold hover:bg-[#3b44b8] w-max cursor-pointer">
    {#if icon}<i class={icon}></i>{/if}
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
    <!-- Login card: light gray panel centered in the layer, actions centered below the fields. -->
    <div class="flex flex-col gap-12 w-full max-w-[440px] mx-auto mt-8 p-24 rounded-[11px] bg-[#f5f6f8]">
      <!-- The phone menu already titles the section in its top row. -->
      {#if !isMobile}<div class="text-[20px] ff-bold text-center text-gray-800 mb-4">Iniciar Sesión</div>{/if}
      {#if forms.codeSentTo}
        {@render codeForm()}
      {:else}
        <Input label="Correo Electrónico" saveOn={forms} save="loginEmail" required={true} />
        <Input label="Contraseña" type="password" saveOn={forms} save="password" />
        <div class="flex flex-col gap-12 items-center mt-4">
          {@render primaryButton("Ingresar", onLoginWithPassword, "icon-[fa--sign-in]")}
          <button class="text-[#4c55d5] hover:underline cursor-pointer" onclick={() => sendCode(forms.loginEmail)}>
            Ingresar con un código enviado a mi correo
          </button>
        </div>
      {/if}
    </div>
  {:else if selectedTab === 2}
    <!-- Register card: same light gray panel as Login, kept full width for the two-column fields. -->
    <div class="flex flex-col gap-12 w-full mt-8 p-24 rounded-[11px] bg-[#f5f6f8]">
      {#if forms.codeSentTo}
        {@render codeForm()}
      {:else}
        {@render profileFields(false)}
        <div class="flex flex-col gap-12 items-center mt-4">
          <p class="text-gray-500 text-[14px] text-center">Te enviaremos un código para confirmar tu correo. Podrás crear una contraseña después.</p>
          {@render primaryButton("Registrarme", onRegister)}
        </div>
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet accountContent()}
  <!-- Folder-tab layout: the menu column's right border is the only divider. The selected section
       shares the panel's gray and overlaps that divider by 1px (-mr-1), so it opens into the panel.
       Every section keeps the same border widths (transparent when idle) to avoid a layout shift. -->
  <div class="flex">
    <div class="flex flex-col w-112 shrink-0 pt-14 border-r border-gray-300">
      {#each accountSections as section (section.id)}
        <button onclick={() => { accountSection = section.id }}
          class={["relative flex flex-col items-center gap-4 -mr-1 px-8 py-10 text-center leading-[1.2]",
            "border-y border-l-4 border-r-0 cursor-pointer",
            accountSection === section.id
              ? "z-1 bg-gray-100 border-gray-300 text-[#4c55d5] ff-bold"
              : "border-transparent text-gray-700 hover:text-[#4c55d5]"].join(" ")}>
          <i class="{section.icon} text-[18px]"></i><span>{section.name}</span>
        </button>
      {/each}
      <button onclick={handleLogout}
        class="flex flex-col items-center gap-4 mt-12 px-8 py-10 text-center leading-[1.2] cursor-pointer text-gray-500 hover:text-gray-800">
        <i class="icon-[fa--sign-out] text-[18px]"></i><span>Cerrar Sesión</span>
      </button>
    </div>
    <div class="grow min-w-0 p-16 bg-gray-100">
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

<!-- One icon + text pair of an order card. The flex box keeps the icon in its own slot: Font Awesome
     glyphs are wider than tall (the truck is 1.23em), and inline they overlapped the next text. -->
{#snippet orderInfoItem(icon: string, text: string, css: string)}
  <span class="inline-flex items-center gap-6 {css}">
    <i class="{icon} shrink-0 opacity-70"></i>{text}
  </span>
{/snippet}

{#snippet ordersEmpty()}
  <div class="flex flex-col items-center justify-center py-12 text-gray-400">
    <i class="icon--supermarket-cart text-[48px] mb-4 opacity-20"></i>
    <p>Aún no has realizado ningún pedido.</p>
  </div>
{/snippet}

{#snippet orderDetail(order: IWebOrder)}
  <WebOrderDetail {order} {productNames} {cityName} />
{/snippet}

<!-- Detail header: the order number with its date beside it. -->
{#snippet orderDetailHeader(order: IWebOrder)}
  <div class="flex flex-wrap items-baseline gap-x-12">
    <span class="ff-bold text-[18px] text-gray-800">Pedido {formatWebOrderNumber(order.ID)}</span>
    <span class="ff-mono text-[14px] text-gray-500">{formatTime(order.Date, 'd-M-Y')}</span>
  </div>
{/snippet}

<!-- Fixed height so the tab keeps its size with 1 or 50 orders; the list scrolls inside, and a
     clicked card opens its detail in a side layer within that same box. -->
{#snippet ordersContent()}
  {#if ordersLoading}
    <!-- Same box as the list, so the tab keeps its size while the orders download. -->
    <div class="flex flex-col items-center justify-center gap-8 h-[min(520px,75vh)] bg-gray-100 rounded-[10px] text-gray-400">
      <i class="icon-[fa--spinner] animate-spin text-[32px]"></i>
      <p>Cargando pedidos...</p>
    </div>
  {:else}
    <CardsDetailLayer items={orders} getKey={(order) => order.ID} card={orderCard} detail={orderDetail}
      detailHeader={orderDetailHeader}
      empty={ordersEmpty} css="h-[min(520px,75vh)] bg-gray-100 rounded-[10px]" openCss="outline-2 outline-[#45495c45]" />
  {/if}
{/snippet}

{#snippet orderCard(order: IWebOrder, isSelected: boolean)}
  {@const orderLines = order.Lines || []}
  {@const progress = webOrderProgress(order)}
  {@const paymentLabel = webOrderPaymentLabel(order.ss)}
  {@const isPaid = order.ss === 2 || order.ss === 4}
  <!-- Two columns: the order summary on the left (number + progress badge, date + total,
       delivery + payment), its 3 most expensive lines on the right. Stacks on narrow screens. -->
  <!-- White on the list's gray background; the open card is ringed instead of bordered. -->
  <div class={["grid grid-cols-1 md:grid-cols-2 gap-x-20 gap-y-10 rounded-[10px] p-14 bg-white shadow-sm",
    isSelected ? "ring-1 ring-[#4c55d5]" : "hover:shadow-md"].join(" ")}>
    <div class="flex flex-col gap-6 min-w-0">
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4">
        <span class="ff-bold text-[17px] text-gray-800">Pedido {formatWebOrderNumber(order.ID)}</span>
        <span class="px-10 py-2 rounded-full border text-[14px] ff-semibold {progressBadgeClass[progress.key]}">
          {progress.label}
        </span>
      </div>
      <!-- Row 2: when and how much. Row 3: how it is delivered and whether it is paid. -->
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4 text-[14px]">
        {@render orderInfoItem("icon-[fa--calendar]", formatTime(order.Date, 'd-M-Y') as string, "ff-mono text-gray-500")}
        <span class="w-5 h-5 rounded-full bg-gray-300"></span>
        <span class="ff-bold text-gray-800">{formatSoles(order.TotalAmount)}</span>
      </div>
      <div class="flex flex-wrap items-center gap-x-10 gap-y-4 text-[14px]">
        {#if order.DeliveryType === DELIVERY_PICKUP}
          {@render orderInfoItem("icon-[fa--shopping-bag]", "Recojo en tienda", "text-gray-600")}
        {:else}
          {@render orderInfoItem("icon-[fa--truck]", "Entrega a domicilio", "text-gray-600")}
        {/if}
        {#if paymentLabel}
          <span class="w-5 h-5 rounded-full bg-gray-300"></span>
          <span class="ff-semibold {isPaid ? 'text-green-700' : 'text-amber-700'}">{paymentLabel}</span>
        {/if}
      </div>
    </div>
    <div class="min-w-0 text-[14px] text-gray-700 md:border-l md:border-gray-100 md:pl-20">
      {#each mostExpensiveOrderLines(orderLines, 3) as line}
        <div class="truncate">{unpackQuantityLine(line.Quantity).units} × {productNames[line.ProductID] || `Producto ${line.ProductID}`}</div>
      {/each}
      {#if orderLines.length > 3}
        <div class="text-gray-500">(y {orderLines.length - 3} más...)</div>
      {/if}
    </div>
  </div>
{/snippet}

{#snippet menuContent()}
  <div class="p-16 flex flex-col min-h-300">
    {#if !isMobile}
      <OptionsStrip
        {options}
        selected={selectedTab}
        keyId="id"
        keyName="name"
        onSelect={(opt: any) => selectedTab = opt.id}
        css="mb-12"
      />
    {/if}

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
{:else}
  {@render menuContent()}
{/if}

<Modal id={addressModalID} title={addressForm.ID > 0 ? "Editar dirección" : "Nueva dirección"}
  size={7} bodyCss="px-16 py-14"
  onSave={() => onSaveAddress()}
  onDelete={addressForm.ID > 0 ? () => onSaveAddress(true) : undefined}
>
  <AddressForm address={addressForm} />
</Modal>
