<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import ColorSelect from '#components/form/ColorSelect.svelte';
  import Input from '#components/form/Input.svelte';
  import T from '#components/misc/T.svelte';
  import { PROJECT_COLORS, STATUS_DELETED, makeEmptyEpic, type IEpic } from './project.ts';

  interface Props {
    epicForms: IEpic[]
    projectID: number
    // Only the project's admins delete stored epics; anyone editing may drop an epic not saved yet.
    canDelete: boolean
  }

  let { epicForms = $bindable(), projectID, canDelete }: Props = $props()

  const keptEpicsCount = $derived(epicForms.filter((epicForm) => epicForm.ss !== STATUS_DELETED).length)

  // A stored epic stays in the forms marked deleted, so the save sends its deletion.
  const removeEpic = (epicForm: IEpic) => {
    if (epicForm.ID) {
      epicForm.ss = STATUS_DELETED
    } else {
      epicForms = epicForms.filter((otherEpicForm) => otherEpicForm !== epicForm)
    }
  }
</script>

<div class="flex flex-col gap-6 mt-12" aria-label="Project epics">
  {#if keptEpicsCount === 0}
    <div class="text-fg-muted py-12 text-center"><T text="This project has no epics yet.|Este proyecto todavía no tiene épicas." /></div>
  {/if}
  {#each epicForms as epicForm (epicForm)}
    {#if epicForm.ss !== STATUS_DELETED}
      <div class="flex items-center gap-8 rounded bg-surface-soft px-10 py-4">
        <ColorSelect saveOn={epicForm} save="Color" options={PROJECT_COLORS} keyId="id" keyName="name" keyColorCss="swatchCss"
          label={`Color of the epic ${epicForm.Name}|Color de la épica ${epicForm.Name}`} />
        <Input saveOn={epicForm} save="Name" placeholder="Epic name|Nombre de la épica" css="grow min-w-0" required={true} />
        {#if canDelete || !epicForm.ID}
          <Button icon="icon-[fa--trash]" color="red" label={`Remove the epic ${epicForm.Name}|Quitar la épica ${epicForm.Name}`}
            onClick={() => removeEpic(epicForm)} />
        {/if}
      </div>
    {/if}
  {/each}
  <Button name="Epic|Épica" icon="icon-[fa--plus]" color="green" css="self-start mt-4"
    label="Adds an epic to the list; it is created on save.|Agrega una épica a la lista; se crea al guardar."
    onClick={() => { epicForms = [...epicForms, makeEmptyEpic(projectID)] }} />
  <div class="text-[14px] text-fg-muted ml-2">
    <T text="Changes are written on Save. The stories of a removed epic stay, without an epic.|Los cambios se graban al Guardar. Las historias de una épica quitada quedan, sin épica." />
  </div>
</div>
