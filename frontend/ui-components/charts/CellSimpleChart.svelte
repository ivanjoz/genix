<script lang="ts">
	interface CellSimpleChartProps {
		values: number[];
		labels?: string[];
		// Y base axis: bars are measured up from this value instead of 0.
		minValue?: number;
		// Bar thickness in px; small by design since this lives in a table cell.
		barWidth?: number;
		// Gap between adjacent bars in px.
		barGap?: number;
		barColor?: string;
		// Per-bar color by index; falls back to barColor where empty/undefined.
		barColors?: (string | undefined)[];
		// When set, the value range (min→max) is split into one band per color and
		// each bar takes the color of the band its value falls in. Overrides barColors.
		colorScale?: string[];
		// Minimum bar spacing between two visible labels; labels closer than this are dropped.
		labelGroup?: number;
		labelColor?: string;
	}

	let {
		values,
		labels = [],
		minValue = 0,
		barWidth = 10,
		barGap = 3,
		barColor = '#4874f5',
		barColors = [],
		colorScale = [],
		labelGroup = 1,
		labelColor = '#374151'
	}: CellSimpleChartProps = $props();

	// Map a value to its color band over the actual data range (independent of minValue baseline).
	const scaleMin = $derived(Math.min(...values));
	const scaleSpan = $derived(Math.max(Math.max(...values) - scaleMin, 1e-9));
	const colorForValue = (rawValue: number): string | undefined => {
		if (colorScale.length === 0) return undefined;
		const ratio = (rawValue - scaleMin) / scaleSpan; // 0 at min, 1 at max
		const bandIndex = Math.min(colorScale.length - 1, Math.floor(ratio * colorScale.length));
		return colorScale[bandIndex];
	};

	// Leading/trailing room so the first and last centered labels aren't clipped by overflow: hidden.
	const edgePadding = 8;
	// Bars start after the leading pad; total width is the bar track plus both edge pads.
	const chartWidth = $derived(edgePadding + Math.max(0, values.length * (barWidth + barGap) - barGap) + edgePadding);

	// Tallest bar defines the top reference; minValue is the bottom (y base axis).
	const maxValue = $derived(Math.max(minValue, ...values));
	// Span only guards against a flat dataset dividing by zero; a tiny epsilon keeps
	// the tallest bar at a full 100% even when the real range is well below 1.
	const valueSpan = $derived(Math.max(maxValue - minValue, 1e-9));

	const bars = $derived.by(() => {
		// Track the last shown label so we can keep at least labelGroup bars between labels.
		let lastLabeledIndex = -Infinity;
		return values.map((rawValue, barIndex) => {
			// Clamp to the baseline so values at/below minValue render as empty bars.
			const valueAboveBase = Math.max(0, rawValue - minValue);
			const rawLabel = labels[barIndex] ?? '';
			const keepLabel = rawLabel !== '' && barIndex - lastLabeledIndex >= labelGroup;
			if (keepLabel) lastLabeledIndex = barIndex;
			return {
				barIndex,
				value: rawValue,
				// Bars consume the full vertical space so short cells stay readable.
				heightPercent: (valueAboveBase / valueSpan) * 100,
				// Each bar is absolutely placed by its index, shifted past the leading pad.
				leftPx: edgePadding + barIndex * (barWidth + barGap),
				color: colorForValue(rawValue) || barColors[barIndex] || barColor,
				label: keepLabel ? rawLabel : ''
			};
		});
	});
</script>

<div
	class="cell-simple-chart"
	style={`--cell-chart-label-color: ${labelColor};`}
	style:width={`${chartWidth}px`}
>
	{#each bars as bar (bar.barIndex)}
		<!-- One absolutely-positioned div per bar; height can reach the full cell. -->
		<div class="cell-simple-chart-bar"
			style:left={`${bar.leftPx}px`}
			style:width={`${barWidth}px`}
			style:height={`${bar.heightPercent}%`}
			style:background={bar.color}
			title={String(bar.value)}
		></div>
		{#if bar.label}
			<!-- Label floats just above its bar, but never past the container ceiling (no overflow). -->
			<span class="cell-simple-chart-label text-14"
				style:left={`${bar.leftPx + barWidth / 2}px`}
				style:bottom={`min(${bar.heightPercent}%, calc(100% - 1em))`}
			>{bar.label}</span>
		{/if}
	{/each}
</div>

<style>
	.cell-simple-chart {
		position: relative;
		height: 100%;
		/* Width is set inline from the exact bar count; clip anything past it. */
		overflow: hidden;
	}

	.cell-simple-chart-bar {
		position: absolute;
		bottom: 0;
		border-radius: 2px 2px 0 0;
	}

	.cell-simple-chart-label {
		position: absolute;
		transform: translateX(-50%);
		z-index: 1;
		color: var(--cell-chart-label-color);
		white-space: nowrap;
		line-height: 1;
		pointer-events: none;
	}
</style>
