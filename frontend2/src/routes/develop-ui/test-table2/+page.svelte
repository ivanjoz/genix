<script lang="ts">
	import QTable2, { type ITableColumn } from '../../../components/QTable2.svelte';

	interface TestRecord {
		id: string;
		edad: number;
		nombre: string;
		apellidos: string;
		numero: number;
		_updated?: boolean;
	}

	let data = $state<TestRecord[]>(makeData());

	// Create columns with render functions
	const columns: ITableColumn<TestRecord>[] = [
		{
			header: 'ID',
			id: 101,
			render: (e) => {
				if (e._updated) {
					return `<span class="updated-indicator">🔄</span> ${e.id}`;
				}
				return e.id;
			}
		},
		{
			header: 'Edad',
			getValue: (e) => e.edad
		},
		{
			header: 'Nombre',
			id: 103,
			getValue: (e) => e.nombre
		},
		{
			header: 'Apellidos',
			getValue: (e) => e.apellidos
		},
		{
			header: 'Edad',
			getValue: (e) => e.edad
		},
		{
			header: 'Actions',
			css: 'text-center',
			getValue: (e) => '✓'
		}
	];

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

	function handleRowClick(record: TestRecord) {
		record.nombre = record.nombre + '_1';
		record._updated = !record._updated;
		data = [...data]; // Trigger reactivity
	}
</script>

<div class="page-container" style="padding: 1rem;">
	<div style="margin-bottom: 1rem;">
		<h2>VIRTUALIZE SCROLL - VIRTUA VERSION</h2>
		<p class="text-sm text-gray-600 mt-2">
			Testing with {data.length.toLocaleString()} rows - Svelte 5 + Virtua Library
		</p>
		<p class="text-sm text-blue-600 mt-1">
			Using <strong>virtua</strong> library instead of TanStack Virtual
		</p>
		<p class="text-sm text-gray-500 mt-1">
			💡 <strong>Tip:</strong> Click any row to update its "Nombre" field and toggle the update indicator!
		</p>
	</div>

	<QTable2 {columns} {data} maxHeight="calc(100vh - 10rem)" onRowClick={handleRowClick} />
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

	.text-blue-600 {
		color: #2563eb;
	}

	.text-gray-500 {
		color: #6c757d;
	}

	.mt-1 {
		margin-top: 0.25rem;
	}

	.mt-2 {
		margin-top: 0.5rem;
	}

	:global(.updated-indicator) {
		color: #dc3545;
		font-size: 1rem;
		margin-right: 0.25rem;
	}

	:global(.flex) {
		display: flex;
	}

	:global(.ai-center) {
		align-items: center;
	}

	:global(.c-red) {
		color: #dc3545;
	}

	:global(.text-center) {
		text-align: center;
	}

	:global(.btn-action) {
		padding: 0.25rem 0.5rem;
		background-color: #4042a3;
		color: white;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		transition: background-color 0.2s;
		font-size: 0.875rem;
	}

	:global(.btn-action:hover) {
		background-color: #2f3280;
	}

	:global(.btn-action i) {
		font-size: 0.875rem;
	}
</style>

