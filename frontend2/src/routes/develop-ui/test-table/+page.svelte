<script lang="ts">
	import { untrack } from 'svelte';
	import { createVirtualizer } from '$lib/virtualizer/index.svelte';
	import CellEditable from '../../../components/CellEditable.svelte';

	interface TestRecord {
		id: string;
		edad: number;
		nombre: string;
		apellidos: string;
		numero: number;
		_updated?: boolean;
	}

	let data = $state<TestRecord[]>(makeData());
	let containerRef = $state<HTMLDivElement>();
	let virtualItems = $state<any[]>([]);
	let totalSize = $state(0);
	let virtualizerStore: ReturnType<typeof createVirtualizer> | null = null;
	let isInitialized = false;
	let dataVersion = $state(0);

	$effect(() => {
		if (containerRef && !virtualizerStore) {
			virtualizerStore = createVirtualizer({
				count: 0, // Initial count, will use getCount() for dynamic updates
				getScrollElement: () => containerRef!,
				estimateSize: () => 34,
				overscan: 15, // Modest increase for smoother scrolling
				getCount: () => data.length
			});

			// Update state when virtualizer changes
			const updateVirtualItems = () => {
				const items = virtualizerStore!.getVirtualItems();
				const size = virtualizerStore!.getTotalSize();
				
				// Create new array reference to ensure Svelte detects the change
				virtualItems = [...items];
				totalSize = size;
			};

			// Subscribe to scroll and resize changes
			const unsubscribe = virtualizerStore.subscribe(updateVirtualItems);

			// Initial update - use requestAnimationFrame to ensure DOM is ready
			requestAnimationFrame(() => {
				updateVirtualItems();
				isInitialized = true;
			});

			return () => {
				isInitialized = false;
				virtualizerStore = null;
				unsubscribe();
			};
		}
	});

	// Watch for data array changes and update version
	let lastDataRef: any = null;
	$effect(() => {
		// Track the data array reference
		const currentData = data;
		
		// Only process if data reference changed and we're initialized
		if (lastDataRef !== null && currentData !== lastDataRef && isInitialized && virtualizerStore) {
			console.log('Data changed, updating virtualizer at current scroll position');
			
			// Use untrack to prevent these updates from triggering the effect again
			untrack(() => {
				// Increment data version (tracked by each row)
				dataVersion++;
				
				// Force immediate recalculation with current scroll position
				// The virtualizer will read from the current scroll offset
				const items = virtualizerStore!.getVirtualItems();
				const size = virtualizerStore!.getTotalSize();
				
				console.log('Updated items:', items.length, 'first index:', items[0]?.index, 'scroll maintained');
				
				virtualItems = [...items];
				totalSize = size;
			});
		}
		
		lastDataRef = currentData;
	});

	function makeData(): TestRecord[] {
		console.log('generando data::');

		const records: TestRecord[] = [];

		for (let i = 0; i < 50000; i++) {
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

	function handleNameUpdate(record: TestRecord, idx: number) {
		record.nombre = record.nombre + '_1';
		data = [...data]; // Trigger reactivity
	}

	function handleApellidosChange(record: TestRecord) {
		record._updated = true;
		data = [...data]; // Trigger reactivity
	}
</script>

<div class="page-container" style="padding: 1rem;">
	<div style="margin-bottom: 1rem;">
		<h2>VIRTUALIZE SCROLL</h2>
		<p class="text-sm text-gray-600 mt-2">
			Testing with {data.length.toLocaleString()} rows - Svelte 5 Virtual Table
		</p>
		<p class="text-sm text-gray-500 mt-1">
			Currently rendering: {virtualItems.length} rows | Total size: {totalSize}px
		</p>
	</div>

	<div bind:this={containerRef} class="table-container">
		<table class="virtual-table">
			<!-- Header -->
			<thead class="table-header">
				<tr class="header-row">
					<th class="table-cell">ID</th>
					<th class="table-cell">Edad</th>
					<th class="table-cell">Nombre</th>
					<th class="table-cell">Apellidos</th>
					<th class="table-cell">Edad</th>
					<th class="table-cell text-center">...</th>
				</tr>
			</thead>

		<!-- Virtual Tbody -->
		<tbody class="virtual-tbody">
			{#each virtualItems as row, i (`${row.index}-${dataVersion}`)}
				{@const firstItemStart = virtualItems[0]?.start || 0}
				{@const isFinal = i === virtualItems.length - 1}
				{@const remainingSize = totalSize - (virtualItems[0]?.size || 34) * virtualItems.length}
				<tr
					class="table-row"
					class:tr-even={row.index % 2 === 0}
					class:tr-odd={row.index % 2 !== 0}
					style="height: {row.size}px; transform: translateY({firstItemStart}px);"
				>
					<td class="table-cell">
						<div class="flex ai-center">
							{#if data[row.index]._updated}
								<div class="c-red" style="margin-left: -4px">
									<i class="icon-arrows-cw"></i>
								</div>
							{/if}
							<div>{data[row.index].id}</div>
						</div>
					</td>
					<td class="table-cell">{data[row.index].edad}</td>
					<td class="table-cell">{data[row.index].nombre}</td>
					<td class="table-cell" style="padding: 0;">
						<CellEditable
							saveOn={data[row.index]}
							save="apellidos"
							onChange={() => handleApellidosChange(data[row.index])}
						/>
					</td>
					<td class="table-cell">{data[row.index].edad}</td>
					<td class="table-cell text-center">
						<button
							class="btn-action"
							aria-label="Update name"
							onclick={(ev) => {
								ev.stopPropagation();
								handleNameUpdate(data[row.index], row.index);
							}}
						>
							<i class="icon-ok"></i>
						</button>
					</td>
				</tr>
				{#if isFinal}
					<tr style="height: {remainingSize}px; visibility: hidden;">
						<td style="border: none;"></td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</table>
</div>
</div>

	<style>
		.page-container {
			min-height: 100vh;
			background-color: #f8f9fa;
		}

		h2 {
			font-size: 1.5rem;
			font-weight: 600;
			margin: 0;
		}

		.text-sm {
			font-size: 0.875rem;
		}

		.text-gray-600 {
			color: #6c757d;
		}

		.mt-2 {
			margin-top: 0.5rem;
		}

		.table-container {
			overflow: auto;
			max-height: calc(100vh - 14rem);
			border: 1px solid #dee2e6;
			border-radius: 8px;
			background-color: white;
			box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		}

		.virtual-table {
			width: 100%;
			border-collapse: collapse;
			background-color: white;
		}

		.table-header {
			position: sticky;
			top: 0;
			z-index: 10;
			background-color: #f8f9fa;
		}

		.header-row {
			width: 100%;
			border-bottom: 2px solid #dee2e6;
			background-color: #f8f9fa;
		}

		.header-row th {
			padding: 0.75rem;
			font-weight: 600;
			font-size: 0.875rem;
			text-align: left;
			border: none;
		}

		.header-row th:nth-child(1) { width: 150px; }
		.header-row th:nth-child(2) { width: 80px; }
		.header-row th:nth-child(3) { width: 200px; }
		.header-row th:nth-child(4) { width: 250px; }
		.header-row th:nth-child(5) { width: 80px; }
		.header-row th:nth-child(6) { width: 100px; }

	/* Virtual table rows styling */
	.table-row {
		display: table-row;
		width: 100%;
		border-bottom: 1px solid #e9ecef;
		transition: background-color 0.15s ease;
		will-change: transform; /* Safe hint for GPU optimization */
	}

	.table-row:hover {
		background-color: #f1f3f5;
	}

	.table-row.tr-even {
		background-color: #ffffff;
	}

	.table-row.tr-odd {
		background-color: #f8f9fa;
	}

	.table-cell {
		padding: 0.5rem 0.75rem;
		display: table-cell;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		vertical-align: middle;
	}

	.table-row td:nth-child(1) { width: 150px; }
	.table-row td:nth-child(2) { width: 80px; }
	.table-row td:nth-child(3) { width: 200px; }
	.table-row td:nth-child(4) { width: 250px; }
	.table-row td:nth-child(5) { width: 80px; }
	.table-row td:nth-child(6) { width: 100px; }

	.flex {
		display: flex;
	}

	.ai-center {
		align-items: center;
	}

	.c-red {
		color: #dc3545;
	}

	.text-center {
		text-align: center;
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

	.btn-action i {
		font-size: 0.875rem;
	}
	</style>

