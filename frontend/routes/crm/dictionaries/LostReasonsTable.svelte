<script lang="ts">
  import { confirmWarn, hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { postCrmLostReason, type CrmLostReasonsService, type ICrmLostReason } from '#services/crm/crm-pipeline.svelte.ts'

  // The list an opportunity picks from when it is closed as lost, edited in place: no side layer.
  let { lostReasons }: { lostReasons: CrmLostReasonsService } = $props()

  // A new reason lives only on screen, at the top of the table, until the user types its name.
  let draftReason = $state<ICrmLostReason | null>(null)
  const tableRows = $derived(draftReason ? [draftReason, ...lostReasons.records] : lostReasons.records)

  const saveReason = async (lostReason: ICrmLostReason, changes: Partial<ICrmLostReason>, rerender: () => void) => {
    showLoading(tr('Saving reason...|Guardando motivo...'))
    try {
      await postCrmLostReason(lostReasons, { ...lostReason, ...changes })
      if (lostReason.ID === 0) draftReason = null
    } catch (error) {
      notifyFailure(error as string)
      rerender()
    }
    hideLoading()
  }

  const deleteReason = async (lostReason: ICrmLostReason) => {
    if (lostReason.ID === 0) { draftReason = null; return }
    if (!(await confirmWarn({ title: tr('Delete|Eliminar'), message: tr(`Delete "${lostReason.Name}"?|¿Eliminar "${lostReason.Name}"?`) }))) return
    await saveReason(lostReason, { ss: 0 }, () => {})
  }

  const columns: ITableColumn<ICrmLostReason>[] = [
    {
      header: 'Reason|Motivo', css: 'px-6', highlight: true,
      getValue: (lostReason) => lostReason.Name,
      // An empty name is rejected before it reaches the server; the cell keeps its previous value.
      onBeforeCellChange: (_, value) => String(value).trim() !== '',
      onCellEdit: (lostReason, value, rerender) => saveReason(lostReason, { Name: String(value).trim() }, rerender),
    },
    {
      header: '', css: 'w-60',
      buttonDeleteHandler: deleteReason,
    },
  ]
</script>

<div class="flex items-center justify-between mb-8" aria-label="Lost reasons toolbar">
  <div class="h3">{tr('Lost reasons|Motivos de pérdida')}</div>
  <Button color="green" icon="icon-[fa--plus]" name="Add|Agregar" label="Adds an empty row at the top of the table to type a new lost reason."
    disabled={!!draftReason} onClick={() => { draftReason = { ID: 0, Name: '', ss: 1, upd: 0 } }} />
</div>
<VTable {columns} data={tableRows} css="w-full" maxHeight="calc(100vh - 8rem - 16px)" />
