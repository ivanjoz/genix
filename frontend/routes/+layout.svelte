<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import TopLayerSelector from '$components/micro/TopLayerSelector.svelte';
	import favicon from '$lib/assets/favicon.svg';
	import { checkIsLogin } from '$lib/security';
	import { doInitServiceWorker } from '$lib/sw-cache';
	import { onMount } from 'svelte';
	import '../app.css';
	import Header from '$components/layout/Header.svelte';
	import Page from '$components/Page.svelte';
	import SideMenu from '$components/layout/SideMenu.svelte';
	import Modules from "$core/modules";
	import { Core, getDeviceType } from '$core/store.svelte';
	import ImageWorker from '../workers/image-worker?worker';
    import { Env } from '../env';

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

	const routesWithoutLayout: string[] = ["login","store"]
	// Check if current route should show Header and SideMenu
	let showLayout = $derived(
		!routesWithoutLayout.some(x => !page.url.pathname.startsWith(x))
	);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Genix - Sistema de Gestión</title>
</svelte:head>

{#if showLayout && !redirectsToLogin}
	<TopLayerSelector></TopLayerSelector>
	<!-- Header with mobile menu toggle -->
	<Header	showMenuButton={true}/>
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
