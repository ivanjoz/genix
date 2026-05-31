<script lang="ts">
	import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
	import Page from '$domain/Page.svelte';
	import T from '$components/misc/T.svelte';
	import { ProductosService } from '$routes/negocio/productos/productos.svelte';
	import SaleOrdersChartsDailySummary from './SaleOrdersChartsDailySummary.svelte';
	import SaleOrdersChartsByProduct from './SaleOrdersChartsByProduct.svelte';
	import { SaleOrdersChartsService } from './sale_orders_charts.svelte';

	type TChartsView = 1 | 2 | 3;
	type TChartViewOption = [TChartsView, string];

	interface IChartMetricForm {
		metricMode?: 'amount' | 'quantity';
	}

	const saleOrdersChartsService = new SaleOrdersChartsService();
	const productosService = new ProductosService(true);
	const chartViewOptions: TChartViewOption[] = [
		[1, 'By Product|Por Producto'],
		[2, 'Daily Summary|Resumen Diario'],
		[3, 'Weekly Summary|Resumen Semanal']
	];

	let chartMetricForm = $state<IChartMetricForm>({ metricMode: 'amount' });
	let view = $state<TChartsView>(1);
</script>

<Page title="Sales Charts|Gráficos de Ventas" fixedFullHeight={true}>
	<div class="flex h-full min-h-0 flex-col">
		<div class="flex">
			<OptionsStrip
				selected={view} 
				options={chartViewOptions}
				useMobileGrid={true}
				css="mb-10"
				itemCss="md:w-160"
				onSelect={(selectedOption: TChartViewOption) => {
					view = selectedOption[0] as TChartsView;
				}}
			/>
		</div>

		<div class="flex min-h-0 flex-1 flex-col">
		{#if view === 1}
			<SaleOrdersChartsByProduct
				chartMetricForm={chartMetricForm}
				saleSummaryRecords={saleOrdersChartsService.records}
				productsByIdMap={productosService.recordsMap}
			/>
		{:else if view === 2}
			<SaleOrdersChartsDailySummary
				chartMetricForm={chartMetricForm}
				saleSummaryRecords={saleOrdersChartsService.records}
				productsByIdMap={productosService.recordsMap}
			/>
		{:else if view === 3}
			<div class="text-gray-500"><T text="Weekly summary pending.|Resumen Semanal pendiente." /></div>
		{/if}
		</div>
	</div>
</Page>
