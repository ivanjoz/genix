<script lang="ts">
import { Core, fetchOnCourse } from '$core/core/store.svelte';
    import ButtonLayer from "../micro/ButtonLayer.svelte";
    import HeaderConfig from "./HeaderConfig.svelte";

	// Props
	const {
		showMenuButton = false
	}: {
		showMenuButton?: boolean;
	} = $props();

	// State
	let showSettings = $state(false);
	let uiTheme = $state<'light' | 'dark'>('light');
	let isReloading = $state(false);

	// Functions
	function toggleSettings() {
		showSettings = !showSettings;
	}

	function toggleTheme(theme: 'light' | 'dark') {
		uiTheme = theme;
		document.body.classList.remove('light', 'dark');
		document.body.classList.add(theme);
		localStorage.setItem('ui-color', theme);
		showSettings = false;
	}

	function handleReload() {
		isReloading = true;
		const now5secodsMore = Math.floor(Date.now() / 1000) + 5;
		localStorage.setItem('force_sync_cache_until', String(now5secodsMore));
		window.location.reload();
	}

	// Initialize theme from localStorage
	$effect(() => {
		const savedTheme = localStorage.getItem('ui-color') || 'light';
		if (savedTheme === 'dark' || savedTheme === 'light') {
			uiTheme = savedTheme;
			document.body.classList.add(savedTheme);
		}
	});
</script>

<header	class="_1 fixed top-0 left-0 right-0 bg-gradient-to-r from-indigo-600 to-indigo-700 
	shadow-md z-150 flex items-center px-6 md:px-16"
>

	<!-- Logo Section (Desktop) -->
	<div class="hidden md:flex items-center justify-center h-full w-56 mr-12">
		<div class="h-40 w-40 bg-black/20 rounded-lg flex items-center justify-center">
			<img src="/images/genix_logo4.svg" alt="Genix Logo" class="w-full h-full p-1" />
		</div>
	</div>

	<!-- Mobile Menu Button -->
	{#if showMenuButton}
		<button type="button"
			class="md:hidden p-8 hover:bg-white/10 rounded-lg transition-colors mr-12 cursor-pointer"
			aria-label="Toggle menu"
			onclick={() => Core.toggleMobileMenu()}
		>
			<span class="text-white text-2xl">☰</span>
		</button>
	{/if}

	<!-- Title -->
	<div class="flex-1 flex items-center">
		{#if Core.pageOptions?.length > 0}
			{#each Core.pageOptions as opt }
			{@const selected = Core.pageOptionSelected == opt.id}
				<button class="_2" class:_3={selected} aria-label={opt.name}
					onclick={() => {
						Core.pageOptionSelected = opt.id
					}}>{opt.name}
				</button>
			{/each}
		{:else}
			<div class="h1 text-white text-lg font-semibold tracking-wide">
				{Core.pageTitle}
			</div>
		{/if}
	</div>

	<!-- Right Actions -->
	<div class="flex items-center gap-8 h-full relative">
		{#if fetchOnCourse.size > 0}
			<div class="pm-loading mr-06">
				<div class="bg"></div>
				<span>{"Cargando..."}</span>
			</div>
		{/if}

		<!-- Settings Dropdown -->
		<div class="relative">
			<ButtonLayer layerClass="md:w-640 md:h-460 px-8 py-6"
				buttonClass="w-40 h-40 rounded-full bg-white/10 hover:bg-white/20 
					flex items-center justify-center transition-colors shadow-sm"
				contentCss="px-4 pb-8 md:px-8 md:py-8"
			>
				{#snippet button()}
					<span class="text-white text-lg icon-cog"></span>
				{/snippet}
				<HeaderConfig />
			</ButtonLayer>
		</div>

		<!-- Reload Button -->
		<button
			class="hidden md:flex w-40 h-40 rounded-full bg-white/10 hover:bg-white/20 
				items-center justify-center transition-colors shadow-sm
				{isReloading ? 'animate-spin' : ''}"
			onclick={handleReload}
			aria-label="Reload"
			disabled={isReloading}
		>
			<span class="text-white text-lg icon-cw"></span>
		</button>
	</div>
</header>

<!-- Click outside to close settings -->
{#if showSettings}
	<div
		class="fixed inset-0 z-30"
		onclick={() => (showSettings = false)}
		onkeydown={(e) => e.key === 'Escape' && (showSettings = false)}
		role="button"
		tabindex="0"
		aria-label="Close settings"
	></div>
{/if}

<style>
	._1 {
		height: var(--header-height);
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	.animate-spin {
		animation: spin 1s linear infinite;
	}

	/* LOADING */
	.pm-loading {
		height: calc(100% - 7px);
		margin-bottom: 1px;
		width: 10rem;
		text-align: left;
		position: relative;
		line-height: 1;
		display: flex;
		z-index: 210;
		padding: 0.6px 8px 0.6px 1.8rem;
		color: #ffffff;
		overflow: hidden;
		align-items: center;
		border-radius: 7px;
	}
	/*
	.pm-loading > .pm-counter {
		height: calc(100% - 8px);
		background-color: black;
		color: #fff324;
		width: 1.4rem;
		display: flex;
		align-items: center;
		justify-content: center;
		position: absolute;
		left: 4px;
	}
	*/
	.pm-loading > .bg {
		position: absolute;
		left: -46px;
		right: 0;
		top: 0;
		bottom: 0;
		z-index: -1;
		background: repeating-linear-gradient(-55deg,#000000 1px,#323538 2px,	#393d42 11px,#292827 12px,#000000 20px);
		animation-name: MOVE-BG;
		animation-duration: 0.4s;
		animation-timing-function: linear;
		animation-iteration-count: infinite;
	}

	._2 {
		color: white;
		min-width: 150px;
		background-color: rgba(0, 0, 0, 0.123);
		height: calc(var(--header-height) - 8px);
		margin-top: 8px;
		margin-right: 8px;
		border-radius: 8px 8px 0 0;
		border-bottom: 5px solid transparent;
		padding-top: 4px;
		opacity: 0.7;
	}
	._2._3 {
		background-color: rgba(0, 0, 0, 0.2);
		border-bottom: 2px solid white;
		border-bottom: 5px solid #2b2b4c;
		font-family: semibold;
		opacity: 1;
	}

	/* LOADING BAR */
	@keyframes MOVE-BG {
		from {
			transform: translateX(0);
		}

		to {
			transform: translateX(46px);
		}
	}
</style>

