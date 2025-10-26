<script lang="ts">
	// Props
	let {
		onMenuToggle,
		title = 'Genix',
		showMenuButton = false
	}: {
		onMenuToggle?: () => void;
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

<header
	class="fixed top-0 left-0 right-0 h-12 bg-gradient-to-r from-indigo-600 to-indigo-700 
		shadow-md z-40 flex items-center px-4"
>
	<!-- Logo Section (Desktop) -->
	<div class="hidden md:flex items-center justify-center h-full w-56 mr-3">
		<div class="h-10 w-10 bg-black/20 rounded-lg flex items-center justify-center">
			<img src="/images/genix_logo4.svg" alt="Genix Logo" class="w-full h-full p-1" />
		</div>
	</div>

	<!-- Mobile Menu Button -->
	{#if showMenuButton}
		<button
			class="md:hidden p-2 hover:bg-white/10 rounded-lg transition-colors mr-3"
			onclick={onMenuToggle}
			aria-label="Toggle menu"
		>
			<span class="text-white text-2xl">☰</span>
		</button>
	{/if}

	<!-- Title -->
	<div class="flex-1 flex items-center">
		<h1 class="text-white text-lg font-semibold tracking-wide">
			{title}
		</h1>
	</div>

	<!-- Right Actions -->
	<div class="flex items-center gap-2">
		<!-- Loading Indicator (placeholder for future implementation) -->
		<!-- 
		{#if isLoading}
			<div class="px-3 py-1 bg-black/20 rounded-lg flex items-center gap-2">
				<div class="w-2 h-2 bg-yellow-400 rounded-full animate-pulse"></div>
				<span class="text-white text-sm">Cargando...</span>
			</div>
		{/if}
		-->

		<!-- Settings Dropdown -->
		<div class="relative">
			<button
				class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 
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
					class="absolute right-0 top-12 mt-1 w-48 bg-white dark:bg-gray-800 
						rounded-lg shadow-xl border border-gray-200 dark:border-gray-700 
						py-2 z-50"
				>
					<!-- Theme Section -->
					<div class="px-3 py-2 border-b border-gray-200 dark:border-gray-700">
						<p class="text-xs text-gray-500 dark:text-gray-400 mb-2">Tema</p>
						<div class="flex gap-2">
							<button
								class="flex-1 px-3 py-1.5 rounded-md text-sm transition-colors
									{uiTheme === 'light'
										? 'bg-indigo-600 text-white'
										: 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'}"
								onclick={() => toggleTheme('light')}
							>
								Claro
							</button>
							<button
								class="flex-1 px-3 py-1.5 rounded-md text-sm transition-colors
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
						class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 
							flex items-center gap-2 text-gray-700 dark:text-gray-200"
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
			class="w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 
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
</style>
