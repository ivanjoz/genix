<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import Header from '../components/Header.svelte';
	import SideMenu from '../components/SideMenu.svelte';
	import { getDeviceType, Core } from '../core/store.svelte';
	import Modules from '../core/modules';
	import { doInitServiceWorker } from '$lib/sw-cache';
	import Page from '../components/Page.svelte';
  import { Env } from '$lib/security';
  import { browser } from '$app/environment';
    import TopLayerSelector from '$components/micro/TopLayerSelector.svelte';

	let { children } = $props();

	if(browser){
		console.log('🔧 Initializing image worker...')
		try {
			Env.imageWorker = new Worker(
				new URL('../workers/image-worker.ts', import.meta.url), { type: 'module' })
			console.log('✅ Image worker initialized successfully:', Env.imageWorker)
		} catch (error) {
			console.error('❌ Failed to initialize image worker:', error)
		}
	}
	
	// Main content margin - menu is always 4.5rem (w-18) by default
	let mainMarginClass = 'ml-18';

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

	// Check if current route should show Header and SideMenu
	let showLayout = $derived(!page.url.pathname.startsWith('/login'));
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Genix - Sistema de Gestión</title>
</svelte:head>

{#if showLayout}
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
{#if Core.isLoading === 0 || !showLayout}
	{@render children?.()}
{/if}
