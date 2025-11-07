<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/stores';
	import Header from '../components/Header.svelte';
	import SideMenu from '../components/SideMenu.svelte';
    import { Core } from '../core/store.svelte';
    import Modules from '../core/modules';

	let { children } = $props();
	
	// Main content margin - menu is always 4.5rem (w-18) by default
	let mainMarginClass = 'ml-18';

	$effect(() => {
		Core.module = Modules[0]
	})

	// Check if current route should show Header and SideMenu
	let showLayout = $derived(!$page.url.pathname.startsWith('/login'));
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Genix - Sistema de Gestión</title>
</svelte:head>

{#if showLayout}
	<!-- Header with mobile menu toggle -->
	<Header	showMenuButton={true}	title="Sistema Genix"/>
	<!-- Side Menu -->
	<SideMenu />
{/if}

<!-- Main Content -->
{@render children?.()}
