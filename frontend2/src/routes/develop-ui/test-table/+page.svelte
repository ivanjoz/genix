<script lang="ts">
	import CellSelector from '../../../components/CellSelector.svelte';
	import { VTable, type ITableColumn } from '../../../components/VTable';
	import Page from "../../../components/Page.svelte";

	interface TestRecord {
		id: string;
		edad: number;
		nombre: string;
		apellidos: string;
		numero: number;
		_updated?: boolean;
	}

	let data = $state<TestRecord[]>(makeData());
	let selectedRecord = $state<TestRecord | undefined>(undefined);
	let useSnippetRenderer = $state(true);

	// Create columns with render functions
	const columns: ITableColumn<TestRecord>[] = [
		{
			header: 'ID',
			id: 101,
			renderHTML: (e) => {
				if (e._updated) {
					return `<span class="updated-indicator">🔄</span> ${e.id}`;
				}
				return e.id;
			},
			css: 'id-column'
		},
		{
			header: 'Personal Information',
			headerCss: 'text-center header-group',
			subcols: [
				{
					header: 'Edad',
					getValue: (e) => e.edad,
					css: 'text-center'
				},
				{
					header: 'Nombre',
					id: 103,
					getValue: (e) => e.nombre,
					css: 'nombre-column'
				}
			]
		},
		{
			header: 'Contact & Details',
			headerCss: 'text-center header-group',
			subcols: [
				{
					header: 'Apellidos',
					getValue: (e) => e.apellidos,
					onEditChange(e, value) {
						e.apellidos = value as string
					},
				},
				{
					header: 'Selector',
					getValue: (e) => e.edad,
					css: 'text-center'
				}
			]
		},
		{
			header: 'Actions',
			id: 'actions',
			css: 'text-center action-column',
			getValue: () => '' // Empty default, will be replaced by snippet
		}
	];

	function makeData(): TestRecord[] {
		console.log('generando data::');

		const records: TestRecord[] = [];

		for (let i = 0; i < 15000; i++) {
			const record: TestRecord = {
				id: makeid(12),
				edad: Math.floor(Math.random() * 100),
				nombre: makeid(18).toLowerCase(),
				apellidos: makeid(23),
				numero: Math.floor(Math.random() * 1000),
				selector: ""
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

	function handleRowClick(record: TestRecord, index: number) {
		console.log('Row clicked:', record, 'at index:', index);
		record.nombre = record.nombre + '_1';
		record._updated = !record._updated;
		selectedRecord = record;
		data = [...data]; // Trigger reactivity
	}

	function isRowSelected(record: TestRecord, selected: TestRecord | number): boolean {
		return record === selected;
	}

	function handleEdit(record: TestRecord) {
		console.log('Edit:', record);
		alert(`Editing: ${record.nombre}`);
	}

	function handleDelete(record: TestRecord) {
		if (confirm(`Delete ${record.nombre}?`)) {
			data = data.filter(e => e !== record);
		}
	}
</script>

<!-- Snippet-based cellRenderer - renders actual Svelte components! -->
{#snippet cellRendererSnippet(record: TestRecord, col: ITableColumn<TestRecord>, value: any )}	
	<!-- Edad > 50 - red with fire emoji -->
	{#if col.header === 'Edad' && record.edad > 50}
		<span style="color: #ef4444; font-weight: 600;">{value} 🔥</span>
	
	<!-- Updated apellidos - yellow highlight -->
	{:else if record._updated && col.header === 'Apellidos'}
		<span style="background-color: #fef3c7; padding: 0.125rem 0.25rem; border-radius: 3px;">
			{value}
		</span>

	{:else if col.header === 'Selector'}
		<CellSelector id={record.id} options={data} keyField="id" valueField="nombre"
			saveOn={record} save="selector"
		/>
		
	<!-- Actions column - real interactive buttons! -->
	{:else if col.id === 'actions'}
		<div class="action-buttons">
			<button class="btn-small btn-edit"
				onclick={(e) => {
					e.stopPropagation();
					handleEdit(record);
				}}
				title="Edit"
			>
				✏️
			</button>
			<button class="btn-small btn-delete"
				onclick={(e) => {
					e.stopPropagation();
					handleDelete(record);
				}}
				title="Delete"
			>
				🗑️
			</button>
		</div>
	
	<!-- Default - just show content -->
	{:else}
		{value}
	{/if}
{/snippet}

<Page>
	<div class="header-section">
		<h2>VTable Component - Snippet-based cellRenderer Demo</h2>
		<p class="text-sm text-gray-600 mt-2">
			Testing with {data.length.toLocaleString()} rows - Svelte 5 + Custom Virtualizer
		</p>
		<p class="text-sm text-blue-600 mt-1">
			Using <strong>cellRendererSnippet</strong> to render actual Svelte components in cells!
		</p>

		<p class="text-sm text-gray-500 mt-2">
			💡 <strong>Tip:</strong> Click any row to update it. Try the Edit/Delete buttons in the Actions column!
		</p>
		{#if selectedRecord}
			<div class="selected-info mt-2">
				<strong>Selected:</strong> {selectedRecord.nombre} (ID: {selectedRecord.id})
				{#if selectedRecord._updated}
					<span class="updated-badge">Updated</span>
				{/if}
			</div>
		{/if}
	</div>

	<VTable {columns} {data} 
		maxHeight="calc(100vh - 16rem)" 
		onRowClick={handleRowClick}
		selected={selectedRecord}
		isSelected={isRowSelected}
		estimateSize={34}
		overscan={15}
		cellRenderer={useSnippetRenderer ? cellRendererSnippet : undefined}
	/>
</Page>

<style>
	.page-container {
		min-height: 100vh;
		background-color: #f8f9fa;
	}

	.header-section {
		margin-bottom: 1.5rem;
	}

	h2 {
		font-size: 1.5rem;
		font-weight: 600;
		margin: 0;
		color: #1a202c;
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

	.selected-info {
		padding: 0.75rem 1rem;
		background-color: #e7f3ff;
		border-left: 4px solid #3b82f6;
		border-radius: 4px;
		color: #1e40af;
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.updated-badge {
		padding: 0.25rem 0.5rem;
		background-color: #22c55e;
		color: white;
		border-radius: 4px;
		font-size: 0.75rem;
		font-weight: 600;
	}

	/* Custom styling for VTable columns */
	:global(.updated-indicator) {
		color: #dc3545;
		font-size: 1rem;
		margin-right: 0.25rem;
	}

	:global(.id-column) {
		font-weight: 600;
		color: #4042a3;
	}

	:global(.nombre-column) {
		font-weight: 500;
	}

	:global(.action-column) {
		width: 100px;
	}

	:global(.header-group) {
		background-color: #e9ecef !important;
		font-weight: 700;
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

	/* New snippet-based action buttons */
	:global(.action-buttons) {
		display: flex;
		gap: 0.25rem;
		justify-content: center;
		align-items: center;
	}

	:global(.btn-small) {
		padding: 0.25rem 0.5rem;
		border: none;
		background: #f3f4f6;
		border-radius: 4px;
		cursor: pointer;
		font-size: 0.875rem;
		transition: all 0.2s;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	:global(.btn-small:hover) {
		background: #e5e7eb;
		transform: scale(1.05);
	}

	:global(.btn-small:active) {
		transform: scale(0.95);
	}

	:global(.btn-edit:hover) {
		background: #dbeafe;
		color: #1e40af;
	}

	:global(.btn-delete:hover) {
		background: #fee2e2;
		color: #dc2626;
	}
</style>

