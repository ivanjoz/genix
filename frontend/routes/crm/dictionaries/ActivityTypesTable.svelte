<script lang="ts">
  import { useUI } from '@genix/ui'
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Modal from '#components/layers/Modal.svelte'
  import T from '#components/misc/T.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { ACTIVITY_ICON_OPTIONS, SYSTEM_ACTIVITY_TYPES } from '#services/crm/crm-activity-types.ts'
  import { postCrmActivityType, type CrmActivityTypesService, type ICrmActivityType } from '#services/crm/crm-activity.svelte.ts'

  // The company's own activity types, edited in place like the lost reasons. The fixed ones
  // (Call, Meeting...) are listed above as read-only chips: every company has them.
  let { activityTypes }: { activityTypes: CrmActivityTypesService } = $props()

  const ICON_MODAL_ID = 1
  const ui = useUI()

  // A new type lives only on screen, at the top of the table, until the user types its name.
  let draftType = $state<ICrmActivityType | null>(null)
  // The row whose icon the modal is choosing.
  let iconPickerRow = $state<ICrmActivityType | null>(null)
  const tableRows = $derived(draftType ? [draftType, ...activityTypes.records] : activityTypes.records)

  const saveType = async (activityType: ICrmActivityType, changes: Partial<ICrmActivityType>, rerender: () => void) => {
    // A draft is saved once it has a name; until then an icon change only updates the row on screen.
    if (activityType.ID === 0 && !(changes.Name ?? activityType.Name)) {
      draftType = { ...activityType, ...changes }
      return
    }
    showLoading(tr('Saving type...|Guardando tipo...'))
    try {
      await postCrmActivityType(activityTypes, { ...activityType, ...changes })
      if (activityType.ID === 0) draftType = null
    } catch (error) {
      notifyFailure(error as string)
      rerender()
    }
    hideLoading()
  }

  const deleteType = async (activityType: ICrmActivityType) => {
    if (activityType.ID === 0) { draftType = null; return }
    if (!(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${activityType.Name}"?|¿Eliminar "${activityType.Name}"?`) }))) return
    await saveType(activityType, { ss: 0 }, () => {})
  }

  const pickIcon = async (icon: string) => {
    const row = iconPickerRow
    ui.closeModal(ICON_MODAL_ID)
    if (row) await saveType(row, { Icon: icon }, () => {})
  }

  const columns: ITableColumn<ICrmActivityType>[] = [
    { id: 'icon', header: 'Icon|Ícono', css: 'w-80 justify-center', useCellRenderer: true },
    {
      header: 'Type|Tipo', css: 'px-6', highlight: true,
      getValue: (activityType) => activityType.Name,
      // An empty name is rejected before it reaches the server; the cell keeps its previous value.
      onBeforeCellChange: (_, value) => String(value).trim() !== '',
      onCellEdit: (activityType, value, rerender) => saveType(activityType, { Name: String(value).trim() }, rerender),
    },
    {
      header: '', css: 'w-60',
      buttonDeleteHandler: deleteType,
    },
  ]
</script>

<div class="flex items-center justify-between mb-8" aria-label="Activity types toolbar">
  <div class="h3">{tr('Activity types|Tipos de actividad')}</div>
  <Button color="green" icon="icon-[fa--plus]" name="Add|Agregar" label="Adds an empty row at the top of the table to type a new activity type."
    disabled={!!draftType} onClick={() => { draftType = { ID: 0, Name: '', Icon: ACTIVITY_ICON_OPTIONS[0].icon, ss: 1, upd: 0 } }} />
</div>

<div class="flex flex-wrap items-center gap-6 mb-10 text-[14px]" aria-label="System activity types">
  <span class="text-fg-muted"><T text="Always available:|Siempre disponibles:" /></span>
  {#each SYSTEM_ACTIVITY_TYPES as systemType (systemType.id)}
    <span class="flex items-center gap-6 px-8 py-2 rounded border border-line bg-surface-soft text-fg-soft">
      <i class={systemType.icon}></i><T text={systemType.name} />
    </span>
  {/each}
</div>

<VTable {columns} data={tableRows} css="w-full" maxHeight="calc(100vh - 12rem)">
  {#snippet cellRenderer(activityType: ICrmActivityType)}
    <button type="button" class="flex items-center justify-center w-full h-full hover:text-purple-solid"
      title={tr('Change icon|Cambiar ícono')}
      onclick={() => { iconPickerRow = activityType; ui.openModal(ICON_MODAL_ID) }}>
      <i class="{activityType.Icon} h-16 w-16"></i>
    </button>
  {/snippet}
</VTable>

<Modal id={ICON_MODAL_ID} size={3} bodyCss="px-16 py-14" title={tr('Choose an icon|Elige un ícono')}>
  <div class="grid grid-cols-8 gap-8" aria-label="Activity icons">
    {#each ACTIVITY_ICON_OPTIONS as iconOption (iconOption.icon)}
      <button type="button"
        class="flex items-center justify-center h-40 rounded border hover:border-line-strong {iconPickerRow?.Icon === iconOption.icon ? 'border-purple-border bg-purple-bg' : 'border-line bg-surface'}"
        aria-label={iconOption.icon} onclick={() => pickIcon(iconOption.icon)}>
        <i class="{iconOption.icon} h-18 w-18"></i>
      </button>
    {/each}
  </div>
</Modal>
