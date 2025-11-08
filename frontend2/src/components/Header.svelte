<script lang="ts">
    import { fetchOnCourse } from "../core/store.svelte";

	// Props
	let {
		title = 'Genix',
		showMenuButton = false
	}: {
		title?: string;
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

	function handleLogout() {
		// Clear session/tokens
		localStorage.clear();
		sessionStorage.clear();
		window.location.href = '/login';
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
	shadow-md z-40 flex items-center px-16"
>
	<!-- Logo Section (Desktop) -->
	<div class="hidden md:flex items-center justify-center h-full w-56 mr-12">
		<div class="h-40 w-40 bg-black/20 rounded-lg flex items-center justify-center">
			<img src="/images/genix_logo4.svg" alt="Genix Logo" class="w-full h-full p-1" />
		</div>
	</div>

	<!-- Mobile Menu Button -->
	{#if showMenuButton}
		<label
			for="mobile-menu-toggle"
			class="md:hidden p-8 hover:bg-white/10 rounded-lg transition-colors mr-12 cursor-pointer"
			aria-label="Toggle menu"
		>
			<span class="text-white text-2xl">☰</span>
		</label>
	{/if}

	<!-- Title -->
	<div class="flex-1 flex items-center">
		<h1 class="text-white text-lg font-semibold tracking-wide">
			{title}
		</h1>
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
			<button
				class="w-40 h-40 rounded-full bg-white/10 hover:bg-white/20 
					flex items-center justify-center transition-colors shadow-sm"
				onclick={toggleSettings}
				aria-label="Settings"
				aria-expanded={showSettings}
			>
				<span class="text-white text-lg">⚙️</span>
			</button>

			{#if showSettings}
				<!-- Settings Dropdown Panel -->
				<div
					class="absolute right-0 top-48 mt-4 w-192 bg-white dark:bg-gray-800 
						rounded-lg shadow-xl border border-gray-200 dark:border-gray-700 
						py-8 z-50"
				>
					<!-- Theme Section -->
					<div class="px-12 py-8 border-b border-gray-200 dark:border-gray-700">
						<p class="text-xs text-gray-500 dark:text-gray-400 mb-8">Tema</p>
						<div class="flex gap-8">
							<button
								class="flex-1 px-12 py-6 rounded-md text-sm transition-colors
									{uiTheme === 'light'
										? 'bg-indigo-600 text-white'
										: 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'}"
								onclick={() => toggleTheme('light')}
							>
								Claro
							</button>
							<button
								class="flex-1 px-12 py-6 rounded-md text-sm transition-colors
									{uiTheme === 'dark'
										? 'bg-indigo-600 text-white'
										: 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'}"
								onclick={() => toggleTheme('dark')}
							>
								Oscuro
							</button>
						</div>
					</div>

					<!-- Actions -->
					<button
						class="w-full px-12 py-8 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 
							flex items-center gap-8 text-gray-700 dark:text-gray-200"
						onclick={handleLogout}
					>
						<span>🚪</span>
						<span>Salir</span>
					</button>
				</div>
			{/if}
		</div>

		<!-- Reload Button -->
		<button
			class="w-40 h-40 rounded-full bg-white/10 hover:bg-white/20 
				flex items-center justify-center transition-colors shadow-sm
				{isReloading ? 'animate-spin' : ''}"
			onclick={handleReload}
			aria-label="Reload"
			disabled={isReloading}
		>
			<span class="text-white text-lg">🔄</span>
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
