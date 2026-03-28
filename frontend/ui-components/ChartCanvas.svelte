<script lang="ts" module>
  interface IChartCanvasRectFrame {
    x: number
    y: number
    width: number
    height: number
    fill: string
  }

  interface IChartCanvasLineFrame {
    name: string
    stroke: string
    strokeWidth: number
    segments: number[][]
    pointSize: number
    points: Array<{ x: number, y: number }>
  }

  interface IChartCanvasLineRange {
    minValue: number
    maxValue: number
  }

  interface IChartCanvasCacheEntry {
    renderKey: string
    stackedBarFrames: IChartCanvasRectFrame[]
    lineFrames: IChartCanvasLineFrame[]
  }

  interface IChartCanvasMetrics {
    pointsCount: number
    columnWidth: number
    maxChartValue: number
    plotWidth: number
  }

  interface IChartCanvasYAxisGuide {
    label: string
    top: number
    transform: string
    labelOffsetPx: number
    hideLabel: boolean
  }

  interface IChartCanvasXAxisLabel {
    key: string
    label: string
    left: number
    width: number
    align: "left" | "center" | "right"
  }

  export interface ChartCanvasSeries {
    type: "bar" | "line"
    values: Array<number | null>
    name: string
    color?: string
    lineWidth?: number
    pointSize?: number
    yAxisMin?: number
    yAxisMax?: number
  }

  export interface ChartCanvasProps {
    id?: number | string
    data: ChartCanvasSeries[]
    dateLabels?: Array<string | number>
    dateLabelFormatter?: (dateLabel: string | number, labelIndex: number) => string
    dateLabelEvery?: number
    useHtmlRendered?: boolean
    className?: string
    style?: string
    height?: number
    fixedPointWidthPx?: number
  }

  const sharedChartCache = new Map<string, IChartCanvasCacheEntry>()
</script>

<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte"

  const {
    id = "",
    data = [],
    dateLabels = [],
    dateLabelFormatter = (dateLabel) => String(dateLabel ?? ""),
    dateLabelEvery = 1,
    useHtmlRendered = false,
    className = "h100 w100",
    style = "",
    height = 64,
    fixedPointWidthPx = undefined,
  }: ChartCanvasProps = $props()

  let containerElement = $state<HTMLDivElement | undefined>(undefined)
  let plotFrameElement = $state<HTMLDivElement | undefined>(undefined)
  let plotCanvasElement = $state<HTMLDivElement | undefined>(undefined)
  let measuredWidth = $state(0)
  let resizeObserver: ResizeObserver | undefined
  let chartLeafer: any
  let pendingRenderFrame = 0
  let lastRenderAt = 0
  let queuedRenderTimeout: ReturnType<typeof setTimeout> | undefined
  let lastRenderKey = ""
  let sharedLeaferPromise: Promise<any> | undefined

  const renderThrottleMs = 250
  const yAxisLabelWidthPx = 28
  const yAxisGuideSpacingPx = 16
  const yAxisPaddingTopPx = 2
  const xAxisLabelHeightPx = 18

  const snapRectStart = (value: number) => Math.round(value)
  const snapRectSize = (value: number) => value > 0 ? Math.max(1, Math.round(value)) : 0
  const snapStrokePoint = (value: number, strokeWidth: number) => {
    const roundedValue = Math.round(value)
    return Math.round(strokeWidth) % 2 === 1 ? roundedValue + 0.5 : roundedValue
  }

  const loadLeafer = () => {
    if (!sharedLeaferPromise) {
      sharedLeaferPromise = import("leafer-ui")
    }
    return sharedLeaferPromise
  }

  const updateMetrics = () => {
    if (!plotFrameElement) { return }
    measuredWidth = Math.max(0, Math.floor(plotFrameElement.clientWidth))
  }

  const getChartMetrics = (): IChartCanvasMetrics => {
    const pointsCount = data.reduce((maxCount, chartSeries) => {
      return Math.max(maxCount, chartSeries.values.length)
    }, 0)
    const barSeries = data.filter((chartSeries) => chartSeries.type === 'bar')
    const lineSeries = data.filter((chartSeries) => chartSeries.type === 'line')
    const maxChartValue = Array.from({ length: pointsCount }, (_, pointIndex) => {
      const stackedBarTotal = barSeries.reduce((stackTotal, chartSeries) => {
        return stackTotal + (chartSeries.values[pointIndex] || 0)
      }, 0)
      const lineMaximum = lineSeries.reduce((maxValue, chartSeries) => {
        return Math.max(maxValue, chartSeries.values[pointIndex] || 0, 0)
      }, 0)
      return Math.max(stackedBarTotal, lineMaximum)
    }).reduce((maxValue, stackValue) => Math.max(maxValue, stackValue), 0)

    return {
      pointsCount,
      columnWidth: pointsCount > 0 ? fixedPointWidthPx || (measuredWidth / pointsCount) : 0,
      maxChartValue,
      plotWidth: pointsCount > 0 ? pointsCount * (fixedPointWidthPx || (measuredWidth / pointsCount)) : measuredWidth,
    }
  }

  const formatYAxisLabel = (value: number) => {
    if (value >= 1000) {
      const compactValue = Number((value / 1000).toFixed(1))
      return `${compactValue}k`
    }
    return String(Math.round(value))
  }

  const buildYAxisGuides = (metrics: IChartCanvasMetrics): IChartCanvasYAxisGuide[] => {
    const guideCount = Math.max(1, Math.floor(height / yAxisGuideSpacingPx))
    let hasRenderedZeroLabel = false

    // Keep the HTML grid deterministic with the same vertical scale the canvas uses.
    return Array.from({ length: guideCount }, (_, guideIndex) => {
      const top = yAxisPaddingTopPx + (guideIndex * yAxisGuideSpacingPx)
      const ratioFromBottom = Math.max(0, 1 - (top / height))
      const rawValue = metrics.maxChartValue * ratioFromBottom
      const roundedValue = Math.round(rawValue)
      const hideLabel = roundedValue === 0 && hasRenderedZeroLabel

      if (roundedValue === 0 && !hasRenderedZeroLabel) {
        hasRenderedZeroLabel = true
      }

      return {
        label: formatYAxisLabel(rawValue),
        top,
        transform: guideIndex === 0 ? 'translateY(0)' : 'translateY(-50%)',
        labelOffsetPx: guideIndex === 0 ? -2 : 2,
        hideLabel,
      }
    })
  }

  const buildStackedBarFrames = (metrics: IChartCanvasMetrics): IChartCanvasRectFrame[] => {
    const barSeries = data.filter((chartSeries) => chartSeries.type === 'bar')

    return Array.from({ length: metrics.pointsCount }, (_, pointIndex) => {
      const columnBarEntries = barSeries.flatMap((chartSeries, seriesIndex) => {
        const pointValue = chartSeries.values[pointIndex] || 0
        if (pointValue <= 0) { return [] }

        return [{
          pointValue,
          fill: chartSeries.color || (seriesIndex % 2 === 0 ? '#ef4444' : '#3b82f6'),
        }]
      })
      const usesStackedLayout = columnBarEntries.length > 1
      let stackedHeight = 0

      // Decide per column if we need stacking; most charts end up mixed instead of globally stacked.
      return columnBarEntries.map((columnBarEntry) => {
        const pointValue = columnBarEntry.pointValue
        const frameHeight = metrics.maxChartValue > 0 ? (pointValue / metrics.maxChartValue) * height : 0
        const frameX = snapRectStart(pointIndex * metrics.columnWidth)
        const frameWidth = snapRectSize(Math.max(1, metrics.columnWidth - 1))
        const frameHeightPx = snapRectSize(frameHeight)
        const frameY = usesStackedLayout
          ? snapRectStart(height - stackedHeight - frameHeight)
          : snapRectStart(height - frameHeight)
        const frame = {
          x: frameX,
          y: frameY,
          width: frameWidth,
          height: frameHeightPx,
          fill: columnBarEntry.fill,
        }
        if (usesStackedLayout) {
          stackedHeight += frameHeightPx
        }
        return frame
      })
    }).flat()
  }

  const getLineRange = (chartSeries: ChartCanvasSeries, metrics: IChartCanvasMetrics): IChartCanvasLineRange => {
    const numericValues = chartSeries.values.filter((pointValue): pointValue is number => {
      return pointValue !== null && pointValue !== undefined
    })

    const fallbackMaxValue = metrics.maxChartValue > 0 ? metrics.maxChartValue : 1
    const rawMinValue = chartSeries.yAxisMin ?? (numericValues.length ? Math.min(...numericValues) : 0)
    const rawMaxValue = chartSeries.yAxisMax ?? (numericValues.length ? Math.max(...numericValues) : fallbackMaxValue)

    if (rawMaxValue > rawMinValue) {
      return {
        minValue: rawMinValue,
        maxValue: rawMaxValue,
      }
    }

    // Keep constant-value lines visible by expanding a tiny local range around the value.
    const centerValue = rawMaxValue || rawMinValue || 0
    const paddingValue = Math.max(Math.abs(centerValue) * 0.1, 1)
    return {
      minValue: centerValue - paddingValue,
      maxValue: centerValue + paddingValue,
    }
  }

  const getLinePointY = (pointValue: number, lineRange: IChartCanvasLineRange) => {
    const valueSpan = lineRange.maxValue - lineRange.minValue
    if (valueSpan <= 0) { return height / 2 }

    const normalizedPointValue = Math.min(lineRange.maxValue, Math.max(lineRange.minValue, pointValue))
    return height - (((normalizedPointValue - lineRange.minValue) / valueSpan) * height)
  }

  const buildLineFrames = (metrics: IChartCanvasMetrics): IChartCanvasLineFrame[] => {
    const lineSeries = data.filter((chartSeries) => chartSeries.type === 'line')

    return lineSeries.map((chartSeries) => {
      const segments: Array<number[]> = []
      let currentSegment: number[] = []
      const lineRange = getLineRange(chartSeries, metrics)
      const strokeWidth = chartSeries.lineWidth || 2

      for (let pointIndex = 0; pointIndex < chartSeries.values.length; pointIndex += 1) {
        const pointValue = chartSeries.values[pointIndex]
        if (pointValue === null || pointValue === undefined) {
          if (currentSegment.length >= 4) { segments.push(currentSegment) }
          currentSegment = []
          continue
        }

        const x = snapStrokePoint((pointIndex * metrics.columnWidth) + (metrics.columnWidth / 2), strokeWidth)
        const y = snapStrokePoint(getLinePointY(pointValue, lineRange), strokeWidth)
        currentSegment.push(x, y)
      }

      if (currentSegment.length >= 4) { segments.push(currentSegment) }

      const lineFrame: IChartCanvasLineFrame = {
        name: chartSeries.name,
        stroke: chartSeries.color || '#0f172a',
        strokeWidth,
        segments,
        pointSize: chartSeries.pointSize || 0,
        points: chartSeries.values.flatMap((pointValue, pointIndex) => {
          if (pointValue === null || pointValue === undefined) { return [] }
          const x = snapStrokePoint((pointIndex * metrics.columnWidth) + (metrics.columnWidth / 2), strokeWidth)
          const y = snapStrokePoint(getLinePointY(pointValue, lineRange), strokeWidth)
          return [{ x, y }]
        }),
      }
      return lineFrame
    })
  }

  const yAxisGuides = $derived.by(() => {
    return buildYAxisGuides(getChartMetrics())
  })

  const chartMetrics = $derived.by(() => {
    // Share one metrics object across HTML guides and the canvas width calculation.
    return getChartMetrics()
  })

  const xAxisLabels = $derived.by((): IChartCanvasXAxisLabel[] => {
    if (!dateLabels.length || chartMetrics.pointsCount <= 0) { return [] }

    const labelStep = Math.max(1, Math.floor(dateLabelEvery || 1))
    const visibleLabels: IChartCanvasXAxisLabel[] = []

    for (let labelIndex = 0; labelIndex < Math.min(dateLabels.length, chartMetrics.pointsCount); labelIndex += labelStep) {
      const spanPointsCount = Math.min(labelStep, chartMetrics.pointsCount - labelIndex)
      visibleLabels.push({
        key: `${labelIndex}-${String(dateLabels[labelIndex] ?? "")}`,
        label: dateLabelFormatter(dateLabels[labelIndex], labelIndex),
        left: labelIndex * chartMetrics.columnWidth,
        width: spanPointsCount * chartMetrics.columnWidth,
        align: labelIndex === 0
          ? "left"
          : labelIndex + spanPointsCount >= chartMetrics.pointsCount
            ? "right"
            : "center",
      })
    }

    return visibleLabels
  })

  const stackedBarFrames = $derived.by(() => {
    return buildStackedBarFrames(chartMetrics)
  })

  const lineFrames = $derived.by(() => {
    return buildLineFrames(chartMetrics)
  })

  const getRenderKey = () => {
    return JSON.stringify({
      measuredWidth,
      height,
      id,
      fixedPointWidthPx: fixedPointWidthPx || 0,
      dateLabels,
      dateLabelEvery,
      useHtmlRendered,
      series: data.map((chartSeries) => ({
        type: chartSeries.type,
        name: chartSeries.name,
        color: chartSeries.color || "",
        lineWidth: chartSeries.lineWidth || 0,
        pointSize: chartSeries.pointSize || 0,
        yAxisMin: chartSeries.yAxisMin ?? null,
        yAxisMax: chartSeries.yAxisMax ?? null,
        values: chartSeries.values,
      })),
    })
  }

  const renderChart = async () => {
    await tick()
    if (!containerElement || !plotCanvasElement || measuredWidth <= 0) { return }

    const nextRenderKey = getRenderKey()
    if (nextRenderKey === lastRenderKey) { return }

    const { Leafer, Rect, Line, Ellipse } = await loadLeafer()
    const metrics = chartMetrics
    const cacheID = String(id || '')
    const cachedChart = cacheID ? sharedChartCache.get(cacheID) : undefined
    const useCachedFrames = cachedChart?.renderKey === nextRenderKey
    const nextStackedBarFrames = useCachedFrames ? cachedChart.stackedBarFrames : buildStackedBarFrames(metrics)
    const nextLineFrames = useCachedFrames ? cachedChart.lineFrames : buildLineFrames(metrics)

    if (cacheID && !useCachedFrames) {
      sharedChartCache.set(cacheID, {
        renderKey: nextRenderKey,
        stackedBarFrames: nextStackedBarFrames,
        lineFrames: nextLineFrames,
      })
    }

    chartLeafer?.destroy()
    chartLeafer = new Leafer({
      view: plotCanvasElement,
      width: metrics.plotWidth,
      height,
      fill: 'transparent',
    })

    // Rebuild from the flat frame list so updates stay deterministic and minimal.
    if (!useHtmlRendered) {
      for (const stackedBarFrame of nextStackedBarFrames) {
        if (!stackedBarFrame.height) { continue }
        chartLeafer.add(new Rect(stackedBarFrame))
      }
    }

    for (const lineFrame of nextLineFrames) {
      for (const segment of lineFrame.segments) {
        if (segment.length < 4) { continue }
        chartLeafer.add(new Line({
          points: segment,
          stroke: lineFrame.stroke,
          strokeWidth: lineFrame.strokeWidth,
        }))
      }

      for (const point of lineFrame.points) {
        if (!lineFrame.pointSize) { continue }
        chartLeafer.add(new Ellipse({
          x: point.x - lineFrame.pointSize,
          y: point.y - lineFrame.pointSize,
          width: lineFrame.pointSize * 2,
          height: lineFrame.pointSize * 2,
          fill: lineFrame.stroke,
        }))
      }
    }

    lastRenderKey = nextRenderKey
  }

  const scheduleRender = () => {
    if (pendingRenderFrame) { return }

    pendingRenderFrame = requestAnimationFrame(() => {
      pendingRenderFrame = 0

      const now = Date.now()
      const waitTime = renderThrottleMs - (now - lastRenderAt)
      if (waitTime > 0) {
        if (queuedRenderTimeout) { clearTimeout(queuedRenderTimeout) }
        queuedRenderTimeout = setTimeout(() => {
          lastRenderAt = Date.now()
          void renderChart()
        }, waitTime)
        return
      }

      lastRenderAt = now
      void renderChart()
    })
  }

  onMount(async () => {
    updateMetrics()

    if (containerElement) {
      resizeObserver = new ResizeObserver(() => {
        updateMetrics()
        scheduleRender()
      })
      resizeObserver.observe(containerElement)
    }

    scheduleRender()
  })

  $effect(() => {
    data
    dateLabels
    dateLabelEvery
    useHtmlRendered
    className
    style
    height
    scheduleRender()
  })

  onDestroy(() => {
    resizeObserver?.disconnect()
    if (pendingRenderFrame) { cancelAnimationFrame(pendingRenderFrame) }
    if (queuedRenderTimeout) { clearTimeout(queuedRenderTimeout) }
    chartLeafer?.destroy()
    chartLeafer = undefined
  })
</script>

<div bind:this={containerElement} class={className} style={`${style};height:${height + (xAxisLabels.length ? xAxisLabelHeightPx : 0)}px`}>
  <div class="relative flex h-full w-full min-w-0 flex-col">
    <div class="relative flex min-h-0 flex-1 w-full min-w-0">
    <div class="relative shrink-0 text-right text-[12px] leading-none text-slate-500" style={`width:${yAxisLabelWidthPx}px`}>
      {#each yAxisGuides as yAxisGuide (yAxisGuide.top)}
        {#if !yAxisGuide.hideLabel}
          <div class="pointer-events-none absolute right-0 pr-4 [&>div]:block" style={`top:${yAxisGuide.top + yAxisGuide.labelOffsetPx}px;transform:${yAxisGuide.transform}`}>
            <div>{yAxisGuide.label}</div>
          </div>
        {/if}
      {/each}
    </div>

    <div class="relative h-full min-w-0 flex-1 overflow-hidden" bind:this={plotFrameElement}>
      <div class="absolute inset-0 [&>div]:pointer-events-none [&>div]:absolute [&>div]:left-0 [&>div]:right-0 [&>div]:border-t [&>div]:border-dashed [&>div]:border-slate-300/80">
        {#each yAxisGuides as yAxisGuide (yAxisGuide.top)}
          <div style={`top:${yAxisGuide.top}px`}></div>
        {/each}
      </div>

      {#if useHtmlRendered}
        <!-- Render bars in the DOM so narrow stacked columns can use the browser's pixel snapping. -->
        <div class="absolute inset-y-0 [&>div]:pointer-events-none [&>div]:absolute" style={`width:${chartMetrics.plotWidth}px;right:0`}>
          {#each stackedBarFrames as stackedBarFrame, frameIndex (`${frameIndex}-${stackedBarFrame.x}-${stackedBarFrame.y}-${stackedBarFrame.height}`)}
            {#if stackedBarFrame.height}
              <div
                style={`left:${stackedBarFrame.x}px;top:${stackedBarFrame.y}px;width:${stackedBarFrame.width}px;height:${stackedBarFrame.height}px;background:${stackedBarFrame.fill}`}
              ></div>
            {/if}
          {/each}
        </div>
      {/if}

      <div
        bind:this={plotCanvasElement}
        class="absolute inset-y-0"
        style={`width:${chartMetrics.plotWidth}px;right:0`}
      ></div>
    </div>
    </div>

    {#if xAxisLabels.length}
      <div class="relative mt-2 flex w-full min-w-0">
        <div class="shrink-0" style={`width:${yAxisLabelWidthPx}px`}></div>
        <div class="relative min-w-0 flex-1 overflow-hidden" style={`height:${xAxisLabelHeightPx}px`}>
          <div class="absolute inset-y-0 [&>div]:pointer-events-none [&>div]:absolute [&>div]:overflow-hidden [&>div]:text-[11px] [&>div]:uppercase [&>div]:leading-none [&>div]:text-slate-500 [&>div]:whitespace-nowrap" style={`width:${chartMetrics.plotWidth}px;right:0`}>
            {#each xAxisLabels as xAxisLabel (xAxisLabel.key)}
              <div
                class={`${xAxisLabel.align === 'left' ? '[&>div]:text-left' : xAxisLabel.align === 'right' ? '[&>div]:text-right' : '[&>div]:text-center'}`}
                style={`left:${xAxisLabel.left}px;width:${xAxisLabel.width}px`}
              >
                <div>{xAxisLabel.label}</div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
