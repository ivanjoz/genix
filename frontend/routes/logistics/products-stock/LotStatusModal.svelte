<script lang="ts">
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify'
  import { useUI } from '@genix/ui'
  import Modal from '#components/layers/Modal.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import DateInput from '#components/form/DateInput.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { untrack } from 'svelte'
  import { lotQualityOptions } from './lot-traceability'
  import { postProductStockLotStatus, type IProductStockLot, type IProductStockLotStatusChange } from './stock-movement'

  // Moves a lot between Liberado, Cuarentena and Rechazado and edits its expiry (M22, M20). The
  // server moves the lot's stock in or out of its rows' Blocked counters.
  let { id, lot, onSaved }: {
    id: number
    lot: IProductStockLot | undefined
    onSaved: (savedLot: IProductStockLot) => void
  } = $props()

  const ui = useUI()
  const qualityOptions = lotQualityOptions.map((option) => ({ id: option.id, name: tr(option.name) }))
  let statusForm = $state<IProductStockLotStatusChange>({ LotID: 0, QualityStatus: 0, ExpirationDate: 0 })

  $effect(() => {
    const openedLot = lot
    untrack(() => {
      statusForm = {
        LotID: openedLot?.ID || 0,
        QualityStatus: openedLot?.QualityStatus || 0,
        ExpirationDate: openedLot?.ExpirationDate || 0,
      }
    })
  })

  const saveLotStatus = async () => {
    if (!statusForm.QualityStatus) {
      notifyFailure(tr('Select the lot status.|Seleccione el estado del lote.'))
      return
    }
    showLoading(tr('Saving lot...|Guardando lote...'))
    try {
      const savedLot = await postProductStockLotStatus({ ...statusForm, ExpirationDate: statusForm.ExpirationDate || 0 })
      ui.closeModal(id)
      onSaved(savedLot)
    } finally {
      hideLoading()
    }
  }
</script>

<Modal {id} size={3} bodyCss="px-16 py-12" isEdit={true} saveButtonLabel="Guardar"
  title={`${tr('Batch|Lote')} ${lot?.Name || ''}`}
  onSave={() => { void saveLotStatus() }}
>
  <div class="grid grid-cols-2 gap-10" aria-label="Lot quality status and expiry form">
    <SearchSelect css="col-span-2 md:col-span-1" label="Status|Estado"
      keyId="id" keyName="name" options={qualityOptions}
      bind:saveOn={statusForm} save="QualityStatus" required={true} />
    <DateInput css="col-span-2 md:col-span-1" label="Expiry|Vencimiento"
      bind:saveOn={statusForm} save="ExpirationDate" />
  </div>
</Modal>
