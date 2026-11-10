<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import { useUI } from '@genix/ui'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { CrmTeamsService, postCrmTeam, type ICrmTeam } from '#services/crm/crm-team.svelte.ts'
  import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
  import { makeEmptyTeam, otherTeamOfUser, teamProblem, toggleTeamMember } from './teams.ts'

  // Teams group sellers for the goal, forecast and scorecard roll-ups. A user is in one team only.
  const ui = useUI()
  const TEAM_LAYER_ID = 1
  const teams = new CrmTeamsService(true)
  const usersService = new UsuariosService()

  let form = $state<ICrmTeam>(makeEmptyTeam())

  const userName = (userID: number) => {
    const user = usersService.usuariosMap.get(userID)
    return user ? `${user.FirstName} ${user.LastName}`.trim() : `#${userID}`
  }
  const memberOptions = $derived(form.MemberUserIDs.map((userID) => ({ id: userID, name: userName(userID) })))

  const openTeam = (team: ICrmTeam) => { form = structuredClone($state.snapshot(team)); ui.openSideLayer(TEAM_LAYER_ID) }

  const saveTeam = async (isDelete = false) => {
    const problem = isDelete ? '' : teamProblem(form)
    if (problem) { notifyFailure(tr(problem)); return }
    showLoading(tr('Saving team...|Guardando equipo...'))
    try {
      await postCrmTeam(teams, { ...$state.snapshot(form), ss: isDelete ? 0 : 1 })
      ui.openSideLayer(0)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<ICrmTeam>[] = [
    { header: 'Team|Equipo', css: 'px-6', highlight: true, getValue: (team) => team.Name },
    { header: 'Leader|Líder', css: 'px-6', getValue: (team) => userName(team.LeaderUserID) },
    { header: 'Members|Miembros', css: 'px-6', getValue: (team) => team.MemberUserIDs.map(userName).join(', ') },
  ]
</script>

<Page title="CRM Teams|Equipos CRM">
  <Layer type="content">
    <div class="flex items-center justify-end mb-6" aria-label="Teams toolbar">
      <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a team."
        onClick={() => openTeam(makeEmptyTeam())} />
    </div>
    <VTable {columns} data={teams.records} css="w-full" maxHeight="calc(100vh - 8rem - 16px)"
      selected={form.ID} isSelected={(team, selectedID) => team.ID === selectedID} onRowClick={openTeam} />
  </Layer>

  <Layer id={TEAM_LAYER_ID} type="side" sideLayerSize={520} title={form.ID > 0 ? form.Name : 'New team|Nuevo equipo'}
    titleCss="h2 mb-6" css="px-12 py-10" onSave={() => saveTeam()} onDelete={form.ID > 0 ? () => saveTeam(true) : undefined}
    onClose={() => { form = makeEmptyTeam() }}>
    <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Team form">
      <Input bind:saveOn={form} save="Name" label="Name|Nombre" required={true} css="col-span-24" />
      <SearchSelect options={memberOptions} keyId="id" keyName="name" label="Leader|Líder" css="col-span-24"
        selected={form.LeaderUserID} onChange={(option) => { form.LeaderUserID = option.id }} />
    </div>
    <div class="h3 mt-16 mb-6">{tr('Members|Miembros')}</div>
    <div class="flex flex-col gap-4" aria-label="Team members">
      {#each usersService.usuarios as user}
        {@const otherTeam = otherTeamOfUser(teams.records, user.ID, form.ID)}
        <div class="flex items-center gap-8">
          {#if otherTeam}
            <span class="text-fg-muted">{`${user.FirstName} ${user.LastName}`.trim()} · {tr('In team|En equipo')} {otherTeam.Name}</span>
          {:else}
            <Checkbox label={`${user.FirstName} ${user.LastName}`.trim()} checked={form.MemberUserIDs.includes(user.ID)}
              onToggle={(isChecked) => { form = toggleTeamMember(form, user.ID, isChecked) }} />
          {/if}
        </div>
      {/each}
    </div>
  </Layer>
</Page>
