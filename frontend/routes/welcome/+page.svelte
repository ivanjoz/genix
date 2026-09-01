<script lang="ts">
  import { onMount } from 'svelte';
  import { browser } from '$app/environment';
  import { useUI } from '@genix/ui';
  import Input from '$components/form/Input.svelte';
  import T from '$components/misc/T.svelte';
  import { Core, setLanguaje, tr } from '$core/store.svelte';
  import { Env, lastLoginCompanyIDStorageKey } from '$core/env';
  import { Notify } from '$libs/helpers';
  import { security } from '$libs/ui-runtime.svelte';
  import { getPublicCompanyName, sendUserLogin, type ILogin } from '$services/login';
  import RegistrationModal from './RegistrationModal.svelte';
  import { featureSections } from './features';

  const REGISTRATION_MODAL_ID = 71;
  const WELCOME_DEVELOPER_MODE_STORAGE_KEY = 'genixWelcomeDeveloperMode';
  const GITHUB_REPOSITORY_URL = 'https://github.com/ivanjoz/genix';
  const ui = useUI();

  const navigationItems = [
    { id: 'home', label: 'Home|Inicio' },
    { id: 'features', label: 'Features|Funcionalidades' },
    { id: 'contact', label: 'Contact|Contacto' },
  ];

  // The pitch above the feature lists. `pending` marks a claim the product does not deliver yet, so
  // this section keeps the same promise as the one below it: nothing planned is sold as finished.
  const platformHighlights = [
    {
      icon: 'icon-[fa--sitemap]',
      title: 'Connected operations|Operaciones conectadas',
      description: 'Sales, stock, purchasing, cash, customers, and the online store share the same data. Register one sale and the stock, the cash register, and the report move on their own.|Ventas, stock, compras, caja, clientes y tienda en línea comparten los mismos datos. Registre una venta y el stock, la caja y el reporte se mueven solos.',
      points: [] as { text: string; pending?: boolean }[],
    },
    {
      icon: 'icon-[fa--bar-chart]',
      title: 'Detailed reporting and tracking|Reportería y seguimiento detallado',
      description: '',
      points: [
        { text: 'Dashboards that start from the whole picture and drill into the detail with one click.|Dashboards que parten de lo general y bajan al detalle con un click.' },
        { text: 'Projected cash flow and stock alerts: the system gets ahead of the problem.|Flujos de caja proyectados y alertas de stock: el sistema se adelanta al problema.', pending: true },
        { text: 'Permission and access management, with a complete log per user.|Gestión de permisos y accesos, con registro completo por usuario.' },
      ],
    },
    {
      icon: 'icon-[fa--life-ring]',
      title: 'AI assistance and support|Asistencia por IA y soporte',
      description: 'The AI assistant operates the ERP for you without leaving the page. Found a bug? Leave your request and an agent takes care of it in less than 24 hours.|El asistente de IA opera el ERP por usted sin salir de la página. ¿Algún error? Deje su solicitud y un agente la atiende en menos de 24 horas.',
      points: [] as { text: string; pending?: boolean }[],
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
  let developerMode = $state(false);

  // The public app has one backend: always use the first configured endpoint instead of exposing
  // a server choice or retaining a selection from a prior browser session.
  const selectDefaultApiEndpoint = () => {
    const defaultEndpointRoute = Env.availableApiEndpoints[0]?.route;
    if (defaultEndpointRoute) Env.setSelectedApiEndpoint(defaultEndpointRoute);
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
    selectDefaultApiEndpoint();
    if (security.checkIsLogin() === 2) {
      Env.navigate('/');
      return;
    }
    // The ID survives a reload in localStorage but the name does not, so resolve it again
    // (memory → IndexedDB → server) instead of leaving the field blank.
    void lookupCompanyName();

    // Landing from the registration email opens the wizard straight on the code check.
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('dev') === 'x') {
      localStorage.setItem(WELCOME_DEVELOPER_MODE_STORAGE_KEY, '1');
    }
    developerMode = localStorage.getItem(WELCOME_DEVELOPER_MODE_STORAGE_KEY) === '1';

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

{#snippet githubMark(sizeClass: string)}
  <svg class={sizeClass} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
    <path fill-rule="evenodd" d="M8 0C3.58 0 0 3.58 0 8C0 11.54 2.29 14.53 5.47 15.59C5.87 15.66 6.02 15.42 6.02 15.21C6.02 15.02 6.01 14.39 6.01 13.72C4 14.09 3.48 13.23 3.32 12.78C3.23 12.55 2.84 11.84 2.5 11.65C2.22 11.5 1.82 11.13 2.49 11.12C3.12 11.11 3.57 11.7 3.72 11.94C4.44 13.15 5.59 12.81 6.05 12.6C6.12 12.08 6.33 11.73 6.56 11.53C4.78 11.33 2.92 10.64 2.92 7.58C2.92 6.71 3.23 5.99 3.74 5.43C3.66 5.23 3.38 4.41 3.82 3.31C3.82 3.31 4.49 3.1 6.02 4.13C6.66 3.95 7.34 3.86 8.02 3.86C8.7 3.86 9.38 3.95 10.02 4.13C11.55 3.09 12.22 3.31 12.22 3.31C12.66 4.41 12.38 5.23 12.3 5.43C12.81 5.99 13.12 6.7 13.12 7.58C13.12 10.65 11.25 11.33 9.47 11.53C9.76 11.78 10.01 12.26 10.01 13.01C10.01 14.08 10 14.94 10 15.21C10 15.42 10.15 15.67 10.55 15.59C13.71 14.53 16 11.53 16 8C16 3.58 12.42 0 8 0Z" />
  </svg>
{/snippet}

<div class="min-h-screen bg-slate-50 text-slate-900 selection:bg-indigo-200 selection:text-indigo-950">
  <header class="fixed inset-x-0 top-0 z-50">
    <nav
      class="border-b border-white/15 bg-slate-950/60 px-14 py-10 shadow-[0_8px_32px_rgba(15,23,42,0.18)] backdrop-blur-2xl md:px-24"
      aria-label={tr('Primary navigation|Navegación principal')}
    >
      <div class="mx-auto flex max-w-1320 items-center justify-between gap-12">
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
          <a
            class="hidden h-38 items-center gap-7 rounded-[10px] px-12 text-sm text-white/80 transition hover:bg-white/10 hover:text-white focus-visible:outline-3 focus-visible:outline-white md:flex"
            href={GITHUB_REPOSITORY_URL}
            target="_blank"
            rel="noreferrer"
          >
            {@render githubMark('size-16')}
            Github
          </a>
          {#if developerMode}
            <button
              class="hidden h-38 items-center rounded-[10px] bg-indigo-600 px-16 text-sm text-white shadow-md shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 md:flex"
              onclick={openRegistration}
            >
              <T text="Register|Regístrese" />
            </button>
          {/if}
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
        <div id="welcome-mobile-menu" class="mx-auto mt-10 grid max-w-1320 gap-4 border-t border-white/15 pt-10 lg:hidden">
          {#each navigationItems as navigationItem}
            <button
              class="rounded-[9px] px-12 py-10 text-left text-white/80 hover:bg-white/10 hover:text-white"
              onclick={() => navigateToSection(navigationItem.id)}
            >
              <T text={navigationItem.label} />
            </button>
          {/each}
          <div class="mt-4 grid gap-8 {developerMode ? 'grid-cols-2' : 'grid-cols-1'}">
            <button class="rounded-[10px] border border-white/25 px-12 py-10 text-white" onclick={focusLogin}>
              <T text="Sign in|Ingresar" />
            </button>
            {#if developerMode}
              <button class="rounded-[10px] bg-indigo-600 px-12 py-10 text-white" onclick={openRegistration}>
                <T text="Register|Regístrese" />
              </button>
            {/if}
          </div>
          <a
            class="mt-4 flex items-center justify-center gap-7 rounded-[10px] border border-white/25 px-12 py-10 text-white"
            href={GITHUB_REPOSITORY_URL}
            target="_blank"
            rel="noreferrer"
          >
            {@render githubMark('size-16')}
            Github
          </a>
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
            <h1 class="ff-display max-w-620 text-4xl font-bold leading-tight md:text-5xl lg:text-[56px]">
              <T text="Run your business from one clear, connected system.|Gestione cada proceso de su empresa" />
            </h1>
            <p class="mt-20 max-w-600 text-lg leading-[1.6] text-indigo-100 md:text-xl">
              <T text="Genix brings sales, inventory, purchasing, finance, and online commerce together for small businesses that want control without complexity.|Genix reúne ventas, inventario, compras, finanzas y comercio electrónico para pequeñas empresas que buscan control sin complejidad." />
            </p>
            {#if developerMode}
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
            {:else}
              <div
                class="mt-28 flex max-w-600 items-start gap-10 rounded-[14px] border border-violet-300/45 bg-slate-900/82 px-16 py-14 text-sm leading-[1.55] text-[#ffeed6] shadow-lg shadow-slate-950/40 backdrop-blur"
                role="status"
              >
                <span class="w-4 shrink-0 self-stretch rounded-[2px] bg-[#ffeed6]" aria-hidden="true"></span>
                <T text="Available from February 8, 2027, with a free-use limit or a paid business plan.|Disponible desde el 8 de Febrero 2027, con límite para uso gratuito o con plan empresarial de pago." />
              </div>
            {/if}
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
              <!-- Keep the field mounted so loading never replaces the form with a mismatched block. -->
              <Input required={true} label="Password|Contraseña" saveOn={loginForm} save="Password" type="password" css="welcome-login-field" inputCss="text-base" disabled={isLoginLoading} />
              <button
                type="submit"
                class="mt-4 flex h-44 w-full items-center justify-center gap-8 rounded-[10px] bg-indigo-600 px-18 text-white shadow-lg shadow-indigo-900/15 transition hover:bg-indigo-700 focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-60"
                aria-label={tr('Submit login credentials|Enviar credenciales de acceso')}
                aria-busy={isLoginLoading}
                disabled={isLoginLoading}
              >
                {#if isLoginLoading}
                  <!-- A restrained ring reads clearly at button scale and matches the glass palette. -->
                  <span class="welcome-login-spinner" aria-hidden="true"></span>
                {:else}
                  <i class="icon-[fa--sign-in]" aria-hidden="true"></i>
                {/if}
                <span class="font-semibold"><T text={isLoginLoading ? 'Signing in...|Ingresando...' : 'Sign in|Ingresar'} /></span>
              </button>
            </form>
        </div>
      </div>
    </section>

    <section class="px-16 py-72 md:px-28 md:py-96" aria-labelledby="what-is-genix">
      <div class="mx-auto max-w-1320 text-center">
        <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-600"><T text="Your business, under control|Su negocio bajo control" /></p>
        <h2 id="what-is-genix" class="ff-display mx-auto mt-12 max-w-760 text-3xl font-bold leading-tight text-slate-950 md:text-4xl">
          <T text="Your business under control, a few clicks away.|El control de su empresa a pocos clicks." />
        </h2>
        <p class="mx-auto mt-16 max-w-760 text-lg leading-[1.6] text-slate-600">
          <T text="Stop chasing your own information across notebooks, spreadsheets, and chats. What happened at the counter, in the warehouse, and in the cash register is one click away, up to date and in one place.|Deje de perseguir su propia información entre cuadernos, hojas de cálculo y conversaciones. Lo que pasó en el mostrador, el almacén y la caja está a un click, al día y en un solo lugar." />
        </p>

        <div class="mt-42 grid gap-14 md:grid-cols-3">
          {#each platformHighlights as platformHighlight}
            <article class="rounded-[18px] border border-slate-200 bg-white p-24 text-left shadow-sm">
              <i class="{platformHighlight.icon} text-2xl text-indigo-600" aria-hidden="true"></i>
              <h3 class="mt-16 text-xl font-semibold"><T text={platformHighlight.title} /></h3>
              {#if platformHighlight.description}
                <p class="mt-8 leading-[1.55] text-slate-600"><T text={platformHighlight.description} /></p>
              {:else}
                <ul class="mt-12 grid gap-9">
                  {#each platformHighlight.points as highlightPoint}
                    <li class="flex gap-9 leading-[1.5] {highlightPoint.pending ? 'text-slate-500' : 'text-slate-600'}">
                      <i
                        class="{highlightPoint.pending ? 'icon-[fa--clock-o] text-amber-500' : 'icon-[fa--check] text-indigo-600'} mt-5 shrink-0"
                        aria-hidden="true"
                      ></i>
                      <span>
                        <span class="sr-only">{highlightPoint.pending ? tr('Coming soon|Próximamente') : tr('Available today|Disponible hoy')}: </span>
                        <T text={highlightPoint.text} />
                      </span>
                    </li>
                  {/each}
                </ul>
              {/if}
            </article>
          {/each}
        </div>
      </div>
    </section>

    <section id="features" class="scroll-mt-110 bg-white" aria-labelledby="features-title">
      <div class="mx-auto max-w-1320 px-16 pb-40 pt-72 md:px-28 md:pb-48 md:pt-96">
        <div class="max-w-760">
          <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-600"><T text="Capabilities|Funcionalidades" /></p>
          <h2 id="features-title" class="ff-display mt-12 text-3xl font-bold leading-tight md:text-4xl"><T text="Everything your business needs, in one system.|Todo lo que su negocio necesita, en un solo sistema." /></h2>
          <p class="mt-14 text-lg leading-[1.6] text-slate-600"><T text="Genix covers the daily life of a small business — selling, buying, producing, and controlling stock and cash — and grows with you into planning, accounting, and online sales.|Genix cubre el día a día de una micro o pequeña empresa: vender, comprar, producir, controlar el stock y la caja, y crece con usted hacia la planificación, la contabilidad y la venta en línea." /></p>
          <!-- The product is unfinished on purpose and says so: the legend explains both marks before
               the first list, so a planned item is never read as something that already works. -->
          <p class="mt-12 leading-[1.55] text-slate-600"><T text="We are building it in the open. Each capability below carries its own mark.|Estamos construyendo el producto a la vista. Cada funcionalidad de abajo lleva su propia marca." /></p>
          <div class="mt-16 flex flex-wrap gap-x-20 gap-y-10 text-sm">
            <span class="flex items-center gap-8 rounded-full border border-slate-200 bg-slate-50 px-12 py-6 text-slate-700">
              <i class="icon-[fa--check] text-indigo-600" aria-hidden="true"></i><T text="Available today|Disponible hoy" />
            </span>
            <span class="flex items-center gap-8 rounded-full border border-slate-200 bg-slate-50 px-12 py-6 text-slate-700">
              <i class="icon-[fa--clock-o] text-amber-500" aria-hidden="true"></i><T text="Coming soon|Próximamente" />
            </span>
          </div>
        </div>
      </div>

      {#each featureSections as featureSection, sectionIndex}
        <article
          id={featureSection.id}
          class="scroll-mt-110 border-t border-slate-200 px-16 py-44 md:px-28 md:py-60 {sectionIndex % 2 === 0 ? 'bg-slate-50' : 'bg-white'}"
          aria-labelledby="{featureSection.id}-title"
        >
          <!-- The illustration alternates sides, so the column template flips with it: moving the
               image with `order` alone would leave the text in the narrow 360px track. -->
          <div
            class="mx-auto grid max-w-1320 items-center gap-28 lg:gap-40 {sectionIndex % 2 === 1
              ? 'lg:grid-cols-[minmax(0,1fr)_minmax(0,400px)]'
              : 'lg:grid-cols-[minmax(0,400px)_minmax(0,1fr)]'}"
          >
            <!-- Decorative: every claim the illustration makes is already written in the list beside it.
                 The square box is reserved by CSS so the lazy load shifts nothing, and `object-contain`
                 letterboxes the one illustration that is 3:2 instead of stretching it. -->
            <img
              class="mx-auto aspect-square w-full max-w-320 object-contain lg:max-w-400 {sectionIndex % 2 === 1 ? 'lg:order-2' : ''}"
              src={featureSection.illustration}
              alt=""
              loading="lazy"
            />
            <div>
              <div class="flex items-center gap-12">
                <div class="flex size-44 shrink-0 items-center justify-center rounded-[12px] bg-indigo-100 text-indigo-700">
                  <i class="{featureSection.icon} text-xl" aria-hidden="true"></i>
                </div>
                <h3 id="{featureSection.id}-title" class="text-2xl font-semibold text-slate-950"><T text={featureSection.title} /></h3>
              </div>
              <p class="mt-14 leading-[1.6] text-slate-600"><T text={featureSection.lead} /></p>
              <ul class="mt-20 grid gap-11 sm:grid-cols-2 sm:gap-x-20">
                {#each featureSection.available as availableFeature}
                  <li class="flex gap-9 leading-[1.5] text-slate-700">
                    <i class="icon-[fa--check] mt-5 shrink-0 text-indigo-600" aria-hidden="true"></i>
                    <span><span class="sr-only">{tr('Available today|Disponible hoy')}: </span><T text={availableFeature} /></span>
                  </li>
                {/each}
                {#each featureSection.pending as pendingFeature}
                  <li class="flex gap-9 leading-[1.5] text-slate-500">
                    <i class="icon-[fa--clock-o] mt-5 shrink-0 text-amber-500" aria-hidden="true"></i>
                    <span><span class="sr-only">{tr('Coming soon|Próximamente')}: </span><T text={pendingFeature} /></span>
                  </li>
                {/each}
              </ul>
            </div>
          </div>
        </article>
      {/each}
    </section>
    <section id="contact" class="scroll-mt-110 px-16 pb-80 pt-24 md:px-28 md:pb-100 md:pt-40" aria-labelledby="contact-title">
      <div class="mx-auto grid max-w-1320 overflow-hidden rounded-[26px] bg-indigo-950 shadow-2xl shadow-indigo-950/15 lg:grid-cols-[0.85fr_1.15fr]">
        <div class="relative overflow-hidden px-24 py-40 text-white md:px-40 md:py-54">
          <div class="absolute -bottom-120 -left-80 size-300 rounded-full bg-violet-500/25 blur-3xl"></div>
          <div class="relative">
            <p class="text-sm font-semibold uppercase tracking-[0.18em] text-indigo-200"><T text="Contact|Contacto" /></p>
            <h2 id="contact-title" class="ff-display mt-12 text-3xl font-bold leading-tight md:text-4xl"><T text="Tell us what your business needs.|Cuéntenos qué necesita su empresa." /></h2>
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
    <div class="mx-auto flex max-w-1320 flex-col items-center justify-between gap-18 text-center text-sm text-slate-500 md:flex-row md:text-left">
      <div class="flex items-center gap-12">
        <img class="h-32 w-100 object-contain" src="/images/genix_logo.svg" alt="Genix" />
        <span><T text="Open-source ERP + e-commerce for small businesses.|ERP + comercio electrónico de código abierto para pequeñas empresas." /></span>
      </div>
      <div class="flex flex-wrap items-center justify-center gap-16">
        <button class="hover:text-indigo-700" onclick={() => navigateToSection('features')}><T text="Features|Funcionalidades" /></button>
        <button class="hover:text-indigo-700" onclick={() => navigateToSection('roadmap')}><T text="Roadmap|Roadmap" /></button>
        <!-- Storyset's free license requires this credit; dropping it needs their paid license. -->
        <a class="hover:text-indigo-700" href="https://storyset.com/" target="_blank" rel="noopener noreferrer">Illustrations by Storyset</a>
        <span>GPL v3</span>
      </div>
    </div>
  </footer>

  <RegistrationModal
    id={REGISTRATION_MODAL_ID}
    presetRequestID={signUpRequestID}
    presetCode={signUpCode}
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
    min-height: 440px;
    display: flex;
    flex-direction: column;
    justify-content: center;
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
    --input-bg-disabled: linear-gradient(145deg, rgb(15 23 42 / 78%), rgb(30 41 59 / 66%));
    --input-border-color-disabled: rgb(255 255 255 / 14%);
    --input-label-color: rgb(226 232 240 / 78%);
    --input-label-color-focus: white;
    --input-label-color-disabled: rgb(226 232 240 / 58%);
    --input-text-color: white;
    --input-text-color-disabled: rgb(226 232 240 / 62%);
    --input-placeholder-color: rgb(226 232 240 / 42%);
    --input-suffix-color: rgb(221 214 254 / 78%);
  }

  /* app.css paints disabled native inputs white; the shell owns the visible glass fill. */
  :global(.welcome-login-field input:disabled) {
    background: transparent;
    color: var(--input-text-color-disabled);
    -webkit-text-fill-color: var(--input-text-color-disabled);
  }

  /* One quiet loading cue is enough; keeping it local avoids changing shared buttons. */
  .welcome-login-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgb(255 255 255 / 38%);
    border-top-color: white;
    border-radius: 999px;
    animation: welcome-login-spin 0.75s linear infinite;
  }

  @keyframes welcome-login-spin {
    to { transform: rotate(360deg); }
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
