<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { Core } from '../core/store.svelte';
	import type { IMenuRecord, IModule } from '../types/menu';

	// State
	const module = $derived(Core.module);
	let menuOpen = $state<[number, string]>([0, '']);
	let currentPathname = $state('');

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
<div class="d-menu fixed left-0 top-0 h-screen bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950 
		text-white shadow-xl transition-all duration-200 ease-in-out z-50 hidden md:block"
	role="navigation"
	aria-label="Main navigation"
>
	<div class="flex items-center h-48 border-b border-gray-800/30 w-full">
		<div class="_1 flex items-center">
			<img class="w-42 h-42" src="/images/genix_logo4.svg" alt="">
			<div class="_2 hidden white ff-bold h2 ml-[-3px]">enix</div>
		</div>
	</div>

	<!-- Menu Items -->
	<div class="flex-1 transition-all duration-300 w-full">
		{#each module.menus as menu}
			{@const isOpen = menuOpen[0] === menu.id}
			{@const optionsCount = menu.options?.length || 0}
			{@const maxHeight = isOpen ? `${optionsCount * 2.8 + 3}rem` : '3rem'}

			<div class="overflow-hidden transition-all duration-400 mb-1"
				style="max-height: {maxHeight}"
			>
				<!-- Menu Header -->
				<button class="w-full h-48 px-12 flex items-center justify-between relative
						text-indigo-300 hover:bg-gray-800/50 transition-colors duration-400
						border-l-4 border-transparent hover:border-indigo-500
						{isOpen ? 'bg-gray-800 border-indigo-500' : ''}"
					onclick={() => toggleMenu(menu.id || 0)}
					aria-expanded={isOpen}
				>
					<div class="flex items-center flex-1 min-w-0 whitespace-nowrap">
						<!-- Minimized view - show only minName -->
						<span class="menu-minimized font-mono font-semibold ml-1">
							{(menu.minName || menu.name.substring(0, 3)).toUpperCase()}
						</span>
						<!-- Expanded view - show full name -->
						<span class="menu-expanded font-mono font-semibold ml-1 tracking-wider">
							{menu.name.toUpperCase()}
						</span>
					</div>

					<!-- Arrow icon (only visible when expanded) -->
					{#if menu.options && menu.options.length > 0}
						<span
							class="menu-arrow absolute right-3 transition-all duration-300"
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
							<button class="submenu-option w-full flex items-center px-0 py-10 relative
								hover:bg-indigo-600/20 transition-all duration-150
								border-l-2 border-transparent
								{isActive ? 'bg-indigo-600/30 border-indigo-400 text-white' : 'text-gray-300'}"
								onclick={() => navigateTo(option.route || '/', menu.id || 0)}
							>
								<!-- Minimized: show icon only centered -->
								<div class="option-minimized flex w-full">
									<i class="{option.icon || "icon-box"} mr-2"></i>
									<div class="font-mono">
										{option.minName || option.name.substring(0, 2)}
									</div>
								</div>
								
								<!-- Expanded: show icon and full name -->
								<div class="option-expanded flex w-full">
									<i class="{option.icon || "icon-box"} mr-2"></i>
									<div class="font-mono">
										{#each option.name.split(' ') as word}
											<span class="mr-4">{word}</span>
										{/each}
									</div>
								</div>

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
	</div>
</div>

<!-- Mobile Menu Toggle (hidden checkbox for CSS-only control) -->
<input type="checkbox" id="mobile-menu-toggle" class="mobile-menu-checkbox" />

<!-- Mobile Menu -->
<div class="mobile-menu-wrapper md:hidden" role="dialog" aria-modal="true">
	<!-- Backdrop -->
	<label for="mobile-menu-toggle" class="mobile-menu-backdrop" aria-label="Close menu"></label>

	<!-- Mobile Menu Panel -->
	<aside class="mobile-menu-panel">
		<!-- Mobile Header -->
		<div class="h-48 flex items-center justify-between px-16 border-b border-gray-800/50">
			<span class="text-lg font-bold tracking-wider text-indigo-400">GENIX</span>
			<label
				for="mobile-menu-toggle"
				class="p-8 hover:bg-gray-800 rounded-lg transition-colors cursor-pointer"
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
			</label>
		</div>

		<!-- Mobile Menu Items -->
		<div class="py-4">
			{#each (module?.menus||[]) as menu}
				{@const isOpen = menuOpen[0] === menu.id}
				{@const optionsCount = menu.options?.length || 0}
				{@const maxHeight = isOpen ? `${optionsCount * 2.8 + 3}rem` : '3rem'}

				<div class="mb-1 overflow-hidden transition-all duration-300"
					style="max-height: {maxHeight}"
				>
					<button	class="w-full h-48 px-16 flex items-center justify-between
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
						<div class="transition-all duration-300">
							{#each menu.options as option}
								{@const isActive = option.route === currentPathname}
								<button	class="w-full flex items-center px-24 py-10 text-sm relative
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
		</div>
	</aside>
</div>

<style>
	.d-menu:hover {

	}
	._1 {
		position: absolute;
		left: 12px;
	}
	.d-menu:hover ._2 {
		display: block;
	}
	/* Desktop Menu - Pure CSS Width Control */
	.d-menu {
		width: var(--menu-min-width);
		overflow: hidden;
	}
	
	.d-menu:hover {
		width: var(--menu-max-width);
		overflow: auto;
	}

	/* Menu Text - Show minimized by default, expanded on hover */
	.menu-minimized {
		display: block;
	}
	
	.menu-expanded {
		display: none;
	}
	
	.d-menu:hover .menu-minimized {
		display: none;
	}
	
	.d-menu:hover .menu-expanded {
		display: block;
	}

	/* Menu Arrow - Hidden by default, visible on hover */
	.menu-arrow {
		opacity: 0;
	}
	
	.d-menu:hover .menu-arrow {
		opacity: 1;
	}

	/* Submenu Options - Show minimized by default, expanded on hover */
	.option-minimized {
		display: flex;
	}
	
	.option-expanded {
		display: none;
	}

	.option-minimized, .option-expanded {
		position: absolute;
		left: 6px;
		align-items: center;
	}

	.option-minimized > div, .option-expanded > div {
		margin-bottom: -2px;
		font-size: 15px;
	}
	
	.d-menu:hover .option-minimized {
		display: none;
	}
	
	.d-menu:hover .option-expanded {
		display: flex;
	}

	/* Submenu padding adjustment */
	.submenu-option {
		height: 38px;
	}
	
	.d-menu:hover .submenu-option {
		padding-left: 8px;
		padding-right: 8px;
	}

	/* Pure CSS Mobile Menu Control */
	.mobile-menu-checkbox {
		display: none;
	}

	.mobile-menu-wrapper {
		position: fixed;
		inset: 0;
		z-index: 50;
		pointer-events: none;
		visibility: hidden;
	}

	.mobile-menu-backdrop {
		position: absolute;
		inset: 0;
		background-color: rgb(0 0 0 / 0.6);
		backdrop-filter: blur(4px);
		opacity: 0;
		transition: opacity 0.3s ease-out;
		cursor: pointer;
	}

	.mobile-menu-panel {
		position: absolute;
		left: 0;
		top: 0;
		height: 100%;
		width: 14rem;
		max-width: 75vw;
		background: linear-gradient(to bottom, rgb(17 24 39), rgb(17 24 39), rgb(3 7 18));
		color: white;
		box-shadow: 0 25px 50px -12px rgb(0 0 0 / 0.25);
		overflow-y: auto;
		transform: translateX(-100%);
		transition: transform 0.3s ease-out;
	}

	/* When checkbox is checked, show the mobile menu */
	.mobile-menu-checkbox:checked ~ .mobile-menu-wrapper {
		pointer-events: auto;
		visibility: visible;
	}

	.mobile-menu-checkbox:checked ~ .mobile-menu-wrapper .mobile-menu-backdrop {
		opacity: 1;
	}

	.mobile-menu-checkbox:checked ~ .mobile-menu-wrapper .mobile-menu-panel {
		transform: translateX(0);
	}
</style>
