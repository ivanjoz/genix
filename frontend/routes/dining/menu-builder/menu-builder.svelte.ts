import { DEFAULT_MENU_THEME, defaultMenuBlocks, type IDiningMenu } from '#core/dining-menu.ts'
import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// The company's active cartas, shared by the builder and the layout settings.
export class DiningMenusService extends GetHandler<IDiningMenu> {
  route = 'dining-menus'
  useCache = { min: 0.2, ver: 1 }

  handler(result: { records?: IDiningMenu[] }): void {
    const menus = (result?.records || []).filter((menu) => menu.ss)
    menus.sort((leftMenu, rightMenu) => leftMenu.Name.localeCompare(rightMenu.Name))
    this.records = menus
    this.recordsMap = new Map(menus.map((menu) => [menu.ID, menu]))
  }

  constructor() {
    super()
    this.fetch()
  }
}

// Creates or updates a carta; ss = 0 deletes it. The answer is the stored carta.
export const postDiningMenu = (menu: IDiningMenu) => {
  return POST({ route: 'dining-menu', data: menu, refreshRoutes: ['dining-menus'] }) as Promise<IDiningMenu>
}

export const makeEmptyMenu = (): IDiningMenu =>
  ({ ID: 0, Name: '', Theme: { ...DEFAULT_MENU_THEME }, Blocks: defaultMenuBlocks(), ss: 1, upd: 0 })

// The carta being edited, the selected block and the saved copy the unsaved-changes check compares to.
export class MenuBuilderState {
  draft = $state<IDiningMenu>()
  // Index of the selected block in draft.Blocks; -1 shows the carta's own settings.
  selectedIndex = $state(-1)
  #savedSnapshot = ''

  isDirty = $derived(!!this.draft && JSON.stringify($state.snapshot(this.draft)) !== this.#savedSnapshot)

  load(menu: IDiningMenu) {
    this.draft = structuredClone(menu)
    this.selectedIndex = -1
    this.#savedSnapshot = JSON.stringify(menu)
  }

  // Replaces the blocks after an edit of menu-builder.ts; selectedIndex follows a drop or a move.
  applyEdit(blocks: IDiningMenu['Blocks'], selectedIndex: number = this.selectedIndex) {
    if (!this.draft) return
    this.draft.Blocks = blocks
    this.selectedIndex = selectedIndex
  }
}
