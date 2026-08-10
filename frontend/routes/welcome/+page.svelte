<script lang="ts">
  import { onMount } from 'svelte';
  import { browser } from '$app/environment';
  import { useUI } from '@genix/ui';
  import Input from '$components/form/Input.svelte';
  import T from '$components/misc/T.svelte';
  import { Core, setLanguaje, tr } from '$core/store.svelte';
  import { Env, lastLoginCompanyIDStorageKey, type IApiEndpointOption } from '$core/env';
  import { Notify } from '$libs/helpers';
  import { security } from '$libs/ui-runtime.svelte';
  import { getPublicCompanyName, sendUserLogin, type ILogin } from '$services/login';
  import RegistrationModal from './RegistrationModal.svelte';

  const REGISTRATION_MODAL_ID = 71;
  const ui = useUI();

  const navigationItems = [
    { id: 'home', label: 'Home|Inicio' },
    { id: 'features', label: 'Features|Funcionalidades' },
    { id: 'roadmap', label: 'Roadmap|Roadmap' },
    { id: 'contact', label: 'Contact|Contacto' },
  ];

  const featureCards = [
    {
      icon: 'icon-[fa--cubes]',
      title: 'Products and customers|Productos y clientes',
      description: 'Manage catalogs, categories, brands, images, customers, suppliers, and Excel imports from one place.|Gestione catálogos, categorías, marcas, imágenes, clientes, proveedores e importaciones de Excel desde un solo lugar.',
    },
    {
      icon: 'icon-[fa--truck]',
      title: 'Inventory and purchasing|Inventario y compras',
      description: 'Control stock by warehouse, lots, serial numbers, movements, purchase orders, receipts, and replenishment planning.|Controle stock por almacén, lotes, series, movimientos, órdenes de compra, recepciones y planificación de reposición.',
    },
    {
      icon: 'icon-[fa--shopping-cart]',
      title: 'Sales and point of sale|Ventas y punto de venta',
      description: 'Create orders, register payments and deliveries, and follow performance through reports, charts, and sales planning.|Cree pedidos, registre pagos y entregas, y siga el rendimiento con reportes, gráficos y planificación de ventas.',
    },
    {
      icon: 'icon-[fa--line-chart]',
      title: 'Cash and finance|Caja y finanzas',
      description: 'Operate cash and bank registers, income, expenses, transfers, reconciliations, schedules, and partial payments.|Opere cajas y bancos, ingresos, gastos, transferencias, conciliaciones, programaciones y pagos parciales.',
    },
    {
      icon: 'icon-[fa--globe]',
      title: 'E-commerce|Comercio electrónico',
      description: 'Design and deploy a storefront with product search, catalog, cart, payment UI, shipping configuration, and custom domains.|Diseñe y publique una tienda con búsqueda de productos, catálogo, carrito, interfaz de pagos, configuración de envíos y dominios propios.',
    },
    {
      icon: 'icon-[fa--magic]',
      title: 'AI, security, and operations|IA, seguridad y operaciones',
      description: 'Use AI-assisted page building and ERP navigation with access profiles, automated jobs, and tenant backups.|Use creación de páginas y navegación ERP asistidas por IA, con perfiles de acceso, tareas automatizadas y copias de seguridad por empresa.',
    },
  ];

  const roadmapGroups = [
    {
      status: 'Available now|Disponible ahora',
      icon: 'icon-[fa--check]',
      accent: 'bg-emerald-50 text-emerald-700 border-emerald-200',
      items: [
        'Core ERP workflows and point of sale|Flujos ERP principales y punto de venta',
        'Inventory, purchasing, and warehouse control|Inventario, compras y control de almacenes',
        'Cash management, users, access, and backups|Gestión de caja, usuarios, accesos y copias de seguridad',
        'Visual storefront builder and custom domains|Constructor visual de tiendas y dominios propios',
      ],
    },
    {
      status: 'In progress|En progreso',
      icon: 'icon-[fa--refresh]',
      accent: 'bg-amber-50 text-amber-700 border-amber-200',
      items: [
        'Projected cash-flow report|Reporte de flujo de caja proyectado',
        'Weekly sales summary|Resumen semanal de ventas',
        'Self-hosted provider completion|Finalización del proveedor autohospedado',
        'Shipping prices applied at checkout|Precios de envío aplicados al checkout',
      ],
    },
    {
      status: 'Next|Próximo',
      icon: 'icon-[fa--arrow-right]',
      accent: 'bg-indigo-50 text-indigo-700 border-indigo-200',
      items: [
        'E-commerce order persistence|Persistencia de pedidos e-commerce',
        'Storefront customer accounts and order history|Cuentas de clientes e historial de pedidos',
        'Streaming AI responses and usage quotas|Respuestas IA en streaming y cuotas de uso',
        'Verified public registration flow|Flujo público de registro verificado',
      ],
    },
  ];

  const DEFAULT_COMPANY_ID = 1;

  const readStoredCompanyID = () => {
    if (!browser) return DEFAULT_COMPANY_ID;
    const storedCompanyID = Number(localStorage.getItem(lastLoginCompanyIDStorageKey));
    return Number.isInteger(storedCompanyID) && storedCompanyID > 0 ? storedCompanyID : DEFAULT_COMPANY_ID;
  };

  // Resolved synchronously so the Input reads the stored tenant on its first render.
  let loginForm = $state<ILogin>({ User: '', Password: '', CompanyID: readStoredCompanyID(), CipherKey: '' });
  let selectedApiEndpointRoute = $state('');
  let apiEndpointOptions = $state<IApiEndpointOption[]>([]);
  let isLoginLoading = $state(false);
  let isCompanyLookupLoading = $state(false);
  let companyName = $state('');
  let companyLookupError = $state('');
  let companyLookupSequence = 0;
  let mobileMenuOpen = $state(false);
  let contactForm = $state({ Name: '', Email: '', Company: '', Message: '' });
  // Filled from /welcome?req=…&code=…, the link the registration email carries.
  let signUpRequestID = $state(0);
  let signUpCode = $state('');

  // Keep endpoint selection identical to the former login page so authentication behavior does not change.
  const syncApiEndpointSelector = () => {
    apiEndpointOptions = [...Env.availableApiEndpoints];
    selectedApiEndpointRoute = Env.selectedApiEndpointRoute || apiEndpointOptions[0]?.route || '';
  };

  const onApiEndpointChange = (selectedEndpoint?: IApiEndpointOption) => {
    if (!selectedEndpoint?.route) return;
    Env.setSelectedApiEndpoint(selectedEndpoint.route);
    selectedApiEndpointRoute = selectedEndpoint.route;
    // A name resolved on another server is no longer authoritative, so resolve it again there.
    companyLookupSequence += 1;
    isCompanyLookupLoading = false;
    companyName = '';
    companyLookupError = '';
    console.info('[WelcomeLogin] API endpoint selected:', selectedEndpoint.route);
    void lookupCompanyName();
  };

  // Company IDs are positive database identifiers; reject values that cannot identify a tenant.
  const isValidCompanyID = (companyID: string | number) => Number.isInteger(Number(companyID)) && Number(companyID) > 0;

  const lookupCompanyName = async () => {
    const companyID = Number(loginForm.CompanyID);
    const lookupSequence = ++companyLookupSequence;
    companyName = '';
    companyLookupError = '';

    if (!isValidCompanyID(companyID)) {
      console.warn('[WelcomeLogin] Company lookup skipped for invalid ID:', companyID);
      return;
    }

    if (browser) localStorage.setItem(lastLoginCompanyIDStorageKey, String(companyID));

    isCompanyLookupLoading = true;
    console.info('[WelcomeLogin] Looking up company:', companyID);
    try {
      const publicCompany = await getPublicCompanyName(companyID);
      if (lookupSequence !== companyLookupSequence) return;

      companyName = publicCompany?.Name?.trim() || '';
      if (!companyName) companyLookupError = tr('Company not found|Empresa no encontrada');
      console.info('[WelcomeLogin] Company lookup completed:', { companyID, companyName });
    } catch (lookupError) {
      if (lookupSequence !== companyLookupSequence) return;

      companyLookupError = tr('Company not found|Empresa no encontrada');
      console.error('[WelcomeLogin] Company lookup failed:', { companyID, lookupError });
    } finally {
      if (lookupSequence === companyLookupSequence) isCompanyLookupLoading = false;
    }
  };

  const submitLogin = async () => {
    // The form is disabled while the request is in flight, but a queued Enter keypress could
    // still reach this handler, so the guard is what actually prevents a duplicate POST.
    if (isLoginLoading) return;

    if (loginForm.User.trim().length < 4 || loginForm.Password.length < 4 || !isValidCompanyID(loginForm.CompanyID)) {
      Notify.failure(tr('Please provide a valid username, password, and company ID.|Debe proporcionar un usuario, una contraseña y un ID de empresa válidos.'));
      return;
    }

    isLoginLoading = true;
    console.info('[WelcomeLogin] Sending credentials to selected endpoint.');
    // Only logged here: the HTTP client already shows the server's own message ("wrong user or
    // password", …), so a second generic notification would just duplicate it.
    const result = await sendUserLogin(loginForm);
    isLoginLoading = false;

    if (result.error) console.error('[WelcomeLogin] Login request failed:', result.error);
  };

  const navigateToSection = (sectionID: string) => {
    mobileMenuOpen = false;
    document.getElementById(sectionID)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  const focusLogin = () => {
    navigateToSection('login');
    setTimeout(() => document.querySelector<HTMLInputElement>('#login-form input')?.focus(), 450);
  };

  const openRegistration = () => {
    mobileMenuOpen = false;
    console.info('[WelcomeRegistration] Opening email registration form.');
    ui.openModal(REGISTRATION_MODAL_ID);
  };

  const handleWindowKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && mobileMenuOpen) mobileMenuOpen = false;
  };

  onMount(() => {
    syncApiEndpointSelector();
    if (security.checkIsLogin() === 2) {
      Env.navigate('/');
      return;
    }
    // The ID survives a reload in localStorage but the name does not, so resolve it again
    // (memory → IndexedDB → server) instead of leaving the field blank.
    void lookupCompanyName();

    // Landing from the registration email opens the wizard straight on the code check.
    const urlParams = new URLSearchParams(window.location.search);
    const requestIDParam = Number(urlParams.get('req'));
    if (Number.isSafeInteger(requestIDParam) && requestIDParam > 0) {
      signUpRequestID = requestIDParam;
      signUpCode = urlParams.get('code') || '';
      console.info('[WelcomeRegistration] Opening registration from email link:', signUpRequestID);
      ui.openModal(REGISTRATION_MODAL_ID);
      // Drop the credentials from the address bar as soon as they are captured: a reload would
      // otherwise re-open the wizard on a code that has already been consumed, and the code would
      // stay in the browser history and in any copied link.
      // The native call and not replaceState() from $app/navigation: on the first load the router
      // is not started yet (it flips the flag after onMount), so that one throws. Passing the
      // current history.state through keeps SvelteKit's own back/forward bookkeeping intact.
      history.replaceState(history.state, '', '/welcome');
    }
  });
</script>

<svelte:head>
  <title>{tr('Genix — ERP and e-commerce for small businesses|Genix — ERP y comercio electrónico para pequeñas empresas')}</title>
  <meta
    name="description"
    content={tr('Manage sales, inventory, purchasing, finance, and e-commerce with an open-source platform built for small businesses.|Gestione ventas, inventario, compras, finanzas y comercio electrónico con una plataforma de código abierto para pequeñas empresas.')}
  />
</svelte:head>

<svelte:window onkeydown={handleWindowKeydown} />

<div class="min-h-screen bg-slate-50 text-slate-900 selection:bg-indigo-200 selection:text-indigo-950">
  <header class="fixed inset-x-0 top-0 z-50">
    <nav
      class="border-b border-white/15 bg-slate-950/60 px-14 py-10 shadow-[0_8px_32px_rgba(15,23,42,0.18)] backdrop-blur-2xl md:px-24"
      aria-label={tr('Primary navigation|Navegación principal')}
    >
      <div class="mx-auto flex max-w-1240 items-center justify-between gap-12">
        <button
          class="flex shrink-0 items-center focus-visible:outline-3 focus-visible:outline-offset-4 focus-visible:outline-indigo-600"
          aria-label={tr('Go to the welcome section|Ir a la sección de bienvenida')}
          onclick={() => navigateToSection('home')}
        >
          <img class="h-36 w-112 object-contain md:h-40 md:w-128" src="/images/genix_logo.svg" alt="Genix" />
        </button>

        <div class="hidden items-center gap-8 lg:flex">
          {#each navigationItems as navigationItem}
            <button
              class="rounded-[9px] px-12 py-8 text-sm text-white/75 transition hover:bg-white/10 hover:text-white focus-visible:outline-3 focus-visible:outline-white"
              onclick={() => navigateToSection(navigationItem.id)}
            >
              <T text={navigationItem.label} />
            </button>
          {/each}
        </div>

        <div class="flex items-center gap-6">
          <div class="flex rounded-[10px] border border-white/20 bg-white/10 p-3" aria-label={tr('Language|Idioma')}>
            <button
              class="rounded-[7px] px-8 py-5 text-xs transition {Core.languaje === 1 ? 'bg-indigo-600 text-white' : 'text-white/65 hover:text-white'}"
              aria-pressed={Core.languaje === 1}
              onclick={() => setLanguaje(1)}
            >ES</button>
            <button
              class="rounded-[7px] px-8 py-5 text-xs transition {Core.languaje === 2 ? 'bg-indigo-600 text-white' : 'text-white/65 hover:text-white'}"
              aria-pressed={Core.languaje === 2}
              onclick={() => setLanguaje(2)}
            >EN</button>
          </div>

          <button
            class="hidden h-38 items-center rounded-[10px] px-12 text-sm text-white/80 transition hover:bg-white/10 hover:text-white focus-visible:outline-3 focus-visible:outline-white md:flex"
            onclick={focusLogin}
          >
            <T text="Sign in|Ingresar" />
          </button>
          <button
            class="hidden h-38 items-center rounded-[10px] bg-indigo-600 px-16 text-sm text-white shadow-md shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 md:flex"
            onclick={openRegistration}
          >
            <T text="Register|Regístrese" />
          </button>
          <button
            class="flex size-38 items-center justify-center rounded-[10px] border border-white/20 bg-white/10 text-white lg:hidden"
            aria-label={tr('Toggle navigation menu|Alternar menú de navegación')}
            aria-expanded={mobileMenuOpen}
            aria-controls="welcome-mobile-menu"
            onclick={() => { mobileMenuOpen = !mobileMenuOpen; }}
          >
            <i class={mobileMenuOpen ? 'icon-[fa--close]' : 'icon-[fa--bars]'} aria-hidden="true"></i>
          </button>
        </div>
      </div>

      {#if mobileMenuOpen}
        <div id="welcome-mobile-menu" class="mx-auto mt-10 grid max-w-1240 gap-4 border-t border-white/15 pt-10 lg:hidden">
          {#each navigationItems as navigationItem}
            <button
              class="rounded-[9px] px-12 py-10 text-left text-white/80 hover:bg-white/10 hover:text-white"
              onclick={() => navigateToSection(navigationItem.id)}
            >
              <T text={navigationItem.label} />
            </button>
          {/each}
          <div class="mt-4 grid grid-cols-2 gap-8">
            <button class="rounded-[10px] border border-white/25 px-12 py-10 text-white" onclick={focusLogin}>
              <T text="Sign in|Ingresar" />
            </button>
            <button class="rounded-[10px] bg-indigo-600 px-12 py-10 text-white" onclick={openRegistration}>
              <T text="Register|Regístrese" />
            </button>
          </div>
        </div>
      {/if}
    </nav>
  </header>

  <main>
    <section id="home" class="relative isolate min-h-screen scroll-mt-64 overflow-hidden bg-slate-950">
      <img
        class="absolute inset-0 -z-30 h-full w-full object-cover object-[60%_center]"
        src="/images/welcome-hero-v3.webp"
        alt={tr('Small business owner managing orders|Propietaria de una pequeña empresa gestionando pedidos')}
      />
      <div class="absolute inset-0 -z-20 bg-linear-to-r from-slate-950/95 via-slate-950/68 to-slate-950/45"></div>
      <div class="absolute inset-0 -z-20 bg-linear-to-b from-slate-950/35 via-transparent to-slate-950/60"></div>
      <div class="absolute -left-80 top-80 -z-10 size-320 rounded-full bg-violet-500/20 blur-3xl"></div>

      <div class="mx-auto grid min-h-screen max-w-1440 items-center gap-38 px-20 pb-58 pt-106 md:px-42 md:pb-72 md:pt-118 lg:grid-cols-[minmax(0,650px)_minmax(360px,420px)] lg:justify-between lg:gap-40">
        <div class="max-w-650 text-white">
            <div class="mb-18 inline-flex items-center gap-8 rounded-full border border-white/20 bg-white/10 px-14 py-7 text-sm backdrop-blur">
              <i class="icon-[fa--code-fork] text-violet-200" aria-hidden="true"></i>
              <T text="ERP + e-commerce|ERP + comercio electrónico" />
            </div>
            <h1 class="max-w-620 text-4xl font-bold leading-tight md:text-5xl lg:text-[56px]">
              <T text="Run your business from one clear, connected system.|Gestione cada proceso de su empresa" />
            </h1>
            <p class="mt-20 max-w-600 text-lg leading-[1.6] text-indigo-100 md:text-xl">
              <T text="Genix brings sales, inventory, purchasing, finance, and online commerce together for small businesses that want control without complexity.|Genix reúne ventas, inventario, compras, finanzas y comercio electrónico para pequeñas empresas que buscan control sin complejidad." />
            </p>
            <div class="mt-28 flex flex-col gap-10 sm:flex-row">
              <button
                class="flex h-46 items-center justify-center gap-8 rounded-[11px] bg-white px-20 text-indigo-800 shadow-lg transition hover:-translate-y-1 hover:bg-indigo-50 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-white"
                onclick={openRegistration}
              >
                <span class="font-semibold"><T text="Register|Regístrese" /></span>
                <i class="icon-[fa--arrow-right]" aria-hidden="true"></i>
              </button>
              <button
                class="flex h-46 items-center justify-center rounded-[11px] border border-white/25 bg-white/10 px-20 text-white backdrop-blur transition hover:bg-white/15 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-white"
                onclick={() => navigateToSection('features')}
              >
                <T text="Discover Genix|Conozca Genix" />
              </button>
            </div>
            <div class="mt-30 flex flex-wrap gap-x-20 gap-y-9 text-sm text-indigo-100">
              <span class="flex items-center gap-7"><i class="icon-[fa--check-circle]" aria-hidden="true"></i><T text="Open source|Código abierto" /></span>
              <span class="flex items-center gap-7"><i class="icon-[fa--check-circle]" aria-hidden="true"></i><T text="Self-hostable|Autohospedable" /></span>
              <span class="flex items-center gap-7"><i class="icon-[fa--check-circle]" aria-hidden="true"></i><T text="Your data, exportable|Sus datos, exportables" /></span>
            </div>
          </div>

        <div id="login" class="welcome-glass-panel scroll-mt-80 rounded-[22px] p-20 text-white md:p-28">
            <div class="mb-20 flex items-center gap-12">
              <div class="flex size-44 items-center justify-center rounded-[12px] border border-white/15 bg-white/10 text-violet-200">
                <i class="icon-[fa--sign-in] text-xl" aria-hidden="true"></i>
              </div>
              <h2 class="text-xl font-semibold text-white"><T text="Sign in|Ingresar" /></h2>
            </div>

            <form
              id="login-form"
              class="grid gap-14"
              aria-label={tr('Login form with username and password|Formulario de acceso con usuario y contraseña')}
              onsubmit={(event) => { event.preventDefault(); void submitLogin(); }}
            >
              {#if apiEndpointOptions.length > 0}
                <fieldset>
                  <legend class="mb-9 text-sm text-violet-100"><T text="Server|Servidor" /></legend>
                  <div class="grid grid-cols-3 gap-10">
                    {#each apiEndpointOptions as apiEndpointOption}
                      <button
                        type="button"
                        class="flex min-h-56 w-full items-center justify-start gap-8 rounded-[12px] border px-10 text-left text-sm transition focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-violet-300 disabled:cursor-not-allowed disabled:opacity-60 {selectedApiEndpointRoute === apiEndpointOption.route
                          ? 'border-violet-300 bg-violet-500/35 text-white shadow-lg shadow-violet-950/20'
                          : 'border-white/20 bg-white/8 text-white/75 hover:border-white/40 hover:bg-white/15 hover:text-white'}"
                        aria-pressed={selectedApiEndpointRoute === apiEndpointOption.route}
                        disabled={isLoginLoading}
                        onclick={() => onApiEndpointChange(apiEndpointOption)}
                      >
                        <i class="icon-[fa--server] shrink-0 -ml-1 -mr-1" aria-hidden="true"></i>
                        <span class="min-w-0 leading-tight">{apiEndpointOption.name}</span>
                      </button>
                    {/each}
                  </div>
                </fieldset>
              {/if}
              <div class="relative mt-8">
                <Input
                  label="Company ID|ID Empresa"
                  saveOn={loginForm}
                  save="CompanyID"
                  type="number"
                  css="welcome-login-field welcome-company-id-field"
                  inputCss="text-base"
                  disabled={isLoginLoading}
                  onChange={() => { void lookupCompanyName(); }}
                />
                <div
                  class="pointer-events-none absolute bottom-0 right-0 flex h-38 w-[65%] items-center justify-end overflow-hidden px-10 text-right text-sm"
                  aria-live="polite"
                >
                  {#if isCompanyLookupLoading}
                    <i class="icon-[fa--refresh] animate-spin text-violet-200" aria-label={tr('Loading company|Buscando empresa')}></i>
                  {:else if companyName}
                    <span class="truncate text-white/80" title={companyName}>{companyName}</span>
                  {:else if companyLookupError}
                    <span class="truncate text-rose-300" title={companyLookupError}>{companyLookupError}</span>
                  {/if}
                </div>
              </div>
              <Input required={true} label="Username|Usuario" saveOn={loginForm} save="User" type="text" css="welcome-login-field" inputCss="text-base" disabled={isLoginLoading} />
              {#if isLoginLoading}
                <!-- Takes the password field's slot at its exact height so nothing shifts. -->
                <div
                  class="flex items-center justify-center gap-10 rounded-[12px] border border-white/20 bg-white/8 text-violet-100"
                  style="height: var(--input-height)"
                  role="status"
                  aria-live="polite"
                >
                  <i class="icon-[fa--refresh] animate-spin text-lg" aria-hidden="true"></i>
                  <span class="text-sm"><T text="Signing in...|Ingresando..." /></span>
                </div>
              {:else}
                <Input required={true} label="Password|Contraseña" saveOn={loginForm} save="Password" type="password" css="welcome-login-field" inputCss="text-base" />
              {/if}
              <button
                type="submit"
                class="mt-4 flex h-44 w-full items-center justify-center gap-8 rounded-[10px] bg-indigo-600 px-18 text-white shadow-lg shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-60"
                aria-label={tr('Submit login credentials|Enviar credenciales de acceso')}
                disabled={isLoginLoading}
              >
                <i class={isLoginLoading ? 'icon-[fa--refresh] animate-spin' : 'icon-[fa--sign-in]'} aria-hidden="true"></i>
                <span class="font-semibold"><T text={isLoginLoading ? 'Signing in...|Ingresando...' : 'Sign in|Ingresar'} /></span>
              </button>
            </form>
        </div>
      </div>
    </section>

    <section class="px-16 py-72 md:px-28 md:py-96" aria-labelledby="what-is-genix">
      <div class="mx-auto max-w-1120 text-center">
        <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-600"><T text="Built for practical growth|Creado para crecer de forma práctica" /></p>
        <h2 id="what-is-genix" class="mx-auto mt-12 max-w-760 text-3xl font-bold leading-tight text-slate-950 md:text-4xl">
          <T text="One platform, from the counter to the online store.|Una plataforma, desde el mostrador hasta la tienda en línea." />
        </h2>
        <p class="mx-auto mt-16 max-w-760 text-lg leading-[1.6] text-slate-600">
          <T text="Genix is an ERP and e-commerce platform for small businesses. It can run in the cloud or on your own infrastructure, while keeping business data portable and under your control.|Genix es una plataforma ERP y de comercio electrónico para pequeñas empresas. Puede funcionar en la nube o en su propia infraestructura, manteniendo los datos del negocio portables y bajo su control." />
        </p>

        <div class="mt-42 grid gap-14 md:grid-cols-3">
          <article class="rounded-[18px] border border-slate-200 bg-white p-24 text-left shadow-sm">
            <i class="icon-[fa--sitemap] text-2xl text-indigo-600" aria-hidden="true"></i>
            <h3 class="mt-16 text-xl font-semibold"><T text="Connected operations|Operaciones conectadas" /></h3>
            <p class="mt-8 leading-[1.55] text-slate-600"><T text="Sales, stock, purchasing, cash, customers, and online commerce share one operational view.|Ventas, stock, compras, caja, clientes y comercio electrónico comparten una sola visión operativa." /></p>
          </article>
          <article class="rounded-[18px] border border-slate-200 bg-white p-24 text-left shadow-sm">
            <i class="icon-[fa--server] text-2xl text-indigo-600" aria-hidden="true"></i>
            <h3 class="mt-16 text-xl font-semibold"><T text="Flexible deployment|Implementación flexible" /></h3>
            <p class="mt-8 leading-[1.55] text-slate-600"><T text="Choose cloud, self-hosted, or a compact single-binary installation as your business requires.|Elija nube, autohospedaje o una instalación compacta en un solo binario, según las necesidades de su empresa." /></p>
          </article>
          <article class="rounded-[18px] border border-slate-200 bg-white p-24 text-left shadow-sm">
            <i class="icon-[fa--shield] text-2xl text-indigo-600" aria-hidden="true"></i>
            <h3 class="mt-16 text-xl font-semibold"><T text="Your data stays yours|Sus datos siguen siendo suyos" /></h3>
            <p class="mt-8 leading-[1.55] text-slate-600"><T text="Tenant isolation, access profiles, and complete backup exports make ownership a product requirement.|El aislamiento por empresa, los perfiles de acceso y las copias de seguridad completas convierten la propiedad de datos en un requisito del producto." /></p>
          </article>
        </div>
      </div>
    </section>

    <section id="features" class="scroll-mt-110 bg-white px-16 py-72 md:px-28 md:py-96" aria-labelledby="features-title">
      <div class="mx-auto max-w-1120">
        <div class="max-w-720">
          <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-600"><T text="Capabilities|Funcionalidades" /></p>
          <h2 id="features-title" class="mt-12 text-3xl font-bold leading-tight md:text-4xl"><T text="The essential workflows in one system.|Los flujos esenciales en un solo sistema." /></h2>
          <p class="mt-14 text-lg leading-[1.6] text-slate-600"><T text="Start with daily operations and expand into planning, automation, and digital commerce as your business grows.|Comience con las operaciones diarias y avance hacia la planificación, automatización y comercio digital a medida que su empresa crece." /></p>
        </div>

        <div class="mt-40 grid gap-14 md:grid-cols-2 lg:grid-cols-3">
          {#each featureCards as featureCard}
            <article class="group rounded-[18px] border border-slate-200 bg-slate-50 p-22 transition hover:-translate-y-2 hover:border-indigo-200 hover:bg-white hover:shadow-xl hover:shadow-indigo-950/8">
              <div class="flex size-44 items-center justify-center rounded-[12px] bg-indigo-100 text-indigo-700 transition group-hover:bg-indigo-600 group-hover:text-white">
                <i class="{featureCard.icon} text-xl" aria-hidden="true"></i>
              </div>
              <h3 class="mt-18 text-xl font-semibold text-slate-950"><T text={featureCard.title} /></h3>
              <p class="mt-9 leading-[1.55] text-slate-600"><T text={featureCard.description} /></p>
            </article>
          {/each}
        </div>
      </div>
    </section>

    <section id="roadmap" class="scroll-mt-110 px-16 py-72 md:px-28 md:py-96" aria-labelledby="roadmap-title">
      <div class="mx-auto max-w-1120">
        <div class="text-center">
          <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-600"><T text="Transparent roadmap|Roadmap transparente" /></p>
          <h2 id="roadmap-title" class="mt-12 text-3xl font-bold leading-tight md:text-4xl"><T text="What works today, and what comes next.|Lo que funciona hoy y lo que viene después." /></h2>
          <p class="mx-auto mt-14 max-w-760 text-lg leading-[1.6] text-slate-600"><T text="Genix is in pre-alpha and under active development. Priorities, interfaces, and schemas may change as the product evolves.|Genix está en etapa pre-alfa y en desarrollo activo. Las prioridades, interfaces y esquemas pueden cambiar mientras evoluciona el producto." /></p>
        </div>

        <div class="mt-40 grid gap-14 lg:grid-cols-3">
          {#each roadmapGroups as roadmapGroup}
            <article class="rounded-[20px] border border-slate-200 bg-white p-22 shadow-sm">
              <div class="inline-flex items-center gap-8 rounded-full border px-12 py-7 text-sm font-semibold {roadmapGroup.accent}">
                <i class={roadmapGroup.icon} aria-hidden="true"></i>
                <T text={roadmapGroup.status} />
              </div>
              <ul class="mt-20 grid gap-13">
                {#each roadmapGroup.items as roadmapItem}
                  <li class="flex gap-10 leading-[1.55] text-slate-600">
                    <i class="icon-[fa--circle] mt-10 text-[6px] text-indigo-400" aria-hidden="true"></i>
                    <span><T text={roadmapItem} /></span>
                  </li>
                {/each}
              </ul>
            </article>
          {/each}
        </div>
      </div>
    </section>

    <section id="contact" class="scroll-mt-110 px-16 pb-80 pt-24 md:px-28 md:pb-100 md:pt-40" aria-labelledby="contact-title">
      <div class="mx-auto grid max-w-1120 overflow-hidden rounded-[26px] bg-indigo-950 shadow-2xl shadow-indigo-950/15 lg:grid-cols-[0.85fr_1.15fr]">
        <div class="relative overflow-hidden px-24 py-40 text-white md:px-40 md:py-54">
          <div class="absolute -bottom-120 -left-80 size-300 rounded-full bg-violet-500/25 blur-3xl"></div>
          <div class="relative">
            <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-200"><T text="Contact|Contacto" /></p>
            <h2 id="contact-title" class="mt-12 text-3xl font-bold leading-tight md:text-4xl"><T text="Tell us what your business needs.|Cuéntenos qué necesita su empresa." /></h2>
            <p class="mt-16 max-w-430 leading-[1.55] text-indigo-100"><T text="Share your current workflow, deployment preference, or the Genix capability you want to evaluate.|Comparta su flujo actual, preferencia de implementación o la funcionalidad de Genix que desea evaluar." /></p>
            <div class="mt-28 grid gap-12 text-sm text-indigo-100">
              <span class="flex items-center gap-10"><i class="icon-[fa--code]" aria-hidden="true"></i><T text="Open-source foundation|Base de código abierto" /></span>
              <span class="flex items-center gap-10"><i class="icon-[fa--database]" aria-hidden="true"></i><T text="Portable business data|Datos empresariales portables" /></span>
              <span class="flex items-center gap-10"><i class="icon-[fa--cloud]" aria-hidden="true"></i><T text="Cloud or self-hosted|Nube o autohospedado" /></span>
            </div>
          </div>
        </div>

        <form
          class="grid grid-cols-24 gap-10 bg-white p-20 md:p-36"
          aria-label={tr('Contact form|Formulario de contacto')}
          onsubmit={(event) => event.preventDefault()}
        >
          <Input saveOn={contactForm} save="Name" label="Name|Nombre" required={true} css="col-span-24 md:col-span-12" />
          <Input saveOn={contactForm} save="Email" label="Email|Correo electrónico" type="email" required={true} css="col-span-24 md:col-span-12" />
          <Input saveOn={contactForm} save="Company" label="Company (optional)|Empresa (opcional)" css="col-span-24" />
          <Input saveOn={contactForm} save="Message" label="How can we help?|¿Cómo podemos ayudarle?" required={true} useTextArea={true} rows={5} css="col-span-24" />
          <div class="col-span-24 mt-4">
            <button
              type="submit"
              class="flex h-44 w-full items-center justify-center gap-8 rounded-[10px] bg-indigo-600 px-18 text-white disabled:cursor-not-allowed disabled:bg-slate-300 md:w-auto"
              disabled={true}
              aria-label={tr('Send contact message|Enviar mensaje de contacto')}
            >
              <i class="icon-[fa--paper-plane-o]" aria-hidden="true"></i>
              <span class="font-semibold"><T text="Send message|Enviar mensaje" /></span>
            </button>
            <p class="mt-9 text-sm text-slate-500"><T text="Contact delivery will be enabled after its destination is configured.|El envío de contacto se habilitará cuando se configure su destino." /></p>
          </div>
        </form>
      </div>
    </section>
  </main>

  <footer class="border-t border-slate-200 bg-white px-16 py-28 md:px-28">
    <div class="mx-auto flex max-w-1120 flex-col items-center justify-between gap-18 text-center text-sm text-slate-500 md:flex-row md:text-left">
      <div class="flex items-center gap-12">
        <img class="h-32 w-100 object-contain" src="/images/genix_logo.svg" alt="Genix" />
        <span><T text="Open-source ERP + e-commerce for small businesses.|ERP + comercio electrónico de código abierto para pequeñas empresas." /></span>
      </div>
      <div class="flex flex-wrap items-center justify-center gap-16">
        <button class="hover:text-indigo-700" onclick={() => navigateToSection('features')}><T text="Features|Funcionalidades" /></button>
        <button class="hover:text-indigo-700" onclick={() => navigateToSection('roadmap')}><T text="Roadmap|Roadmap" /></button>
        <span>GPL v3</span>
      </div>
    </div>
  </footer>

  <RegistrationModal
    id={REGISTRATION_MODAL_ID}
    presetRequestID={signUpRequestID}
    presetCode={signUpCode}
    {apiEndpointOptions}
    {selectedApiEndpointRoute}
    {onApiEndpointChange}
  />
</div>

<style>
  /* Keep the login readable while allowing the photographic backdrop to show through. */
  .welcome-glass-panel {
    position: relative;
    overflow: hidden;
    background: rgb(15 23 42 / 34%);
    border: 1px solid rgb(255 255 255 / 24%);
    box-shadow: 0 16px 60px rgb(0 0 0 / 40%), inset 0 0 4px 2px rgb(255 255 255 / 10%);
    backdrop-filter: blur(25px) saturate(100%);
    -webkit-backdrop-filter: blur(25px) saturate(100%);
  }

  /* Two quiet highlights create the reflected-edge effect of physical glass. */
  .welcome-glass-panel::before,
  .welcome-glass-panel::after {
    content: '';
    position: absolute;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    border-radius: inherit;
  }

  .welcome-glass-panel::before {
    background: linear-gradient(to left top, rgb(255 255 255 / 7%), transparent 50%);
  }

  .welcome-glass-panel::after {
    background: linear-gradient(to bottom, rgb(255 255 255 / 5%), transparent 100%);
  }

  .welcome-glass-panel > :global(*) {
    position: relative;
    z-index: 1;
  }

  /* Keep shared inputs legible on this glass panel without changing forms elsewhere. */
  :global(.welcome-login-field) {
    --input-bg: linear-gradient(145deg, rgb(15 23 42 / 72%), rgb(30 41 59 / 58%));
    --input-border-color: rgb(255 255 255 / 20%);
    --input-border-color-hover: rgb(255 255 255 / 38%);
    --input-border-color-focus: rgb(196 181 253 / 85%);
    --input-border-color-invalid: rgb(248 113 113 / 75%);
    --input-border-color-invalid-focus: rgb(252 165 165 / 95%);
    --input-ring-color: rgb(167 139 250 / 32%);
    --input-ring-color-invalid: rgb(248 113 113 / 28%);
    --input-shadow-color: rgb(0 0 0 / 28%);
    --input-label-color: rgb(226 232 240 / 78%);
    --input-label-color-focus: white;
    --input-text-color: white;
    --input-placeholder-color: rgb(226 232 240 / 42%);
    --input-suffix-color: rgb(221 214 254 / 78%);
  }

  /* Reserve the right 65% for the resolved company name and center the ID on the left. */
  :global(.welcome-company-id-field input) {
    padding-right: 65%;
    text-align: center;
  }

  :global(html) {
    scroll-behavior: smooth;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(html) {
      scroll-behavior: auto;
    }
  }
</style>
