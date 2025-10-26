<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import Header from '../components/Header.svelte';
	import SideMenu from '../components/SideMenu.svelte';
    import { Core } from '../core/store.svelte';
    import Modules from '../core/modules';

	let { children } = $props();

	// State for mobile menu
	let isMobileMenuOpen = $state(false);
	
	// Main content margin - menu is always 4.5rem (w-18) by default
	let mainMarginClass = 'ml-18';

	$effect(() => {
		Core.module = Modules[0]
	})
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Genix - Sistema de Gestión</title>
</svelte:head>

<!-- Header with mobile menu toggle -->
<Header
	showMenuButton={true}
	onMenuToggle={() => (isMobileMenuOpen = !isMobileMenuOpen)}
	title="Sistema Genix"
/>

<!-- Side Menu -->
<SideMenu bind:isMobileOpen={isMobileMenuOpen} />

<!-- Main Content -->
<main
	class="pt-12 min-h-screen bg-gray-50 
		transition-all duration-300
		{mainMarginClass}
		md:{mainMarginClass}
		p-4 md:p-6"
>
	<div class="max-w-7xl mx-auto">
		{@render children?.()}
	</div>
</main>
