<script lang="ts">
	import * as Infinitable from 'svelte-infinitable';
	import type { InfiniteHandler } from 'svelte-infinitable/types';
	import { onMount } from 'svelte';

	interface TestRecord {
		id: string;
		edad: number;
		nombre: string;
		apellidos: string;
		numero: number;
		_updated?: boolean;
	}

	let items = $state<TestRecord[]>([]);
	let isLoading = $state(false);
	let errorMessage = $state('');
	let page = $state(1);
	const pageSize = 100;

	// Generate mock data - same function from test-table2
	function makeData(): TestRecord[] {
		const records: TestRecord[] = [];

		for (let i = 0; i < pageSize; i++) {
			const record: TestRecord = {
				id: makeid(12),
				edad: Math.floor(Math.random() * 100),
				nombre: makeid(18),
				apellidos: makeid(23),
				numero: Math.floor(Math.random() * 1000)
			};
			records.push(record);
		}
		return records;
	}

	function makeid(length: number): string {
		let result = '';
		const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
		const charactersLength = characters.length;
		let counter = 0;
		while (counter < length) {
			result += characters.charAt(Math.floor(Math.random() * charactersLength));
			counter += 1;
		}
		return result;
	}

	// Initial load
	async function loadInitialData() {
		isLoading = true;
		errorMessage = '';

		try {
			// Simulate loading delay
			await new Promise((resolve) => setTimeout(resolve, 500));
			const data = makeData();
			items = data;
			isLoading = false;
		} catch (e) {
			errorMessage = 'Failed to load initial data';
			isLoading = false;
		}
	}

	const onInfinite: InfiniteHandler = async ({ loaded, completed, error }) => {
		try {
			// Simulate network delay
			await new Promise((resolve) => setTimeout(resolve, 300));

			const newData = makeData();

			// For demo purposes, stop loading after 500 items
			if (items.length >= 500) {
				completed(newData);
			} else {
				loaded(newData);
			}
		} catch (e) {
			console.error('Error loading items:', e);
			error();
		}
	};

	// Load initial data on mount
	onMount(async () => {
		if (items.length === 0) {
			await loadInitialData();
		}
	});

	function handleRowClick(record: TestRecord) {
		record.nombre = record.nombre + '_1';
		record._updated = !record._updated;
		items = [...items]; // Trigger reactivity
	}

</script>

<div class="page-container">
	<div class="header-section">
		<h1>Test Table 3 - Virtual Table (Svelte Infinitable)</h1>
		<p class="subtitle">Virtual table component with infinite scrolling using svelte-infinitable - Same data as test-table2</p>
		<div class="info-section">
			<div class="info-item">
				<span class="label">Total Rows:</span>
				<span class="value">{items.length.toLocaleString()}</span>
			</div>
			<div class="info-item">
				<span class="label">Row Height:</span>
				<span class="value">36px</span>
			</div>
			<div class="info-item">
				<span class="label">Status:</span>
				<span class="value status-badge" class:loading={isLoading}>
					{isLoading ? 'Loading...' : 'Ready'}
				</span>
			</div>
		</div>
	</div>

	{#if errorMessage}
		<div class="error-message">
			{errorMessage}
		</div>
	{/if}

	<div class="table-wrapper">
		<Infinitable.Root
			bind:items
			rowHeight={36}
			{onInfinite}
			class="virtual-table"
			overscan={10}
		>
			{#snippet headers()}
				<tr class="header-row">
					<th class="col-id">ID</th>
					<th class="col-edad">Edad</th>
					<th class="col-nombre">Nombre</th>
					<th class="col-apellidos">Apellidos</th>
					<th class="col-numero">Número</th>
					<th class="col-actions">Actions</th>
				</tr>
			{/snippet}

			{#snippet children({ item, index })}
				{@const data = items[index]}
				<tr class="data-row" onmouseenter={() => handleRowClick(data)}>
					<td class="col-id">
						{#if data._updated}
							<span class="updated-indicator">🔄</span>
						{/if}
						{data.id}
					</td>
					<td class="col-edad">{data.edad}</td>
					<td class="col-nombre">{data.nombre}</td>
					<td class="col-apellidos">{data.apellidos}</td>
					<td class="col-numero">{data.numero}</td>
					<td class="col-actions">
						<button class="btn-action" onclick={() => handleRowClick(data)}>✓</button>
					</td>
				</tr>
			{/snippet}

			{#snippet loader()}
				<div class="loader-message">
					<div class="spinner"></div>
					<span>Loading more rows...</span>
				</div>
			{/snippet}

			{#snippet completed()}
				<div class="completed-message">
					✓ All {items.length.toLocaleString()} rows loaded
				</div>
			{/snippet}

			{#snippet empty()}
				<div class="empty-message">
					No items found
				</div>
			{/snippet}

			{#snippet loadingEmpty()}
				<div class="loading-empty-message">
					<div class="spinner"></div>
					<span>Loading data...</span>
				</div>
			{/snippet}
		</Infinitable.Root>
	</div>
</div>

<style>
	.page-container {
		min-height: 100vh;
		background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
		padding: 2rem;
	}

	.header-section {
		background: white;
		border-radius: 8px;
		padding: 2rem;
		margin-bottom: 2rem;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
	}

	h1 {
		margin: 0 0 0.5rem 0;
		font-size: 2rem;
		font-weight: 700;
		color: #1a202c;
	}

	.subtitle {
		margin: 0.5rem 0 1.5rem 0;
		font-size: 1rem;
		color: #4a5568;
	}

	.info-section {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1.5rem;
		margin-top: 1.5rem;
	}

	.info-item {
		display: flex;
		flex-direction: column;
		padding: 1rem;
		background: #f7fafc;
		border-radius: 6px;
		border-left: 4px solid #4299e1;
	}

	.label {
		font-size: 0.875rem;
		font-weight: 600;
		color: #718096;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.value {
		margin-top: 0.5rem;
		font-size: 1.5rem;
		font-weight: 700;
		color: #2d3748;
	}

	.status-badge {
		display: inline-block;
		padding: 0.5rem 1rem;
		background: #c6f6d5;
		color: #22543d;
		border-radius: 4px;
		font-size: 0.875rem;
		width: fit-content;
	}

	.status-badge.loading {
		background: #bee3f8;
		color: #2c5282;
		animation: pulse 1.5s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.7;
		}
	}

	.error-message {
		background: #fed7d7;
		color: #742a2a;
		padding: 1rem;
		border-radius: 6px;
		margin-bottom: 1rem;
		border-left: 4px solid #fc8181;
	}

	.table-wrapper {
		background: white;
		border-radius: 8px;
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		overflow: hidden;
	}

	:global(.virtual-table) {
		width: 100%;
		max-height: 600px;
		border-collapse: collapse;
	}

	:global(.virtual-table table) {
		width: 100%;
		border-collapse: collapse;
	}

	.header-row {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		font-weight: 600;
		position: sticky;
		top: 0;
		z-index: 10;
	}

	.header-row th {
		padding: 0.75rem 1rem;
		text-align: left;
		font-size: 0.875rem;
		white-space: nowrap;
		border-bottom: 2px solid #e2e8f0;
	}

	.data-row {
		border-bottom: 1px solid #e2e8f0;
		transition: background-color 0.15s ease;
		height: 36px;
	}

	.data-row:hover {
		background-color: #f7fafc;
	}

	.data-row:nth-child(even) {
		background-color: #fafbfc;
	}

	.data-row td {
		padding: 0.5rem 1rem;
		font-size: 0.875rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.col-id {
		width: 15%;
		font-weight: 600;
		color: #2d3748;
	}

	.col-edad {
		width: 10%;
		text-align: center;
	}

	.col-nombre {
		width: 18%;
	}

	.col-apellidos {
		width: 20%;
	}

	.col-numero {
		width: 12%;
		text-align: center;
	}

	.col-actions {
		width: 12%;
		text-align: center;
	}

	.loader-message {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		padding: 2rem;
		color: #4a5568;
		background: #f7fafc;
	}

	.spinner {
		width: 20px;
		height: 20px;
		border: 3px solid #e2e8f0;
		border-top-color: #4299e1;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.updated-indicator {
		color: #dc3545;
		font-size: 1rem;
		margin-right: 0.25rem;
	}

	.btn-action {
		padding: 0.25rem 0.5rem;
		background-color: #4042a3;
		color: white;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		transition: background-color 0.2s;
		font-size: 0.875rem;
	}

	.btn-action:hover {
		background-color: #2f3280;
	}

	.completed-message {
		padding: 1.5rem;
		text-align: center;
		color: #22543d;
		background: #c6f6d5;
		font-weight: 500;
	}

	.empty-message {
		padding: 3rem;
		text-align: center;
		color: #718096;
		font-size: 1rem;
	}

	.loading-empty-message {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		padding: 4rem 2rem;
		color: #4a5568;
		background: #f7fafc;
	}

	@media (max-width: 768px) {
		.page-container {
			padding: 1rem;
		}

		.header-section {
			padding: 1.5rem;
		}

		h1 {
			font-size: 1.5rem;
		}

		.info-section {
			grid-template-columns: 1fr;
		}

		:global(.virtual-table) {
			max-height: 400px;
		}

		.col-numero {
			display: none;
		}

		.col-actions {
			width: 15%;
		}

		.col-id {
			width: 18%;
		}

		.col-nombre {
			width: 22%;
		}

		.col-apellidos {
			width: 25%;
		}
	}
</style>
