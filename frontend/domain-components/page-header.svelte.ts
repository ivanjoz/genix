import type { Snippet } from 'svelte'

// What the open page adds to the header around its title: Page's `header`, `subtitle` and back
// button props. The header lives in the layout, so the page hands it over through this shared state.
export interface PageHeaderContent {
  // Replaces the title / tabs with the page's own content.
  content?: Snippet
  // A second line under the title.
  subtitle?: string
  // Set when the page shows the back button left of the title.
  onBack?: () => void
  // Page's `useMenuSearchBox`: its options drawn as a search box in place of the title.
  menuSearchBox?: {
    options: { id: number, name: string }[]
    selectedOption?: number
    onSelectOption?: (optionID: number) => void
  }
}

// $state.raw: Page clears the object only while it is still the one it published, an identity
// check a deep $state proxy would break.
class PageHeader {
  current = $state.raw<PageHeaderContent>({})
}

export const pageHeader = new PageHeader()
