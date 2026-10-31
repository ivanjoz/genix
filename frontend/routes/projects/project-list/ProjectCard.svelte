<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Card from '#components/cards/Card.svelte';
  import Avatar from '#components/misc/Avatar.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import { PROJECT_ROLE_ADMIN, isProjectArchived, projectCode, projectColor, projectRole, type IProject } from '../project.ts';
  import { projectDatesLabel } from './project-list.ts';

  interface Props {
    project: IProject
    currentUserID: number
    userName: (userID: number) => string
    onOpen: (project: IProject) => void
    onEdit?: (project: IProject) => void
  }

  const { project, currentUserID, userName, onOpen, onEdit }: Props = $props()

  const color = $derived(projectColor(project.Color))
  const currentUserRole = $derived(projectRole(project, currentUserID))
  const datesLabel = $derived(projectDatesLabel(project))
  // The card shows up to 6 avatars, then "+N".
  const visibleMembers = $derived(project.Members.slice(0, 6))
</script>

<Card label={`Open project ${project.Name}|Abrir proyecto ${project.Name}`} onClick={() => onOpen(project)}
  css="flex flex-col min-h-170 rounded-lg bg-surface border border-line overflow-hidden cursor-pointer
    transition-shadow hover:shadow-md hover:border-line-strong {isProjectArchived(project) ? 'opacity-70' : ''}">
  <div class="h-6 {color.swatchCss}"></div>
  <div class="flex flex-col gap-6 px-14 py-10 grow">
    <div class="flex items-center gap-8">
      <Badge text={projectCode(project)} size="small" css="ff-bold {color.softCss}" />
      {#if isProjectArchived(project)}
        <Badge text="Archived|Archivado" size="small" css="bg-surface-strong text-fg-soft" />
      {/if}
      {#if currentUserRole}
        <Badge text={currentUserRole === PROJECT_ROLE_ADMIN ? 'Admin|Administrador' : 'Member|Miembro'} size="small" css="bg-accent-bg text-accent-fg" />
      {/if}
      {#if onEdit}
        <Button icon="icon-[fa--pencil]" label={`Edit project ${project.Name}|Editar proyecto ${project.Name}`}
          css="ml-auto" onClick={() => onEdit(project)} />
      {/if}
    </div>
    <div class="h3 ff-bold leading-tight">{project.Name}</div>
    {#if project.Description}
      <div class="text-[14px] text-fg-muted line-clamp-2">{project.Description}</div>
    {/if}
    <div class="flex items-center gap-8 mt-auto pt-6">
      <div class="flex -space-x-6">
        {#each visibleMembers as member (member.UserID)}
          <!-- The color tells the role: admins solid, members gray. -->
          <Avatar name={userName(member.UserID)} colorSeed={member.UserID} size={28} css="border-2 border-surface"
            colorCss={member.Role === PROJECT_ROLE_ADMIN ? 'bg-accent-solid text-on-solid' : 'bg-surface-strong text-fg-soft'} />
        {/each}
        {#if project.Members.length > visibleMembers.length}
          <span class="w-28 h-28 rounded-full border-2 border-surface bg-surface-muted text-fg-muted flex items-center justify-center text-[14px]">
            +{project.Members.length - visibleMembers.length}
          </span>
        {/if}
      </div>
      {#if datesLabel}
        <span class="ml-auto text-[14px] text-fg-muted whitespace-nowrap">{datesLabel}</span>
      {/if}
    </div>
  </div>
</Card>
