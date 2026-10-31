<script lang="ts">
  import { tr } from '#core/store.svelte.ts';

  // ChartCanvas draws no legend: with two or more series the identity of each color is listed
  // here, in text ink with the colored mark beside it.
  let { entries }: { entries: { name: string, color: string }[] } = $props()
</script>

<div class="flex flex-wrap gap-x-14 gap-y-4 text-[14px] text-fg-muted">
  {#each entries as legendEntry (legendEntry.name)}
    <span class="flex items-center gap-6">
      <span class="inline-block h-10 w-10 shrink-0 rounded-[3px]" style:background={legendEntry.color}></span>
      {tr(legendEntry.name)}
    </span>
  {/each}
</div>
