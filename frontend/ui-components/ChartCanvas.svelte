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

  export interface ChartCanvasSeries {
    type: "bar" | "line"
    values: Array<number | null>
    name: string
    color?: string
    lineWidth?: number
    pointSize?: number
  }

  export interface ChartCanvasProps {
    id?: number | string
    data: ChartCanvasSeries[]
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
      let stackedHeight = 0

      // Keep each series section in one column so the mini chart stays cheap to draw.
      return barSeries.map((chartSeries, seriesIndex) => {
        const pointValue = chartSeries.values[pointIndex] || 0
        const frameHeight = metrics.maxChartValue > 0 ? (pointValue / metrics.maxChartValue) * height : 0
        const frame = {
          x: pointIndex * metrics.columnWidth,
          y: height - stackedHeight - frameHeight,
          width: Math.max(1, metrics.columnWidth - 1),
          height: frameHeight,
          fill: chartSeries.color || (seriesIndex % 2 === 0 ? '#ef4444' : '#3b82f6'),
        }
        stackedHeight += frameHeight
        return frame
      })
    }).flat()
  }

  const buildLineFrames = (metrics: IChartCanvasMetrics): IChartCanvasLineFrame[] => {
    const lineSeries = data.filter((chartSeries) => chartSeries.type === 'line')

    return lineSeries.map((chartSeries) => {
      const segments: Array<number[]> = []
      let currentSegment: number[] = []

      for (let pointIndex = 0; pointIndex < chartSeries.values.length; pointIndex += 1) {
        const pointValue = chartSeries.values[pointIndex]
        if (pointValue === null || pointValue === undefined) {
          if (currentSegment.length >= 4) { segments.push(currentSegment) }
          currentSegment = []
          continue
        }

        const x = (pointIndex * metrics.columnWidth) + (metrics.columnWidth / 2)
        const clampedPointValue = Math.max(0, pointValue)
        const y = metrics.maxChartValue > 0 ? height - ((clampedPointValue / metrics.maxChartValue) * height) : height
        currentSegment.push(x, y)
      }

      if (currentSegment.length >= 4) { segments.push(currentSegment) }

      const lineFrame: IChartCanvasLineFrame = {
        name: chartSeries.name,
        stroke: chartSeries.color || '#0f172a',
        strokeWidth: chartSeries.lineWidth || 2,
        segments,
        pointSize: chartSeries.pointSize || 0,
        points: chartSeries.values.flatMap((pointValue, pointIndex) => {
          if (pointValue === null || pointValue === undefined) { return [] }
          const x = (pointIndex * metrics.columnWidth) + (metrics.columnWidth / 2)
          const clampedPointValue = Math.max(0, pointValue)
          const y = metrics.maxChartValue > 0 ? height - ((clampedPointValue / metrics.maxChartValue) * height) : height
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

  const getRenderKey = () => {
    return JSON.stringify({
      measuredWidth,
      height,
      id,
      fixedPointWidthPx: fixedPointWidthPx || 0,
      series: data.map((chartSeries) => ({
        type: chartSeries.type,
        name: chartSeries.name,
        color: chartSeries.color || "",
        lineWidth: chartSeries.lineWidth || 0,
        pointSize: chartSeries.pointSize || 0,
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
    const stackedBarFrames = useCachedFrames ? cachedChart.stackedBarFrames : buildStackedBarFrames(metrics)
    const lineFrames = useCachedFrames ? cachedChart.lineFrames : buildLineFrames(metrics)

    if (cacheID && !useCachedFrames) {
      sharedChartCache.set(cacheID, {
        renderKey: nextRenderKey,
        stackedBarFrames,
        lineFrames,
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
    for (const stackedBarFrame of stackedBarFrames) {
      if (!stackedBarFrame.height) { continue }
      chartLeafer.add(new Rect(stackedBarFrame))
    }

    for (const lineFrame of lineFrames) {
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

<div bind:this={containerElement} class={className} style={`${style};height:${height}px`}>
  <div class="relative flex h-full w-full min-w-0">
    <div class="relative shrink-0 text-right text-[12px] leading-none text-slate-500" style={`width:${yAxisLabelWidthPx}px`}>
      {#each yAxisGuides as yAxisGuide (yAxisGuide.top)}
        {#if !yAxisGuide.hideLabel}
          <div class="pointer-events-none absolute right-0 pr-4" style={`top:${yAxisGuide.top + yAxisGuide.labelOffsetPx}px;transform:${yAxisGuide.transform}`}>
            {yAxisGuide.label}
          </div>
        {/if}
      {/each}
    </div>

    <div class="relative h-full min-w-0 flex-1 overflow-hidden" bind:this={plotFrameElement}>
      {#each yAxisGuides as yAxisGuide (yAxisGuide.top)}
        <div class="pointer-events-none absolute left-0 right-0 border-t border-dashed border-slate-300/80" style={`top:${yAxisGuide.top}px`}></div>
      {/each}

      <div
        bind:this={plotCanvasElement}
        class="absolute inset-y-0"
        style={`width:${chartMetrics.plotWidth}px;right:0`}
      ></div>
    </div>
  </div>
</div>
