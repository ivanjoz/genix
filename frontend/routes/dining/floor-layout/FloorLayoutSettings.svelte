<script lang="ts">
  import { notifyFailure } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import SelectBox from '#components/form/SelectBox.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
  import { SharedListsService } from '#services/business/shared-lists.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import type { IDiningStation } from '../dining.ts'
  import { DiningMenusService } from '../menu-builder/menu-builder.svelte.ts'
  import { makeStation, setDefaultStation, stationDeleteBlocker } from './floor-layout.ts'
  import type { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  // The "Parámetros" tab, in place of the plan: the layout's settings and its prep stations with
  // their routing, each in its own card. Every edit lands on the draft and records an undo step.
  // Deleting the layout is the page's (it also resets the picker and reloads the list): it is only
  // offered here, next to the layout's own settings.
  let { editor, isSaving, onDeleteLayout }: { editor: FloorLayoutEditorState, isSaving: boolean, onDeleteLayout: () => void } = $props()

  const warehousesService = new WarehousesService()
  const productsService = new ProductsService(true)
  const sharedLists = new SharedListsService([1], true)
  const menusService = new DiningMenusService()
  // 0 is "no carta chosen": the QR page then shows the default one (all the restaurant's products).
  const menuOptions = $derived([{ ID: 0, Name: 'Default (all products)|Predeterminada (todos los productos)' }, ...menusService.records])

  const sellableProducts = $derived(productsService.records.filter((product) => product.ss === 1))
  const categories = $derived(sharedLists.ListaRecordsMap.get(1) || [])

  function addStation() {
    if (!editor.draft) return
    editor.draft.Stations = [...(editor.draft.Stations || []), makeStation(editor.draft.Stations || [])]
    editor.checkpoint()
  }

  function deleteStation(station: IDiningStation) {
    const draft = editor.draft
    if (!draft) return
    const blocker = stationDeleteBlocker(draft, station.ID)
    if (blocker) { notifyFailure(tr(blocker)); return }
    draft.Stations = (draft.Stations || []).filter((candidate) => candidate.ID !== station.ID)
    editor.checkpoint()
  }

  const stationColumns: ITableColumn<IDiningStation>[] = [
    { header: 'Station|Estación', field: 'Name', css: 'px-6', getValue: (station) => station.Name,
      onCellEdit: (station, value) => { station.Name = String(value || '').trim(); editor.checkpoint() } },
    { id: 'default', header: 'Default|Defecto', width: '76px', css: 'text-center',
      getValue: (station) => station.IsDefault ? '✓' : '',
      onCellClick: (station, _index, rerender) => {
        if (!editor.draft) return
        setDefaultStation(editor.draft.Stations || [], station.ID)
        editor.checkpoint()
        rerender()
      } },
    { header: 'Products|Productos', field: 'ProductIDs', css: 'px-6',
      getCellOptions: () => sellableProducts, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      onCellMultiSelect: (station, selectedIDs) => { station.ProductIDs = selectedIDs.map(Number); editor.checkpoint() } },
    { header: 'Categories|Categorías', field: 'CategoryIDs', css: 'px-6',
      getCellOptions: () => categories, cellOptionsKeyId: 'ID', cellOptionsKeyName: 'Name',
      onCellMultiSelect: (station, selectedIDs) => { station.CategoryIDs = selectedIDs.map(Number); editor.checkpoint() } },
    { id: 'actions', header: '', width: '36px', css: 'text-center px-4', buttonDeleteHandler: deleteStation },
  ]
</script>

{#if editor.draft}
  {@const draft = editor.draft}
  <div class="flex flex-col gap-12">
    <div class="grid grid-cols-24 gap-8 p-12 rounded-md border border-line bg-surface" aria-label="Layout settings">
      <div class="col-span-24 h3">{tr('Layout|Salón')}</div>
      <Input saveOn={draft} save="Name" css="col-span-24 md:col-span-8" label="Name|Nombre" required onChange={() => editor.checkpoint()} />
      <SearchSelect saveOn={draft} save="WarehouseID" css="col-span-24 md:col-span-8" label="Warehouse of its products|Almacén de sus productos"
        options={(warehousesService.Almacenes || []).filter((warehouse) => warehouse.ss)} keyId="ID" keyName="Name"
        placeholder=":: seleccione ::" onChange={() => editor.checkpoint()} />
      <Input saveOn={draft} save="WaitWarnMinutes" type="number" css="col-span-12 md:col-span-4" label="Warn after (min)|Alerta a los (min)" onChange={() => editor.checkpoint()} />
      <Input saveOn={draft} save="WaitLateMinutes" type="number" css="col-span-12 md:col-span-4" label="Late after (min)|Retraso a los (min)" onChange={() => editor.checkpoint()} />
      <SelectBox saveOn={draft} save="MenuID" css="col-span-24 md:col-span-8" label="Menu shown on the QR|Carta que se muestra en el QR"
        options={menuOptions} keyId="ID" keyName="Name" onChange={() => editor.checkpoint()} />
      <Checkbox checked={!!draft.QRRequiresConfirmation} css="col-span-24 md:col-span-8 self-end"
        onToggle={(isChecked) => { draft.QRRequiresConfirmation = isChecked; editor.checkpoint() }}
        label="QR orders wait for a waiter's confirmation|Los pedidos por QR esperan la confirmación de un mozo" />
      {#if draft.ID}
        <div class="col-span-24 md:col-span-16 flex justify-end items-end">
          <Button color="red" mode="subtle" icon="icon-[fa--trash]" name="Delete|Eliminar" label="Deletes this layout."
            disabled={isSaving} onClick={onDeleteLayout} />
        </div>
      {/if}
    </div>

    <div class="flex flex-col gap-8 p-12 rounded-md border border-line bg-surface" aria-label="Prep stations and the products and categories routed to each">
      <div class="flex items-start justify-between gap-10">
        <div>
          <div class="h3">{tr('Stations|Estaciones')}</div>
          <div class="text-fg-muted">
            {tr('A product goes to the station that names it, else the one naming its category, else the default.|Un producto va a la estación que lo nombra, si no a la que nombra su categoría, si no a la por defecto.')}
          </div>
        </div>
        <Button color="green" icon="icon-[fa--plus]" name="Station|Estación" label="Adds a prep station."
          css="shrink-0" onClick={addStation} />
      </div>
      <VTable columns={stationColumns} data={draft.Stations || []} disableVirtualizer={true} estimateSize={40}
        emptyMessage="No stations.|No hay estaciones." />
    </div>
  </div>
{/if}
