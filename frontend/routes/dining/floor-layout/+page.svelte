<script lang="ts">
  import { untrack } from 'svelte'
  import { confirmWarn, notifySuccess } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import OptionsStrip from '#components/navigation/OptionsStrip.svelte'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { DiningLayoutsService, postDiningLayout } from '../dining.svelte.ts'
  import type { IDiningLayout } from '../dining.ts'
  import FloorLayoutEditor from './FloorLayoutEditor.svelte'
  import FloorLayoutSettings from './FloorLayoutSettings.svelte'
  import FloorLayoutShapeRow from './FloorLayoutShapeRow.svelte'
  import FloorLayoutTools from './FloorLayoutTools.svelte'
  import { makeEmptyLayout, trimGridToShapes } from './floor-layout.ts'
  import { FloorLayoutEditorState, lastOpenedLayoutID, rememberOpenedLayout } from './floor-layout.svelte.ts'

  const layoutsService = new DiningLayoutsService()
  const editor = new FloorLayoutEditorState()
  let layoutPicker = $state({ LayoutID: 0 })
  let isSaving = $state(false)

  // "Diseño" puts the tools on the left and the selected shape's row over the plan; "Parámetros"
  // puts the layout settings and stations in place of the plan.
  const TAB_DESIGN = 1, TAB_SETTINGS = 2
  let editorTab = $state(TAB_DESIGN)

  // A reload reopens the layout last opened, once the list has it; only if nothing is open yet.
  let hasRestoredLastLayout = false
  $effect(() => {
    const lastLayout = layoutsService.recordsMap.get(lastOpenedLayoutID())
    if (hasRestoredLastLayout || !lastLayout) return
    hasRestoredLastLayout = true
    untrack(() => {
      if (editor.draft) return
      editor.load(lastLayout)
      layoutPicker.LayoutID = lastLayout.ID
    })
  })

  // Leaving a draft with changes asks first; the picker goes back when the user stays.
  async function confirmDiscardChanges(): Promise<boolean> {
    if (!editor.isDirty) return true
    return await confirmWarn({
      title: tr('Unsaved changes|Cambios sin guardar'),
      message: tr('The layout has unsaved changes. Discard them?|El salón tiene cambios sin guardar. ¿Descartarlos?'),
    })
  }

  async function openLayout(layoutID: number) {
    const layout = layoutsService.recordsMap.get(layoutID)
    if (!layout || layout.ID === editor.draft?.ID) return
    if (!(await confirmDiscardChanges())) { layoutPicker.LayoutID = editor.draft?.ID || 0; return }
    editor.load(layout)
    rememberOpenedLayout(layout.ID)
  }

  async function startNewLayout() {
    if (!(await confirmDiscardChanges())) return
    layoutPicker.LayoutID = 0
    editor.load(makeEmptyLayout())
  }

  async function saveLayout() {
    if (!editor.draft || isSaving) return
    isSaving = true
    try {
      // Stored as the used area plus its margin: the free space left while editing is dropped.
      const layoutToSave = $state.snapshot(editor.draft) as IDiningLayout
      const trimShift = trimGridToShapes(layoutToSave)
      const savedLayout = await postDiningLayout(layoutToSave)
      // The answer carries the IDs the backend gave the new shapes: editing goes on from it.
      editor.load(savedLayout, { x: editor.viewAnchor.x + trimShift.x, y: editor.viewAnchor.y + trimShift.y })
      layoutPicker.LayoutID = savedLayout.ID
      rememberOpenedLayout(savedLayout.ID)
      await layoutsService.fetchOnline()
      notifySuccess(tr('Layout saved.|Salón guardado.'))
    } catch (saveError) {
      console.error('[floor-layout] save error', { layoutID: editor.draft.ID, saveError })
    } finally {
      isSaving = false
    }
  }

  async function deleteLayout() {
    const draft = editor.draft
    if (!draft?.ID || isSaving) return
    const confirmed = await confirmWarn({
      title: tr('Delete layout|Eliminar salón'),
      message: tr(`Delete "${draft.Name}"? Its tables stop taking orders.|¿Eliminar "${draft.Name}"? Sus mesas dejan de recibir pedidos.`),
    })
    if (!confirmed) return
    isSaving = true
    try {
      await postDiningLayout({ ...($state.snapshot(draft) as IDiningLayout), ss: 0 })
      editor.draft = undefined
      layoutPicker.LayoutID = 0
      rememberOpenedLayout(0)
      await layoutsService.fetchOnline()
      notifySuccess(tr('Layout deleted.|Salón eliminado.'))
    } catch (deleteError) {
      console.error('[floor-layout] delete error', { layoutID: draft.ID, deleteError })
    } finally {
      isSaving = false
    }
  }
</script>

<Page title="Floor layout|Diseño de salones" noPadding>
  <Layer type="content">
    <div class="flex flex-col lg:flex-row">
      <!-- Left column: which layout, its save, and the tab that drives both columns. Edge to
           edge and as tall as the screen, so it reads as the page's side panel. -->
      <div class="w-full lg:w-340 shrink-0 flex flex-col gap-12 p-12 bg-surface border-b lg:border-b-0 lg:border-r border-line lg:min-h-[calc(100vh-var(--header-height))]">
        <div class="flex items-end gap-8" aria-label="Pick or create a layout">
          <SearchSelect saveOn={layoutPicker} save="LayoutID" css="flex-1 min-w-0" label="Layout|Salón"
            options={layoutsService.records} keyId="ID" keyName="Name" placeholder=":: seleccione ::"
            onChange={() => { void openLayout(layoutPicker.LayoutID) }} />
          <Button color="green" icon="icon-[fa--plus]" label="Starts drawing a new layout.|Crear un plano nuevo." onClick={startNewLayout} />
        </div>
        {#if editor.draft}
          <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" label="Saves the whole layout.|Guardar el plano completo." css="w-full"
            disabled={isSaving} onClick={saveLayout} />
          <OptionsStrip selected={editorTab} useMobileGrid containerCss="w-full" buttonCss="flex-1 justify-center"
            options={[[TAB_DESIGN, 'Design|Diseño'], [TAB_SETTINGS, 'Settings|Parámetros']]}
            onSelect={(option) => { editorTab = option[0] as number }} />
          {#if editorTab === TAB_DESIGN}
            <FloorLayoutTools {editor} />
          {:else}
            <div class="text-fg-muted">
              {tr('The layout settings and its prep stations are shown on the right.|Los parámetros del salón y sus estaciones de preparación se muestran a la derecha.')}
            </div>
          {/if}
          {#if editor.isDirty}<div class="text-amber-fg">{tr('Unsaved changes|Cambios sin guardar')}</div>{/if}
        {/if}
      </div>

      <!-- Right column, as tall as the screen. Diseño: the selected shape's bar tight over the plan,
           which takes the rest of the height. Parámetros: only the settings, scrolling if long. -->
      <div class="flex-1 min-w-0 w-full flex flex-col px-12 pb-12 lg:h-[calc(100vh-var(--header-height))] {editorTab === TAB_DESIGN ? 'pt-8 gap-4' : 'pt-12 overflow-auto'}">
        {#if editor.draft}
          {#if editorTab === TAB_SETTINGS}
            <FloorLayoutSettings {editor} {isSaving} onDeleteLayout={deleteLayout} />
          {:else}
            <FloorLayoutShapeRow {editor} />
            <FloorLayoutEditor {editor} />
          {/if}
        {:else}
          <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
            {tr('Pick a layout or start a new one.|Elija un salón o cree uno nuevo.')}
          </div>
        {/if}
      </div>
    </div>
  </Layer>
</Page>
