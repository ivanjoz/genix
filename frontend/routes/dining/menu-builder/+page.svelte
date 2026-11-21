<script lang="ts">
  import { confirmWarn, notifyFailure, notifySuccess } from '@genix/ui/notify'
  import Button from '#components/buttons/Button.svelte'
  import SearchSelect from '#components/form/SearchSelect.svelte'
  import Layer from '#components/layers/Layer.svelte'
  import { menuCatalogProducts, type IDiningMenu } from '#core/dining-menu.ts'
  import { tr } from '#core/store.svelte.ts'
  import Page from '#domain/Page.svelte'
  import { SharedListsService } from '#services/business/shared-lists.svelte.ts'
  import { ProductsService } from '#services/production/products.svelte.ts'
  import { DiningMenuSources } from '../floor-service/floor-service.svelte.ts'
  import MenuBlockEditor from './MenuBlockEditor.svelte'
  import MenuBlocksPalette from './MenuBlocksPalette.svelte'
  import MenuCanvas from './MenuCanvas.svelte'
  import { DiningMenusService, MenuBuilderState, makeEmptyMenu, postDiningMenu } from './menu-builder.svelte.ts'
  import { incompleteBlockMessage } from './menu-builder.ts'

  const menusService = new DiningMenusService()
  const productsService = new ProductsService(true)
  const sharedLists = new SharedListsService([1], true)
  // Only its exchange rate is used: a USD-priced product shows in soles, as the customer sees it.
  const menuSources = new DiningMenuSources()
  const builder = new MenuBuilderState()
  let menuPicker = $state({ MenuID: 0 })
  let isSaving = $state(false)

  const catalog = $derived(menuCatalogProducts(productsService.records, menuSources.penRate))
  const categories = $derived((sharedLists.ListaRecordsMap.get(1) || []).map((category) => ({ ID: category.ID, Name: category.Name })))

  // Leaving a draft with changes asks first; the picker goes back when the user stays.
  async function confirmDiscardChanges(): Promise<boolean> {
    if (!builder.isDirty) return true
    return await confirmWarn({
      title: tr('Unsaved changes|Cambios sin guardar'),
      message: tr('The menu has unsaved changes. Discard them?|La carta tiene cambios sin guardar. ¿Descartarlos?'),
    })
  }

  async function openMenu(menuID: number) {
    const menu = menusService.recordsMap.get(menuID)
    if (!menu || menu.ID === builder.draft?.ID) return
    if (!(await confirmDiscardChanges())) { menuPicker.MenuID = builder.draft?.ID || 0; return }
    builder.load(menu)
  }

  async function startNewMenu() {
    if (!(await confirmDiscardChanges())) return
    menuPicker.MenuID = 0
    builder.load(makeEmptyMenu())
  }

  async function saveMenu() {
    if (!builder.draft || isSaving) return
    const menuToSave = $state.snapshot(builder.draft) as IDiningMenu
    const incompleteMessage = incompleteBlockMessage(menuToSave)
    if (incompleteMessage) { notifyFailure(incompleteMessage); return }
    isSaving = true
    try {
      const savedMenu = await postDiningMenu(menuToSave)
      builder.load(savedMenu)
      menuPicker.MenuID = savedMenu.ID
      await menusService.fetchOnline()
      notifySuccess(tr('Menu saved.|Carta guardada.'))
    } catch (saveError) {
      console.error('[menu-builder] save error', { menuID: builder.draft.ID, saveError })
    } finally {
      isSaving = false
    }
  }

  async function deleteMenu() {
    const draft = builder.draft
    if (!draft?.ID || isSaving) return
    const confirmed = await confirmWarn({
      title: tr('Delete menu|Eliminar carta'),
      message: tr(`Delete "${draft.Name}"?|¿Eliminar "${draft.Name}"?`),
    })
    if (!confirmed) return
    isSaving = true
    try {
      await postDiningMenu({ ...($state.snapshot(draft) as IDiningMenu), ss: 0 })
      builder.draft = undefined
      menuPicker.MenuID = 0
      await menusService.fetchOnline()
      notifySuccess(tr('Menu deleted.|Carta eliminada.'))
    } catch (deleteError) {
      console.error('[menu-builder] delete error', { menuID: draft.ID, deleteError })
    } finally {
      isSaving = false
    }
  }
</script>

<Page title="Menu|Carta" noPadding>
  <Layer type="content">
    <div class="flex flex-col lg:flex-row">
      <!-- Left: which carta, its save, and everything that can be dragged onto it. -->
      <div class="w-full lg:w-300 shrink-0 flex flex-col gap-12 p-12 bg-surface border-b lg:border-b-0 lg:border-r border-line lg:h-[calc(100vh-var(--header-height))]">
        <div class="flex items-end gap-8" aria-label="Pick or create a menu">
          <SearchSelect saveOn={menuPicker} save="MenuID" css="flex-1 min-w-0" label="Menu|Carta"
            options={menusService.records} keyId="ID" keyName="Name" placeholder=":: seleccione ::"
            onChange={() => { void openMenu(menuPicker.MenuID) }} />
          <Button color="green" icon="icon-[fa--plus]" label="Starts a new menu." onClick={startNewMenu} />
        </div>
        {#if builder.draft}
          <div class="flex gap-8">
            <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar" label="Saves the menu." css="flex-1"
              disabled={isSaving} onClick={saveMenu} />
            {#if builder.draft.ID}
              <Button color="red" mode="subtle" icon="icon-[fa--trash]" label="Deletes this menu." disabled={isSaving} onClick={deleteMenu} />
            {/if}
          </div>
          {#if builder.isDirty}<div class="text-amber-fg">{tr('Unsaved changes|Cambios sin guardar')}</div>{/if}
          <div class="min-h-0 flex-1">
            <MenuBlocksPalette products={catalog} {categories} />
          </div>
        {/if}
      </div>

      {#if builder.draft}
        <MenuCanvas {builder} menu={builder.draft} products={catalog} {categories} />
        <div class="w-full lg:w-340 shrink-0 bg-surface border-t lg:border-t-0 lg:border-l border-line lg:h-[calc(100vh-var(--header-height))] overflow-auto">
          <MenuBlockEditor {builder} menu={builder.draft} products={catalog} {categories} />
        </div>
      {:else}
        <div class="flex-1 p-12">
          <div class="p-12 rounded-md bg-surface-soft text-fg-muted">
            {tr('Pick a menu or start a new one.|Elija una carta o cree una nueva.')}
          </div>
        </div>
      {/if}
    </div>
  </Layer>
</Page>
