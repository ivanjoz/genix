<script lang="ts">
  import Input from '#components/form/Input.svelte';
  import InputList from '#components/form/InputList.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { STORY_PRIORITIES, STORY_TYPES, type IStory } from '../project.ts';
  import type { ProjectBacklog, ProjectPeople } from '../project.svelte.ts';
  import AttachmentFiles from '#domain/attachments/AttachmentFiles.svelte';
  import type { IStoryDetail } from './story.ts';
  import { uploadAttachment } from './story.svelte.ts';

  interface Props {
    backlog: ProjectBacklog
    people: ProjectPeople
    storyForm: IStory
    // null while the detail of an existing story loads.
    detailForm: IStoryDetail | null
    uploadingCount: number
    canWork: boolean
  }

  let { backlog, people, storyForm = $bindable(), detailForm = $bindable(), uploadingCount = $bindable(), canWork }: Props = $props()

  // genix-ui selects show option names as given: translate them here.
  const typeOptions = STORY_TYPES.map((type) => ({ id: type.id, name: tr(type.name) }))
  const priorityOptions = STORY_PRIORITIES.map((priority) => ({ id: priority.id, name: tr(priority.name) }))
  // Epic and assignee are optional: SearchSelect shows a 0 as empty, so "none" is its placeholder
  // and clearing the select.
  const epicOptions = $derived(backlog.epics.map((epic) => ({ ID: epic.ID, Name: epic.Name })))
  const assigneeOptions = $derived((backlog.project?.Members || []).map((member) => ({ ID: member.UserID, Name: people.userName(member.UserID) })))
</script>

<div class="grid grid-cols-24 gap-10 mt-12" aria-label="Story form">
  <Input label="Title|Título" saveOn={storyForm} save="Title" css="col-span-24" required={true} disabled={!canWork} />
  <SearchSelect label="Type|Tipo" saveOn={storyForm} save="Type" options={typeOptions} keyId="id" keyName="name" notEmpty={true}
    css="col-span-12 md:col-span-8" disabled={!canWork} />
  <SearchSelect label="Priority|Prioridad" saveOn={storyForm} save="Priority" options={priorityOptions} keyId="id" keyName="name" notEmpty={true}
    css="col-span-12 md:col-span-8" disabled={!canWork} />
  <Input label="Estimated hours|Horas estimadas" saveOn={storyForm} save="EstimatedHours" type="number" postValue="h"
    css="col-span-24 md:col-span-8" disabled={!canWork} />
  <SearchSelect label="Epic|Épica" saveOn={storyForm} save="EpicID" options={epicOptions} keyId="ID" keyName="Name"
    placeholder={tr('No epic|Sin épica')} css="col-span-24 md:col-span-12" disabled={!canWork} />
  <SearchSelect label="Assignee|Responsable" saveOn={storyForm} save="AssigneeID" options={assigneeOptions} keyId="ID" keyName="Name"
    placeholder={tr('Unassigned|Sin responsable')} css="col-span-24 md:col-span-12" disabled={!canWork} />
  <Input label="Short description|Descripción corta" saveOn={storyForm} save="Summary"
    css="col-span-24" disabled={!canWork} />

  {#if detailForm}
    <div class="col-span-24 min-h-200 mt-6">
      <!-- The editor reads its value once: it is drawn again for each version of the detail. -->
      {#key detailForm}
        {#await import('#components/editor/HTMLEditor.svelte') then htmlEditorModule}
          {@const HTMLEditor = htmlEditorModule.default}
          <HTMLEditor saveOn={detailForm} save="Description" css="min-h-200"
            placeholder="Write a description of the story...|Escriba una descripción de la historia..." />
        {/await}
      {/key}
    </div>
    <div class="col-span-24">
      <AttachmentFiles bind:attachments={detailForm.Attachments!} bind:uploadingCount
        uploadFile={(file) => uploadAttachment(storyForm.ProjectID, file)}
        canEdit={canWork} />
    </div>
    <div class="col-span-24 mt-6 mb-16">
      <InputList bind:items={detailForm.Checklist!} textKey="Text" checkKey="Done" makeItem={() => ({ Text: '', Done: false })}
        title="Acceptance criteria|Criterios de aceptación" countLabel="Done|Cumplidos"
        placeholder="Acceptance criterion|Criterio de aceptación" emptyText="No acceptance criteria.|Sin criterios de aceptación."
        disabled={!canWork} />
    </div>
  {:else}
    <div class="col-span-24 text-fg-muted"><T text="Loading...|Cargando..." /></div>
  {/if}
</div>
