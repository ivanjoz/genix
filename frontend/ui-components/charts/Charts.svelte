<script lang="ts" module>
  import type {
    Axis,
    BarOptions,
    CandlestickOptions,
    Chart,
    Data,
    LegendOptions,
    LineOptions,
    Padding,
    PointOptions,
    TooltipOptions
  } from "billboard.js"

  export interface ChartLine {
    value: number
    opacity?: number
    width?: number
    range?: [number | string, number | string]
    yPosition?: number
    dasharray?: number
    color?: string
    text?: string
    textPadding?: number
    fontSize?: number
    fontColor?: string
    rangeOffset?: number
  }

  export interface D3ChartProps {
    data: Data
    axis?: Axis
    className?: string
    categories?: Array<string | number>
    lines?: ChartLine[]
    legend?: LegendOptions
    tooltip?: TooltipOptions
    candlestick?: CandlestickOptions
    line?: LineOptions
    point?: PointOptions
    padding?: Padding
    onclick?: (this: Chart, event: Event) => void
    forceWidth?: number
    bar?: BarOptions
    style?: string
  }
</script>

<script lang="ts">
  import { onDestroy, onMount } from "svelte"
  import type { Chart, ChartTypes, Data } from "billboard.js"

  interface BillboardRuntime {
    bb: { generate: (config: Record<string, unknown>) => Chart }
    spline: () => ChartTypes
    line: () => ChartTypes
    area: () => ChartTypes
    areaSpline: () => ChartTypes
    bar: () => ChartTypes
    step: () => ChartTypes
    bubble: () => ChartTypes
    candlestick: () => ChartTypes
    scatter: () => ChartTypes
  }

  const {
    data,
    axis,
    className = "h100 w100",
    categories = [],
    lines = [],
    legend,
    tooltip,
    candlestick,
    line,
    point,
    padding,
    onclick,
    forceWidth,
    bar,
    style = ""
  }: D3ChartProps = $props()

  let chartContainerElement = $state<HTMLDivElement | undefined>(undefined)
  // Keep chart instance/runtime refs non-reactive to avoid effect self-trigger loops.
  let generatedChart: Chart | undefined
  let billboardRuntime: BillboardRuntime | undefined
  let isChartRuntimeReady = false
  // Use plain counters for tracing loops without adding reactive dependencies.
  let renderChartRunsCount = 0
  let chartEffectRunsCount = 0

  let runtimePromise: Promise<BillboardRuntime> | null = null

  const assignChartType = (runtime: BillboardRuntime, chartType: ChartTypes): ChartTypes => {
    if (chartType === "spline") return runtime.spline()
    if (chartType === "line") return runtime.line()
    if (chartType === "area") return runtime.area()
    if (chartType === "area-spline") return runtime.areaSpline()
    if (chartType === "bar") return runtime.bar()
    if (chartType === "step") return runtime.step()
    if (chartType === "bubble") return runtime.bubble()
    if (chartType === "candlestick") return runtime.candlestick()
    if (chartType === "scatter") return runtime.scatter()
    return chartType
  }

  const normalizeDataTypes = (runtime: BillboardRuntime, sourceData: Data): Data => {
    // Clone before normalization to avoid mutating caller state.
    const normalizedData: Data = {
      ...sourceData,
      columns: sourceData.columns ? [...sourceData.columns] : sourceData.columns,
      rows: sourceData.rows ? [...sourceData.rows] : sourceData.rows,
      json: sourceData.json ? { ...(sourceData.json as Record<string, unknown>) } : sourceData.json,
      types: sourceData.types ? { ...sourceData.types } : sourceData.types
    }

    if (typeof normalizedData.type === "string") {
      normalizedData.type = assignChartType(runtime, normalizedData.type)
    }

    if (normalizedData.types) {
      for (const typeName in normalizedData.types) {
        const typeValue = normalizedData.types[typeName]
        normalizedData.types[typeName] = assignChartType(runtime, typeValue)
      }
    }

    return normalizedData
  }

  const drawOverlayLines = (chart: Chart) => {
    // Billboard doesn't expose this part with strict types; use internal info as in the original abstraction.
    const chartWithInternals = chart as Chart & { internal?: { state?: { current?: { maxTickSize?: { y?: { domain?: [number, number] } } } } } }
    const yAxisDomain = chartWithInternals.internal?.state?.current?.maxTickSize?.y?.domain || [0, 0]
    const yAxisMin = yAxisDomain[0]
    const yAxisMax = yAxisDomain[1]
    const yAxisGap = yAxisMax - yAxisMin

    const svgChart = chart.$.svg.select("g.bb-chart")
    const chartNode = svgChart.node() as SVGElement | null
    if (!chartNode) return

    chartNode.removeAttribute("clip-path")
    const { height, width } = chartNode.getBoundingClientRect()
    const hasCategories = (categories || []).length > 0
    const stepSize = hasCategories ? width / categories.length : 0

    for (let lineIndex = 0; lineIndex < lines.length; lineIndex += 1) {
      const chartLine = lines[lineIndex]
      const lineId = `line-${lineIndex}`
      const textId = `line-t${lineIndex}`
      svgChart.select(`#${lineId}`)?.remove()
      svgChart.select(`#${textId}`)?.remove()

      if (chartLine.range?.length === 2) {
        const rectangleHeight = 24
        const rangeOffset = chartLine.rangeOffset || 0

        let xStartCategoryIndex = categories.findIndex(category => category === chartLine.range?.[0])
        let xEndCategoryIndex = categories.findIndex(category => category === chartLine.range?.[1])

        if (xStartCategoryIndex === -1 && xEndCategoryIndex > 0) {
          xStartCategoryIndex = 0
        } else if (xEndCategoryIndex === -1 && xStartCategoryIndex >= 0) {
          xEndCategoryIndex = categories.length - 1
        }

        if (xStartCategoryIndex === -1 || xEndCategoryIndex === -1) {
          continue
        }

        const xStartPixel = Math.ceil(xStartCategoryIndex * stepSize)
        const xEndPixel = Math.floor((xEndCategoryIndex * stepSize) + stepSize)
        const yStartPixel = (chartLine.yPosition || 0) - rectangleHeight

        svgChart.append("rect")
          .attr("id", lineId)
          .attr("class", "default")
          .attr("x", xStartPixel + rangeOffset)
          .attr("width", xEndPixel - xStartPixel - rangeOffset)
          .attr("y", yStartPixel)
          .attr("height", rectangleHeight)
          .attr("fill", chartLine.color || "black")
          .attr("opacity", chartLine.opacity || 1)

        const svgText = svgChart.append("text")
          .attr("id", textId)
          .attr("x", xStartPixel + ((xEndPixel - xStartPixel) / 2))
          .attr("y", yStartPixel + rectangleHeight / 2)
          .attr("dy", "0.35em")
          .attr("text-anchor", "middle")
          .style("font-size", `${chartLine.fontSize || 15}px`)
          .text(chartLine.text || "My Label")

        if (chartLine.fontColor) {
          svgText.attr("fill", chartLine.fontColor)
          svgText.attr("class", "svg-tt")
        }

        continue
      }

      if (yAxisGap === 0) {
        continue
      }

      const lineHeight = Math.round(height - (((chartLine.value - yAxisMin) / yAxisGap) * height - (chartLine.width || 1) / 2))

      const svgLine = svgChart.append("line")
        .attr("id", lineId)
        .attr("class", "default")
        .attr("x1", 0).attr("y1", lineHeight)
        .attr("x2", width).attr("y2", lineHeight)
        .attr("stroke-width", chartLine.width || 1)
        .attr("stroke", chartLine.color || "black")

      if (chartLine.dasharray) {
        svgLine.attr("stroke-dasharray", chartLine.dasharray)
      }
      if (chartLine.opacity) {
        svgLine.attr("stroke-opacity", chartLine.opacity)
      }

      if (chartLine.text) {
        const svgText = svgChart.append("text")
          .attr("id", textId)
          .attr("x", chartLine.textPadding || 4)
          .attr("y", lineHeight - 4)
          .attr("font-size", `${chartLine.fontSize || 12}px`)
          .text(chartLine.text)

        if (chartLine.fontColor) {
          svgText.attr("fill", chartLine.fontColor)
        }
      }
    }
  }

  const loadBillboardRuntime = async (): Promise<BillboardRuntime> => {
    if (!runtimePromise) {
      runtimePromise = Promise.all([
        import("billboard.js"),
        import("billboard.js/dist/billboard.css")
      ]).then(([module]) => {
        return {
          bb: module.bb,
          spline: module.spline,
          line: module.line,
          area: module.area,
          areaSpline: module.areaSpline,
          bar: module.bar,
          step: module.step,
          bubble: module.bubble,
          candlestick: module.candlestick,
          scatter: module.scatter
        }
      })
    }
    return runtimePromise
  }

  const renderChart = () => {
    renderChartRunsCount += 1
    console.debug("[Charts] renderChart run:", renderChartRunsCount, {
      isChartRuntimeReady,
      hasRuntime: !!billboardRuntime,
      hasContainer: !!chartContainerElement,
      columnsCount: data?.columns?.length || 0
    })
    if (!isChartRuntimeReady || !billboardRuntime || !chartContainerElement) {
      return
    }

    const normalizedData = normalizeDataTypes(billboardRuntime, data)
    generatedChart?.destroy()

    generatedChart = billboardRuntime.bb.generate({
      bindto: chartContainerElement,
      data: normalizedData,
      axis,
      legend,
      bar,
      tooltip,
      candlestick,
      line,
      padding,
      point,
      onclick,
      onafterinit: function () {
        if (forceWidth) {
          this.resize({ width: forceWidth })
        }
      },
      onrendered: function () {
        console.debug("[Charts] onrendered callback")
        drawOverlayLines(this)
      }
    })
  }

  onMount(async () => {
    billboardRuntime = await loadBillboardRuntime()
    isChartRuntimeReady = true
    renderChart()
  })

  $effect(() => {
    chartEffectRunsCount += 1
    console.debug("[Charts] props effect run:", chartEffectRunsCount)
    data
    axis
    legend
    tooltip
    candlestick
    line
    point
    padding
    onclick
    forceWidth
    bar
    categories
    lines
    if (!isChartRuntimeReady) return
    renderChart()
  })

  onDestroy(() => {
    generatedChart?.destroy()
    generatedChart = undefined
  })
</script>

<div bind:this={chartContainerElement} class={className} style={style}></div>
