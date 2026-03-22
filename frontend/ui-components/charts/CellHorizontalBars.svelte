<script lang="ts">
	type CellHorizontalBarsEntry = [number, number] | readonly [number, number] | readonly number[];

	interface CellHorizontalBarsProps {
		values: CellHorizontalBarsEntry[];
		maxValue: number;
		logScaleFactor?: number;
		totalBarColor?: string;
		pendingBarColor?: string;
	}

	let {
		values,
		maxValue,
		logScaleFactor = 0,
		totalBarColor = '#4874f5',
		pendingBarColor = '#e67676'
	}: CellHorizontalBarsProps = $props();

	// Keep percentage computation centralized so the caller can tune log compression strength.
	const calculateWidthPercent = (rawValue: number): number => {
		const safeMaxValue = maxValue > 0 ? maxValue : 1;
		const normalizedValue = Math.max(0, rawValue);
		const normalizedLogScaleFactor = Math.max(0, logScaleFactor);
		if (normalizedLogScaleFactor > 0) {
			const scaledValue = Math.log1p(normalizedValue * normalizedLogScaleFactor);
			const scaledMaxValue = Math.log1p(safeMaxValue * normalizedLogScaleFactor);
			return Math.min(100, (scaledValue / (scaledMaxValue || 1)) * 100);
		}
		return Math.min(100, (normalizedValue / safeMaxValue) * 100);
	};

	// Bars count is fully dynamic and driven by incoming values.
	const barsCount = $derived(Math.max(1, values.length));
	const barHeightPercent = $derived(100 / barsCount);

	const rowsWithPercents = $derived.by(() => {
		return values.map((entryValues, pairIndex) => {
			// Normalize broad input shapes to a strict [total, pending] numeric pair.
			const totalSales = Number(entryValues?.[0] || 0);
			const unpaidSales = Number(entryValues?.[1] || 0);
			const normalizedTotalSales = Math.max(0, totalSales);
			const normalizedUnpaidSales = Math.min(normalizedTotalSales, Math.max(0, unpaidSales));
			return {
				pairIndex,
				totalSales: normalizedTotalSales,
				unpaidSales: normalizedUnpaidSales,
				totalWidthPercent: calculateWidthPercent(normalizedTotalSales),
				unpaidWidthPercent: calculateWidthPercent(normalizedUnpaidSales)
			};
		});
	});
</script>

<div class="weekly-sales-bars-flat"
	style={`--weekly-sales-total-color: ${totalBarColor}; --weekly-sales-pending-color: ${pendingBarColor};`}
>
	{#each rowsWithPercents as chartRow, chartRowIndex (chartRow.pairIndex)}
		<div class="weekly-sales-bars-total"
			style={`top: ${barHeightPercent * chartRowIndex}%; height: ${barHeightPercent}%;`}
			title={`Total: ${Math.round(chartRow.totalSales)}`}
			style:width={`${chartRow.totalWidthPercent}%`}
		></div>
		<div class="weekly-sales-bars-unpaid"
			style={`top: ${barHeightPercent * chartRowIndex}%; height: ${barHeightPercent}%;`}
			title={`Pendiente: ${Math.round(chartRow.unpaidSales)}`}
			style:width={`${chartRow.unpaidWidthPercent}%`}
		></div>
	{/each}
</div>

<style>
	.weekly-sales-bars-flat {
		position: relative;
		width: 100%;
		height: 100%;
	}

	.weekly-sales-bars-total,
	.weekly-sales-bars-unpaid {
		position: absolute;
		left: 0;
	}

	.weekly-sales-bars-total {
		background: var(--weekly-sales-total-color);
		opacity: 0.9;
		z-index: 1;
	}

	.weekly-sales-bars-unpaid {
		background: var(--weekly-sales-pending-color);
		z-index: 2;
	}
</style>
