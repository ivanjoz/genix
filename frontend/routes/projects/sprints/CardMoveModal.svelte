<script lang="ts">
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte';
  import Input from '#components/form/Input.svelte';
  import Modal from '#components/layers/Modal.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import type { IDropPosition } from '#libs/drag-drop.ts';
  import { useUI } from '@genix/ui';
  import { notifyFailure } from '@genix/ui/notify';
  import { STORY_STATES, storyCode } from '../project.ts';
  import { needsMoveReason, type ISprintCard } from './sprints.ts';

  // One modal for both ways a card moves without a plain drop: the reason a card is dropped on
  // Stopped (the state is fixed), and the mobile "Move to…" menu (the state is chosen here).
  interface Props {
    modalID: number
    onMove: (card: ISprintCard, toState: number, dropPosition: IDropPosition | null, reason: string) => void
  }

  const { modalID, onMove }: Props = $props()

  const ui = useUI()
  const stateOptions = Object.entries(STORY_STATES).filter(([state]) => Number(state) > 0)
    .map(([state, stateInfo]) => ({ id: Number(state), name: tr(stateInfo.name) }))

  let movedCard = $state<ISprintCard | null>(null)
  // null: the menu, the card goes last in the chosen column.
  let dropPosition = $state<IDropPosition | null>(null)
  let isStateFixed = $state(false)
  let moveForm = $state({ toState: 0, reason: '' })

  export const openReasonModal = (card: ISprintCard, toState: number, droppedAt: IDropPosition) => {
    movedCard = card
    dropPosition = droppedAt
    isStateFixed = true
    moveForm = { toState, reason: '' }
    ui.openModal(modalID)
  }

  export const openMoveMenu = (card: ISprintCard) => {
    movedCard = card
    dropPosition = null
    isStateFixed = false
    moveForm = { toState: card.sprintStory.State, reason: '' }
    ui.openModal(modalID)
  }

  const isReasonNeeded = $derived(!!movedCard && needsMoveReason(movedCard.sprintStory, moveForm.toState))

  const confirmMove = () => {
    if (!movedCard || !moveForm.toState) { return }
    if (isReasonNeeded && !moveForm.reason.trim()) {
      notifyFailure(tr('Say why the story is stopped.|Indica por qué se detiene la historia.'))
      return
    }
    onMove(movedCard, moveForm.toState, dropPosition, moveForm.reason.trim())
    ui.closeModal(modalID)
  }
</script>

<Modal id={modalID} size={2} saveButtonLabel={isStateFixed ? 'Stop|Detener' : 'Move|Mover'} saveIcon="icon-[fa--check]"
  title={movedCard ? `${storyCode(movedCard.story)} · ${movedCard.story.Title}` : ''} onSave={confirmMove}>
  <div class="flex flex-col gap-12">
    {#if !isStateFixed}
      <div>
        <div class="text-[14px] text-fg-muted mb-6 ml-2"><T text="Move to|Mover a" /></div>
        <CheckboxOptions saveOn={moveForm} save="toState" options={stateOptions} keyId="id" keyName="name" useButtons={true} />
      </div>
    {/if}
    {#if isReasonNeeded}
      <Input label="Why is it stopped?|¿Por qué se detiene?" saveOn={moveForm} save="reason" useTextArea={true} rows={3}
        required={true} focusOnOpen={isStateFixed} />
    {/if}
  </div>
</Modal>
