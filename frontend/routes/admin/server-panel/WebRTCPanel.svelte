<script lang="ts">
	import { browser } from '$app/environment';
	import { webRTCManager, useWebRTC } from '$core/lib/webrtc/manager';
	import { Core } from '$core/store.svelte';

	const webRTC = useWebRTC();

	// UI state
	let showAdvanced = $state(false);
	let showMessages = $state(true);
	let testMessageType = $state('ping');
	let customMessage = $state('');
	let sendInterval = $state(null as NodeJS.Timeout | null);
	let autoSendEnabled = $state(false);

	// Message history
	let messages = $state<Array<{
		id: number;
		direction: 'sent' | 'received';
		type: string;
		content: any;
		timestamp: Date;
	}>>([]);

	// Statistics
	let stats = $state({
		messagesSent: 0,
		messagesReceived: 0,
		lastPingTime: null as number | null,
		pingLatency: null as number | null
	});

	// Subscribe to messages
	let unsubscribe = () => {};

	if (browser) {
		unsubscribe = webRTC.onMessage('*', (message) => {
			handleIncomingMessage(message);
		});
	}

	// Handle incoming messages
	function handleIncomingMessage(message: any) {
		const now = new Date();
		let latency: number | null = null;

		// Try to extract latency from message
		if (message && typeof message === 'object') {
			if (message.type === 'pong' && message.originalTimestamp) {
				latency = now.getTime() - message.originalTimestamp;
				stats.pingLatency = latency;
			} else if (message.timestamp) {
				latency = now.getTime() - message.timestamp;
			}
		}

		// Handle "Echo: " prefix for text messages
		if (typeof message === 'string' && message.startsWith('Echo: ')) {
			try {
				const jsonStr = message.substring(6);
				const echoed = JSON.parse(jsonStr);
				if (echoed.timestamp) {
					latency = now.getTime() - echoed.timestamp;
				}
			} catch (e) {
				// Not JSON or no timestamp
			}
		}

		const msgType = typeof message === 'string' ? 'text' : (message?.type || 'unknown');

		messages = [
			{
				id: messages.length + 1,
				direction: 'received',
				type: msgType,
				content: message,
				timestamp: now,
				latency
			},
			...messages
		].slice(0, 100); // Keep last 100 messages

		stats.messagesReceived++;
	}

	// Send a test message
	function sendTestMessage(type: string = testMessageType, content?: any) {
		if (!Core.webRTCConnected) {
			console.warn('WebRTC not connected');
			return;
		}

		const now = new Date();
		const message = {
			timestamp: now.getTime(),
			...(content || {
				type,
				message: customMessage || `Test ${type} message`,
				testId: Math.random().toString(36).substr(2, 9)
			})
		};

		// Add to sent messages
		messages = [
			{
				id: messages.length + 1,
				direction: 'sent',
				type,
				content: message,
				timestamp: now,
				latency: null
			},
			...messages
		].slice(0, 100);

		stats.messagesSent++;
		webRTC.send(message);

		// For ping messages, track the time
		if (type === 'ping') {
			stats.lastPingTime = now.getTime();
		}
	}

	// Send JSON payload
	function sendJsonPayload() {
		try {
			const payload = JSON.parse(customMessage);
			sendTestMessage('json', payload);
		} catch (err) {
			alert('Invalid JSON: ' + err);
		}
	}

	// Toggle auto-send
	function toggleAutoSend() {
		autoSendEnabled = !autoSendEnabled;

		if (autoSendEnabled) {
			sendInterval = setInterval(() => {
				sendTestMessage('auto_ping');
			}, 5000);
		} else if (sendInterval) {
			clearInterval(sendInterval);
			sendInterval = null;
		}
	}

	// Clear messages
	function clearMessages() {
		messages = [];
		stats = {
			messagesSent: 0,
			messagesReceived: 0,
			lastPingTime: null,
			pingLatency: null
		};
	}

	// Reconnect
	function reconnect() {
		webRTC.reconnect();
	}

	// Disconnect
	function disconnect() {
		webRTC.disconnect();
	}

	// Cleanup
	$effect(() => {
		return () => {
			unsubscribe();
			if (sendInterval) {
				clearInterval(sendInterval);
			}
		};
	});

	// Derived values
	const statusColor = $derived.by(() => {
		if (Core.webRTCConnected) return 'text-green-600 bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800 dark:text-green-400';
		if (Core.webRTCConnecting) return 'text-yellow-600 bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800 dark:text-yellow-400';
		return 'text-red-600 bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800 dark:text-red-400';
	});

	const connectionStatus = $derived.by(() => {
		if (Core.webRTCConnected) return 'Connected';
		if (Core.webRTCConnecting) return 'Connecting...';
		return 'Disconnected';
	});

	const bridgeInfo = $derived(() => webRTCManager.getConnectionStatus());
</script>

<div class="p-6 space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-2xl font-bold text-gray-900 dark:text-gray-100">WebRTC Test Panel</h2>
			<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
				Test and debug P2P connection to home lab server
			</p>
		</div>

		<!-- Status Badge -->
		<div class="flex items-center gap-2 px-4 py-2 rounded-lg border {statusColor}">
			<div class="w-3 h-3 rounded-full {Core.webRTCConnected ? 'bg-green-500' : Core.webRTCConnecting ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'}"></div>
			<span class="font-semibold">{connectionStatus}</span>
		</div>
	</div>

	<!-- Connection Controls -->
	<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
		<div class="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
			<h3 class="font-semibold text-gray-900 dark:text-gray-100 mb-3">Connection Actions</h3>
			<div class="flex flex-wrap gap-2">
				{#if !Core.webRTCConnected && !Core.webRTCConnecting}
					<button
						onclick={reconnect}
						class="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
					>
						Connect
					</button>
				{:else if Core.webRTCConnected}
					<button
						onclick={disconnect}
						class="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg transition-colors"
					>
						Disconnect
					</button>
				{/if}

				<button
					onclick={() => showAdvanced = !showAdvanced}
					class="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg transition-colors"
				>
					{showAdvanced ? 'Hide' : 'Show'} Advanced
				</button>

				<button
					onclick={() => showMessages = !showMessages}
					class="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg transition-colors"
				>
					{showMessages ? 'Hide' : 'Show'} Messages
				</button>
			</div>
		</div>

		<div class="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
			<h3 class="font-semibold text-gray-900 dark:text-gray-100 mb-3">Statistics</h3>
			<div class="grid grid-cols-2 gap-4 text-sm">
				<div>
					<div class="text-gray-600 dark:text-gray-400">Messages Sent</div>
					<div class="font-mono font-semibold text-gray-900 dark:text-gray-100">{stats.messagesSent}</div>
				</div>
				<div>
					<div class="text-gray-600 dark:text-gray-400">Messages Received</div>
					<div class="font-mono font-semibold text-gray-900 dark:text-gray-100">{stats.messagesReceived}</div>
				</div>
				{#if stats.pingLatency !== null}
					<div>
						<div class="text-gray-600 dark:text-gray-400">Ping Latency</div>
						<div class="font-mono font-semibold {stats.pingLatency < 100 ? 'text-green-600' : 'text-yellow-600'}">
							{stats.pingLatency}ms
						</div>
					</div>
				{/if}
				{#if Core.webRTCReconnectAttempts > 0}
					<div>
						<div class="text-gray-600 dark:text-gray-400">Reconnect Attempts</div>
						<div class="font-mono font-semibold text-gray-900 dark:text-gray-100">
							{Core.webRTCReconnectAttempts}/5
						</div>
					</div>
				{/if}
			</div>
		</div>
	</div>

	<!-- Send Message Controls -->
	{#if Core.webRTCConnected}
		<div class="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
			<h3 class="font-semibold text-gray-900 dark:text-gray-100 mb-3">Send Test Messages</h3>

			<!-- Quick Actions -->
			<div class="flex flex-wrap gap-2 mb-4">
				<button
					onclick={() => sendTestMessage('ping')}
					class="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
					disabled={!Core.webRTCConnected}
				>
					Send Ping
				</button>

				<button
					onclick={() => sendTestMessage('hello', { type: 'hello', message: 'Hello from Genix!' })}
					class="px-4 py-2 bg-green-500 hover:bg-green-600 text-white rounded-lg transition-colors"
					disabled={!Core.webRTCConnected}
				>
					Send Hello
				</button>

				<button
					onclick={() => sendTestMessage('data_request', { type: 'data_request', resource: 'test' })}
					class="px-4 py-2 bg-purple-500 hover:bg-purple-600 text-white rounded-lg transition-colors"
					disabled={!Core.webRTCConnected}
				>
					Request Data
				</button>

				<button
					onclick={() => toggleAutoSend()}
					class="px-4 py-2 {autoSendEnabled ? 'bg-orange-500 hover:bg-orange-600' : 'bg-gray-500 hover:bg-gray-600'} text-white rounded-lg transition-colors"
					disabled={!Core.webRTCConnected}
				>
					{autoSendEnabled ? 'Stop' : 'Start'} Auto Ping
				</button>

				<button
					onclick={clearMessages}
					class="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg transition-colors"
				>
					Clear Messages
				</button>
			</div>

			<!-- Custom Message -->
			<div class="flex gap-2">
				<select
					bind:value={testMessageType}
					class="px-3 py-2 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100"
				>
					<option value="ping">Ping</option>
					<option value="hello">Hello</option>
					<option value="data_request">Data Request</option>
					<option value="custom">Custom</option>
					<option value="json">JSON</option>
				</select>

				<input
					type="text"
					bind:value={customMessage}
					placeholder="Enter custom message or JSON"
					class="flex-1 px-3 py-2 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100"
					disabled={!Core.webRTCConnected}
				/>

				<button
					onclick={() => {
						if (testMessageType === 'json') {
							sendJsonPayload();
						} else {
							sendTestMessage();
						}
					}}
					class="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors"
					disabled={!Core.webRTCConnected}
				>
					Send
				</button>
			</div>
		</div>
	{/if}

	<!-- Advanced Info -->
	{#if showAdvanced}
		<div class="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
			<h3 class="font-semibold text-gray-900 dark:text-gray-100 mb-3">Advanced Information</h3>

			<div class="space-y-3">
				{#if webRTC.bridge}
					<div>
						<div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Bridge Instance</div>
						<div class="font-mono text-xs bg-gray-100 dark:bg-gray-900 p-2 rounded overflow-x-auto">
							<pre>{JSON.stringify({
								isConnected: webRTC.bridge.isConnected(),
								config: {
									wsUrl: webRTC.bridge['config']?.wsUrl,
									targetId: webRTC.bridge['config']?.targetId,
									trickle: webRTC.bridge['config']?.trickle,
									timeout: webRTC.bridge['config']?.timeout
								}
							}, null, 2)}</pre>
						</div>
					</div>
				{/if}

				{#if Core.webRTCError}
					<div>
						<div class="text-sm text-red-600 dark:text-red-400 mb-1">Last Error</div>
						<div class="font-mono text-xs bg-red-50 dark:bg-red-900/20 p-2 rounded text-red-900 dark:text-red-300">
							{Core.webRTCError}
						</div>
					</div>
				{/if}

				{#if Env.SIGNALING_ENDPOINT}
					<div>
						<div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Signaling Endpoint</div>
						<div class="font-mono text-xs bg-gray-100 dark:bg-gray-900 p-2 rounded break-all">
							{Env.SIGNALING_ENDPOINT}
						</div>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- Messages Log -->
	{#if showMessages}
		<div class="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
			<div class="flex items-center justify-between mb-3">
				<h3 class="font-semibold text-gray-900 dark:text-gray-100">Messages Log</h3>
				<span class="text-sm text-gray-600 dark:text-gray-400">
					{messages.length} messages
				</span>
			</div>

			{#if messages.length === 0}
				<div class="text-center py-8 text-gray-500 dark:text-gray-400">
					No messages yet. Send a test message to get started!
				</div>
			{:else}
				<div class="space-y-2 max-h-450 overflow-y-auto">
					{#each messages as msg}
						<div class="flex gap-3 p-3 rounded-lg {msg.direction === 'sent'
							? 'bg-blue-50 dark:bg-blue-900/20 border-l-4 border-blue-500'
							: 'bg-green-50 dark:bg-green-900/20 border-l-4 border-green-500'}">
							<div class="flex-shrink">
								<span class="text-2xl {msg.direction === 'sent' ? 'text-blue-500' : 'text-green-500'}">
									{msg.direction === 'sent' ? '📤' : '📥'}
								</span>
							</div>
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase">
										{msg.direction}
									</span>
									<span class="text-xs px-2 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
										{msg.type}
									</span>
									{#if typeof msg.latency === 'number'}
										<span class="text-xs font-mono font-semibold {msg.latency < 100 ? 'text-green-600' : 'text-yellow-600'}">
											{msg.latency}ms
										</span>
									{/if}
									<span class="text-xs text-gray-500 dark:text-gray-400 ml-auto">
										{msg.timestamp.toLocaleTimeString()}
									</span>
								</div>
								<div class="font-mono text-xs text-gray-800 dark:text-gray-200 break-all">
									<pre class="whitespace-pre-wrap">{typeof msg.content === 'string'
										? msg.content
										: JSON.stringify(msg.content, null, 2)}</pre>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	pre {
		margin: 0;
		white-space: pre-wrap;
		word-wrap: break-word;
	}
</style>
