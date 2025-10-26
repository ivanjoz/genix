<script lang="ts" module>
	type CSSProperties = Record<string, string | number>;

	export interface ITableColumn<T> {
		id?: number;
		header: string | (() => string);
		headerCss?: string;
		headerStyle?: CSSProperties;
		cellStyle?: CSSProperties;
		css?: string;
		cardCss?: string;
		field?: string;
		subcols?: ITableColumn<T>[];
		cardColumn?: [number, (1 | 2 | 3)?];
		cardRender?: (e: T, idx: number, rerender: (ids?: number[]) => void) => any;
		getValue?: (e: T, idx: number) => string | number;
		render?: (e: T, idx: number, rerender: (ids?: number[]) => void) => any;
		_colspan?: number;
	}

	export interface IQTable2Props<T> {
		columns: ITableColumn<T>[];
		data: T[];
		maxHeight?: string;
		css?: string;
		style?: CSSProperties;
		tableStyle?: CSSProperties;
		tableCss?: string;
		styleMobile?: CSSProperties;
		selected?: number | string | T;
		isSelected?: (e: T, c: number | string | T) => boolean;
		onRowClick?: (e: T) => void;
		filterText?: string;
		makeFilter?: (e: T) => string;
		filterKeys?: string[];
		deviceType?: number; // 1: desktop, 2: tablet, 3: mobile
	}
</script>

<script lang="ts" generics="T">
	import { VList } from 'virtua/svelte';

	let {
		columns = $bindable(),
		data = $bindable(),
		maxHeight = 'calc(100vh - 8rem - 12px)',
		css = '',
		style = {},
		tableStyle = {},
		tableCss = '',
		styleMobile = {},
		selected,
		isSelected,
		onRowClick,
		filterText = '',
		makeFilter,
		filterKeys = [],
		deviceType = 1
	}: IQTable2Props<T> = $props();

	let records = $state<T[]>([]);

	// Build card columns layout
	const cardColumns = $derived.by(() => {
		let cardColumnsFlat: ITableColumn<T>[] = [];

		for (const co of columns) {
			if (co.subcols) {
				for (const sc of co.subcols) {
					cardColumnsFlat.push(sc);
				}
			} else {
				cardColumnsFlat.push(co);
			}
		}

		cardColumnsFlat = cardColumnsFlat.filter((x) => x.cardColumn && x.cardColumn.length > 0);
		cardColumnsFlat.sort((a, b) => a.cardColumn![0] - b.cardColumn![0]);

		const cardColumnsMap: Map<number, ITableColumn<T>[]> = new Map();
		for (const e of cardColumnsFlat) {
			cardColumnsMap.has(e.cardColumn![0])
				? cardColumnsMap.get(e.cardColumn![0])!.push(e)
				: cardColumnsMap.set(e.cardColumn![0], [e]);
		}

		const cardCols: ITableColumn<T>[][] = [];
		for (const cols of cardColumnsMap.values()) {
			cols.sort((a, b) => (a.cardColumn![1] || 1) - (b.cardColumn![1] || 1));
			cardCols.push(cols);
		}
		return cardCols;
	});

	// Check if we should use card view (mobile)
	const isCardView = $derived(deviceType === 3 && cardColumns.length > 0);

	// Filter data
	const filteredRecords = $derived.by(() => {
		let filterFn = makeFilter;

		if (!filterFn && filterKeys?.length > 0) {
			filterFn = (e: T) => {
				let content: string[] = [];
				for (let key of filterKeys) {
					const val = (e as any)[key];
					if (val && (typeof val === 'string' || typeof val === 'number')) {
						content.push(String(val));
					}
				}
				return content.join(' ').toLowerCase();
			};
		}

		if (!filterFn || !filterText) {
			return data;
		}

		let recordsFiltered: T[] = [];
		const filterTexts = filterText.split(' ').map((x) => x.toLowerCase());

		for (let e of data) {
			const text = filterFn(e);
			if (!text) continue;

			// Simple include check
			const textLower = text.toLowerCase();
			if (filterTexts.every((ft) => textLower.includes(ft))) {
				recordsFiltered.push(e);
			}
		}

		return recordsFiltered;
	});

	// Update records when filtered
	$effect(() => {
		records = filteredRecords;
	});

	// Process columns for headers
	const processedColumns = $derived.by(() => {
		const columns1: ITableColumn<T>[] = [];
		const columns2: ITableColumn<T>[] = [];

		for (let column of columns) {
			column._colspan = column?.subcols?.length || 0;
			if (column._colspan) {
				for (let sc of column.subcols!) {
					columns2.push(sc);
				}
			}
			columns1.push(column);
		}

		return { columns1, columns2 };
	});

	// Helper to highlight text
	function highlightString(text: string, searchTerms: string[]): string {
		if (!searchTerms.length) return text;

		let result = text;
		for (const term of searchTerms) {
			if (!term) continue;
			const regex = new RegExp(`(${term})`, 'gi');
			result = result.replace(regex, '<mark>$1</mark>');
		}
		return result;
	}

	// Get cell content
	function getCellContent(
		column: ITableColumn<T>,
		record: T,
		index: number,
		rerender: (ids?: number[]) => void
	): any {
		let content: string | number | any = '';

		if (column.render) {
			content = column.render(record, index, rerender);
		} else if (column.getValue) {
			content = column.getValue(record, index);
		}

		return content;
	}

	// Check if row is selected
	function isRowSelected(record: T): boolean {
		if (!selected || !isSelected) return false;
		return isSelected(record, selected);
	}

	// Get header content
	function getHeaderContent(column: ITableColumn<T>): string {
		if (typeof column.header === 'function') {
			return column.header();
		}
		return column.header;
	}

	const divClass = $derived(isCardView ? 'qtable-cards w100' : `qtable-c${css ? ' ' + css : ''}`);

	const containerStyle = $derived.by(() => {
		const baseStyle = {
			'max-height': maxHeight,
			...style
		};

		if (isCardView) {
			return { ...baseStyle, ...styleMobile };
		}

		return baseStyle;
	});

	// Helper to convert style object to string
	function styleToString(styleObj?: CSSProperties): string {
		if (!styleObj) return '';
		return Object.entries(styleObj)
			.map(([key, value]) => `${key}: ${value}`)
			.join('; ');
	}

	// Calculate column widths with minmax for flexible sizing
	const columnWidths = $derived(
		columns.map(() => 'minmax(20px, auto)').join(' ')
	);
</script>

<div class={divClass} style={styleToString(containerStyle)}>
	{#if !isCardView}
		<div class="qtable-grid-wrapper">
			<!-- Fixed Header -->
			<div class="qtable-header">
				<div class="qtable-row header-row" style="grid-template-columns: {columnWidths};">
					{#each processedColumns.columns1 as column}
						<div
							class={`qtable-cell ${column.headerCss || ''}`}
							style={styleToString(column.headerStyle)}
						>
							{getHeaderContent(column)}
						</div>
					{/each}
				</div>
				{#if processedColumns.columns2.length > 0}
					<div class="qtable-row header-row" style="grid-template-columns: {columnWidths};">
						{#each processedColumns.columns2 as column}
							<div class={`qtable-cell ${column.headerCss || ''}`} style={styleToString(column.headerStyle)}>
								{getHeaderContent(column)}
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Virtual List Body -->
			{#if records.length === 0}
				<div class="empty-message flex ai-center">No se encontraron registros.</div>
			{:else}
				<div class="qtable-body-wrapper">
					<VList data={records} itemSize={34} style="height: {maxHeight}; overflow: auto;">
						{#snippet children(item, index)}
							{@const record = item}
							{@const selected = isRowSelected(record)}

							<div
								role="button"
								tabindex="0"
								class="qtable-row data-row"
								class:selected
								class:tr-even={index % 2 === 0}
								class:tr-odd={index % 2 !== 0}
								style="grid-template-columns: {columnWidths}; width: 100%;"
								onclick={() => onRowClick?.(record)}
								onkeydown={(ev) => {
									if (ev.key === 'Enter' || ev.key === ' ') {
										ev.preventDefault();
										onRowClick?.(record);
									}
								}}
							>
								{#each columns as column}
									{@const content = getCellContent(column, record, index, () => {})}
									<div class={`qtable-cell ${column.css || ''}`} style={styleToString(column.cellStyle)}>
										{#if typeof content === 'string' && column.field && filterText && filterKeys?.includes(column.field)}
											<span class="_highlight">
												{@html highlightString(content, filterText.split(' '))}
											</span>
										{:else}
											{content}
										{/if}
									</div>
								{/each}
							</div>
						{/snippet}
					</VList>
				</div>
			{/if}
		</div>
	{:else}
		<!-- Card View for Mobile -->
		<div class="p-rel w100">
			{#if records.length === 0}
				<div class="empty-message flex ai-center">No se encontraron registros.</div>
			{:else}
				<VList data={records} itemSize={50}>
					{#snippet children(item, index)}
						{@const record = item}
						{@const selected = isRowSelected(record)}

						<div
							role="button"
							tabindex="0"
							class={`w100 flex-column card-ct mb-04${selected ? ' selected' : ''}`}
							onclick={(ev) => {
								ev.stopPropagation();
								onRowClick?.(record);
							}}
							onkeydown={(ev) => {
								if (ev.key === 'Enter' || ev.key === ' ') {
									ev.preventDefault();
									onRowClick?.(record);
								}
							}}
						>
							{#each cardColumns as cols}
								<div class="flex ai-center jc-between">
									{#each cols as col}
										{@const content = col.cardRender
											? col.cardRender(record, index, () => {})
											: col.render
												? col.render(record, -1, () => {})
												: col.getValue
													? col.getValue(record, index)
													: ''}
										<div class={col.cardCss || ''}>{content}</div>
									{/each}
								</div>
							{/each}
						</div>
					{/snippet}
				</VList>
			{/if}
		</div>
	{/if}
</div>

<style>
	.qtable-c {
		overflow: hidden;
		position: relative;
		border: 1px solid #dee2e6;
		border-radius: 8px;
		background-color: white;
	}

	.qtable-grid-wrapper {
		width: 100%;
		position: relative;
		overflow-x: auto;
	}

	.qtable-header {
		position: sticky;
		top: 0;
		z-index: 10;
		background-color: #f8f9fa;
		border-bottom: 2px solid #dee2e6;
		min-width: 100%;
	}

	.qtable-body-wrapper {
		width: 100%;
		min-width: 100%;
	}

	.qtable-body-wrapper :global(> div) {
		width: 100% !important;
	}

	.qtable-row {
		display: grid;
		width: 100%;
		min-width: 100%;
		border-bottom: 1px solid #e9ecef;
		transition: background-color 0.15s ease;
	}

	.data-row {
		cursor: pointer;
	}

	.header-row {
		cursor: default;
		font-weight: 600;
		font-size: 0.875rem;
	}

	.qtable-cell {
		padding: 0.5rem 0.75rem;
		display: flex;
		align-items: center;
		overflow: hidden;
		text-overflow: ellipsis;
		min-width: 0; /* Important for text truncation in grid */
	}

	.header-row .qtable-cell {
		font-weight: 600;
		padding: 0.75rem;
		white-space: nowrap;
	}

	.qtable-row.data-row:hover {
		background-color: #f1f3f5;
	}

	.qtable-row.selected {
		background-color: #e7f5ff;
	}

	.qtable-row.tr-even {
		background-color: #ffffff;
	}

	.qtable-row.tr-odd {
		background-color: #f8f9fa;
	}

	.empty-message {
		padding: 2rem;
		text-align: center;
		color: #6c757d;
		font-size: 0.875rem;
	}

	/* Card view styles */
	.qtable-cards {
		overflow: auto;
		position: relative;
		padding: 0.5rem;
	}

	.card-ct {
		background: white;
		border-radius: 0.5rem;
		padding: 0.75rem;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.card-ct:hover {
		box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
	}

	.card-ct.selected {
		background-color: #e7f5ff;
		border: 1px solid #339af0;
	}

	/* Utility classes */
	.w100 {
		width: 100%;
	}

	.h100 {
		height: 100%;
	}

	.flex {
		display: flex;
	}

	.flex-column {
		flex-direction: column;
	}

	.ai-center {
		align-items: center;
	}

	.jc-between {
		justify-content: space-between;
	}

	.p-rel {
		position: relative;
	}

	.mb-04 {
		margin-bottom: 0.5rem;
	}

	._highlight :global(mark) {
		background-color: #ffec99;
		padding: 0 0.125rem;
		border-radius: 2px;
	}
</style>

