<script lang="ts">
	import { browser } from '$app/environment';
	import { Core } from '$core/store.svelte';
import { webRTCManager, useWebRTC } from '$libs/webrtc/manager';

	// Click outside action
	function clickOutside(node: HTMLElement, callback: () => void) {
		const handleClick = (event: MouseEvent) => {
			if (node && !node.contains(event.target as HTMLElement) && !event.defaultPrevented) {
				callback();
			}
		};
		document.addEventListener('click', handleClick, true);
		return {
			destroy() {
				document.removeEventListener('click', handleClick, true);
			}
		};
	}

	// Use the WebRTC manager
	const webRTC = useWebRTC();

	// Subscribe to state changes
	let unsubscribe = () => {};

	// Local state for UI
	let showDetails = $state(false);
	let isHovering = $state(false);

	if (browser) {
		unsubscribe = webRTC.subscribe(() => {
			// State changes will be reactive through Core store
		});
	}

	// Derived status
	const status = $derived.by(() => {
		if (Core.webRTCConnected) return 'connected';
		if (Core.webRTCConnecting) return 'connecting';
		return 'disconnected';
	});

	// Status configuration
	const statusConfig = $derived.by(() => {
		switch (status) {
			case 'connected':
				return {
					color: 'bg-green-500',
					text: 'Connected',
					subtext: 'P2P tunnel active',
					icon: 'icon-check-circle',
					pulse: false
				};
			case 'connecting':
				return {
					color: 'bg-yellow-500',
					text: 'Connecting...',
					subtext: 'Establishing P2P tunnel',
					icon: 'icon-loader',
					pulse: true
				};
			case 'disconnected':
				return {
					color: 'bg-red-500',
					text: 'Disconnected',
					subtext: Core.webRTCError || 'Connection lost',
					icon: 'icon-x-circle',
					pulse: false
				};
		}
	});

	// Handle reconnect click
	function handleReconnect() {
		if (!Core.webRTCConnected && !Core.webRTCConnecting) {
			webRTC.reconnect();
		}
	}

	// Cleanup on destroy
	$effect(() => {
		return () => {
			unsubscribe();
		};
	});
</script>

<div class="inline-flex items-center gap-2">
	<!-- Status indicator button -->
	<button
		class="relative flex items-center gap-2 px-3 py-1.5 rounded-lg transition-all duration-200 hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer group"
		onmouseenter={() => isHovering = true}
		onmouseleave={() => isHovering = false}
		onclick={() => showDetails = !showDetails}
	>
		<!-- Status dot -->
		<div class="relative">
			<div class="w-3 h-3 rounded-full {statusConfig.color} {statusConfig.pulse ? 'animate-pulse' : ''}"></div>
			{#if statusConfig.pulse}
				<div class="absolute inset-0 w-3 h-3 rounded-full {statusConfig.color} animate-ping opacity-75"></div>
			{/if}
		</div>

		<!-- Status text -->
		<span class="text-sm font-medium text-gray-700 dark:text-gray-300">
			{statusConfig.text}
		</span>

		<!-- Chevron icon -->
		<svg
			class="w-4 h-4 text-gray-500 dark:text-gray-400 transition-transform duration-200 {showDetails ? 'rotate-180' : ''}"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	<!-- Details dropdown -->
	{#if showDetails}
		<div class="absolute top-full right-0 mt-2 w-72 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50">
			<div class="p-4">
				<!-- Header -->
				<div class="flex items-center gap-3 mb-3">
					<div class="w-8 h-8 rounded-full {statusConfig.color} {statusConfig.pulse ? 'animate-pulse' : ''} flex items-center justify-center">
						<svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							{#if status === 'connected'}
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
							{:else if status === 'connecting'}
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
							{:else}
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							{/if}
						</svg>
					</div>
					<div>
						<h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
							P2P Connection
						</h3>
						<p class="text-xs text-gray-500 dark:text-gray-400">
							{statusConfig.subtext}
						</p>
					</div>
				</div>

				<!-- Connection info -->
				<div class="space-y-2 text-sm">
					<div class="flex justify-between">
						<span class="text-gray-600 dark:text-gray-400">Status:</span>
						<span class="font-medium text-gray-900 dark:text-gray-100">
							{statusConfig.text}
						</span>
					</div>

					{#if Core.webRTCReconnectAttempts > 0}
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Reconnect attempts:</span>
							<span class="font-medium text-gray-900 dark:text-gray-100">
								{Core.webRTCReconnectAttempts}/5
							</span>
						</div>
					{/if}

					{#if Core.webRTCError}
						<div class="flex flex-col gap-1">
							<span class="text-gray-600 dark:text-gray-400">Error:</span>
							<span class="text-red-600 dark:text-red-400 text-xs break-all">
								{Core.webRTCError}
							</span>
						</div>
					{/if}
				</div>

				<!-- Action button -->
				{#if status === 'disconnected'}
					<button
						onclick={() => {
							handleReconnect();
							showDetails = false;
						}}
						class="w-full mt-4 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium rounded-lg transition-colors duration-200"
					>
						Reconnect
					</button>
				{:else if status === 'connected'}
					<button
						onclick={() => {
							webRTC.disconnect();
							showDetails = false;
						}}
						class="w-full mt-4 px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 text-sm font-medium rounded-lg transition-colors duration-200"
					>
						Disconnect
					</button>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Click outside to close dropdown -->
	<div use:clickOutside={() => showDetails = false}></div>
</div>

<style>
	/* Prevent text selection on click */
	button {
		user-select: none;
		-webkit-user-select: none;
	}
</style>
