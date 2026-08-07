<script lang="ts">
  import { onMount } from 'svelte';
  import { useUI } from '@genix/ui';
  import Input from '$components/form/Input.svelte';
  import SearchSelect from '$components/form/SearchSelect.svelte';
  import T from '$components/misc/T.svelte';
  import { Core, setLanguaje, tr } from '$core/store.svelte';
  import { Env, type IApiEndpointOption } from '$core/env';
  import { Notify } from '$libs/helpers';
  import { security } from '$libs/ui-runtime.svelte';
  import { sendUserLogin, type ILogin } from '$services/login';
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

  let loginForm = $state<ILogin>({ User: '', Password: '', CompanyID: 1, CipherKey: '' });
  let selectorForm = $state({ selectedApiEndpointRoute: '' });
  let apiEndpointOptions = $state<IApiEndpointOption[]>([]);
  let isLoginLoading = $state(false);
  let mobileMenuOpen = $state(false);
  let contactForm = $state({ Name: '', Email: '', Company: '', Message: '' });

  // Keep endpoint selection identical to the former login page so authentication behavior does not change.
  const syncApiEndpointSelector = () => {
    apiEndpointOptions = [...Env.availableApiEndpoints];
    selectorForm.selectedApiEndpointRoute = Env.selectedApiEndpointRoute || apiEndpointOptions[0]?.route || '';
  };

  const onApiEndpointChange = (selectedEndpoint?: IApiEndpointOption) => {
    if (!selectedEndpoint?.route) return;
    Env.setSelectedApiEndpoint(selectedEndpoint.route);
    selectorForm.selectedApiEndpointRoute = selectedEndpoint.route;
    console.info('[WelcomeLogin] API endpoint selected:', selectedEndpoint.route);
  };

  const submitLogin = async () => {
    if (loginForm.User.trim().length < 4 || loginForm.Password.length < 4) {
      Notify.failure(tr('Please provide a valid username and password.|Debe proporcionar un usuario y una contraseña válidos.'));
      return;
    }

    isLoginLoading = true;
    Notify.info(tr('Sending credentials...|Enviando credenciales...'));
    console.info('[WelcomeLogin] Sending credentials to selected endpoint.');
    const result = await sendUserLogin(loginForm);
    isLoginLoading = false;

    if (result.error) {
      console.error('[WelcomeLogin] Login request failed:', result.error);
      Notify.failure(tr('Login error|Error al iniciar sesión'));
    }
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
    if (security.checkIsLogin() === 2) Env.navigate('/');
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
  <header class="sticky top-0 z-50 px-10 pt-10 md:px-24 md:pt-16">
    <nav
      class="mx-auto max-w-1240 rounded-[18px] border border-white/70 bg-white/75 px-14 py-10 shadow-xl shadow-indigo-950/8 backdrop-blur-xl md:px-20"
      aria-label={tr('Primary navigation|Navegación principal')}
    >
      <div class="flex items-center justify-between gap-12">
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
              class="rounded-[9px] px-12 py-8 text-sm text-slate-600 transition hover:bg-indigo-50 hover:text-indigo-700 focus-visible:outline-3 focus-visible:outline-indigo-600"
              onclick={() => navigateToSection(navigationItem.id)}
            >
              <T text={navigationItem.label} />
            </button>
          {/each}
        </div>

        <div class="flex items-center gap-6">
          <div class="flex rounded-[10px] border border-slate-200 bg-white/80 p-3" aria-label={tr('Language|Idioma')}>
            <button
              class="rounded-[7px] px-8 py-5 text-xs transition {Core.languaje === 1 ? 'bg-indigo-600 text-white' : 'text-slate-500 hover:text-indigo-700'}"
              aria-pressed={Core.languaje === 1}
              onclick={() => setLanguaje(1)}
            >ES</button>
            <button
              class="rounded-[7px] px-8 py-5 text-xs transition {Core.languaje === 2 ? 'bg-indigo-600 text-white' : 'text-slate-500 hover:text-indigo-700'}"
              aria-pressed={Core.languaje === 2}
              onclick={() => setLanguaje(2)}
            >EN</button>
          </div>

          <button
            class="hidden h-38 items-center rounded-[10px] px-12 text-sm text-indigo-700 transition hover:bg-indigo-50 focus-visible:outline-3 focus-visible:outline-indigo-600 md:flex"
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
            class="flex size-38 items-center justify-center rounded-[10px] border border-slate-200 bg-white text-slate-700 lg:hidden"
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
        <div id="welcome-mobile-menu" class="mt-10 grid gap-4 border-t border-slate-200/80 pt-10 lg:hidden">
          {#each navigationItems as navigationItem}
            <button
              class="rounded-[9px] px-12 py-10 text-left text-slate-700 hover:bg-indigo-50"
              onclick={() => navigateToSection(navigationItem.id)}
            >
              <T text={navigationItem.label} />
            </button>
          {/each}
          <div class="mt-4 grid grid-cols-2 gap-8">
            <button class="rounded-[10px] border border-indigo-200 px-12 py-10 text-indigo-700" onclick={focusLogin}>
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
    <section id="home" class="scroll-mt-110 px-16 pb-72 pt-38 md:px-28 md:pb-96 md:pt-54">
      <div class="relative mx-auto max-w-1240 overflow-hidden rounded-[30px] bg-indigo-950 shadow-2xl shadow-indigo-950/20">
        <img
          class="absolute inset-0 h-full w-full object-cover opacity-35"
          src="/images/background-1.webp"
          alt={tr('Small business owner managing orders|Propietaria de una pequeña empresa gestionando pedidos')}
        />
        <div class="absolute inset-0 bg-linear-to-r from-indigo-950 via-indigo-950/90 to-indigo-900/55"></div>
        <div class="absolute -left-80 -top-100 size-320 rounded-full bg-violet-500/25 blur-3xl"></div>

        <div class="relative grid items-center gap-38 px-20 py-40 md:px-42 md:py-58 lg:grid-cols-[minmax(0,1.15fr)_minmax(360px,0.85fr)] lg:gap-54">
          <div class="max-w-650 text-white">
            <div class="mb-18 inline-flex items-center gap-8 rounded-full border border-white/20 bg-white/10 px-14 py-7 text-sm backdrop-blur">
              <i class="icon-[fa--code-fork] text-violet-200" aria-hidden="true"></i>
              <T text="Open-source ERP + e-commerce|ERP + comercio electrónico de código abierto" />
            </div>
            <h1 class="max-w-620 text-4xl font-bold leading-tight md:text-5xl lg:text-[56px]">
              <T text="Run your business from one clear, connected system.|Dirija su empresa desde un sistema claro y conectado." />
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

          <div id="login" class="scroll-mt-120 rounded-[22px] border border-white/35 bg-white/92 p-20 shadow-2xl shadow-black/25 backdrop-blur-xl md:p-28">
            <div class="mb-20">
              <div class="mb-14 flex size-44 items-center justify-center rounded-[12px] bg-indigo-50 text-indigo-700">
                <i class="icon-[fa--sign-in] text-xl" aria-hidden="true"></i>
              </div>
              <h2 class="text-2xl font-bold text-slate-900"><T text="Sign in to your workspace|Ingrese a su espacio de trabajo" /></h2>
              <p class="mt-6 text-sm leading-[1.45] text-slate-500"><T text="Use your Genix credentials and select your server.|Use sus credenciales de Genix y seleccione su servidor." /></p>
            </div>

            <form
              id="login-form"
              class="grid gap-14"
              aria-label={tr('Login form with username and password|Formulario de acceso con usuario y contraseña')}
              onsubmit={(event) => { event.preventDefault(); void submitLogin(); }}
            >
              <Input required={true} label="Username|Usuario" saveOn={loginForm} save="User" type="text" inputCss="text-base" />
              <Input required={true} label="Password|Contraseña" saveOn={loginForm} save="Password" type="password" inputCss="text-base" />
              {#if apiEndpointOptions.length > 0}
                <SearchSelect
                  css="w-full"
                  inputCss="text-base"
                  label="Server|Servidor"
                  saveOn={selectorForm}
                  save="selectedApiEndpointRoute"
                  options={apiEndpointOptions}
                  keyId="route"
                  keyName="name"
                  onChange={onApiEndpointChange}
                />
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

  <RegistrationModal id={REGISTRATION_MODAL_ID} />
</div>

<style>
  :global(html) {
    scroll-behavior: smooth;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(html) {
      scroll-behavior: auto;
    }
  }
</style>
