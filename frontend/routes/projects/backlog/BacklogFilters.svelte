<script lang="ts">
  import ButtonLayer from '#components/buttons/ButtonLayer.svelte';
  import Checkbox from '#components/form/Checkbox.svelte';
  import FilterInput from '#components/form/FilterInput.svelte';
  import SearchSelect from '#components/form/SearchSelect.svelte';
  import { tr } from '#core/store.svelte.ts';
  import type { ProjectBacklog, ProjectPeople } from '../project.svelte.ts';
  import { backlogFilterOptions, hasActiveFilters, type IBacklogFilters } from './backlog.ts';

  interface Props {
    filters: IBacklogFilters
    backlog: ProjectBacklog
    people: ProjectPeople
  }

  let { filters = $bindable(), backlog, people }: Props = $props()

  // The text search lives outside the layer: only the filters hidden in it color the button,
  // so the user sees there is something filtering the list while the layer is closed. Both states
  // are `bx-*` buttons so only the color changes, never the size, border or shadow.
  const hasLayerFilters = $derived(hasActiveFilters({ ...filters, text: '' }))
  const filterOptions = $derived(backlogFilterOptions(backlog.epics, backlog.project?.Members || [], (userID) => people.userName(userID), tr))
</script>

<!-- The active filter tags are not here: they show in BacklogView's summary line, under this button. -->
<div class="flex items-center gap-8 flex-1 md:flex-none">
  <ButtonLayer icon="icon-[fa--filter]" buttonClass={hasLayerFilters ? 'bx-purple min-w-40' : 'bx-blue min-w-40'}
    layerClass="md:w-300" contentCss="p-12 flex flex-col gap-12"
    label="Opens the backlog filters.|Abre los filtros del backlog.">
    <SearchSelect label="Type|Tipo" placeholder="All types|Todos los tipos" options={filterOptions.typeOptions} keyId="id" keyName="name" saveOn={filters} save="typeID" css="w-full" />
    <SearchSelect label="State|Estado" placeholder="All states|Todos los estados" options={filterOptions.stateOptions} keyId="id" keyName="name" saveOn={filters} save="state" css="w-full" />
    <SearchSelect label="Epic|Épica" placeholder="All epics|Todas las épicas" options={filterOptions.epicOptions} keyId="ID" keyName="Name" saveOn={filters} save="epicID" css="w-full" />
    <SearchSelect label="Assignee|Responsable" placeholder="Anyone|Cualquiera" options={filterOptions.assigneeOptions} keyId="ID" keyName="Name" saveOn={filters} save="assigneeID" css="w-full" />
    <Checkbox label="To plan only|Solo por planificar" saveOn={filters} save="onlyPlannable" />
  </ButtonLayer>
  <FilterInput label="Filter stories|Filtrar historias" size="small" css="flex-1 md:flex-none md:w-200" icon="icon-[fa--search]" bind:value={filters.text} />
</div>
