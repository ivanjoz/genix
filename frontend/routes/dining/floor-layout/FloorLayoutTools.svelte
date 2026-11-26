<script lang="ts">
  import Button from '#components/buttons/Button.svelte'
  import NumericSlider from '#components/form/NumericSlider.svelte'
  import { tr } from '#core/store.svelte.ts'
  import { LAYOUT_TOOL_DRAG_TYPE, MAX_CELL_SIZE, MIN_CELL_SIZE, type LayoutTool } from './floor-layout.ts'
  import type { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  // The "Diseño" tab of the left column: the palette of elements to drag onto the plan, undo/redo and
  // the plan's zoom. The drop and the gestures live in FloorLayoutEditor; the actions on the
  // selection, in FloorLayoutShapeRow.
  let { editor }: { editor: FloorLayoutEditorState } = $props()

  const PALETTE: { tool: LayoutTool, icon: string, name: string, label: string }[] = [
    { tool: 'table-rect', icon: 'icon-[fa--square-o]', name: 'Table|Mesa', label: 'A square table.' },
    { tool: 'table-round', icon: 'icon-[fa--circle-thin]', name: 'Round|Redonda', label: 'A round table.' },
    { tool: 'rect', icon: 'icon-[fa--square]', name: 'Block|Bloque', label: 'A block: bar, column, planter.' },
    { tool: 'line', icon: 'icon-[fa--minus]', name: 'Wall|Pared', label: 'A wall or divider, 10 cells long.' },
    { tool: 'label', icon: 'icon-[fa--font]', name: 'Text|Texto', label: 'A text.' },
    { tool: 'station', icon: 'icon-[fa--fire]', name: 'Station|Estación', label: 'Where a prep station is.' },
    { tool: 'pickup', icon: 'icon-[fa--hand-paper-o]', name: 'Pickup|Recojo', label: 'Where waiters collect a station\'s dishes.' },
  ]
</script>

<div class="flex flex-col gap-12">
  <div class="text-fg-muted">{tr('Drag an element onto the plan.|Arrastre un elemento al plano.')}</div>
  <div class="grid grid-cols-2 gap-6" role="list" aria-label="Elements to drag onto the floor plan">
    {#each PALETTE as paletteItem (paletteItem.tool)}
      <div draggable="true" role="listitem" aria-label={tr(paletteItem.label)} title={tr(paletteItem.label)}
        class="flex items-center gap-8 h-36 px-10 rounded-md border border-line bg-surface cursor-grab select-none hover:border-blue-solid hover:bg-blue-bg active:cursor-grabbing"
        ondragstart={(event) => {
          event.dataTransfer?.setData(LAYOUT_TOOL_DRAG_TYPE, paletteItem.tool)
          if (event.dataTransfer) event.dataTransfer.effectAllowed = 'copy'
        }}>
        <i class="{paletteItem.icon} text-fg-muted"></i>{tr(paletteItem.name)}
      </div>
    {/each}
  </div>

  <!-- Zoom and history share a row. The slider keeps its own width (its label is centred over the
       track, so it must not be stretched); the cell size updates live while it moves. -->
  <div class="flex items-center justify-between gap-10">
    <NumericSlider saveOn={editor} save="cellSize" min={MIN_CELL_SIZE} max={MAX_CELL_SIZE} step={2} referenceWidth={170}
      label="Zoom|Zoom" suffix="px" scaleLabels={[MIN_CELL_SIZE, 16, MAX_CELL_SIZE]} />
    <div class="flex items-center gap-8 shrink-0" aria-label="Undo and redo">
      <Button icon="icon-[fa--undo]" label="Undoes the last change (Ctrl+Z).|Deshacer el último cambio (Ctrl+Z)." height="compact" color="blue" mode="white"
        disabled={!editor.canUndo} onClick={() => editor.undo()} />
      <Button icon="icon-[fa--repeat]" label="Redoes the undone change (Ctrl+Y).|Rehacer el cambio deshecho (Ctrl+Y)." height="compact" color="blue" mode="white"
        disabled={!editor.canRedo} onClick={() => editor.redo()} />
    </div>
  </div>
</div>
