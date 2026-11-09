<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import ColorPicker from '#components/form/ColorPicker.svelte'
  import ColorPickerCell from '#components/form/ColorPickerCell.svelte'
  import Input from '#components/form/Input.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import {
    postCrmPipeline, type CrmCustomFieldsService, type CrmPipelinesService, type ICrmPipeline, type ICrmPipelineStage,
  } from '#services/crm/crm-pipeline.svelte.ts'
  import {
    activeStages, makeEmptyPipeline, makeEmptyStage, moveItem, removeStage, setFieldRequired, toggleField,
  } from './settings.ts'

  // A pipeline is its stages and the custom fields it shows. Members are not edited here: the
  // access page owns them, so the two pages cannot overwrite each other.
  let { pipelines, customFields }: { pipelines: CrmPipelinesService, customFields: CrmCustomFieldsService } = $props()

  const ui = useUI()
  const PIPELINE_LAYER_ID = 1
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
    { id: 'actions', header: '', width: '130px', css: 'px-4', getValue: () => '' },
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
  sideLayerSize={760}
  title={pipelineForm.ID > 0 ? pipelineForm.Name : 'New pipeline|Nuevo pipeline'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => savePipeline()}
  onDelete={pipelineForm.ID > 0 ? () => savePipeline(true) : undefined}
  onClose={() => { pipelineForm = makeEmptyPipeline() }}
>
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Pipeline form">
    <Input bind:saveOn={pipelineForm} save="Name" label="Name|Nombre" required={true} css="col-span-24 md:col-span-16" />
    <ColorPicker bind:saveOn={pipelineForm} save="Color" label="Color" css="col-span-24 md:col-span-8" />
    <Input bind:saveOn={pipelineForm} save="Description" label="Description|Descripción" css="col-span-24" />
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
  <div class="flex flex-col gap-6" aria-label="Pipeline custom fields">
    {#each customFields.records as customField}
      {@const picked = pickedField(customField.ID)}
      <div class="flex items-center gap-16">
        <Checkbox label={customField.Name} checked={!!picked} css="w-256"
          onToggle={() => pipelineForm.Fields = toggleField(pipelineForm.Fields, customField.ID)} />
        {#if picked}
          <Checkbox label="Required|Obligatorio" checked={picked.Required}
            onToggle={(isRequired) => pipelineForm.Fields = setFieldRequired(pipelineForm.Fields, customField.ID, isRequired)} />
        {/if}
      </div>
    {:else}
      <div class="text-gray-500">{tr('No custom fields yet; create them in the Custom fields tab.|Aún no hay campos; créelos en la pestaña Campos personalizados.')}</div>
    {/each}
  </div>
</Layer>
