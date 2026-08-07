<script lang="ts">
	import { browser } from '$app/environment';
	import { navigating, page } from '$app/state';
	import TopLayerDatePicker from '$components/layers/TopLayerDatePicker.svelte';
	import TopLayerSelector from '$components/layers/TopLayerSelector.svelte';
	import { Env } from '$core/env';
	import Modules from '$core/modules';
	import { security } from '$libs/ui-runtime.svelte';
	import { Core, getDeviceType, tr } from '$core/store.svelte';
	import AppHeader from '$domain/AppHeader.svelte';
	import favicon from '$libs/assets/favicon.svg?raw';
	import PageLoading from '$domain/PageLoading.svelte';
	import SideMenu from '$domain/SideMenu.svelte';
	import { Notify } from '$libs/helpers';
	import { doInitServiceWorker } from '@genix/ui/service-worker';
	import { onMount } from 'svelte';
	import { provideUi } from '@genix/ui';
	import { genixUiRuntime } from '$libs/ui-runtime.svelte';
	import './app.css';
	import { fetchAccessListCatalog, getAccessEntriesForRoute } from './security/access-profiles/access-list-catalog';
	import './tailwind.css';
	// Shared typography (Open Sans desktop / Inter mobile) — imported last so its
	// ≤749px remap wins the cascade. Same file the storefront uses.
	import '../styles/fonts.css';

	// Registered here, not in the runtime options: this layout is the authenticated app, so
	// the catalog (and backend/access_list.yml with it) stays out of the storefront bundle.
	security.setRouteAccessResolver(getAccessEntriesForRoute);

	let { children } = $props();
	const ui = provideUi(genixUiRuntime);
	ui.state.deviceType = getDeviceType();

	const redirectsToLogin = $derived.by(() => {
		// Skip auth check for store routes (public-facing)
		if(page.url.pathname.startsWith('/store')){
			return false
		}
		// Chrome-less template preview used by the headless review agent — no login needed.
		if(page.url.pathname.startsWith('/webpage-builder/template-preview')){
			return false
		}
		if(["/login"].includes(page.url.pathname)){
			return false
		}
		return security.checkIsLogin() !== 2
	})

	let lastDeniedRoute = $state('')
	let accessCatalogReady = $state(false)
	let accessCatalogLoading = $state(false)
	let accessCatalogFailed = $state(false)

	const loadAccessCatalog = async () => {
		if (accessCatalogReady || accessCatalogLoading || accessCatalogFailed) { return }
		accessCatalogLoading = true
		try {
			await fetchAccessListCatalog()
			accessCatalogReady = true
		} catch (error) {
			accessCatalogFailed = true
			console.error('[access-list] Failed to load access catalog', error)
			Notify.failure(tr('Unable to load access permissions.|No se pudieron cargar los permisos de acceso.'))
		} finally {
			accessCatalogLoading = false
		}
	}

	onMount(() => {
		if(redirectsToLogin){ Env.navigate("/login") }
	})

	$effect(() => {
		if (browser && !redirectsToLogin && page.url.pathname !== '/login' && !accessCatalogReady && !accessCatalogFailed) {
			void loadAccessCatalog()
		}
	})

	$effect(() => {
		Core.module = Modules[0]

		window.addEventListener('resize', () => {
			const newDeviceType = getDeviceType()
			if (newDeviceType !== ui.state.deviceType) { ui.state.deviceType = newDeviceType }
		})

		doInitServiceWorker().then(() => {
			Core.isLoading = 0
		})

	})

	$effect(() => {
		if (!browser || redirectsToLogin || !accessCatalogReady) { return }

		const currentPath = page.url.pathname
		if (security.canAccessRoute(currentPath)) {
			lastDeniedRoute = ''
			return
		}

		if (lastDeniedRoute === currentPath) { return }
		lastDeniedRoute = currentPath

		const accessNames = getAccessEntriesForRoute(currentPath)
			.map((accessEntry) => accessEntry.name)
			.join(', ')
		Notify.failure(tr(`You don't have access "${accessNames}" to visit ${currentPath}|No posee el acceso "${accessNames}" para acceder a ${currentPath}`))
		Env.navigate('/')
	})

	const routesWithoutLayout: string[] = ["/login","/initial-data","/store","/webpage-builder/template-preview"]
	// Check if current route should show Header and SideMenu
	let showLayout = $derived(
		!routesWithoutLayout.some(x => page.url.pathname.startsWith(x))
	);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Genix - Sistema de Gestión</title>
</svelte:head>

{#if showLayout && !redirectsToLogin && accessCatalogReady}
	<TopLayerSelector></TopLayerSelector>
	<TopLayerDatePicker></TopLayerDatePicker>
	<!-- Header with mobile menu toggle -->
	<AppHeader showMenuButton={true}/>
	<!-- Side Menu -->
	<SideMenu />
	<!-- Boot (service worker init) and route transitions share one loader: in both cases the
	     content region has nothing to show yet. `navigating.to` is set the instant goto() runs. -->
	{#if Core.isLoading > 0 || navigating.to}
		<PageLoading path={navigating.to?.url.pathname ?? ''} />
	{/if}
{/if}

<!-- Main Content -->
{#if (Core.isLoading === 0 || !showLayout /* ??? */) && !redirectsToLogin && (accessCatalogReady || !showLayout)}
	{@render children?.()}
{/if}
