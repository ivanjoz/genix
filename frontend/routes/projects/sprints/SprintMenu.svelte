<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte';
  import Layer from '#components/layers/Layer.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import T from '#components/misc/T.svelte';
  import { formatTime } from '#libs/helpers.ts';
  import { goto } from '$app/navigation';
  import { useUI } from '@genix/ui';
  import { SPRINT_STATE_ACTIVE, SPRINT_STATE_PLANNED, type ISprint } from '../project.ts';
  import { formatHours } from '../backlog/backlog.ts';
  import { BOARD_COLUMNS, SPRINT_STATES, sprintDaysLeft, sprintProgress, type ISprintCard } from './sprints.ts';

  // The board's menu, a thin side layer: the sprint's progress, how the board groups its cards,
  // starting / closing the sprint and a link to its reports.
  interface Props {
    sprint: ISprint
    cards: ISprintCard[]
    boardLayout: { mode: number }
    layoutOptions: { id: number, name: string }[]
    canAdmin: boolean
    today: number
    layerID: number
    onStart: () => void
    onClose: () => void
  }

  const { sprint, cards, boardLayout, layoutOptions, canAdmin, today, layerID, onStart, onClose }: Props = $props()
  const ui = useUI()

  const progress = $derived(sprintProgress(cards))
  const finishedPercent = $derived(progress.committedHours ? Math.round(progress.finishedHours / progress.committedHours * 100) : 0)

  // The start and close flows open their own dialog: the menu gets out of the way first.
  const runFromMenu = (action: () => void) => {
    ui.openSideLayer(0)
    action()
  }
</script>

<Layer type="side" id={layerID} sideLayerSize={360} title={sprint.Name} titleCss="h2 mb-6" css="px-8 py-8 md:px-14 md:py-10">
  <div class="flex flex-col gap-16 mt-8">
    <section class="flex flex-col gap-6 text-[14px]">
      <div class="flex items-center gap-8">
        <Badge text={SPRINT_STATES[sprint.State]?.name} css={SPRINT_STATES[sprint.State]?.badgeCss} />
        <span class="text-fg-muted">{formatTime(sprint.StartDate, 'd/m/Y')} → {formatTime(sprint.EndDate, 'd/m/Y')}</span>
      </div>
      {#if sprint.State === SPRINT_STATE_ACTIVE && today > sprint.EndDate}
        <span class="text-red-fg"><T text="Past its end date|Pasó su fecha de fin" /></span>
      {:else if sprint.State === SPRINT_STATE_ACTIVE}
        <span class="text-fg-muted"><b>{sprintDaysLeft(sprint, today)}</b> <T text="days left|días restantes" /></span>
      {/if}
      {#if sprint.Goal}<div class="text-fg-soft">{sprint.Goal}</div>{/if}
    </section>

    <section class="flex flex-col gap-6">
      <div class="ff-bold"><T text="Progress|Avance" /></div>
      <div class="flex items-center text-[14px] text-fg-muted">
        <span><b>{progress.finishedCount}</b> / {progress.storiesCount} <T text="stories finished|historias finalizadas" /></span>
        <span class="ml-auto"><b>{formatHours(progress.finishedHours)}</b> / {formatHours(progress.committedHours)}</span>
      </div>
      <div class="h-6 rounded-full bg-surface-muted overflow-hidden" title="{finishedPercent}%">
        <div class="h-full bg-green-solid" style:width="{finishedPercent}%"></div>
      </div>
      <div class="flex flex-col gap-2 text-[14px] mt-4">
        {#each BOARD_COLUMNS as column (column.id)}
          {@const columnCards = cards.filter((card) => column.states.includes(card.sprintStory.State))}
          <div class="flex items-center gap-8">
            <span class="text-fg-soft grow"><T text={column.name} /></span>
            <span class="text-fg-muted">{formatHours(sprintProgress(columnCards).committedHours)}</span>
            <b class="w-24 text-right">{columnCards.length}</b>
          </div>
        {/each}
      </div>
    </section>

    <section class="flex flex-col gap-6">
      <div class="ff-bold"><T text="Group the board|Agrupar el tablero" /></div>
      <CheckboxOptions saveOn={boardLayout} save="mode" options={layoutOptions} keyId="id" keyName="name" useButtonsSlim={true} />
    </section>

    <section class="flex flex-col gap-8">
      {#if canAdmin && sprint.State === SPRINT_STATE_PLANNED}
        <Button name="Start sprint|Iniciar sprint" icon="icon-[fa--play]" color="green" label="Makes this the active sprint.|Lo vuelve el sprint activo." onClick={() => runFromMenu(onStart)} />
      {/if}
      {#if canAdmin && sprint.State === SPRINT_STATE_ACTIVE}
        <Button name="Close sprint|Cerrar sprint" icon="icon-[fa--flag-checkered]" color="orange" label="Closes the sprint and decides where its unfinished stories go.|Cierra el sprint y decide a dónde van sus historias no culminadas." onClick={() => runFromMenu(onClose)} />
      {/if}
      {#if sprint.State !== SPRINT_STATE_PLANNED}
        <Button name="Reports|Reportes" icon="icon-[fa--line-chart]" label="Opens the burndown and the reports of this sprint.|Abre el burndown y los reportes de este sprint."
          onClick={() => { void goto(`/projects/reports/sprints?project=${sprint.ProjectID}&sprint=${sprint.ID}`) }} />
      {/if}
    </section>
  </div>
</Layer>
