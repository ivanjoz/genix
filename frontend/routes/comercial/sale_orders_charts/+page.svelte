<script lang="ts">
	import Checkbox from '$components/Checkbox.svelte';
	import CheckboxOptions from '$components/CheckboxOptions.svelte';
	import CellHorizontalBars from '$components/charts/CellHorizontalBars.svelte';
	import OptionsStrip from '$components/OptionsStrip.svelte';
	import TableGrid from '$components/vTable/TableGrid.svelte';
	import type { TableGridColumn } from '$components/vTable/tableGridTypes';
	import Page from '$domain/Page.svelte';
	import { FechaHelper } from '$libs/fecha';
	import { formatN, formatTime } from '$libs/helpers';
	import { untrack } from 'svelte';
	import { ProductosService } from '$routes/negocio/productos/productos.svelte';
	import { SaleOrdersChartsService } from './sale_orders_charts.svelte';

	type TChartsView = 1 | 2 | 3;
	type TChartViewOption = [TChartsView, string];

	interface IWeeklySalesColumn {
		key: string;
		label: string;
		weekStartFechaUnix: number;
	}

	interface IProductWeeklySalesRow {
		productID: number;
		productName: string;
		weeklyTotalsByWeekKey: Record<string, Array<[number, number]>>;
		totalAmount: number;
	}

	type TChartMetricMode = 'amount' | 'quantity';

	interface IChartMetricForm {
		metricMode?: TChartMetricMode;
		useLogScale?: boolean;
	}

	const saleOrdersChartsService = new SaleOrdersChartsService();
	const productosService = new ProductosService();
	const fechaHelper = new FechaHelper();
	const TOTAL_WEEKS_TO_RENDER = 10;
	const LOG_SCALE_FACTOR = 0.08;
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
	const logScaleFactor = $derived(chartMetricForm.useLogScale ? LOG_SCALE_FACTOR : 0);

	// FechaHelper range helpers operate with internal offset-based week code (YYYYWW-like).
	// Domain week code arrives as YYWW (e.g. 2610 => week 10 of 2026), so normalize it.
	const toRangeWeekCode = (domainWeekCode: number): number => {
		if (!domainWeekCode) return 0;
		return domainWeekCode < 10000 ? domainWeekCode + 200000 : domainWeekCode;
	};

	// Normalize mixed backend date formats to internal "fecha unix" day units.
	const toFechaUnix = (rawFechaValue: number): number => {
		if (!rawFechaValue) return 0;

		// Some routes already expose fechaUnix day values directly (around 10k-100k).
		if (rawFechaValue > 10_000 && rawFechaValue < 100_000) {
			return Math.floor(rawFechaValue);
		}

		// If backend sends milliseconds, convert to seconds before FechaHelper.toFechaUnix.
		const normalizedFechaValue = rawFechaValue > 1_000_000_000_000
			? Math.floor(rawFechaValue / 1000)
			: rawFechaValue;

		return fechaHelper.toFechaUnix(normalizedFechaValue);
	};

	const weekColumns = $derived.by((): IWeeklySalesColumn[] => {
		const summaryRecords = saleOrdersChartsService.records;
		const latestFechaUnixInSummary = summaryRecords.reduce((latestFechaUnix, summaryRecord) => {
			const recordFechaUnix = toFechaUnix(summaryRecord.Fecha || 0);
			return Math.max(latestFechaUnix, recordFechaUnix);
		}, 0);

		// Anchor the 10-week window to the latest available sales week, not always "today".
		const anchorWeek = latestFechaUnixInSummary > 0
			? fechaHelper.toWeek(latestFechaUnixInSummary)
			: fechaHelper.weekCurrent();
		const rangeAnchorWeekCode = toRangeWeekCode(anchorWeek.code);

		const weeksRange = fechaHelper.makeWeekRangeFromWeek(
			rangeAnchorWeekCode,
			0,
			TOTAL_WEEKS_TO_RENDER
		);

		const generatedColumns = weeksRange.map((currentSemana) => {
			const weekStartFechaUnix = currentSemana.fechaUnix;
			return {
				key: `week-${weekStartFechaUnix}`,
				// Show only week start label.
				label: `${formatTime(weekStartFechaUnix, 'd-M')}`,
				weekStartFechaUnix
			};
		});
		// Display columns from most recent week to oldest week.
		generatedColumns.sort((leftWeek, rightWeek) => rightWeek.weekStartFechaUnix - leftWeek.weekStartFechaUnix);

		console.debug('[sale_orders_charts] generated week columns', {
			anchorWeekCode: anchorWeek.code,
			rangeAnchorWeekCode,
			latestFechaUnixInSummary,
			summaryRecordsCount: summaryRecords.length,
			totalColumns: generatedColumns.length,
			weekStartUnixList: generatedColumns.map((weekColumn) => weekColumn.weekStartFechaUnix),
			weekLabels: generatedColumns.map((weekColumn) => weekColumn.label)
		});

		return generatedColumns;
	});

	const weeklySalesRows = $derived.by((): IProductWeeklySalesRow[] => {
		const summaryRecords = saleOrdersChartsService.records;
		const productsByIdMap = productosService.recordsMap;
		const weekColumnsByStartFechaUnix = new Map<number, IWeeklySalesColumn>();
		const productRowsByID = new Map<number, IProductWeeklySalesRow>();
		let matchedSummaryRecordsCount = 0;
		let unmatchedSummaryRecordsCount = 0;
		const unmatchedRecordsPreview: Array<{
			fechaRaw: number;
			fechaUnix: number;
			weekStartFechaUnix: number;
			weekCode: number;
		}> = [];
		const normalizationPreview: Array<{
			fechaRaw: number;
			fechaUnix: number;
			weekStartFechaUnix: number;
			weekCode: number;
		}> = [];

		for (const weekColumn of weekColumns) {
			weekColumnsByStartFechaUnix.set(weekColumn.weekStartFechaUnix, weekColumn);
		}

		for (const summaryRecord of summaryRecords) {
			const recordFechaUnix = toFechaUnix(summaryRecord.Fecha || 0);
			const recordWeek = fechaHelper.toWeek(recordFechaUnix);
			const recordWeekStartFechaUnix = recordWeek.fechaUnix;
			const matchingWeekColumn = weekColumnsByStartFechaUnix.get(recordWeekStartFechaUnix);
			if (normalizationPreview.length < 10) {
				normalizationPreview.push({
					fechaRaw: summaryRecord.Fecha || 0,
					fechaUnix: recordFechaUnix,
					weekStartFechaUnix: recordWeekStartFechaUnix,
					weekCode: recordWeek.code
				});
			}
			if (!matchingWeekColumn) {
				unmatchedSummaryRecordsCount += 1;
				if (unmatchedRecordsPreview.length < 10) {
					unmatchedRecordsPreview.push({
						fechaRaw: summaryRecord.Fecha || 0,
						fechaUnix: recordFechaUnix,
						weekStartFechaUnix: recordWeekStartFechaUnix,
						weekCode: recordWeek.code
					});
				}
				continue;
			}
			matchedSummaryRecordsCount += 1;

			const productIDs = summaryRecord.ProductIDs || [];
			const weeklyAmountsInCents = summaryRecord.TotalAmount || [];
			const weeklyUnpaidAmountsInCents = summaryRecord.TotalDebtAmount || [];
			const weeklySoldQuantities = summaryRecord.Quantity || [];
			const weeklyPendingQuantities = summaryRecord.QuantityPendingDelivery || [];

			for (let recordIndex = 0; recordIndex < productIDs.length; recordIndex += 1) {
				const productID = productIDs[recordIndex] || 0;
				if (!productID) continue;

				const rowAmountInCurrency = (weeklyAmountsInCents[recordIndex] || 0) / 100;
				let productRow = productRowsByID.get(productID);

				if (!productRow) {
					const productRecord = productsByIdMap.get(productID);
					// Initialize all 7 daily slots per week so zero-sales days still render as empty bars.
					const rowTotalsByWeekKey: Record<string, Array<[number, number]>> = {};
					for (const weekColumn of weekColumns) {
						rowTotalsByWeekKey[weekColumn.key] = Array.from({ length: 7 }, () => [0, 0] as [number, number]);
					}

					productRow = {
						productID,
						productName: productRecord?.Nombre || `Producto #${productID}`,
						weeklyTotalsByWeekKey: rowTotalsByWeekKey,
						totalAmount: 0
					};
					productRowsByID.set(productID, productRow);
				}

				// Resolve which day (0..6) inside the week this summary record belongs to.
				// Keep metric switch centralized so amount/quantity modes use the same rendering pipeline.
				const metricTotalValue = selectedMetricMode === 'quantity'
					? (weeklySoldQuantities[recordIndex] || 0)
					: rowAmountInCurrency;
				const metricPendingValue = selectedMetricMode === 'quantity'
					? (weeklyPendingQuantities[recordIndex] || 0)
					: (weeklyUnpaidAmountsInCents[recordIndex] || 0) / 100;
				const dayIndexInWeek = Math.max(0, Math.min(6, recordFechaUnix - recordWeekStartFechaUnix));
				productRow.weeklyTotalsByWeekKey[matchingWeekColumn.key][dayIndexInWeek][0] += metricTotalValue;
				productRow.weeklyTotalsByWeekKey[matchingWeekColumn.key][dayIndexInWeek][1] += metricPendingValue;
				productRow.totalAmount += metricTotalValue;
			}
		}
		console.debug('[sale_orders_charts] week match stats', {
			totalSummaryRecords: summaryRecords.length,
			selectedMetricMode,
			matchedSummaryRecordsCount,
			unmatchedSummaryRecordsCount,
			weekColumnsCount: weekColumns.length,
			normalizationPreview,
			unmatchedRecordsPreview
		});

		return [...productRowsByID.values()].sort((leftRow, rightRow) => {
			return rightRow.totalAmount - leftRow.totalAmount;
		});
	});

	const weeklyBarsMaxValue = $derived.by(() => {
		const maxTotalSales = weeklySalesRows.reduce((maxValue, weeklySalesRow) => {
			for (const weekColumn of weekColumns) {
				const currentWeekDailyTotals = weeklySalesRow.weeklyTotalsByWeekKey[weekColumn.key] || [];
				for (const [dailyTotalSales] of currentWeekDailyTotals) {
					maxValue = Math.max(maxValue, dailyTotalSales || 0);
				}
			}
			return maxValue;
		}, 0);

		console.debug('[sale_orders_charts] weekly bars max value', {
			maxTotalSales,
			selectedMetricMode,
			logScaleFactor,
			rowsCount: weeklySalesRows.length,
			weeksCount: weekColumns.length
		});

		return maxTotalSales;
	});

	const tableColumns = $derived.by((): TableGridColumn<IProductWeeklySalesRow>[] => {
		const baseColumns: TableGridColumn<IProductWeeklySalesRow>[] = [
			{
				id: 'productName', cellCss: "py-2 px-6",
				header: 'Producto',
				width: 'minmax(280px, 1.8fr)',
				getValue: (rowRecord) => rowRecord.productName
			}
		];

		for (const weekColumn of weekColumns) {
			baseColumns.push({
				id: weekColumn.key,
				header: weekColumn.label,
				width: '100px',
				useCellRenderer: true,
				align: 'right',
				getValue: (rowRecord) => {
					// Keep weekly sum as cell title fallback while rendering daily bars in the snippet.
					const weekAmount = (rowRecord.weeklyTotalsByWeekKey[weekColumn.key] || []).reduce((sumAmount, [dailyTotal]) => {
						return sumAmount + (dailyTotal || 0);
					}, 0);
					if (selectedMetricMode === 'quantity') {
						return formatN(weekAmount, 0);
					}
					return `S/ ${formatN(weekAmount, 2)}`;
				}
			});
		}

		return baseColumns;
	});

	let effectRunsCount = 0;
	$effect(() => {
		const recordsCount = saleOrdersChartsService.records.length;
		const rowsCount = weeklySalesRows.length;
		const weeksCount = weekColumns.length;

		untrack(() => {
			effectRunsCount += 1;
			console.debug('[sale_orders_charts] weekly table rebuild', {
				effectRunsCount,
				selectedMetricMode,
				logScaleFactor,
				recordsCount,
				rowsCount,
				weeksCount
			});
		});
	});
</script>

<Page title="Gráficos de Ventas">
	<div class="flex">
		<OptionsStrip 
			selected={view}
			options={chartViewOptions}
			useMobileGrid={true} 
			css="mb-10" itemCss="md:w-160"
			onSelect={(selectedOption: TChartViewOption) => {
				// Keep the tab state explicit so each chart section renders independently.
				view = selectedOption[0] as TChartsView;
			}}
		/>
	</div>
	<div class="p-10">
		{#if view === 1}
			<div>
				<div class="flex ">
					<CheckboxOptions
						options={chartMetricSelectionOptions}
						saveOn={chartMetricForm}
						save={'metricMode'}
						keyId={'ID'}
						keyName={'Nombre'}
						type="single"
						css="mb-10"
					/>
					<div class="mr-8 ml-8">|</div>
					<Checkbox label="Escala Log." bind:saveOn={chartMetricForm} save="useLogScale" css="mb-10" />
				</div>
				{#if saleOrdersChartsService.records.length === 0}
					<div class="text-gray-500">No hay registros de ventas para mostrar.</div>
				{:else if weeklySalesRows.length === 0}
					<div class="text-gray-500">No se encontraron productos con ventas en las últimas 10 semanas.</div>
				{:else}
					<TableGrid
						columns={tableColumns}
						data={weeklySalesRows}
						height="560px"
						rowHeight={48}
					>
						{#snippet cellRenderer(rowRecord, columnDefinition)}
							{@const cellValues = rowRecord.weeklyTotalsByWeekKey[String(columnDefinition.id)] || Array.from({ length: 7 }, () => [0, 0] as [number, number])}
							{@const weeklyTotalSales = cellValues.reduce((sumAmount, [dailyTotalSales]) => {
								return sumAmount + (dailyTotalSales || 0);
							}, 0)}
							<div class="relative h-full w-full pt-10 pr-2">
								<div class="absolute top-2 right-2 text-[13px] leading-none font-semibold text-slate-700">
									{weeklyTotalSales ? formatN(weeklyTotalSales) : ""}
								</div>
								<CellHorizontalBars
									values={cellValues}
									maxValue={weeklyBarsMaxValue}
									logScaleFactor={logScaleFactor}
								/>
							</div>
						{/snippet}
					</TableGrid>
				{/if}
			</div>
		{:else if view === 2}
			<div class="text-gray-500">Resumen Diario pendiente.</div>
		{:else if view === 3}
			<div class="text-gray-500">Resumen Semanal pendiente.</div>
		{/if}
	</div>
</Page>
