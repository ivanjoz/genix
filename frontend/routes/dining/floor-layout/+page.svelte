<script lang="ts">
  import { confirmWarn, notifySuccess } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { DiningLayoutsService, postDiningLayout } from '../dining.svelte.ts'
  import type { IDiningLayout } from '../dining.ts'
  import FloorLayoutEditor from './FloorLayoutEditor.svelte'
  import FloorLayoutSidePanel from './FloorLayoutSidePanel.svelte'
  import { makeEmptyLayout } from './floor-layout.ts'
  import { FloorLayoutEditorState } from './floor-layout.svelte.ts'

  const layoutsService = new DiningLayoutsService()
  const editor = new FloorLayoutEditorState()
  let layoutPicker = $state({ LayoutID: 0 })
  let isSaving = $state(false)

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
      const savedLayout = await postDiningLayout($state.snapshot(editor.draft) as IDiningLayout)
      // The answer carries the IDs the backend gave the new shapes: editing goes on from it.
      editor.load(savedLayout)
      layoutPicker.LayoutID = savedLayout.ID
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
      await layoutsService.fetchOnline()
      notifySuccess(tr('Layout deleted.|Salón eliminado.'))
    } catch (deleteError) {
      console.error('[floor-layout] delete error', { layoutID: draft.ID, deleteError })
    } finally {
      isSaving = false
    }
  }
</script>

<Page title="Floor layout|Diseño de salones">
  <Layer type="content">
    <div class="flex flex-wrap items-end gap-10 mb-10" aria-label="Pick, create, save or delete a layout">
      <SearchSelect saveOn={layoutPicker} save="LayoutID" css="w-full md:w-280" label="Layout|Salón"
        options={layoutsService.records} keyId="ID" keyName="Name" placeholder=":: seleccione ::"
        onChange={() => { void openLayout(layoutPicker.LayoutID) }} />
      <Button icon="icon-[fa--plus]" name="New layout|Nuevo salón" label="Starts drawing a new layout." onClick={startNewLayout} />
      {#if editor.draft}
        <Button color="green" icon="icon-[fa--floppy-o]" name="Save|Guardar" label="Saves the whole layout."
          disabled={isSaving} onClick={saveLayout} />
        {#if editor.draft.ID}
          <Button color="red" mode="subtle" icon="icon-[fa--trash]" name="Delete|Eliminar" label="Deletes this layout."
            disabled={isSaving} onClick={deleteLayout} />
        {/if}
        {#if editor.isDirty}<span class="text-amber-fg">{tr('Unsaved changes|Cambios sin guardar')}</span>{/if}
      {/if}
    </div>

    {#if editor.draft}
      <div class="flex flex-col lg:flex-row gap-16 items-start">
        <div class="flex-1 min-w-0 w-full"><FloorLayoutEditor {editor} /></div>
        <div class="w-full lg:w-420 shrink-0"><FloorLayoutSidePanel {editor} /></div>
      </div>
    {:else}
      <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
        {tr('Pick a layout or start a new one.|Elija un salón o cree uno nuevo.')}
      </div>
    {/if}
  </Layer>
</Page>
