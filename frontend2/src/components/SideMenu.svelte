<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { Core } from '../core/store.svelte';
	import type { IMenuRecord, IModule } from '../types/menu';

	// Props
	type Props = {
		isMobileOpen?: boolean;
	};

	let { isMobileOpen = $bindable(false) }: Props = $props();

	// State
	const module = $derived(Core.module);
	let menuOpen = $state<[number, string]>([0, '']);
	let isMenuHover = $state(false);
	let currentPathname = $state('');

	// Computed - menu is always minimized by default (4.5rem), expands on hover (14rem)
	let menuWidthClass = $derived(isMenuHover ? 'w-56' : 'w-18');

	// Functions
	function getMenuOpenFromRoute(mod: IModule, pathname: string): [number, string] {
		for (let menu of mod.menus) {
			for (let opt of menu.options || []) {
				if (opt.route === pathname) {
					return [menu.id || 0, opt.route];
				}
			}
		}
		return [0, ''];
	}

	function toggleMenu(menuId: number) {
		if (menuOpen[0] === menuId) {
			menuOpen = [0, ''];
		} else {
			menuOpen = [menuId, menuOpen[1]];
		}
	}

	function navigateTo(route: string, menuId: number) {
		menuOpen = [menuId, route];
		goto(route);
		if (isMobileOpen) {
			isMobileOpen = false;
		}
	}

	// Lifecycle
	$effect(() => {
		if ($page?.url?.pathname) {
			currentPathname = $page.url.pathname;
			if (module) {
				menuOpen = getMenuOpenFromRoute(module, currentPathname);
			}
		}
	});
</script>

<!-- Desktop Menu -->
<div
	class="fixed left-0 top-0 h-screen bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950 
		text-white shadow-xl transition-all duration-300 ease-in-out z-50
		{menuWidthClass} hidden md:block"
	style="overflow: {isMenuHover ? 'auto' : 'hidden'}"
	onmouseenter={() => (isMenuHover = true)}
	onmouseleave={() => (isMenuHover = false)}
	role="navigation"
	aria-label="Main navigation"
>
	<!-- Header spacer (3rem height) -->
	<div class="h-12 border-b border-gray-800/30"></div>

	<!-- Menu Items -->
	<div class="flex-1 transition-all duration-300" style="min-width: {isMenuHover ? '14rem' : '4.5rem'}">
		{#if module?.menus}
			{#each module.menus as menu}
				{@const isOpen = menuOpen[0] === menu.id}
				{@const optionsCount = menu.options?.length || 0}
				{@const maxHeight = isOpen ? `${optionsCount * 2.8 + 3}rem` : '3rem'}

				<div
					class="overflow-hidden transition-all duration-300 ease-in-out mb-1"
					style="max-height: {maxHeight}"
				>
					<!-- Menu Header -->
					<button
						class="w-full h-12 px-3 flex items-center justify-between relative
							text-indigo-300 hover:bg-gray-800/50 transition-colors duration-200
							border-l-4 border-transparent hover:border-indigo-500
							{isOpen ? 'bg-gray-800 border-indigo-500' : ''}"
						onclick={() => toggleMenu(menu.id || 0)}
						aria-expanded={isOpen}
					>
						<div class="flex items-center flex-1 min-w-0">
							{#if !isMenuHover}
								<!-- Minimized view - show only minName -->
								<span class="text-xs font-mono font-semibold ml-1">
									{menu.minName || menu.name.substring(0, 3).toUpperCase()}
								</span>
							{:else}
								<!-- Expanded view - show full name -->
								<span class="text-sm font-mono font-semibold ml-1 tracking-wider">
									{menu.name.toUpperCase()}
								</span>
							{/if}
						</div>

						<!-- Arrow icon (only visible when expanded) -->
						{#if menu.options && menu.options.length > 0}
							<span
								class="absolute right-3 transition-all duration-300 text-xs"
								style="opacity: {isMenuHover ? 1 : 0}"
								class:rotate-180={isOpen}
							>
								<i class="icon-down-open-1"></i>
							</span>
						{/if}
					</button>

					<!-- Submenu Options -->
					{#if menu.options && isOpen}
						<div class="transition-all duration-200">
							{#each menu.options as option}
								{@const isActive = option.route === currentPathname}
								<button
									class="w-full flex items-center py-2.5 text-sm relative
										hover:bg-indigo-600/20 transition-all duration-150
										border-l-2 border-transparent
										{isActive ? 'bg-indigo-600/30 border-indigo-400 text-white' : 'text-gray-300'}"
									style="padding-left: {isMenuHover ? '1.5rem' : '0'}; padding-right: {isMenuHover
										? '1.5rem'
										: '0'}"
									onclick={() => navigateTo(option.route || '/', menu.id || 0)}
								>
									{#if !isMenuHover}
										<!-- Minimized: show icon only centered -->
										<div class="w-full flex justify-center">
											{#if option.icon}
												<i class="{option.icon} text-lg"></i>
											{:else}
												<span class="text-xs font-mono">
													{option.minName || option.name.substring(0, 2)}
												</span>
											{/if}
										</div>
									{:else}
										<!-- Expanded: show icon and full name -->
										<div class="flex items-center w-full">
											{#if option.icon}
												<i class="{option.icon} text-base mr-2"></i>
											{/if}
											<span class="font-mono text-xs">
												{#each option.name.split(' ') as word}
													<span class="mr-1">{word}</span>
												{/each}
											</span>
										</div>
									{/if}

									<!-- Active indicator -->
									{#if isActive}
										<div
											class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-3/4 bg-indigo-400 rounded-r"
										></div>
									{/if}
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/each}
		{/if}
	</div>
</div>

<!-- Mobile Menu -->
{#if isMobileOpen}
	<div class="fixed inset-0 z-50 md:hidden" role="dialog" aria-modal="true">
		<!-- Backdrop -->
		<div
			class="absolute inset-0 bg-black/60 backdrop-blur-sm"
			onclick={() => (isMobileOpen = false)}
			onkeydown={(e) => e.key === 'Escape' && (isMobileOpen = false)}
			role="button"
			tabindex="0"
			aria-label="Close menu"
		></div>

		<!-- Mobile Menu Panel -->
		<aside
			class="absolute left-0 top-0 h-full w-56 max-w-[75vw]
				bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950 
				text-white shadow-2xl overflow-y-auto animate-slide-in"
		>
			<!-- Mobile Header -->
			<div class="h-12 flex items-center justify-between px-4 border-b border-gray-800/50">
				<span class="text-lg font-bold tracking-wider text-indigo-400">GENIX</span>
				<button
					class="p-2 hover:bg-gray-800 rounded-lg transition-colors"
					onclick={() => (isMobileOpen = false)}
					aria-label="Close menu"
				>
					<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			</div>

			<!-- Mobile Menu Items -->
			<div class="py-4">
				{#if module?.menus}
					{#each module.menus as menu}
						{@const isOpen = menuOpen[0] === menu.id}
						{@const optionsCount = menu.options?.length || 0}
						{@const maxHeight = isOpen ? `${optionsCount * 2.8 + 3}rem` : '3rem'}

						<div
							class="mb-1 overflow-hidden transition-all duration-300"
							style="max-height: {maxHeight}"
						>
							<button
								class="w-full h-12 px-4 flex items-center justify-between
									text-indigo-300 hover:bg-gray-800/50 transition-colors
									border-l-4 border-transparent hover:border-indigo-500 relative
									{isOpen ? 'bg-gray-800 border-indigo-500' : ''}"
								onclick={() => toggleMenu(menu.id || 0)}
							>
								<span class="text-sm font-mono font-semibold tracking-wider">
									{menu.name.toUpperCase()}
								</span>
								{#if menu.options && menu.options.length > 0}
									<span class="transition-transform duration-300 text-xs" class:rotate-180={isOpen}>
										<i class="icon-down-open-1"></i>
									</span>
								{/if}
							</button>

							{#if menu.options && isOpen}
								<div class="transition-all duration-200">
									{#each menu.options as option}
										{@const isActive = option.route === currentPathname}
										<button
											class="w-full flex items-center px-6 py-2.5 text-sm relative
												hover:bg-indigo-600/20 transition-all
												border-l-2 border-transparent
												{isActive
													? 'bg-indigo-600/30 border-indigo-400 text-white font-medium'
													: 'text-gray-300'}"
											onclick={() => navigateTo(option.route || '/', menu.id || 0)}
										>
											<div class="flex items-center w-full">
												{#if option.icon}
													<i class="{option.icon} text-base mr-2"></i>
												{/if}
												<span class="font-mono text-xs">
													{#each option.name.split(' ') as word}
														<span class="mr-1">{word}</span>
													{/each}
												</span>
											</div>

											{#if isActive}
												<div
													class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-3/4 bg-indigo-400 rounded-r"
												></div>
											{/if}
										</button>
									{/each}
								</div>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		</aside>
	</div>
{/if}

<style>
	/* Mobile slide animation */
	@keyframes slide-in {
		from {
			transform: translateX(-100%);
			opacity: 0.5;
		}
		to {
			transform: translateX(0);
			opacity: 1;
		}
	}

	:global(.animate-slide-in) {
		animation: slide-in 0.3s ease-out;
	}
</style>
