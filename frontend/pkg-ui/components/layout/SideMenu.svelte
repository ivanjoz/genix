<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
import { Core } from '$core/core/store.svelte';
import { IMenuRecord, IModule } from '$core/types/menu';

	// State
	const module = $derived(Core.module);
	let menuOpen = $state<[number, string]>([0, '']);
	let currentPathname = $state('');
	let mobileMenuPanel: HTMLElement;
	let mobileMenuBackdrop: HTMLElement;

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

	function applyActiveStylesInstant(button: HTMLElement) {
		// Find and remove active state from the currently active button (only one can be active)
		const activeButton = mobileMenuPanel?.querySelector('.mobile-menu-option.is-active');
		if (activeButton) {
			activeButton.classList.remove('is-active');
		}
		
		// Apply active state to clicked button
		button.classList.add('is-active');
	}

	async function navigateTo(route: string, menuId: number, buttonElement?: HTMLElement) {
		
		// On mobile: Apply styles instantly to clicked button before view transition
		if (Core.mobileMenuOpen && buttonElement) {
			applyActiveStylesInstant(buttonElement);
		}
		
		goto(route);

		if (Core.mobileMenuOpen) {
			toggleMobileMenu(true);
		}

		menuOpen = [menuId, route];
	}

	// Animation duration in milliseconds - should match CSS animation duration
	const ANIMATION_DURATION = 350;

	const toggleMobileMenu = (close?: boolean) => {
		if(!mobileMenuPanel){ return }
		mobileMenuPanel.style.setProperty("view-transition-name", "mobile-side-menu")

		setTimeout(() => {
			mobileMenuPanel.style.setProperty("view-transition-name", "")
		}, ANIMATION_DURATION)

		if (Core.mobileMenuOpen || close) {
			if (document.startViewTransition) {
				document.startViewTransition(() => {
					Core.mobileMenuOpen = 0;
				});
			} else {
				Core.mobileMenuOpen = 0;
			}
		} else {
			if (document.startViewTransition) {
				document.startViewTransition(() => {
					Core.mobileMenuOpen = 1;
				});
			} else {
				Core.mobileMenuOpen = 1;
			}
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

	// Register the toggle function in the Core store
	$effect(() => {
		Core.toggleMobileMenu = toggleMobileMenu;
	});
</script>

<!-- Desktop Menu -->
<div class="d-menu fixed left-0 top-0 h-screen bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950 
		text-white shadow-xl transition-all duration-200 ease-in-out z-300 hidden md:block"
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
			{@const menuHeight = isOpen ? `${optionsCount * 38 + 48}px` : '48px'}

			<div class="overflow-hidden transition-all duration-400 mb-1"
				style="height: {menuHeight}"
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

<!-- Mobile Menu -->
<div class="mobile-menu-wrapper md:hidden {Core.mobileMenuOpen ? 'is-open' : ''}" role="dialog" aria-modal="true">
	<!-- Backdrop -->
	<button type="button" class="mobile-menu-backdrop" aria-label="Close menu" 
		onclick={() => toggleMobileMenu(true)} bind:this={mobileMenuBackdrop}></button>

	<!-- Mobile Menu Panel -->
	<aside class="mobile-menu-panel" bind:this={mobileMenuPanel}>
		<!-- Mobile Header -->
		<div class="mobile-header h-48 flex items-center px-6 justify-between">
			<div class="mobile-header-logo">
				<img src="/images/genix_logo3.svg" alt="Genix" class="size-36" />
				<span class="logo-text">GENIX</span>
			</div>
			<button
				class="close-button size-32"
				aria-label="Close menu"
				onclick={() => toggleMobileMenu(true)}
			>
				<i class="icon-cancel"></i>
			</button>
		</div>

		<!-- Mobile Menu Items -->
		<div class="mobile-menu-content">
			{#each (module?.menus||[]) as menu}
				{@const isOpen = menuOpen[0] === menu.id}
				
				<div class="mobile-menu-group">
					<button	
						class="mobile-menu-group-button {isOpen ? 'is-open' : ''}"
						onclick={() => toggleMenu(menu.id || 0)}
					>
						<span class="menu-group-title">{menu.name.toUpperCase()}</span>
						{#if menu.options && menu.options.length > 0}
							<span class="menu-group-chevron" class:rotated={isOpen}>
								<i class="icon-down-open-1"></i>
							</span>
						{/if}
					</button>

					{#if menu.options && isOpen}
						<div class="mobile-menu-options-grid">
							{#each menu.options as option}
								{@const isActive = option.route === currentPathname}
								<button	
									class="mobile-menu-option {isActive ? 'is-active' : ''}"
									onclick={(e) => navigateTo(option.route || '/', menu.id || 0, e.currentTarget)}
									onmousedown={(e) => e.currentTarget.classList.add('is-pressed')}
									onmouseup={(e) => e.currentTarget.classList.remove('is-pressed')}
									onmouseleave={(e) => e.currentTarget.classList.remove('is-pressed')}
									ontouchstart={(e) => e.currentTarget.classList.add('is-pressed')}
									ontouchend={(e) => e.currentTarget.classList.remove('is-pressed')}
								>
									{#if option.icon}
										<i class="{option.icon} option-icon"></i>
									{/if}
									<span class="option-text">{option.name}</span>
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

	/* Mobile Menu Control */
	.mobile-menu-wrapper {
		position: fixed;
		inset: 0;
		z-index: 301;
		pointer-events: none;
	}

	.mobile-menu-backdrop {
		position: absolute;
		inset: 0;
		background-color: rgb(0 0 0 / 0.5);
		backdrop-filter: blur(2px);
		opacity: 0;
		cursor: pointer;
		border: none;
		padding: 0;
		z-index: 1;
		pointer-events: none;
		transition: opacity 0.35s ease-in-out;
	}

	.mobile-menu-wrapper.is-open .mobile-menu-backdrop {
		pointer-events: all;
		opacity: 1;
	}

	.mobile-menu-panel {
		position: absolute;
		left: 0;
		top: 0;
		height: 100%;
		width: 78vw;
		background: white;
		color: #2c2b2e;
		box-shadow: 4px 0 24px rgba(0, 0, 0, 0.15);
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		/* Don't transform by default - let view transitions handle it */
		opacity: 0;
		z-index: 301;
	}

	/* When mobile menu is open */
	.mobile-menu-wrapper.is-open {
		pointer-events: auto;
	}

	.mobile-menu-wrapper.is-open .mobile-menu-panel {
		opacity: 1;
		z-index: 2;
	}

	/* Mobile Header */
	.mobile-header {
		border-bottom: 1px solid #e5e7eb;
		background: white;
	}

	.mobile-header-logo {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.logo-text {
		font-size: 18px;
		font-weight: 700;
		font-family: bold;
		color: #1f2937;
		letter-spacing: 0.5px;
	}

	.close-button {
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 50%;
		background: #f3f4f6;
		color: #374151;
		border: none;
		cursor: pointer;
		transition: all 0.2s;
	}

	.close-button:hover {
		background: #e5e7eb;
	}

	.close-button svg {
		width: 16px;
		height: 16px;
	}

	/* Mobile Menu Content */
	.mobile-menu-content {
		flex: 1;
		padding: 4px 0;
		overflow-y: auto;
	}

	.mobile-menu-group {
		margin-bottom: 2px;
	}

	.mobile-menu-group-button {
		width: 100%;
		padding: 10px 12px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: white;
		border: none;
		border-left: 4px solid transparent;
		cursor: pointer;
		transition: all 0.2s;
		text-align: left;
	}

	.mobile-menu-group-button:hover {
		background: #f9fafb;
	}

	.mobile-menu-group-button.is-open {
		background: #f3f4f7;
		border-left-color: #8b5cf6;
	}

	.menu-group-title {
		font-size: 15px;
		font-weight: 600;
		font-family: semibold;
		color: #1f2937;
		letter-spacing: 0.3px;
		text-transform: uppercase;
	}

	.menu-group-chevron {
		color: #4b5563;
		font-size: 14px;
		transition: transform 0.3s ease;
		display: flex;
		align-items: center;
	}

	.menu-group-chevron.rotated {
		transform: rotate(180deg);
	}

	/* Options Grid */
	.mobile-menu-options-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 6px;
		padding: 6px 10px 12px 10px;
		background: white;
		animation: fadeIn 0.25s ease-out;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.mobile-menu-option {
    padding: 4px 8px 6px 8px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    background: white;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    cursor: pointer;
    min-height: 70px;
    justify-content: center;
	}

	.mobile-menu-option:hover:not(.is-active) {
		background: #f9fafb;
		border-color: #9ca3af;
		transform: translateY(-1px);
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
		transition: transform 0.2s, box-shadow 0.2s, background 0.2s, border-color 0.2s;
	}

	.mobile-menu-option.is-pressed:not(.is-active) {
		background: #e5e7eb;
		border-color: #9ca3af;
		transform: translateY(0);
		box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.1);
		transition: transform 0.2s, box-shadow 0.2s, background 0.2s, border-color 0.2s;
	}

	.mobile-menu-option.is-active {
		background: #ede9fe !important;
		border-color: #8b5cf6 !important;
		box-shadow: 0 2px 6px rgba(139, 92, 246, 0.25) !important;
		transition: none !important;
	}

	.mobile-menu-option.is-active.is-pressed {
		background: #ddd6fe !important;
		border-color: #7c3aed !important;
		box-shadow: inset 0 2px 4px rgba(124, 58, 237, 0.2) !important;
		transition: none !important;
	}

	.option-icon {
		font-size: 20px;
		color: #4b5563;
	 /*	margin-bottom: 2px; */
	}

	.mobile-menu-option.is-active .option-icon {
		color: #7c3aed !important;
		transition: none !important;
	}

	.option-text {
		font-size: 15px;
		font-family: main;
		color: #1f2937;
		text-align: center;
		line-height: 1.1;
		word-break: break-word;
	}

	.mobile-menu-option.is-active .option-text {
		color: #5b21b6 !important;
		transition: none !important;
	}

	/* View Transitions for Mobile Menu */
	@keyframes slide-out {
		from {
			transform: translateX(0);
			opacity: 1;
		}
		to {
			transform: translateX(-100%);
			opacity: 1;
		}
	}

	@keyframes slide-in {
		from {
			transform: translateX(-100%);
		}
		to {
			transform: translateX(0);
		}
	}

	/* When closing: OLD snapshot slides out to the left */
	::view-transition-old(mobile-side-menu) {
		animation: slide-out 0.35s ease-in-out forwards;
		animation-fill-mode: forwards;
	}

	/* When opening: NEW snapshot slides in from the left */
	::view-transition-new(mobile-side-menu) {
		animation: slide-in 0.35s ease-in-out;
	}

	/* Prevent the default fade out on OLD snapshot */
	::view-transition-image-pair(mobile-side-menu) {
		isolation: auto;
	}
</style>
