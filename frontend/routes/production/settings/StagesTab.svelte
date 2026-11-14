<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import {
    postProductionStage, ProductionStagesService, type IProductionStage,
  } from '#services/production/production-settings.svelte.ts'

  // The stages of the production orders board ("Corte", "Aparado", "Armado"), left to right by
  // position. A stage is information only: moving an order between stages moves no stock.
  const ui = useUI()
  const STAGE_LAYER_ID = 2
  const stages = new ProductionStagesService(true)

  let stageForm = $state({} as IProductionStage)

  const openStage = (stage: IProductionStage) => {
    stageForm = { ...stage }
    ui.openSideLayer(STAGE_LAYER_ID)
  }

  // A new stage goes after the last one, ten apart, so one can be slotted in between later.
  const nextStagePosition = () => Math.max(0, ...stages.records.map((stage) => stage.Position)) + 10

  const saveStage = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({
      title: tr('Delete|Eliminar'), message: tr(`Delete "${stageForm.Name}"?|¿Eliminar "${stageForm.Name}"?`),
    }))) return
    showLoading(tr('Saving stage...|Guardando etapa...'))
    try {
      await postProductionStage({ ...stageForm, ss: isDelete ? 0 : 1 })
      ui.openSideLayer(0)
      stageForm = {} as IProductionStage
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  const columns: ITableColumn<IProductionStage>[] = [
    { header: 'Position|Posición', headerCss: 'w-90', css: 'text-center', getValue: (stage) => stage.Position },
    { header: 'Stage|Etapa', css: 'px-6', highlight: true, getValue: (stage) => stage.Name },
  ]
</script>

<Layer type="content">
  <div class="flex items-center justify-between mb-6 gap-12 flex-wrap" aria-label="Stages toolbar with create button">
    <div class="text-gray-600">{tr('The board shows one column per stage, ordered by position.|El tablero muestra una columna por etapa, ordenadas por posición.')}</div>
    <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a stage."
      onClick={() => openStage({ ID: 0, Name: '', Position: nextStagePosition(), ss: 1, upd: 0 })} />
  </div>
  <VTable
    {columns}
    data={stages.records}
    css="w-full"
    maxHeight="calc(100vh - 8rem - 16px)"
    selected={stageForm.ID}
    isSelected={(stage, selectedID) => stage.ID === selectedID}
    onRowClick={openStage}
    emptyMessage="No stages yet.|Aún no hay etapas."
  />
</Layer>

<Layer
  id={STAGE_LAYER_ID}
  type="side"
  sideLayerSize={480}
  title={stageForm.ID > 0 ? stageForm.Name : 'New stage|Nueva etapa'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => saveStage()}
  onDelete={stageForm.ID > 0 ? () => saveStage(true) : undefined}
  onClose={() => { stageForm = {} as IProductionStage }}
>
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Stage form">
    <Input bind:saveOn={stageForm} save="Name" label="Name|Nombre" required={true} css="col-span-24 md:col-span-16" />
    <Input bind:saveOn={stageForm} save="Position" label="Position|Posición" type="number" css="col-span-24 md:col-span-8" />
  </div>
</Layer>
