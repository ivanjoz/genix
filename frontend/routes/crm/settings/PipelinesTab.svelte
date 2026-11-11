<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import ColorPicker from '#components/form/ColorPicker.svelte'
  import ColorPickerCell from '#components/form/ColorPickerCell.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { GOAL_PERIOD } from '#services/crm/crm-goal-metric.ts'
  import { STANDARD_FIELD } from '#services/crm/crm-standard-field.ts'
  import {
    FORECAST_CATEGORY, postCrmPipeline, type CrmCustomFieldsService, type CrmPipelinesService, type ICrmPipeline, type ICrmPipelineStage,
  } from '#services/crm/crm-pipeline.svelte.ts'
  import { TASK_ASSIGN, TASK_KIND } from '#services/crm/crm-task.ts'
  import type { CrmActivityTypesService } from '#services/crm/crm-activity.svelte.ts'
  import type { CrmEmailTemplatesService } from '#services/crm/crm-email-template.svelte.ts'
  import { UsuariosService } from '../../security/users-profiles/users-profiles.svelte'
  import { activityTypeOptions as buildActivityTypeOptions } from '../pipeline/pipeline.ts'
  import {
    activeStages, buildTaskTemplateRows, changeStageTemplates, makeEmptyPipeline, makeEmptyStage, makeEmptyTaskTemplate, moveItem,
    moveTemplateToStage, removeStage, setFieldRequiredFrom, setStageApprover, setStageStandardField, setTemplateAssignment, toggleField,
    type IStageTaskTemplateRow,
  } from './settings.ts'

  // A pipeline is its stages and the custom fields it shows. Members are not edited here: the
  // access page owns them, so the two pages cannot overwrite each other.
  let { pipelines, customFields, activityTypes, emailTemplates }: {
    pipelines: CrmPipelinesService, customFields: CrmCustomFieldsService, activityTypes: CrmActivityTypesService,
    emailTemplates: CrmEmailTemplatesService
  } = $props()

  const ui = useUI()
  const PIPELINE_LAYER_ID = 1
  const GOAL_PERIOD_OPTIONS = [
    { id: GOAL_PERIOD.MONTHLY, name: tr('Monthly|Mensuales') },
    { id: GOAL_PERIOD.WEEKLY, name: tr('Weekly|Semanales') },
    { id: GOAL_PERIOD.DAILY_BY_MONTH, name: tr('Daily | Month|Diaria | Mes') },
    { id: GOAL_PERIOD.DAILY_BY_WEEK, name: tr('Daily | Week|Diaria | Semana') },
  ]
  const FORECAST_CATEGORY_OPTIONS = [
    { id: FORECAST_CATEGORY.PIPELINE, name: tr('Pipeline|Pipeline') },
    { id: FORECAST_CATEGORY.BEST_CASE, name: tr('Best case|Mejor caso') },
    { id: FORECAST_CATEGORY.COMMIT, name: tr('Commit|Compromiso') },
  ]
  let pipelineForm = $state(makeEmptyPipeline())
  const pickedField = (fieldID: number) => pipelineForm.Fields.find((field) => field.FieldID === fieldID)

  const openPipeline = (pipeline: ICrmPipeline) => {
    pipelineForm = structuredClone($state.snapshot(pipeline))
    ui.openSideLayer(PIPELINE_LAYER_ID)
  }

  const savePipeline = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${pipelineForm.Name}"?|¿Eliminar "${pipelineForm.Name}"?`) }))) return
    showLoading(tr('Saving pipeline...|Guardando pipeline...'))
    try {
      const savedPipeline = await postCrmPipeline(pipelines, { ...$state.snapshot(pipelineForm), ss: isDelete ? 0 : 1 })
      if (isDelete) ui.openSideLayer(0)
      else pipelineForm = structuredClone(savedPipeline)
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  // The table shows the live stages (a soft-deleted one stays in the record but not on screen) and
  // edits them in place, so every handler finds the stage's position in the full list first.
  const visibleStages = $derived(pipelineForm.Stages.filter((stage) => stage.Status !== 0))
  const stageIndexOf = (stage: ICrmPipelineStage) => pipelineForm.Stages.indexOf(stage)
  const numberOf = (value: string | number) => {
    const parsed = parseInt(String(value || '0'))
    return isNaN(parsed) ? 0 : parsed
  }

  const stageColumns: ITableColumn<ICrmPipelineStage>[] = [
    {
      id: 'name', header: 'Stage|Etapa', css: 'px-4', getValue: (stage) => stage.Name,
      onCellEdit: (stage, value) => { stage.Name = String(value) },
    },
    { id: 'color', header: 'Color', width: '120px', css: 'px-4', getValue: (stage) => stage.Color },
    {
      id: 'probability', header: 'Probability %|Probabilidad %', width: '120px', align: 'right',
      cellInputType: 'number', getValue: (stage) => stage.Probability,
      onCellEdit: (stage, value) => { stage.Probability = numberOf(value) },
    },
    {
      id: 'staleDays', header: 'Stale days|Días estancada', width: '120px', align: 'right',
      cellInputType: 'number', getValue: (stage) => stage.StaleDays,
      onCellEdit: (stage, value) => { stage.StaleDays = numberOf(value) },
    },
    {
      id: 'forecastCategory', header: 'Forecast', css: 'px-4', field: 'ForecastCategory', cellSelectMode: 'pick',
      cellOptions: FORECAST_CATEGORY_OPTIONS, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      getValue: (stage) => stage.ForecastCategory,
      onCellSelect: (stage, value) => { stage.ForecastCategory = Number(value) },
    },
    { id: 'requiredStandard', header: 'Requires|Exige', width: '210px', css: 'px-4', getValue: () => '' },
    { id: 'actions', header: '', width: '130px', css: 'px-4', getValue: () => '' },
  ]

  // A field can be required from a stage that is already saved: a stage added in this edit has no ID
  // yet, and the server allocates the IDs on save.
  const requiredFromOptions = $derived([
    { id: 0, name: tr('Optional|Opcional') },
    ...visibleStages.filter((stage) => stage.ID > 0).map((stage) => ({ id: stage.ID, name: stage.Name })),
  ])
  // Option 0 clears the approver / makes the task go to the card's owner.
  const usersService = new UsuariosService()
  const userOptions = $derived(usersService.usuarios.map((user) => ({ id: user.ID, name: `${user.FirstName} ${user.LastName}`.trim() })))
  const approverOptions = $derived([{ id: 0, name: tr('No approval|Sin aprobación') }, ...userOptions])
  const assignOptions = $derived([{ id: 0, name: tr('Card owner|Responsable de la tarjeta') }, ...userOptions])
  // Activity type and template names are bilingual ("Call|Llamada"): the select cells show them as given.
  const activityTypeOptions = $derived(buildActivityTypeOptions(activityTypes.records)
    .map((activityType) => ({ ...activityType, name: ui.translate(activityType.name) })))
  const emailTemplateOptions = $derived(emailTemplates.records.map((emailTemplate) => ({ id: emailTemplate.ID, name: ui.translate(emailTemplate.Name) })))
  const TASK_KIND_OPTIONS = [
    { id: TASK_KIND.ACTIVITY, name: tr('Log activity|Registrar actividad') },
    { id: TASK_KIND.SEND_EMAIL, name: tr('Send email|Enviar correo') },
  ]
  const hasUnsavedStage = $derived(visibleStages.some((stage) => stage.ID === 0))

  const INFORMATION_VIEW = 1
  const TASKS_VIEW = 2
  const APPROVALS_VIEW = 3
  let layerView = $state(INFORMATION_VIEW)

  // Templates propose a task every time a card enters their stage. The table flattens them across
  // stages; the option lists are read per row so they follow the users and types as they load.
  const taskTemplateRows = $derived(buildTaskTemplateRows(pipelineForm.Stages))
  const stageOptions = $derived(visibleStages.map((stage) => ({ id: stageIndexOf(stage), name: ui.translate(stage.Name) })))
  const taskTemplateColumns: ITableColumn<IStageTaskTemplateRow>[] = [
    {
      id: 'stage', header: 'Stage|Etapa', css: 'px-6', field: 'StageIndex', cellSelectMode: 'pick',
      getValue: (row) => ui.translate(pipelineForm.Stages[row.StageIndex].Name),
      getCellOptions: () => stageOptions, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      // A saved template keeps its stage: its ID only means something inside that stage.
      disableCellInteractions: (row) => row.template.ID > 0,
      onCellSelect: (row, value) => {
        pipelineForm.Stages = moveTemplateToStage(pipelineForm.Stages, row.StageIndex, row.TemplateIndex, Number(value))
      },
    },
    {
      id: 'title', header: 'Task|Tarea', headerCss: 'w-[30%]', css: 'px-6', getValue: (row) => row.template.Title,
      onCellEdit: (row, value) => { row.template.Title = String(value) },
    },
    {
      id: 'kind', header: 'Kind|Tipo', css: 'px-6', field: 'Kind', cellSelectMode: 'pick',
      cellOptions: TASK_KIND_OPTIONS, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (row, value) => { row.template.Kind = Number(value) },
    },
    {
      // What the task asks for: the activity to log, or the template the email starts from.
      id: 'source', header: 'Activity / Template|Actividad / Plantilla', css: 'px-6', field: 'SourceID', cellSelectMode: 'pick', cellSelectColumns: 2,
      getCellOptions: (row) => row.Kind === TASK_KIND.SEND_EMAIL ? emailTemplateOptions : activityTypeOptions,
      cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (row, value) => {
        if (row.Kind === TASK_KIND.SEND_EMAIL) row.template.EmailTemplateID = Number(value) || 0
        else row.template.ActivityTypeID = Number(value) || 0
      },
    },
    {
      id: 'dueDays', header: 'Due (days)|Vence (días)', align: 'right', cellInputType: 'number',
      getValue: (row) => row.template.DueDays,
      onCellEdit: (row, value) => { row.template.DueDays = numberOf(value) },
    },
    {
      id: 'assign', header: 'Assign to|Asignar a', css: 'px-6', field: 'AssignUserID',
      getCellOptions: () => assignOptions, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (row, value) => {
        const assignUserID = Number(value) || 0
        pipelineForm.Stages[row.StageIndex].TaskTemplates[row.TemplateIndex] = setTemplateAssignment(
          row.template, assignUserID ? TASK_ASSIGN.USER : TASK_ASSIGN.OWNER, assignUserID)
      },
    },
    { id: 'actions', header: '', headerCss: 'w-[52px]', css: 'px-4', getValue: () => '' },
  ]

  // An approver gates a forward move into the stage.
  const approvalColumns: ITableColumn<ICrmPipelineStage>[] = [
    { header: 'Stage|Etapa', css: 'px-6', getValue: (stage) => ui.translate(stage.Name) },
    {
      header: 'Approver|Aprobador', headerCss: 'w-[50%]', css: 'px-6', field: 'ApproverUserID',
      getCellOptions: () => approverOptions, cellOptionsKeyId: 'id', cellOptionsKeyName: 'name',
      onCellSelect: (stage, value) => {
        pipelineForm.Stages = setStageApprover(pipelineForm.Stages, stageIndexOf(stage), Number(value) || 0)
      },
    },
  ]

  const columns: ITableColumn<ICrmPipeline>[] = [
    { header: 'Pipeline', css: 'px-6', highlight: true, getValue: (e) => e.Name },
    { header: 'Stages|Etapas', css: 'px-6 text-center', getValue: (e) => activeStages(e).length },
    { header: 'Custom fields|Campos', css: 'px-6 text-center', getValue: (e) => e.Fields.length },
    { header: 'Members|Miembros', css: 'px-6 text-center', getValue: (e) => e.Members.length },
  ]
</script>

<Layer type="content">
  <div class="flex items-center justify-end mb-6" aria-label="Pipelines toolbar">
    <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a pipeline."
      onClick={() => openPipeline(makeEmptyPipeline())} />
  </div>
  <VTable {columns} data={pipelines.records} css="w-full" maxHeight="calc(100vh - 8rem - 16px)"
    selected={pipelineForm.ID} isSelected={(pipeline, selectedID) => pipeline.ID === selectedID} onRowClick={openPipeline} />
</Layer>

<Layer
  id={PIPELINE_LAYER_ID}
  type="side"
  sideLayerSize={1040}
  title={pipelineForm.ID > 0 ? pipelineForm.Name : 'New pipeline|Nuevo pipeline'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  options={[[INFORMATION_VIEW, 'Information|Información'], [TASKS_VIEW, 'Tasks|Tareas'], [APPROVALS_VIEW, 'Approvals|Aprobaciones']]}
  bind:selected={layerView}
  onSave={() => savePipeline()}
  onDelete={pipelineForm.ID > 0 ? () => savePipeline(true) : undefined}
  onClose={() => { pipelineForm = makeEmptyPipeline(); layerView = INFORMATION_VIEW }}
>
  {#if layerView === TASKS_VIEW}
  <div class="flex items-center justify-between mt-8 mb-6">
    <div class="text-fg-muted">{tr('Each task is proposed every time a card enters its stage.|Cada tarea se propone cada vez que una tarjeta entra a su etapa.')}</div>
    <!-- A new template starts in the first stage; its Stage cell moves it while it is unsaved. -->
    <Button color="green" icon="icon-[fa--plus]" name="Task|Tarea" label="Adds a task template to the first stage." disabled={visibleStages.length === 0}
      onClick={() => { pipelineForm.Stages = changeStageTemplates(pipelineForm.Stages, stageIndexOf(visibleStages[0]), (templates) => [...templates, makeEmptyTaskTemplate()]) }} />
  </div>
  <div aria-label="Stage task templates">
    <VTable columns={taskTemplateColumns} data={taskTemplateRows} disableVirtualizer={true} estimateSize={42}
      emptyMessage="No task templates yet.|Aún no hay tareas.">
      {#snippet cellRenderer(row, column, defaultContent)}
        {#if column.id === 'actions'}
          <div class="flex justify-center">
            <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the task template."
              onClick={() => { pipelineForm.Stages = changeStageTemplates(pipelineForm.Stages, row.StageIndex, (templates) => templates.filter((_, index) => index !== row.TemplateIndex)) }} />
          </div>
        {:else}
          <!-- VTable routes every column with an id here once its select is disabled (a saved template's stage). -->
          {defaultContent}
        {/if}
      {/snippet}
    </VTable>
  </div>
  {:else if layerView === APPROVALS_VIEW}
  <div class="text-fg-muted mt-8 mb-6">{tr('Moving a card forward into a stage with an approver waits for their approval.|Avanzar una tarjeta a una etapa con aprobador espera su aprobación.')}</div>
  <div aria-label="Stage approvals">
    <VTable columns={approvalColumns} data={visibleStages} disableVirtualizer={true} estimateSize={42}
      emptyMessage="No stages yet.|Aún no hay etapas." />
  </div>
  {:else}
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Pipeline form">
    <Input bind:saveOn={pipelineForm} save="Name" label="Name|Nombre" required={true} css="col-span-24 md:col-span-16" />
    <ColorPicker bind:saveOn={pipelineForm} save="Color" label="Color" css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={pipelineForm} save="Description" label="Description|Descripción" css="col-span-24 md:col-span-16" />
    <SearchSelect options={GOAL_PERIOD_OPTIONS} keyId="id" keyName="name" label="Goals|Metas" css="col-span-24 md:col-span-8"
      selected={pipelineForm.GoalPeriodType} onChange={(option) => { pipelineForm.GoalPeriodType = option.id }} />
  </div>

  <div class="flex items-center justify-between mt-16 mb-6">
    <h3 class="font-bold">{tr('Stages (board columns)|Etapas (columnas del tablero)')}</h3>
    <Button color="green" icon="icon-[fa--plus]" name="Stage|Etapa" label="Adds an empty stage at the end."
      onClick={() => { pipelineForm.Stages = [...pipelineForm.Stages, makeEmptyStage()] }} />
  </div>
  <div aria-label="Pipeline stages">
    <VTable columns={stageColumns} data={visibleStages} disableVirtualizer={true} estimateSize={42}
      emptyMessage="No stages yet.|Aún no hay etapas.">
      {#snippet cellRenderer(stage, column)}
        {#if column.id === 'color'}
          <ColorPickerCell bind:saveOn={pipelineForm.Stages[stageIndexOf(stage)]} save="Color" />
        {:else if column.id === 'requiredStandard'}
          <!-- Required to enter this stage or any later one. -->
          <div class="flex gap-10">
            <Checkbox label="Amount|Monto" checked={stage.RequiredStandardFields.includes(STANDARD_FIELD.AMOUNT)}
              onToggle={(isRequired) => { pipelineForm.Stages = setStageStandardField(pipelineForm.Stages, stageIndexOf(stage), STANDARD_FIELD.AMOUNT, isRequired) }} />
            <Checkbox label="Close|Cierre" checked={stage.RequiredStandardFields.includes(STANDARD_FIELD.EXPECTED_CLOSE_DATE)}
              onToggle={(isRequired) => { pipelineForm.Stages = setStageStandardField(pipelineForm.Stages, stageIndexOf(stage), STANDARD_FIELD.EXPECTED_CLOSE_DATE, isRequired) }} />
          </div>
        {:else if column.id === 'actions'}
          <div class="flex gap-4 justify-end">
            <Button color="blue" mode="subtle" useCircle height="tiny" icon="icon-[fa--arrow-up]" label="Moves the stage one place up."
              onClick={() => { pipelineForm.Stages = moveItem(pipelineForm.Stages, stageIndexOf(stage), -1) }} />
            <Button color="blue" mode="subtle" useCircle height="tiny" icon="icon-[fa--arrow-down]" label="Moves the stage one place down."
              onClick={() => { pipelineForm.Stages = moveItem(pipelineForm.Stages, stageIndexOf(stage), 1) }} />
            <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the stage; the server refuses it while it holds open opportunities."
              onClick={() => { pipelineForm.Stages = removeStage(pipelineForm.Stages, stageIndexOf(stage)) }} />
          </div>
        {/if}
      {/snippet}
    </VTable>
  </div>

  <h3 class="mt-16 mb-6 font-bold">{tr('Custom fields shown|Campos personalizados visibles')}</h3>
  {#if hasUnsavedStage}
    <div class="text-fg-muted mb-6">
      {tr('Save the pipeline to require a field from a stage you just added.|Guarde el pipeline para exigir un campo desde una etapa recién agregada.')}
    </div>
  {/if}
  <div class="flex flex-col gap-6" aria-label="Pipeline custom fields">
    {#each customFields.records as customField}
      {@const picked = pickedField(customField.ID)}
      <div class="flex items-center gap-16">
        <Checkbox label={customField.Name} checked={!!picked} css="w-256"
          onToggle={() => pipelineForm.Fields = toggleField(pipelineForm.Fields, customField.ID)} />
        {#if picked}
          <SearchSelect options={requiredFromOptions} keyId="id" keyName="name" label="Required from|Obligatorio desde" css="w-224"
            selected={picked.RequiredFromStageID}
            onChange={(option) => pipelineForm.Fields = setFieldRequiredFrom(pipelineForm.Fields, customField.ID, option.id)} />
        {/if}
      </div>
    {:else}
      <div class="text-gray-500">{tr('No custom fields yet; create them in the Custom fields tab.|Aún no hay campos; créelos en la pestaña Campos personalizados.')}</div>
    {/each}
  </div>
  {/if}
</Layer>
