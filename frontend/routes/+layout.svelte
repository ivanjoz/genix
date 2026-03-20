<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import TopLayerSelector from '$components/TopLayerSelector.svelte';
	import { Env } from '$core/env';
	import Modules from '$core/modules';
	import { canUserAccessRoute, checkIsLogin } from '$core/security';
	import { Core, getDeviceType } from '$core/store.svelte';
	import AppHeader from '$domain/AppHeader.svelte';
	import favicon from '$domain/assets/favicon.svg?raw';
	import '$domain/libs/fontello-embedded.css';
	import Page from '$domain/Page.svelte';
	import SideMenu from '$domain/SideMenu.svelte';
	import { Notify } from '$libs/helpers';
	import { doInitServiceWorker } from '$libs/sw-cache';
	import ImageWorker from '$libs/workers/image-worker?worker';
	import { onMount } from 'svelte';
	import './app.css';
	import { getAccessEntriesForRoute } from './seguridad/perfiles-accesos/access-list-catalog';
	import './tailwind.css';

	let { children } = $props();

	if(browser){
		console.log('🔧 Initializing image worker...')
		try {
			Env.ImageWorkerClass = ImageWorker
			Env.imageWorker = new ImageWorker()
			console.log('✅ Image worker initialized successfully')
		} catch (error) {
			console.error('❌ Failed to initialize image worker:', error)
		}
	}

	const redirectsToLogin = $derived.by(() => {
		// Skip auth check for store routes (public-facing)
		if(page.url.pathname.startsWith('/store')){
			return false
		}
		if(["/login"].includes(page.url.pathname)){
			return false
		}
		return checkIsLogin() !== 2
	})

	let lastDeniedRoute = $state('')

	onMount(() => {
		if(redirectsToLogin){ Env.navigate("/login") }
	})

	$effect(() => {
		Core.module = Modules[0]
		console.log("imageWorker",Env, Env.imageWorker)

		window.addEventListener('resize', () => {
			const newDeviceType = getDeviceType()
			if (newDeviceType !== Core.deviceType) { Core.deviceType = newDeviceType }
		})

		doInitServiceWorker().then(() => {
			Core.isLoading = 0
		})

	})

	$effect(() => {
		if (!browser || redirectsToLogin) { return }

		const currentPath = page.url.pathname
		if (canUserAccessRoute(currentPath)) {
			lastDeniedRoute = ''
			return
		}

		if (lastDeniedRoute === currentPath) { return }
		lastDeniedRoute = currentPath

		const accessNames = getAccessEntriesForRoute(currentPath)
			.map((accessEntry) => accessEntry.name)
			.join(', ')
		Notify.failure(`No posee el acceso "${accessNames}" para acceder a ${currentPath}`)
		Env.navigate('/')
	})

	const routesWithoutLayout: string[] = ["/login","/store"]
	// Check if current route should show Header and SideMenu
	let showLayout = $derived(
		!routesWithoutLayout.some(x => page.url.pathname.startsWith(x))
	);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Genix - Sistema de Gestión</title>
</svelte:head>

{#if showLayout && !redirectsToLogin}
	<TopLayerSelector></TopLayerSelector>
	<!-- Header with mobile menu toggle -->
	<AppHeader showMenuButton={true}/>
	<!-- Side Menu -->
	<SideMenu />
	{#if Core.isLoading > 0}
		<Page title="...">
			<div class="p-12"><h2>Cargando...</h2></div>
		</Page>
	{/if}
{/if}

<!-- Main Content -->
{#if (Core.isLoading === 0 || !showLayout /* ??? */) && !redirectsToLogin}
	{@render children?.()}
{/if}
