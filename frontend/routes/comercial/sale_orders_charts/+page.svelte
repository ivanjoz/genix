<script lang="ts">
	import ChartCanvas, { type ChartCanvasSeries } from '$components/ChartCanvas.svelte';
	import CheckboxOptions from '$components/CheckboxOptions.svelte';
	import OptionsStrip from '$components/OptionsStrip.svelte';
	import Page from '$domain/Page.svelte';
	import { FechaHelper } from '$libs/fecha';
	import { formatN, formatTime } from '$libs/helpers';
	import { untrack } from 'svelte';
	import { ProductosService } from '$routes/negocio/productos/productos.svelte';
	import { SaleOrdersChartsService } from './sale_orders_charts.svelte';

	type TChartsView = 1 | 2 | 3;
	type TChartViewOption = [TChartsView, string];
	type TChartMetricMode = 'amount' | 'quantity';

	interface IChartMetricForm {
		metricMode?: TChartMetricMode;
	}

	interface IProductChartCard {
		productID: number;
		productName: string;
		paidValues: number[];
		unpaidValues: number[];
		priceValues: Array<number | null>;
		priceAxisMin: number;
		priceAxisMax: number;
		totalMetricValue: number;
		totalPaidMetricValue: number;
		totalUnpaidMetricValue: number;
		priceReference: number;
	}

	const saleOrdersChartsService = new SaleOrdersChartsService();
	const productosService = new ProductosService();
	const fechaHelper = new FechaHelper();
	const TOTAL_DAYS_TO_RENDER = 45;
	const chartViewOptions: TChartViewOption[] = [
		[1, 'Por Producto'],
		[2, 'Resumen Diario'],
		[3, 'Resumen Semanal']
	];
	const chartMetricSelectionOptions: Array<{ ID: TChartMetricMode; Nombre: string }> = [
		{ ID: 'amount', Nombre: 'Por Monto Facturado' },
		{ ID: 'quantity', Nombre: 'Por Cantidad' }
	];

	let chartMetricForm = $state<IChartMetricForm>({ metricMode: 'amount' });
	let view = $state<TChartsView>(1);

	const selectedMetricMode = $derived<TChartMetricMode>(chartMetricForm.metricMode || 'amount');

	const getLineAxisRangeWithMargin = (lineValues: Array<number | null>) => {
		const numericLineValues = lineValues.filter((lineValue): lineValue is number => {
			return lineValue !== null && lineValue !== undefined;
		});

		if (!numericLineValues.length) {
			return {
				minValue: 0,
				maxValue: 1
			};
		}

		const minLineValue = Math.min(...numericLineValues);
		const maxLineValue = Math.max(...numericLineValues);
		const lowerMargin = Math.max(Math.abs(minLineValue) * 0.2, 1);
		const upperMargin = Math.max(Math.abs(maxLineValue) * 0.2, 1);

		return {
			minValue: minLineValue - lowerMargin,
			maxValue: maxLineValue + upperMargin
		};
	};

	// Normalize mixed backend date formats to internal "fecha unix" day units.
	const toFechaUnix = (rawFechaValue: number): number => {
		if (!rawFechaValue) return 0;

		if (rawFechaValue > 10_000 && rawFechaValue < 100_000) {
			return Math.floor(rawFechaValue);
		}

		const normalizedFechaValue = rawFechaValue > 1_000_000_000_000
			? Math.floor(rawFechaValue / 1000)
			: rawFechaValue;

		return fechaHelper.toFechaUnix(normalizedFechaValue);
	};

	const last45FechaUnix = $derived.by(() => {
		const latestFechaUnixInSummary = saleOrdersChartsService.records.reduce((latestFechaUnix, summaryRecord) => {
			return Math.max(latestFechaUnix, toFechaUnix(summaryRecord.Fecha || 0));
		}, 0);
		const anchorFechaUnix = latestFechaUnixInSummary || fechaHelper.fechaUnixCurrent();

		// Keep every card aligned to the same oldest->newest window.
		return Array.from({ length: TOTAL_DAYS_TO_RENDER }, (_, dayIndex) => {
			return anchorFechaUnix - (TOTAL_DAYS_TO_RENDER - 1) + dayIndex;
		});
	});

	const chartWindowSummary = $derived.by(() => {
		const firstFechaUnix = last45FechaUnix[0] || 0;
		const lastFechaUnix = last45FechaUnix[last45FechaUnix.length - 1] || 0;

		return {
			firstFechaUnix,
			lastFechaUnix
		};
	});

	const productChartCards = $derived.by((): IProductChartCard[] => {
		const productChartsByID = new Map<number, IProductChartCard>();
		const fechaIndexByUnix = new Map<number, number>();
		let recordsInsideWindowCount = 0;
		let recordsOutsideWindowCount = 0;

		for (let dayIndex = 0; dayIndex < last45FechaUnix.length; dayIndex += 1) {
			fechaIndexByUnix.set(last45FechaUnix[dayIndex], dayIndex);
		}

		for (const summaryRecord of saleOrdersChartsService.records) {
			const recordFechaUnix = toFechaUnix(summaryRecord.Fecha || 0);
			const recordDayIndex = fechaIndexByUnix.get(recordFechaUnix);
			if (recordDayIndex === undefined) {
				recordsOutsideWindowCount += 1;
				continue;
			}
			recordsInsideWindowCount += 1;

			const productIDs = summaryRecord.ProductIDs || [];
			const totalAmountsInCents = summaryRecord.TotalAmount || [];
			const unpaidAmountsInCents = summaryRecord.TotalDebtAmount || [];
			const soldQuantities = summaryRecord.Quantity || [];
			const pendingQuantities = summaryRecord.QuantityPendingDelivery || [];

			for (let recordIndex = 0; recordIndex < productIDs.length; recordIndex += 1) {
				const productID = productIDs[recordIndex] || 0;
				if (!productID) { continue; }

				const productRecord = productosService.recordsMap.get(productID);
				const totalMetricValue = selectedMetricMode === 'quantity'
					? (soldQuantities[recordIndex] || 0)
					: (totalAmountsInCents[recordIndex] || 0) / 100;
				const unpaidMetricValue = selectedMetricMode === 'quantity'
					? (pendingQuantities[recordIndex] || 0)
					: (unpaidAmountsInCents[recordIndex] || 0) / 100;
				const paidMetricValue = Math.max(0, totalMetricValue - unpaidMetricValue);

				let productChartCard = productChartsByID.get(productID);
				if (!productChartCard) {
					const productPrice = (productRecord?.PrecioFinal || productRecord?.Precio || 0) / 100;
					const priceValues = Array.from({ length: TOTAL_DAYS_TO_RENDER }, () => productPrice > 0 ? productPrice : null);
					const priceAxisRange = getLineAxisRangeWithMargin(priceValues);
					productChartCard = {
						productID,
						productName: productRecord?.Nombre || `Producto #${productID}`,
						paidValues: Array.from({ length: TOTAL_DAYS_TO_RENDER }, () => 0),
						unpaidValues: Array.from({ length: TOTAL_DAYS_TO_RENDER }, () => 0),
						priceValues,
						priceAxisMin: priceAxisRange.minValue,
						priceAxisMax: priceAxisRange.maxValue,
						totalMetricValue: 0,
						totalPaidMetricValue: 0,
						totalUnpaidMetricValue: 0,
						priceReference: productPrice
					};
					productChartsByID.set(productID, productChartCard);
				}

				productChartCard.paidValues[recordDayIndex] += paidMetricValue;
				productChartCard.unpaidValues[recordDayIndex] += unpaidMetricValue;
				productChartCard.totalMetricValue += totalMetricValue;
				productChartCard.totalPaidMetricValue += paidMetricValue;
				productChartCard.totalUnpaidMetricValue += unpaidMetricValue;
			}
		}

		const sortedCards = [...productChartsByID.values()].sort((leftCard, rightCard) => {
			return rightCard.totalMetricValue - leftCard.totalMetricValue;
		});

		untrack(() => {
			console.debug('[sale_orders_charts] product cards rebuild', {
				selectedMetricMode,
				totalDays: TOTAL_DAYS_TO_RENDER,
				recordsCount: saleOrdersChartsService.records.length,
				recordsInsideWindowCount,
				recordsOutsideWindowCount,
				cardsCount: sortedCards.length,
				firstVisibleDate: chartWindowSummary.firstFechaUnix,
				lastVisibleDate: chartWindowSummary.lastFechaUnix,
				priceAxisPreview: sortedCards.slice(0, 3).map((productChartCard) => ({
					productID: productChartCard.productID,
					priceReference: productChartCard.priceReference,
					priceAxisMin: productChartCard.priceAxisMin,
					priceAxisMax: productChartCard.priceAxisMax
				}))
			});
		});

		return sortedCards;
	});

	const chartLegendSeries = $derived.by((): ChartCanvasSeries[] => {
		// Reuse the exact palette shown in every card so the legend stays honest.
		return [
			{ name: 'Ventas no pagadas', type: 'bar', values: [1], color: '#ef4444' },
			{ name: 'Ventas pagadas', type: 'bar', values: [1], color: '#3b82f6' },
			{ name: 'Precio', type: 'line', values: [1], color: '#111827', lineWidth: 2 }
		];
	});

	const formatMetricValue = (metricValue: number) => {
		if (!metricValue) { return '0'; }
		return selectedMetricMode === 'quantity'
			? formatN(metricValue, 0)
			: `S/ ${formatN(metricValue, 2)}`;
	};
</script>

<Page title="Gráficos de Ventas">
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

	<div class="p-10">
		{#if view === 1}
			<div class="mb-12 flex flex-wrap items-center justify-between gap-10">
				<CheckboxOptions
					options={chartMetricSelectionOptions}
					saveOn={chartMetricForm}
					save={'metricMode'}
					keyId={'ID'}
					keyName={'Nombre'}
					type="single"
				/>

				<div class="flex items-center gap-12 text-[13px] text-slate-600">
					<div>{productChartCards.length} productos</div>
					<div>{TOTAL_DAYS_TO_RENDER} días</div>
				</div>
			</div>

			{#if saleOrdersChartsService.records.length === 0}
				<div class="text-gray-500">No hay registros de ventas para mostrar.</div>
			{:else if productChartCards.length === 0}
				<div class="text-gray-500">No se encontraron productos con ventas en los últimos 45 días.</div>
			{:else}
				<div class="mb-14 flex flex-wrap items-center gap-12 rounded-[14px] border border-slate-200 bg-white px-14 py-12">
					{#each chartLegendSeries as chartLegendItem (chartLegendItem.name)}
						<div class="flex items-center gap-8 text-[13px] text-slate-600">
							<div class={`h-10 w-18 rounded-[3px] ${chartLegendItem.type === 'line' ? 'bg-slate-900' : ''}`}
								style={chartLegendItem.type === 'bar'
									? `background:${chartLegendItem.color || '#000000'}`
									: 'height:2px;background:#111827'}
							></div>
							<span>{chartLegendItem.name}</span>
						</div>
					{/each}
				</div>

				<div class="grid grid-cols-1 gap-12 md:grid-cols-2 xl:grid-cols-3">
					{#each productChartCards as productChartCard (productChartCard.productID)}
						<div class="rounded-[12px] border border-slate-200 bg-white p-8 shadow-[0_10px_24px_rgba(15,23,42,0.04)]">
							<div class="mb-8">
								<div class="line-clamp-2 font-semibold leading-[1.2] text-slate-800">
									{productChartCard.productName}
								</div>
							</div>

							<div class="mb-8 flex flex-wrap items-center gap-8 rounded-[10px] bg-slate-50 px-8 py-6 text-[14px] leading-none">
								{#if productChartCard.totalPaidMetricValue > 0}
									<div class="flex items-center gap-6 text-slate-800">
										<div class="h-10 w-10 rounded-[3px] bg-[#3b82f6]"></div>
										<div class="font-semibold">{formatMetricValue(productChartCard.totalPaidMetricValue)}</div>
									</div>
								{/if}
								{#if productChartCard.totalUnpaidMetricValue > 0}
									<div class="flex items-center gap-6 text-slate-800">
										<div class="h-10 w-10 rounded-[3px] bg-[#ef4444]"></div>
										<div class="font-semibold">{formatMetricValue(productChartCard.totalUnpaidMetricValue)}</div>
									</div>
								{/if}
								<div class="flex items-center gap-6 text-slate-800">
									<i class="icon-tag text-slate-600"></i>
									<div class="font-semibold">
										{productChartCard.priceReference ? `S/ ${formatN(productChartCard.priceReference, 2)}` : 'Sin precio'}
									</div>
								</div>
							</div>

							<ChartCanvas
								id={`sale-orders-product-${productChartCard.productID}-${selectedMetricMode}`}
								height={80}
								className="h-full w-full"
								dateLabels={last45FechaUnix}
								dateLabelEvery={6}
								dateLabelFormatter={(fechaUnix) => String(formatTime(Number(fechaUnix || 0), 'd-M') || '')}
								useHtmlRendered={true}
								data={[
									{ name: 'Ventas no pagadas', type: 'bar', values: productChartCard.unpaidValues, color: '#ef4444' },
									{ name: 'Ventas pagadas', type: 'bar', values: productChartCard.paidValues, color: '#3b82f6' },
									{
										name: 'Precio',
										type: 'line',
										values: productChartCard.priceValues,
										color: '#111827',
										lineWidth: 1.5,
										yAxisMin: productChartCard.priceAxisMin,
										yAxisMax: productChartCard.priceAxisMax
									}
								]}
							/>
						</div>
					{/each}
				</div>
			{/if}
		{:else if view === 2}
			<div class="text-gray-500">Resumen Diario pendiente.</div>
		{:else if view === 3}
			<div class="text-gray-500">Resumen Semanal pendiente.</div>
		{/if}
	</div>
</Page>
