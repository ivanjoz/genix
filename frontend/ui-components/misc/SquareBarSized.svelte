<script lang="ts">
  interface SquareBarSizedProps {
    label: string
    value: string
    size?: number
    background?: string
    backgroundColor?: string
    sublabel?: string
    useStripedLines?: string
  }

  let {
    label,
    value,
    size = 0,
    background = '',
    backgroundColor = '#dbeafe',
    sublabel = '',
    useStripedLines = '',
  }: SquareBarSizedProps = $props()

  // Clamp the incoming ratio so visual scaling remains predictable.
  const normalizedSize = $derived.by(() => {
    const numericSize = Number(size || 0)
    return Math.max(0, Math.min(1, numericSize))
  })

  const MIN_BAR_HEIGHT_PX = 6
  const TOTAL_BAR_HEIGHT_PX = 132

  // Keep a visible minimum bar while preserving true proportional scaling.
  const filledHeightPx = $derived.by(() => {
    return MIN_BAR_HEIGHT_PX + (normalizedSize * (TOTAL_BAR_HEIGHT_PX - MIN_BAR_HEIGHT_PX))
  })

  const filledHeightPercent = $derived.by(() => {
    return (filledHeightPx / TOTAL_BAR_HEIGHT_PX) * 100
  })

  const stripedBackgroundStyle = $derived.by(() => {
    return `repeating-linear-gradient(135deg, ${useStripedLines} 0px, ${useStripedLines} 7px, #ffffff 7px, #ffffff 14px)`
  })

  // Render the remaining capacity above the filled block using diagonal stripes.
  const remainingHeightPercent = $derived.by(() => {
    return Math.max(0, 100 - filledHeightPercent)
  })

  const useOutsideContent = $derived.by(() => {
    return normalizedSize <= 0.5
  })

  const filledBackground = $derived.by(() => {
    return background || backgroundColor
  })
</script>

<div class="flex h-full w-full items-end">
  <div class="relative h-full w-full overflow-visible">
    {#if useStripedLines && remainingHeightPercent > 0}
      <div
        class="absolute inset-x-0 top-0 rounded-t-[12px]"
        style={`height:${remainingHeightPercent}%;background:${stripedBackgroundStyle}`}
      ></div>
    {/if}

    <div
      class="absolute inset-x-0 bottom-0 z-[1]"
      style={`height:${filledHeightPercent}%;background:${filledBackground};border-top-left-radius:${(!useStripedLines || remainingHeightPercent === 0) ? '12px' : '0px'};border-top-right-radius:${(!useStripedLines || remainingHeightPercent === 0) ? '12px' : '0px'}`}
    ></div>

    {#if useOutsideContent}
      <div
        class="absolute inset-x-0 z-[2] flex flex-col items-center justify-center gap-2 px-8 text-center"
        style={`top:0;height:${remainingHeightPercent}%`}
      >
        <div class="text-[12px] leading-none text-slate-700">{label}</div>
        <div class="text-[18px] leading-[1.1] ff-bold text-slate-900">{value}</div>
        {#if sublabel}
          <div class="text-[12px] leading-none text-slate-700">{sublabel}</div>
        {/if}
      </div>
    {:else}
      <div
        class="absolute inset-x-0 bottom-0 z-[2] flex flex-col items-center justify-center gap-6 px-10 py-8 text-center"
        style={`height:${filledHeightPercent}%`}
      >
        <div class="text-[12px] leading-none text-slate-700">{label}</div>
        <div class="text-[18px] leading-[1.1] ff-bold text-slate-900">{value}</div>
        {#if sublabel}
          <div class="text-[12px] leading-none text-slate-700">{sublabel}</div>
        {/if}
      </div>
    {/if}
  </div>
</div>
