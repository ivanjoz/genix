<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Checkbox from '#components/form/Checkbox.svelte';
  import CheckboxOptions from '#components/form/CheckboxOptions.svelte';
  import DateInput from '#components/form/DateInput.svelte';
  import Input from '#components/form/Input.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import {
    PROJECT_COLORS, PROJECT_ROLE_MEMBER, PROJECT_ROLE_OPTIONS, PROJECT_STATE_ACTIVE, PROJECT_STATE_ARCHIVED, type IProject,
  } from './project.ts';
  import type { ProjectPeople } from './project.svelte.ts';

  interface Props {
    projectForm: IProject
    people: ProjectPeople
  }

  let { projectForm = $bindable(), people }: Props = $props()

  // genix-ui selects show option names as given: translate them once here.
  const roleOptions = PROJECT_ROLE_OPTIONS.map((roleOption) => ({ ...roleOption, name: tr(roleOption.name) }))
  const colorOptions = PROJECT_COLORS.map((color) => ({ id: color.id, name: tr(color.name) }))
  const memberUserIDs = $derived(projectForm.Members.map((member) => member.UserID))

  const addMember = (userID: number) => {
    if (!userID || memberUserIDs.includes(userID)) { return }
    projectForm.Members = [...projectForm.Members, { UserID: userID, Role: PROJECT_ROLE_MEMBER }]
  }

  const removeMember = (userID: number) => {
    projectForm.Members = projectForm.Members.filter((member) => member.UserID !== userID)
  }
</script>

<div class="grid grid-cols-24 gap-10 mt-12" aria-label="Project form">
  <Input label="Name|Nombre" saveOn={projectForm} save="Name" css="col-span-24" required={true} />
  <Input label="Description|Descripción" saveOn={projectForm} save="Description" css="col-span-24" useTextArea={true} rows={3} />
  <DateInput label="Start|Inicio" saveOn={projectForm} save="StartDate" css="col-span-12" />
  <DateInput label="End|Fin" saveOn={projectForm} save="EndDate" css="col-span-12" />
  <div class="col-span-24">
    <div class="text-[14px] text-fg-muted mb-4 ml-2"><T text="Color" /></div>
    <CheckboxOptions saveOn={projectForm} save="Color" options={colorOptions} keyId="id" keyName="name" useButtons={true} />
  </div>
  {#if projectForm.ID}
    <div class="col-span-24">
      <Checkbox label="Archived (read only, hidden from the list)|Archivado (solo lectura, oculto de la lista)"
        checked={projectForm.State === PROJECT_STATE_ARCHIVED}
        onToggle={(isArchived) => { projectForm.State = isArchived ? PROJECT_STATE_ARCHIVED : PROJECT_STATE_ACTIVE }} />
    </div>
  {/if}

  <div class="ff-bold h3 col-span-24 ml-8 mt-8"><T text="Team|Equipo" /></div>
  <SearchSelect label="Add a member|Agregar integrante" options={people.candidates} keyId="ID" keyName="Name"
    avoidIDs={memberUserIDs} clearOnSelect={true} css="col-span-24"
    onChange={(candidate) => { if (candidate) { addMember(candidate.ID) } }} />
  {#if people.candidates.length === 0}
    <div class="col-span-24 text-[14px] text-fg-muted ml-2">
      <T text="Only users with the Edit level of Projects can join a team.|Solo los usuarios con el nivel Editar de Proyectos pueden integrar un equipo." />
    </div>
  {/if}
  <div class="col-span-24 flex flex-col gap-6">
    {#each projectForm.Members as member (member.UserID)}
      <div class="flex items-center gap-8 rounded bg-surface-soft px-10 py-4">
        <span class="grow min-w-0 truncate">{people.userName(member.UserID)}</span>
        <SearchSelect saveOn={member} save="Role" options={roleOptions} keyId="id" keyName="name" notEmpty={true} css="w-170" />
        <Button icon="icon-[fa--trash]" label={`Remove ${people.userName(member.UserID)} from the team|Quitar a ${people.userName(member.UserID)} del equipo`}
          color="red" onClick={() => removeMember(member.UserID)} />
      </div>
    {/each}
  </div>
</div>
