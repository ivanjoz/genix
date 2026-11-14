<script lang="ts">
  import { notifyFailure } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import Checkbox from '#components/form/Checkbox.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import VTable from '#components/vTable/VTable.svelte'
  import type { ITableColumn } from '#components/vTable/types.ts'
  import { tr } from '#core/store.svelte.ts'
  import { WarehousesService } from '#routes/business/branches-warehouses/branches-warehouses.svelte.ts'
  import { SharedListsService } from '#services/business/shared-lists.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import { SHAPE_KIND, TABLE_OUTLINE, isLineShape, shapeKindLabels, type IDiningStation, type ILayoutShape } from '../dining.ts'
  import {
    applyShapeGeometryFields, fitShapeInGrid, makeStation, setDefaultStation, shapeGeometryFields,
    stationDeleteBlocker, type IShapeGeometryFields,
  } from './floor-layout.ts'
  import type { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  // The editor's side panel: the selected shape's properties, the layout's settings and its prep
  // stations with their routing. Every edit lands on the draft and records an undo step.
  let { editor }: { editor: FloorLayoutEditorState } = $props()

  const warehousesService = new WarehousesService()
  const productsService = new ProductsService(true)
  const sharedLists = new SharedListsService([1], true)

  const PANEL_SHAPE = 1, PANEL_LAYOUT = 2, PANEL_STATIONS = 3
  let panel = $state(PANEL_SHAPE)

  const selectedShapes = $derived(editor.selectedShapes())
  const selectedShape = $derived(selectedShapes.length === 1 ? selectedShapes[0] : undefined)
  const sellableProducts = $derived(productsService.records.filter((product) => product.ss === 1))
  const categories = $derived(sharedLists.ListaRecordsMap.get(1) || [])
  const outlineOptions = [{ ID: TABLE_OUTLINE.RECT, Name: 'Cuadrada' }, { ID: TABLE_OUTLINE.ROUND, Name: 'Redonda' }]

  function onShapeEdited() {
    editor.checkpoint()
  }

  // A typed coordinate or size may leave the grid: it is pulled back in before it is recorded.
  function onGeometryEdited(shape: ILayoutShape, fields: IShapeGeometryFields) {
    if (!editor.draft) return
    applyShapeGeometryFields(shape, fields, editor.draft.GridWidth, editor.draft.GridHeight)
    editor.checkpoint()
  }

  // A smaller grid pulls every shape back inside it.
  function onLayoutEdited() {
    const draft = editor.draft
    if (!draft) return
    for (const shape of draft.Shapes || []) fitShapeInGrid(shape, draft.GridWidth, draft.GridHeight)
    editor.checkpoint()
  }

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

<div class="flex flex-col gap-10">
  <OptionsStrip selected={panel} useMobileGrid
    options={[[PANEL_SHAPE, 'Shape|Elemento'], [PANEL_LAYOUT, 'Layout|Salón'], [PANEL_STATIONS, 'Stations|Estaciones']]}
    onSelect={(option) => { panel = option[0] as number }} />

  {#if editor.draft}
    {@const draft = editor.draft}
    {#if panel === PANEL_SHAPE}
      {#if selectedShape}
        {@const shape = selectedShape}
        <!-- Rebuilt from Points on every change, so a drag on the plan shows in the X/Y/size fields too. -->
        {@const geometry = shapeGeometryFields(shape)}
        <!-- Keyed by shape: the fields rebind when another shape is selected. -->
        {#key shape.ShapeID}
          <div class="grid grid-cols-24 gap-8" aria-label="Properties of the selected shape">
            <div class="col-span-24 h3">{tr(shapeKindLabels[shape.Kind])}</div>
            {#if shape.Kind === SHAPE_KIND.TABLE || shape.Kind === SHAPE_KIND.LABEL}
              <Input saveOn={shape} save="Label" css="col-span-24 md:col-span-12"
                label={shape.Kind === SHAPE_KIND.TABLE ? 'Table name|Nombre de mesa' : 'Text|Texto'} onChange={onShapeEdited} />
            {/if}
            {#if shape.Kind === SHAPE_KIND.TABLE}
              <Input saveOn={shape} save="Seats" type="number" css="col-span-12 md:col-span-6" label="Seats|Asientos"
                onChange={onShapeEdited} />
              <SearchSelect saveOn={shape} save="Outline" css="col-span-12 md:col-span-6" label="Shape|Forma"
                options={outlineOptions} keyId="ID" keyName="Name" onChange={onShapeEdited} />
            {/if}
            {#if shape.Kind === SHAPE_KIND.STATION || shape.Kind === SHAPE_KIND.PICKUP}
              <SearchSelect saveOn={shape} save="StationID" css="col-span-24" label="Station|Estación"
                options={draft.Stations || []} keyId="ID" keyName="Name" onChange={onShapeEdited} />
            {/if}
            <Input saveOn={geometry} save="X" type="number" css="col-span-6" label="X" onChange={() => onGeometryEdited(shape, geometry)} />
            <Input saveOn={geometry} save="Y" type="number" css="col-span-6" label="Y" onChange={() => onGeometryEdited(shape, geometry)} />
            {#if isLineShape(shape)}
              <Input saveOn={geometry} save="X2" type="number" css="col-span-6" label="X2" onChange={() => onGeometryEdited(shape, geometry)} />
              <Input saveOn={geometry} save="Y2" type="number" css="col-span-6" label="Y2" onChange={() => onGeometryEdited(shape, geometry)} />
            {:else}
              <Input saveOn={geometry} save="W" type="number" css="col-span-6" label="Width|Ancho" onChange={() => onGeometryEdited(shape, geometry)} />
              <Input saveOn={geometry} save="H" type="number" css="col-span-6" label="Height|Alto" onChange={() => onGeometryEdited(shape, geometry)} />
            {/if}
          </div>
        {/key}
      {:else if selectedShapes.length > 1}
        <div class="text-fg-muted">{tr(`${selectedShapes.length} shapes selected.|${selectedShapes.length} elementos seleccionados.`)}</div>
      {:else}
        <div class="text-fg-muted">
          {tr('Pick a tool and click the plan to place a shape, or click a shape to edit it.|Elija una herramienta y haga clic en el plano para colocar un elemento, o haga clic en un elemento para editarlo.')}
        </div>
      {/if}
    {:else if panel === PANEL_LAYOUT}
      <div class="grid grid-cols-24 gap-8" aria-label="Layout settings">
        <Input saveOn={draft} save="Name" css="col-span-24" label="Name|Nombre" required onChange={onLayoutEdited} />
        <SearchSelect saveOn={draft} save="WarehouseID" css="col-span-24" label="Warehouse of its products|Almacén de sus productos"
          options={(warehousesService.Almacenes || []).filter((warehouse) => warehouse.ss)} keyId="ID" keyName="Name"
          placeholder=":: seleccione ::" onChange={onLayoutEdited} />
        <Input saveOn={draft} save="GridWidth" type="number" css="col-span-12" label="Width (cells)|Ancho (celdas)" onChange={onLayoutEdited} />
        <Input saveOn={draft} save="GridHeight" type="number" css="col-span-12" label="Height (cells)|Alto (celdas)" onChange={onLayoutEdited} />
        <Input saveOn={draft} save="WaitWarnMinutes" type="number" css="col-span-12" label="Warn after (min)|Alerta a los (min)" onChange={onLayoutEdited} />
        <Input saveOn={draft} save="WaitLateMinutes" type="number" css="col-span-12" label="Late after (min)|Retraso a los (min)" onChange={onLayoutEdited} />
        <Checkbox checked={!!draft.QRRequiresConfirmation} css="col-span-24"
          onToggle={(isChecked) => { draft.QRRequiresConfirmation = isChecked; onLayoutEdited() }}
          label="QR orders wait for a waiter's confirmation|Los pedidos por QR esperan la confirmación de un mozo" />
      </div>
    {:else}
      <div class="flex flex-col gap-8" aria-label="Prep stations and the products and categories routed to each">
        <div class="text-fg-muted">
          {tr('A product goes to the station that names it, else the one naming its category, else the default.|Un producto va a la estación que lo nombra, si no a la que nombra su categoría, si no a la por defecto.')}
        </div>
        <VTable columns={stationColumns} data={draft.Stations || []} disableVirtualizer={true} estimateSize={40}
          emptyMessage="No stations.|No hay estaciones." />
        <Button icon="icon-[fa--plus]" name="Add station|Agregar estación" label="Adds a prep station." css="self-start"
          onClick={addStation} />
      </div>
    {/if}
  {/if}
</div>
