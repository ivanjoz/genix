<script lang="ts">
	import { VTable, type ITableColumn } from '../../../components/VTable';

	interface TestRecord {
		id: string;
		name: string;
		age: number;
		email: string;
		phone: string;
		address: string;
		city: string;
		country: string;
	}

	// Generate test data
	function makeData(count: number): TestRecord[] {
		const records: TestRecord[] = [];
		const names = ['John', 'Jane', 'Bob', 'Alice', 'Charlie', 'David', 'Emma', 'Frank'];
		const cities = ['New York', 'London', 'Paris', 'Tokyo', 'Sydney', 'Berlin', 'Madrid'];
		const countries = ['USA', 'UK', 'France', 'Japan', 'Australia', 'Germany', 'Spain'];

		for (let i = 0; i < count; i++) {
			records.push({
				id: `ID-${i + 1000}`,
				name: names[i % names.length] + ' ' + (i + 1),
				age: 20 + (i % 50),
				email: `user${i}@example.com`,
				phone: `+1-555-${String(i).padStart(4, '0')}`,
				address: `${i + 100} Main Street`,
				city: cities[i % cities.length],
				country: countries[i % countries.length]
			});
		}
		return records;
	}

	let data = $state<TestRecord[]>(makeData(10000));
	let selectedRecord = $state<TestRecord | undefined>(undefined);

	// Example 1: Simple columns without subcolumns
	const simpleColumns: ITableColumn<TestRecord>[] = [
		{
			header: 'ID',
			field: 'id',
			css: 'text-left',
			headerCss: 'text-left'
		},
		{
			header: 'Name',
			field: 'name',
			css: 'text-left font-semibold'
		},
		{
			header: 'Age',
			field: 'age',
			css: 'text-center',
			headerCss: 'text-center'
		},
		{
			header: 'Email',
			field: 'email',
			css: 'text-left'
		},
		{
			header: 'Phone',
			field: 'phone',
			css: 'text-left'
		}
	];

	// Example 2: Columns with subcolumns (multi-level header)
	const complexColumns: ITableColumn<TestRecord>[] = [
		{
			header: 'ID',
			field: 'id',
			css: 'text-left'
		},
		{
			header: 'Personal Info',
			headerCss: 'text-center',
			subcols: [
				{
					header: 'Name',
					field: 'name',
					css: 'font-semibold'
				},
				{
					header: 'Age',
					field: 'age',
					css: 'text-center'
				}
			]
		},
		{
			header: 'Contact',
			headerCss: 'text-center',
			subcols: [
				{
					header: 'Email',
					field: 'email'
				},
				{
					header: 'Phone',
					field: 'phone'
				}
			]
		},
		{
			header: 'Location',
			headerCss: 'text-center',
			subcols: [
				{
					header: 'City',
					field: 'city'
				},
				{
					header: 'Country',
					field: 'country'
				}
			]
		}
	];

	// Example 3: Columns with renderHTML (explicit HTML rendering)
	const customColumns: ITableColumn<TestRecord>[] = [
		{
			header: 'ID',
			renderHTML: (record) => {
				// Using renderHTML for explicit HTML rendering
				return `<span class="badge">${record.id}</span>`;
			}
		},
		{
			header: 'Full Details',
			headerCss: 'text-center',
			subcols: [
				{
					header: 'Name',
					renderHTML: (record, idx) => {
						// Explicit HTML rendering with renderHTML
						return `<div class="name-cell">
							<strong>${record.name}</strong>
							<small style="display: block; color: #666;">${record.email}</small>
						</div>`;
					},
					cellStyle: { 'min-width': '200px' }
				},
				{
					header: 'Age',
					renderHTML: (record) => {
						const color = record.age < 30 ? '#22c55e' : record.age < 50 ? '#3b82f6' : '#ef4444';
						return `<span style="color: ${color}; font-weight: 600;">${record.age}</span>`;
					},
					css: 'text-center'
				}
			]
		},
		{
			header: 'Contact & Location',
			headerCss: 'text-center',
			subcols: [
				{
					header: 'Phone',
					field: 'phone'
				},
				{
					header: 'Address',
					getValue: (record) => `${record.city}, ${record.country}`
				}
			]
		}
	];

	let currentColumns = $state<ITableColumn<TestRecord>[]>(simpleColumns);
	let currentExample = $state<'simple' | 'complex' | 'custom'>('simple');

	function switchExample(example: 'simple' | 'complex' | 'custom') {
		currentExample = example;
		switch (example) {
			case 'simple':
				currentColumns = simpleColumns;
				break;
			case 'complex':
				currentColumns = complexColumns;
				break;
			case 'custom':
				currentColumns = customColumns;
				break;
		}
	}

	function handleRowClick(record: TestRecord, index: number) {
		selectedRecord = record;
		console.log('Row clicked:', record, 'at index:', index);
	}

	function isRowSelected(record: TestRecord, selected: TestRecord | number): boolean {
		return record === selected;
	}
</script>

<div class="page-container">
	<div class="header-section">
		<h1>VTable Component Demo</h1>
		<p class="subtitle">Virtual scrolling table with {data.length.toLocaleString()} records</p>
		
		<div class="button-group">
			<button 
				class="demo-btn"
				class:active={currentExample === 'simple'}
				onclick={() => switchExample('simple')}
			>
				Simple Columns
			</button>
			<button 
				class="demo-btn"
				class:active={currentExample === 'complex'}
				onclick={() => switchExample('complex')}
			>
				Multi-level Headers
			</button>
			<button 
				class="demo-btn"
				class:active={currentExample === 'custom'}
				onclick={() => switchExample('custom')}
			>
				Custom HTML (renderHTML)
			</button>
		</div>

		{#if selectedRecord}
			<div class="selected-info">
				<strong>Selected:</strong> {selectedRecord.name} ({selectedRecord.email})
			</div>
		{/if}
	</div>

	<div class="table-wrapper">
		<VTable
			{data}
			columns={currentColumns}
			maxHeight="calc(100vh - 16rem)"
			onRowClick={handleRowClick}
			selected={selectedRecord}
			isSelected={isRowSelected}
		/>
	</div>
</div>

<style>
	.page-container {
		padding: 2rem;
		min-height: 100vh;
		background-color: #f8f9fa;
	}

	.header-section {
		margin-bottom: 1.5rem;
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		margin: 0 0 0.5rem 0;
		color: #1a202c;
	}

	.subtitle {
		color: #6c757d;
		margin: 0 0 1rem 0;
		font-size: 0.95rem;
	}

	.button-group {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.demo-btn {
		padding: 0.5rem 1rem;
		background-color: white;
		border: 2px solid #dee2e6;
		border-radius: 6px;
		cursor: pointer;
		font-weight: 500;
		transition: all 0.2s;
	}

	.demo-btn:hover {
		background-color: #f8f9fa;
		border-color: #4042a3;
	}

	.demo-btn.active {
		background-color: #4042a3;
		color: white;
		border-color: #4042a3;
	}

	.selected-info {
		padding: 0.75rem 1rem;
		background-color: #e7f3ff;
		border-left: 4px solid #3b82f6;
		border-radius: 4px;
		color: #1e40af;
	}

	/* Custom styles for rendered content */
	:global(.badge) {
		padding: 0.25rem 0.5rem;
		background-color: #e7f3ff;
		color: #1e40af;
		border-radius: 4px;
		font-size: 0.85rem;
		font-weight: 600;
	}

	:global(.name-cell) {
		padding: 0.25rem 0;
	}

	:global(.text-center) {
		text-align: center;
	}

	:global(.text-left) {
		text-align: left;
	}

	:global(.font-semibold) {
		font-weight: 600;
	}
</style>

