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
  }: ChartCanvasProps = $props()

  let containerElement = $state<HTMLDivElement | undefined>(undefined)
  let measuredWidth = $state(0)
  let resizeObserver: ResizeObserver | undefined
  let chartLeafer: any
  let pendingRenderFrame = 0
  let lastRenderAt = 0
  let queuedRenderTimeout: ReturnType<typeof setTimeout> | undefined
  let lastRenderKey = ""
  let sharedLeaferPromise: Promise<any> | undefined

  const renderThrottleMs = 250

  const loadLeafer = () => {
    if (!sharedLeaferPromise) {
      sharedLeaferPromise = import("leafer-ui")
    }
    return sharedLeaferPromise
  }

  const updateMetrics = () => {
    if (!containerElement) { return }
    measuredWidth = Math.max(0, Math.floor(containerElement.clientWidth))
  }

  const buildStackedBarFrames = (): IChartCanvasRectFrame[] => {
    const barSeries = data.filter((chartSeries) => chartSeries.type === 'bar')
    const pointsCount = data.reduce((maxCount, chartSeries) => {
      return Math.max(maxCount, chartSeries.values.length)
    }, 0)
    const maxStackValue = Array.from({ length: pointsCount }, (_, pointIndex) => {
      return barSeries.reduce((stackTotal, chartSeries) => stackTotal + (chartSeries.values[pointIndex] || 0), 0)
    }).reduce((maxValue, stackValue) => Math.max(maxValue, stackValue), 0)
    const columnWidth = pointsCount > 0 ? measuredWidth / pointsCount : 0

    return Array.from({ length: pointsCount }, (_, pointIndex) => {
      let stackedHeight = 0

      // Keep each series section in one column so the mini chart stays cheap to draw.
      return barSeries.map((chartSeries, seriesIndex) => {
        const pointValue = chartSeries.values[pointIndex] || 0
        const frameHeight = maxStackValue > 0 ? (pointValue / maxStackValue) * height : 0
        const frame = {
          x: pointIndex * columnWidth,
          y: height - stackedHeight - frameHeight,
          width: Math.max(1, columnWidth - 1),
          height: frameHeight,
          fill: chartSeries.color || (seriesIndex % 2 === 0 ? '#ef4444' : '#3b82f6'),
        }
        stackedHeight += frameHeight
        return frame
      })
    }).flat()
  }

  const buildLineFrames = (): IChartCanvasLineFrame[] => {
    const pointsCount = data.reduce((maxCount, chartSeries) => {
      return Math.max(maxCount, chartSeries.values.length)
    }, 0)
    const barSeries = data.filter((chartSeries) => chartSeries.type === 'bar')
    const lineSeries = data.filter((chartSeries) => chartSeries.type === 'line')
    const maxStackValue = Array.from({ length: pointsCount }, (_, pointIndex) => {
      const stackedBarTotal = barSeries.reduce((stackTotal, chartSeries) => stackTotal + (chartSeries.values[pointIndex] || 0), 0)
      const lineMaximum = lineSeries.reduce((maxValue, chartSeries) => Math.max(maxValue, chartSeries.values[pointIndex] || 0), 0)
      return Math.max(stackedBarTotal, lineMaximum)
    }).reduce((maxValue, stackValue) => Math.max(maxValue, stackValue), 0)
    const columnWidth = pointsCount > 0 ? measuredWidth / pointsCount : 0

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

        const x = (pointIndex * columnWidth) + (columnWidth / 2)
        const y = maxStackValue > 0 ? height - ((pointValue / maxStackValue) * height) : height
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
          const x = (pointIndex * columnWidth) + (columnWidth / 2)
          const y = maxStackValue > 0 ? height - ((pointValue / maxStackValue) * height) : height
          return [{ x, y }]
        }),
      }
      return lineFrame
    })
  }

  const getRenderKey = () => {
    return JSON.stringify({
      measuredWidth,
      height,
      id,
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
    if (!containerElement || measuredWidth <= 0) { return }

    const nextRenderKey = getRenderKey()
    if (nextRenderKey === lastRenderKey) { return }

    const { Leafer, Rect, Line, Ellipse } = await loadLeafer()
    const cacheID = String(id || '')
    const cachedChart = cacheID ? sharedChartCache.get(cacheID) : undefined
    const useCachedFrames = cachedChart?.renderKey === nextRenderKey
    const stackedBarFrames = useCachedFrames ? cachedChart.stackedBarFrames : buildStackedBarFrames()
    const lineFrames = useCachedFrames ? cachedChart.lineFrames : buildLineFrames()

    if (cacheID && !useCachedFrames) {
      sharedChartCache.set(cacheID, {
        renderKey: nextRenderKey,
        stackedBarFrames,
        lineFrames,
      })
    }

    chartLeafer?.destroy()
    chartLeafer = new Leafer({
      view: containerElement,
      width: measuredWidth,
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

<div bind:this={containerElement} class={className} style={`${style};height:${height}px`}></div>
