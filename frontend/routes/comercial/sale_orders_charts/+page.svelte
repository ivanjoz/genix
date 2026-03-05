<script lang="ts">
	import Charts from '$components/Charts.svelte';
	import Page from '$domain/Page.svelte';
	import { formatN, formatTime } from '$libs/helpers';
	import { untrack } from 'svelte';
	import { SaleOrdersChartsService } from './sale_orders_charts.svelte';

	const saleOrdersChartsService = new SaleOrdersChartsService();
	// Keep reference-stable props to avoid re-render churn in Charts.svelte effects.
	const stableLegendConfig = { show: false };
	const stableBarConfig = { width: { ratio: 0.7 } };
	const stableEmptyCategories: Array<string | number> = [];
	const stableEmptyLines: Array<never> = [];

	let chartData = $state({
		columns: [['Ventas'] as Array<string | number>],
		type: 'bar' as const
	});

	let chartAxis = $state({
		x: {
			type: 'category' as const,
			categories: [] as string[]
		},
		y: {
			tick: {
				format: (value: number) => `S/ ${formatN(value, 2)}`
			}
		}
	});
	let lastChartSignature = '';
	let effectRunsCount = 0;
	let chartStateUpdatesCount = 0;

	$effect(() => {
		const summaryRecords = saleOrdersChartsService.records;
		effectRunsCount += 1;
		untrack(() => {
			const chartCategories = summaryRecords.map((summaryRecord) => {
				return String(formatTime(summaryRecord.Fecha, 'd-M') || summaryRecord.Fecha);
			});

			const salesAmountsByDate = summaryRecords.map((summaryRecord) => {
				const totalAmountInCents = (summaryRecord.TotalAmount_32 || []).reduce((accAmount, currentAmount) => {
					return accAmount + (currentAmount || 0);
				}, 0);
				return totalAmountInCents / 100;
			});
			// Skip state writes when chart payload did not change to prevent reactive loops.
			const nextChartSignature = `${chartCategories.join('|')}::${salesAmountsByDate.join('|')}`;
			console.debug('[sale_orders_charts] effect run:', effectRunsCount, {
				rows: summaryRecords.length,
				nextChartSignature,
				lastChartSignature
			});
			if (nextChartSignature === lastChartSignature) {
				console.debug('[sale_orders_charts] skipped chart state update (same signature)');
				return;
			}
			lastChartSignature = nextChartSignature;
			chartStateUpdatesCount += 1;
			console.debug('[sale_orders_charts] applying chart state update:', chartStateUpdatesCount);

			chartData = {
				columns: [['Ventas', ...salesAmountsByDate]],
				type: 'bar'
			};
			chartAxis = {
				x: {
					type: 'category',
					categories: chartCategories
				},
				y: {
					tick: {
						format: (value: number) => `S/ ${formatN(value, 2)}`
					}
				}
			};
		});
	});
</script>

<Page title="Ventas por Fecha">
	<div class="p-10">
		<div class="bg-white border border-gray-200 rounded-md p-10">
			<h3 class="h3 mb-8">Resumen Diario de Ventas</h3>
			{#if saleOrdersChartsService.records.length === 0}
				<div class="text-gray-500">No hay registros de ventas para graficar.</div>
			{:else}
				<Charts
					data={chartData}
					axis={chartAxis}
					categories={stableEmptyCategories}
					lines={stableEmptyLines}
					className="w100"
					style="height: 340px;"
					legend={stableLegendConfig}
					bar={stableBarConfig}
				/>
			{/if}
		</div>
	</div>
</Page>
