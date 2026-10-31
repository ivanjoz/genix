<script lang="ts">
  import ChartCanvas, { type ChartCanvasSeries } from '#components/charts/ChartCanvas.svelte';
  import T from '#components/misc/T.svelte';
  import { formatTime, todayUnixDay } from '#libs/helpers.ts';
  import ChartLegend from '#domain/charts/ChartLegend.svelte';
  import type { ISprint } from '../../project.ts';
  import { formatHours } from '../../backlog/backlog.ts';
  import { sUnixTimeToUnixMillis } from '../../backlog/backlog.utils.ts';
  import { sprintBurndown, type ISprintCard } from '../../sprints/sprints.ts';

  interface Props {
    sprint: ISprint
    cards: ISprintCard[]
  }

  const { sprint, cards }: Props = $props()

  const MAX_AXIS_LABELS = 8
  const REMAINING_COLOR = '#4874f5'
  const IDEAL_COLOR = '#94a3b8'

  const burndown = $derived(sprintBurndown(sprint, cards, todayUnixDay(), (sUnixTime) => todayUnixDay(new Date(sUnixTimeToUnixMillis(sUnixTime)))))
  const hasHours = $derived(burndown.remaining.some((hours) => !!hours) || burndown.ideal[0] > 0)
  const series = $derived<ChartCanvasSeries[]>([
    { type: 'line', name: 'Ideal|Ideal', values: burndown.ideal, color: IDEAL_COLOR, lineWidth: 2 },
    { type: 'line', name: 'Hours left|Horas restantes', values: burndown.remaining, color: REMAINING_COLOR, lineWidth: 3, pointSize: 3 },
  ])
</script>

<div class="flex flex-col gap-8 rounded-lg border border-line bg-surface p-12 mb-24">
  <div class="flex items-center gap-12 flex-wrap">
    <ChartLegend entries={series.map((line) => ({ name: line.name, color: line.color || '' }))} />
    <span class="text-[14px] text-fg-muted md:ml-auto">
      <T text="Estimated hours of the stories not finished yet, at the end of each day.|Horas estimadas de las historias aún no finalizadas, al final de cada día." />
    </span>
  </div>
  {#if hasHours}
    <!-- Whole hours on the axis: a 4-hour sprint otherwise repeats rounded labels. -->
    <ChartCanvas data={series} dateLabels={burndown.days} height={260} className="w-full" showTooltip showBottomBaseline yAxisStepSize={1}
      dateLabelEvery={Math.max(1, Math.ceil(burndown.days.length / MAX_AXIS_LABELS))}
      dateLabelFormatter={(day) => String(formatTime(Number(day), 'd/m'))}
      tooltipValueFormatter={(value) => formatHours(value)} />
  {:else}
    <div class="text-fg-muted text-center py-24">
      <T text="The stories of this sprint have no estimated hours: there is nothing to burn down.|Las historias de este sprint no tienen horas estimadas: no hay nada que mostrar." />
    </div>
  {/if}
  <div class="text-[14px] text-fg-muted">
    <T text="A story added after the start raises the line on that day; a finished one that is reopened counts again.|Una historia agregada después del inicio sube la línea ese día; una finalizada que se reabre vuelve a contar." />
  </div>
</div>
