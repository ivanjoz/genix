<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { postCrmCustomField, type CrmCustomFieldsService, type ICrmCustomField, type ICrmCustomFieldOption } from '#services/crm/crm-pipeline.svelte.ts'
  import { CUSTOM_FIELD_TYPE_OPTIONS, customFieldHasOptions, makeEmptyCustomField, makeEmptyOption, removeOption } from './settings.ts'

  // The catalog is company-wide; each pipeline picks the fields it shows (PipelinesTab).
  let { customFields }: { customFields: CrmCustomFieldsService } = $props()

  const ui = useUI()
  const FIELD_LAYER_ID = 2
  const typeOptions = CUSTOM_FIELD_TYPE_OPTIONS.map((option) => ({ ...option, name: tr(option.name) }))
  const typeNames = new Map(typeOptions.map((option) => [option.id, option.name]))
  let fieldForm = $state(makeEmptyCustomField())

  const openField = (customField: ICrmCustomField) => {
    fieldForm = structuredClone($state.snapshot(customField))
    ui.openSideLayer(FIELD_LAYER_ID)
  }

  const saveField = async (isDelete = false) => {
    if (isDelete && !(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${fieldForm.Name}"?|¿Eliminar "${fieldForm.Name}"?`) }))) return
    showLoading(tr('Saving field...|Guardando campo...'))
    try {
      const savedField = await postCrmCustomField(customFields, { ...$state.snapshot(fieldForm), ss: isDelete ? 0 : 1 })
      if (isDelete) ui.openSideLayer(0)
      else fieldForm = structuredClone({ ...savedField, Options: savedField.Options ?? [] })
    } catch (error) {
      notifyFailure(error as string)
    }
    hideLoading()
  }

  // The table shows the live options (a soft-deleted one stays in the record but not on screen), so
  // removing one finds its position in the full list first.
  const visibleOptions = $derived(fieldForm.Options.filter((option) => option.Status !== 0))
  const optionColumns: ITableColumn<ICrmCustomFieldOption>[] = [
    {
      id: 'name', header: 'Option|Opción', css: 'px-6', getValue: (option) => option.Name,
      onCellEdit: (option, value) => { option.Name = String(value) },
    },
    { id: 'actions', header: '', headerCss: 'w-[52px]', css: 'px-4', getValue: () => '' },
  ]

  const columns: ITableColumn<ICrmCustomField>[] = [
    { header: 'Field|Campo', css: 'px-6', highlight: true, getValue: (e) => e.Name },
    { header: 'Type|Tipo', css: 'px-6', getValue: (e) => typeNames.get(e.Type) || '' },
    { header: 'Options|Opciones', css: 'px-6 text-center', getValue: (e) => e.Options.filter((option) => option.Status !== 0).length || '' },
  ]
</script>

<Layer type="content">
  <div class="flex items-center justify-end mb-6" aria-label="Custom fields toolbar">
    <Button color="green" icon="icon-[fa--plus]" name="New|Nuevo" label="Opens the side layer to create a custom field."
      onClick={() => openField(makeEmptyCustomField())} />
  </div>
  <VTable {columns} data={customFields.records} css="w-full" maxHeight="calc(100vh - 8rem - 16px)"
    selected={fieldForm.ID} isSelected={(customField, selectedID) => customField.ID === selectedID} onRowClick={openField} />
</Layer>

<Layer
  id={FIELD_LAYER_ID}
  type="side"
  sideLayerSize={520}
  title={fieldForm.ID > 0 ? fieldForm.Name : 'New field|Nuevo campo'}
  titleCss="h2 mb-6"
  css="px-12 py-10"
  contentCss="px-0"
  onSave={() => saveField()}
  onDelete={fieldForm.ID > 0 ? () => saveField(true) : undefined}
  onClose={() => { fieldForm = makeEmptyCustomField() }}
>
  <div class="grid grid-cols-24 gap-10 mt-8" aria-label="Custom field form">
    <Input bind:saveOn={fieldForm} save="Name" label="Name|Nombre" required={true} css="col-span-24 md:col-span-14" />
    <SearchSelect bind:saveOn={fieldForm} save="Type" label="Type|Tipo" required={true}
      options={typeOptions} keyId="id" keyName="name" css="col-span-24 md:col-span-10" />
    <div class="col-span-24 text-gray-600">
      {tr('The type cannot change once an opportunity holds a value.|El tipo no se puede cambiar cuando una oportunidad ya tiene un valor.')}
    </div>
  </div>
  {#if customFieldHasOptions(fieldForm.Type)}
    <div class="flex items-center justify-between mt-16 mb-6">
      <h3 class="font-bold">{tr('Options|Opciones')}</h3>
      <Button color="green" icon="icon-[fa--plus]" name="Option|Opción" label="Adds an empty option."
        onClick={() => { fieldForm.Options = [...fieldForm.Options, makeEmptyOption()] }} />
    </div>
    <div aria-label="Field options">
      <VTable columns={optionColumns} data={visibleOptions} disableVirtualizer={true} estimateSize={42}
        emptyMessage="No options yet.|Aún no hay opciones.">
        {#snippet cellRenderer(option, column, defaultContent)}
          {#if column.id === 'actions'}
            <div class="flex justify-center">
              <Button color="red" mode="subtle" useCircle height="tiny" icon="icon-[mdi--delete]" label="Removes the option; opportunities keep their stored value."
                onClick={() => { fieldForm.Options = removeOption(fieldForm.Options, fieldForm.Options.indexOf(option)) }} />
            </div>
          {:else}
            {defaultContent}
          {/if}
        {/snippet}
      </VTable>
    </div>
  {/if}
</Layer>
