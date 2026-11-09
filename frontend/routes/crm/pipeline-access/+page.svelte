<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import SelectBox from '#components/form/SelectBox.svelte'
  import Page from '#domain/Page.svelte'
  import { tr } from '#core/store.svelte.ts'
  import {
    CrmPipelinesService, postCrmPipelineMembers, type ICrmPipelineMember,
  } from '#services/crm/crm-pipeline.svelte.ts'
  import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
  import { LEVEL_OPTIONS, levelOfUser, pipelinesWithChangedMembers, setMemberLevel } from './pipeline-access.ts'

  // Users (rows) × pipelines (columns): each cell is the user's level in that pipeline. Edits are
  // kept as a draft per pipeline and saved together, one request per pipeline that changed.
  const pipelines = new CrmPipelinesService(true)
  const usersService = new UsuariosService()
  const levelOptions = LEVEL_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))

  let draftMembersByPipeline = $state({} as Record<number, ICrmPipelineMember[]>)
  // GET.users already answers only the active users.
  const activeUsers = $derived(usersService.usuarios)
  const changedPipelines = $derived(pipelinesWithChangedMembers(pipelines.records, draftMembersByPipeline))

  const membersOf = (pipelineID: number, storedMembers: ICrmPipelineMember[]) =>
    draftMembersByPipeline[pipelineID] ?? storedMembers

  const saveChanges = async () => {
    showLoading(tr('Saving access...|Guardando accesos...'))
    try {
      for (const pipeline of changedPipelines) {
        await postCrmPipelineMembers(pipelines, pipeline.ID, $state.snapshot(draftMembersByPipeline[pipeline.ID]))
      }
      draftMembersByPipeline = {}
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }
</script>

<Page title="Pipelines: Access|Flujos: Accesos">
  <div class="flex items-center justify-between mb-8 gap-12" aria-label="Pipeline access toolbar">
    <div class="text-gray-600">
      {tr('A user only sees the pipelines they are assigned to. Edit can create and move opportunities; Full can also edit any opportunity and reassign its owner.|Un usuario solo ve los pipelines que tiene asignados. Editar puede crear y mover oportunidades; Completo también edita cualquier oportunidad y reasigna su responsable.')}
    </div>
    <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" disabled={changedPipelines.length === 0}
      label="Saves the member levels of every pipeline that was edited." onClick={saveChanges} />
  </div>

  <div class="overflow-auto" aria-label="Users by pipelines access grid">
    <table class="border-collapse">
      <thead>
        <tr>
          <th class="text-left px-8 py-6">{tr('User|Usuario')}</th>
          {#each pipelines.records as pipeline}
            <th class="px-8 py-6 text-left">{pipeline.Name}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each activeUsers as user}
          <tr class="border-t border-gray-200">
            <td class="px-8 py-4">{user.FirstName} {user.LastName}</td>
            {#each pipelines.records as pipeline}
              <td class="px-8 py-4">
                <SelectBox options={levelOptions} keyId="id" keyName="name" css="w-144"
                  selected={levelOfUser(membersOf(pipeline.ID, pipeline.Members), user.ID)}
                  onChange={(option) => draftMembersByPipeline[pipeline.ID] =
                    setMemberLevel(membersOf(pipeline.ID, pipeline.Members), user.ID, option?.id ?? 0)} />
              </td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</Page>
