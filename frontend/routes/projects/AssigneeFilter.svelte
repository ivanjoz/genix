<script lang="ts">
  import AvatarToggleGroup from '#components/form/AvatarToggleGroup.svelte';
  import { tr } from '#core/store.svelte.ts';
  import type { ProjectPeople } from './project.svelte.ts';

  // The people filter of the sprint board and the Gantt: one avatar per person holding stories
  // (0 = unassigned). Several can be chosen; none chosen shows everyone.
  interface Props {
    assigneeIDs: number[]
    people: ProjectPeople
    selectedAssigneeIDs: number[]
  }

  let { assigneeIDs, people, selectedAssigneeIDs = $bindable() }: Props = $props()

  const assigneeOptions = $derived(assigneeIDs.map((assigneeID) => ({
    id: assigneeID, name: assigneeID ? people.userName(assigneeID) : tr('Unassigned|Sin responsable'),
  })))
</script>

<AvatarToggleGroup options={assigneeOptions} bind:selected={selectedAssigneeIDs} label="Filter by assignee|Filtrar por responsable"
  optionLabel={(assigneeName) => `Show only the stories of ${assigneeName}|Mostrar solo las historias de ${assigneeName}`} />
