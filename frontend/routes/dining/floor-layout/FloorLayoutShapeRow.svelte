<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import Input from '#components/form/Input.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { SHAPE_KIND, TABLE_OUTLINE, shapeKindLabels } from '../dining.ts'
  import type { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  // One row above the plan with what can be edited of the selected shape: a table's name, seats and
  // outline, a text's words, a mark's station, labelled beside each field, and at its end the actions
  // on the selection (duplicate, rotate, delete). Position and size are edited on the plan only. The row keeps its height while nothing is selected, so selecting a shape
  // never shifts the plan under the pointer that is starting to drag it.
  let { editor }: { editor: FloorLayoutEditorState } = $props()

  const selectedShapes = $derived(editor.selectedShapes())
  const selectedShape = $derived(selectedShapes.length === 1 ? selectedShapes[0] : undefined)
  const outlineOptions = [{ ID: TABLE_OUTLINE.RECT, Name: 'Cuadrada' }, { ID: TABLE_OUTLINE.ROUND, Name: 'Redonda' }]
</script>

<div class="flex items-center gap-12 h-56 px-12 rounded-md border border-line bg-surface/70" aria-label="Properties of the selected shape">
  {#if selectedShape && editor.draft}
    {@const shape = selectedShape}
    <div class="h3 shrink-0">{tr(shapeKindLabels[shape.Kind])}</div>
    <!-- Keyed by shape: the fields rebind when another shape is selected. -->
    {#key shape.ShapeID}
      {#if shape.Kind === SHAPE_KIND.TABLE}
        <Input saveOn={shape} save="Label" css="w-150" labelPosition="side" size="small" label="Table name|Nombre de mesa" onChange={() => editor.checkpoint()} />
        <Input saveOn={shape} save="Seats" type="number" css="w-70" labelPosition="side" size="small" label="Seats|Asientos" onChange={() => editor.checkpoint()} />
        <SearchSelect saveOn={shape} save="Outline" css="w-130" labelPosition="side" size="small" label="Shape|Forma"
          options={outlineOptions} keyId="ID" keyName="Name" onChange={() => editor.checkpoint()} />
      {:else if shape.Kind === SHAPE_KIND.LABEL}
        <Input saveOn={shape} save="Label" css="w-240" labelPosition="side" size="small" label="Text|Texto" onChange={() => editor.checkpoint()} />
      {:else if shape.Kind === SHAPE_KIND.STATION || shape.Kind === SHAPE_KIND.PICKUP}
        <SearchSelect saveOn={shape} save="StationID" css="w-200" labelPosition="side" size="small" label="Station|Estación"
          options={editor.draft.Stations || []} keyId="ID" keyName="Name" onChange={() => editor.checkpoint()} />
      {:else}
        <div class="text-fg-muted">{tr('Move or resize it on the plan.|Muévalo o cambie su tamaño en el plano.')}</div>
      {/if}
    {/key}
  {:else if selectedShapes.length > 1}
    <div class="text-fg-muted">{tr(`${selectedShapes.length} shapes selected.|${selectedShapes.length} elementos seleccionados.`)}</div>
  {:else}
    <div class="text-fg-muted">
      {tr('Drag an element from the palette onto the plan, or click one on the plan to edit it.|Arrastre un elemento de la paleta al plano, o haga clic en uno del plano para editarlo.')}
    </div>
  {/if}
  {#if selectedShapes.length}
    <div class="ml-auto flex items-center gap-10 shrink-0" aria-label="Actions on the selected shapes">
      <Button icon="icon-[fa--clone]" label="Duplicates the selected shapes (Ctrl+D)." height="compact" color="blue" mode="white"
        onClick={() => editor.duplicateSelection()} />
      <Button icon="icon-[fa--refresh]" label="Rotates the selected shapes 90°." height="compact" color="blue" mode="white"
        onClick={() => editor.rotateSelection()} />
      <Button icon="icon-[fa--trash]" label="Deletes the selected shapes (Delete)." height="compact" color="red" mode="subtle" useCircle
        onClick={() => editor.deleteSelection()} />
    </div>
  {/if}
</div>
