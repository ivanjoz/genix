<script lang="ts">
	import VirtualList from 'svelte-tiny-virtual-list';
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
	const pageSize = 500;
	const rowHeight = 36;

	// Generate mock data
	function makeData(count: number): TestRecord[] {
		const records: TestRecord[] = [];

		for (let i = 0; i < count; i++) {
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
			const data = makeData(pageSize);
			items = data;
			isLoading = false;
		} catch (e) {
			errorMessage = 'Failed to load initial data';
			isLoading = false;
		}
	}

	function handleRowClick(record: TestRecord) {
		record.nombre = record.nombre + '_1';
		record._updated = !record._updated;
		items = [...items]; // Trigger reactivity
	}

	// Convert absolute positioning style to transform translateY
	function convertToTransform(style: string): string {
		// Extract the top value from "position:absolute;top:XXpx;"
		const topMatch = style.match(/top:(\d+)px/);
		if (topMatch) {
			const topValue = topMatch[1];
			// Replace absolute positioning with transform
			return style
				.replace(/position:absolute;/, '')
				.replace(/top:\d+px;?/, '')
				+ `transform:translateY(${topValue}px);`;
		}
		return style;
	}

	// Load initial data on mount
	onMount(async () => {
		if (items.length === 0) {
			await loadInitialData();
		}
	});
</script>

<div class="page-container">
	<div class="header-section">
		<h1>Test Table 4 - Virtual Table (Svelte Tiny Virtual List)</h1>
		<p class="subtitle">Virtual table component using svelte-tiny-virtual-list library with fixed header</p>
		<div class="info-section">
			<div class="info-item">
				<span class="label">Total Rows:</span>
				<span class="value">{items.length.toLocaleString()}</span>
			</div>
			<div class="info-item">
				<span class="label">Row Height:</span>
				<span class="value">{rowHeight}px</span>
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
		<div class="table-container">
			<div class="table-header">
				<div class="table-row header-row">
					<div class="col-id">ID</div>
					<div class="col-edad">Edad</div>
					<div class="col-nombre">Nombre</div>
					<div class="col-apellidos">Apellidos</div>
					<div class="col-numero">Número</div>
					<div class="col-actions">Actions</div>
				</div>
			</div>

			<div class="virtual-list-wrapper">
				<VirtualList
					width="100%"
					height={600}
					itemCount={items.length}
					itemSize={rowHeight}
					overscanCount={10}
				>
					<div slot="item" let:index let:style>
						{@const data = items[index]}
						{@const transformStyle = convertToTransform(style)}
						<div class="table-row data-row" role="button" tabindex="0" style={transformStyle} onmouseenter={() => handleRowClick(data)}>
							<div class="col-id">
								{#if data._updated}
									<span class="updated-indicator">🔄</span>
								{/if}
								{data.id}
							</div>
							<div class="col-edad">{data.edad}</div>
							<div class="col-nombre">{data.nombre}</div>
							<div class="col-apellidos">{data.apellidos}</div>
							<div class="col-numero">{data.numero}</div>
							<div class="col-actions">
								<button class="btn-action" onclick={() => handleRowClick(data)}>✓</button>
							</div>
						</div>
					</div>
				</VirtualList>
			</div>
		</div>
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

	.table-container {
		position: relative;
		width: 100%;
	}

	.table-header {
		position: sticky;
		top: 0;
		z-index: 10;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		font-weight: 600;
		height: 36px;
	}

	.table-row {
		display: grid;
		grid-template-columns: 15% 10% 18% 20% 12% 12%;
		gap: 0;
		padding: 0.75rem 1rem;
		text-align: left;
		font-size: 0.875rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.header-row {
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		font-weight: 600;
		border-bottom: 2px solid #e2e8f0;
	}

	.data-row {
		border-bottom: 1px solid #e2e8f0;
		transition: background-color 0.15s ease;
		height: 36px;
		display: grid;
		grid-template-columns: 15% 10% 18% 20% 12% 12%;
		gap: 0;
		padding: 0.5rem 1rem;
		font-size: 0.875rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.data-row:hover {
		background-color: #f7fafc;
	}

	.data-row:nth-child(even) {
		background-color: #fafbfc;
	}

	.col-id {
		font-weight: 600;
		color: #2d3748;
	}

	.col-edad {
		text-align: center;
	}

	.col-numero {
		text-align: center;
	}

	.col-actions {
		text-align: center;
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

	.virtual-list-wrapper {
		width: 100%;
		border-top: 1px solid #e2e8f0;
	}

	:global(.virtual-list-wrapper) {
		width: 100%;
	}

	:global(.virtual-list-inner) {
		width: 100%;
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
