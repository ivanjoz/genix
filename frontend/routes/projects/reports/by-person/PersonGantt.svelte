<script lang="ts">
  import { goto } from '$app/navigation';
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte';
  import Gantt from '#components/gantt/Gantt.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import type { ISprint } from '../../project.ts';
  import { personGantt, type IAssignedStoryRow } from './by-person.ts';

  interface Props {
    rows: IAssignedStoryRow[]
    sprintsByProject: Map<number, ISprint[]>
  }

  const { rows, sprintsByProject }: Props = $props()

  const WEEK_ZOOM = 1
  const MONTH_ZOOM = 2
  const zoomOptions = [{ id: WEEK_ZOOM, name: tr('Weeks|Semanas') }, { id: MONTH_ZOOM, name: tr('Months|Meses') }]
  let ganttView = $state({ zoom: WEEK_ZOOM })

  const gantt = $derived(personGantt(rows, sprintsByProject, tr))
</script>

<!-- Fills the height the page leaves it: the Gantt's rows scroll inside it, not the page. -->
<div class="flex flex-col grow min-h-0">
<div class="flex items-center gap-12 flex-wrap mb-10 shrink-0">
  <CheckboxOptions saveOn={ganttView} save="zoom" options={zoomOptions} keyId="id" keyName="name" useButtonsSlim={true} />
  <span class="text-[14px] text-fg-muted">
    <T text="Each project groups its stories; each story shows the sprints it was in. Click a bar to open its sprint.|Cada proyecto agrupa sus historias; cada historia muestra los sprints en los que estuvo. Haz clic en una barra para abrir su sprint." />
  </span>
  {#if gantt.unplannedCount}
    <span class="text-[14px] text-fg-muted ml-auto">{gantt.unplannedCount} <T text="stories not planned yet are left out|historias sin planificar no se muestran" /></span>
  {/if}
</div>

<Gantt rows={gantt.rows} zoom={ganttView.zoom === MONTH_ZOOM ? 'month' : 'week'} labelWidth={320} css="mb-8 min-h-0 overflow-y-auto"
  emptyText="None of these stories has been in a sprint yet.|Ninguna de estas historias ha estado en un sprint todavía."
  onBarClick={(row, bar) => {
    if (bar.sprintID) { void goto(`/projects/sprints?project=${row.projectID}&sprint=${bar.sprintID}`) }
  }} />
</div>
