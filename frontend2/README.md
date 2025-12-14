# Genix Frontend 2

Frontend using Svelte 5, SvelteKit, and Tailwind CSS.

**Tailwind Configuration:** `--spacing` is set to `1px`.

---

## UI Components

### Page

Container for authenticated pages. All non-public pages must be wrapped in this component. Redirects to `/login` if not authenticated.

If the page is subdivided into sections, use the `options` and `selected` props. These options are rendered in the top main menu, so a maximum of 3 options is recommended because they are stacked horizontally.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `title` | `string` | Yes | Page title displayed in header |
| `children` | `Snippet` | Yes | Page content |
| `options` | `{id: number, name: string}[]` | No | Tab options rendered in header menu (max 3 recommended) |
| `sideLayerSize` | `number` | No | Width of side layer in pixels |

**Example:**
```svelte
<script>
  let view = $state(1)
</script>

<Page title="New Page"
  options={[[1,"Section 1"],[2,"Section 2"]]}
  selected={view}
>
  {#if view === 1}
    <div>Content 1</div>
  {/if}
</Page>
```

---

### OptionsStrip

Horizontal menu for creating sections. Can be used to render options menu within the page itself, inside a Layer, or in small components.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `options` | `T[]` | Yes | Array of options. Format: `[id, name]` or `[id, name, [mobileLine1, mobileLine2]]` |
| `selected` | `number` | Yes | Currently selected option ID |
| `onSelect` | `(e: T) => void` | Yes | Callback when option is selected |
| `keyId` | `keyof T` | No | Key for option ID (if not using array format) |
| `keyName` | `keyof T` | No | Key for option name (if not using array format) |
| `buttonCss` | `string` | No | CSS classes for buttons |
| `css` | `string` | No | CSS classes for container |
| `useMobileGrid` | `boolean` | No | Use grid layout on mobile |

**Example:**
```svelte
<script>
  let view = $state(1)
</script>

<Page title="New Page">
  <OptionsStrip selected={view}
    options={[[1,"Section 1"],[2,"Section 2"]]}
  />
  {#if view === 1}
    <div>Content 1</div>
  {/if}
</Page>
```

---

### Layer and Modal

There are two ways to render overlayed information: **Modals** and **Layers**.

- A **Layer** is a container element that appears on the side or bottom of the screen.
- A **Modal** is a centered overlay dialog.

Both components can show action buttons (Save, Delete) using handler props. Providing `onSave` automatically shows a save button; providing `onDelete` shows a delete button. Both always display a close button.

The Layer can show a horizontal menu bar for conditional rendering using the `options` prop.

**Layer Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `id` | `number` | No | Layer identifier for open/close |
| `type` | `"side" \| "bottom" \| "content"` | Yes | Layer position type |
| `children` | `Snippet` | Yes | Layer content |
| `title` | `string` | No | Layer title |
| `titleCss` | `string` | No | CSS for title |
| `css` | `string` | No | CSS for container |
| `contentCss` | `string` | No | CSS for content area |
| `options` | `[number, string, string[]?][]` | No | Tab options for internal sections |
| `selected` | `number` | No | Bindable selected option ID |
| `onSave` | `() => void` | No | Shows save button, called on click |
| `onDelete` | `() => void` | No | Shows delete button, called on click |
| `onClose` | `() => void` | No | Called when layer closes |
| `saveButtonName` | `string` | No | Custom save button label |
| `saveButtonIcon` | `string` | No | Custom save button icon class |
| `contentOverflow` | `boolean` | No | Allow content overflow |

Use `Core.openSideLayer(id)` to open and `Core.hideSideLayer()` to close.

**Modal Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `id` | `number` | Yes | Modal identifier |
| `title` | `string \| Snippet` | Yes | Modal title (string or Svelte snippet) |
| `size` | `1-9` | Yes | Modal width (1=600px to 9=1000px) |
| `children` | `Snippet` | No | Modal content |
| `css` | `string` | No | CSS for modal container |
| `headCss` | `string` | No | CSS for header |
| `bodyCss` | `string` | No | CSS for body |
| `isEdit` | `boolean` | No | Changes save button to "Actualizar" |
| `onSave` | `() => void` | No | Shows save button, called on click |
| `onDelete` | `() => void` | No | Shows delete button, called on click |
| `onClose` | `() => void` | No | Called when modal closes |
| `saveIcon` | `string` | No | Custom save button icon |
| `saveButtonLabel` | `string` | No | Custom save button label |

Use `Core.openModal(id)` to open and `closeModal(id)` to close.

**Example:**
```svelte
<script>
  import { Core } from "$core/store.svelte"
  let view = $state(1)
  let layerView = $state(1)
</script>

<Page title="New Page">
  <button class="bx-green"
    onclick={() => { Core.openSideLayer(1) }}
  >
    <i class="icon-plus"></i>
  </button>
  <Layer id={1} bind:selected={layerView} title="I am a layer"
    options={[[1,"Section 1"],[2,"Section 2"]]} 
    onSave={() => { Core.hideSideLayer() }}
  >
    {#if layerView === 1}
      <div>Content Section 1</div>
    {/if}
  </Layer>
  <button class="bx-green"
    onclick={() => { Core.openModal(1) }}
  >
    <i class="icon-plus"></i>
  </button>
  <Modal title="New Modal" id={1} onSave={() => { doSomething() }}>
    <div>Content</div>
  </Modal>
</Page>
```

---

## Form Components

Located in `src/components/`. The most important components for making forms are: `Input`, `SearchSelect`, `SearchCard`, and `Checkbox`. Use `SearchSelect` instead of native `<select>` as it provides autocompletion.

### Universal Props Pattern

All form components share this binding pattern:
- **`saveOn`**: A reference to the form object (bindable). The component reads and writes to this object.
- **`save`**: The field name (key) in the form object where the value will be stored.

### Options Pattern (for SearchSelect, SearchCard)

Components that need an `options` array use these props:
- **`keyName`**: The key in each option object used to render the display value. The rendered value is `option[keyName]`.
- **`keyId`**: The key in each option object used to get the ID. When selected, the component saves `option[keyId]` to the form via `saveOn[save] = option[keyId]`.

---

### Input

Text/number input with validation support.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `saveOn` | `T` | Yes | Form object reference (bindable) |
| `save` | `keyof T` | Yes | Field name to save value |
| `label` | `string` | No | Input label |
| `css` | `string` | No | CSS for container |
| `inputCss` | `string` | No | CSS for input element |
| `type` | `string` | No | Input type (default: "search") |
| `placeholder` | `string` | No | Placeholder text |
| `required` | `boolean` | No | Shows validation indicator |
| `disabled` | `boolean` | No | Disables input |
| `validator` | `(v: string \| number) => boolean` | No | Custom validation function |
| `onChange` | `() => void` | No | Called on value change (on blur) |
| `postValue` | `string \| ElementAST[]` | No | Content after input |
| `baseDecimals` | `number` | No | Decimal precision for storage |
| `transform` | `(v: string \| number) => string \| number` | No | Transform value on blur |
| `useTextArea` | `boolean` | No | Render as textarea |
| `rows` | `number` | No | Textarea rows |
| `dependencyValue` | `number \| string` | No | Triggers refresh when changed |

---

### SearchSelect

Autocomplete dropdown. Use instead of native `<select>` as it provides autocompletion.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `saveOn` | `T` | No | Form object reference (bindable) |
| `save` | `keyof T` | No | Field name to save selected ID |
| `options` | `E[]` | Yes | Array of option objects |
| `keyId` | `keyof E` | Yes | Key for option ID (saved to form) |
| `keyName` | `keyof E` | Yes | Key for option display name (rendered) |
| `label` | `string` | No | Input label |
| `css` | `string` | No | CSS for container |
| `inputCss` | `string` | No | CSS for input |
| `optionsCss` | `string` | No | CSS for dropdown |
| `placeholder` | `string` | No | Placeholder text |
| `selected` | `number \| string` | No | Selected ID (bindable) |
| `onChange` | `(e: E) => void` | No | Called when selection changes |
| `required` | `boolean` | No | Shows validation indicator |
| `disabled` | `boolean` | No | Disables input |
| `notEmpty` | `boolean` | No | Prevents clearing selection |
| `clearOnSelect` | `boolean` | No | Clears input after selection |
| `avoidIDs` | `(number \| string)[]` | No | IDs to exclude from options |
| `max` | `number` | No | Max options to display (default: 100) |
| `icon` | `string` | No | Custom dropdown icon class |
| `showLoading` | `boolean` | No | Shows loading state |

---

### SearchCard

Multi-select input that renders selected options as removable cards. The component displays an input with a space below where selected options appear as cards. The options are saved in the form as an array of IDs.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `saveOn` | `T` | No | Form object reference (bindable) |
| `save` | `keyof T` | No | Field name to save array of IDs |
| `options` | `E[]` | Yes | Array of option objects |
| `keyId` | `keyof E` | Yes | Key for option ID (saved to array) |
| `keyName` | `keyof E` | Yes | Key for option display name (rendered in cards) |
| `label` | `string` | No | Placeholder text |
| `css` | `string` | No | CSS for container |
| `inputCss` | `string` | No | CSS for search input |
| `optionsCss` | `string` | No | CSS for dropdown |
| `cardCss` | `string` | No | CSS for cards container |
| `onChange` | `(e: (string \| number)[]) => void` | No | Called when selection changes |

---

### Checkbox

Boolean toggle checkbox.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `saveOn` | `T` | No | Form object reference (bindable) |
| `save` | `keyof T` | No | Field name to save boolean |
| `label` | `string` | No | Checkbox label |
| `css` | `string` | No | CSS for container |

### Form Layout Rules

- **Grid System:** Always use a 24-column grid (`grid-cols-24`) for form layouts.
- **Spacing:** Use `gap-10` (which translates to 10px in this project) for spacing between inputs. Do not use individual margins (e.g., `mb-10`) on input components.
- **Sizing:** Use `col-span-N` classes to define the width of each input relative to the 24-column grid.

**Form Components Example:**
```svelte
<script>
  const countries = [{ id: 1, value: "UK" }, { id: 2, value: "Canada" }, { id: 3, value: "Germany" }]
  const roles = [{ id: 1, name: "Customer" }, { id: 2, name: "Manager" }, { id: 3, name: "Supervisor" }]

  let userForm = $state({ name: "", countryID: 0, active: false, roleIDs: [] })
</script>

<div class="grid grid-cols-24 gap-10">
  <Input label="Name" bind:saveOn={userForm} save="name" css="col-span-24 md:col-span-12"/>
  <SearchSelect label="Country" bind:saveOn={userForm} save="countryID" 
    options={countries} keyId="id" keyName="value" css="col-span-24 md:col-span-12"
  />
  <Checkbox label="Is Admin" bind:saveOn={userForm} save="active" 
    css="col-span-24" 
  />
  <SearchCard label="Roles" bind:saveOn={userForm} save="roleIDs" 
    options={roles} keyId="id" keyName="name" css="col-span-24"
  />
</div>
```

---

## VTable

Versatile table component with filtering and custom cell rendering. Requires a column definition array of `ITableColumn<T>` elements.

**Props:**
| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `columns` | `ITableColumn<T>[]` | Yes | Column definitions |
| `data` | `T[]` | Yes | Data array |
| `maxHeight` | `string` | No | Max table height |
| `css` | `string` | No | CSS for container div |
| `tableCss` | `string` | No | CSS for table element |
| `selected` | `T` | No | Selected row |
| `isSelected` | `(row: T, selected: T \| number) => boolean` | No | Function to identify selected row |
| `cellRenderer` | `CellRendererSnippet<T>` | No | Svelte 5 snippet for custom cell rendering |
| `filterText` | `string` | No | Text to filter rows |
| `getFilterContent` | `(row: T) => string` | No | Builds string for filter comparison |
| `useFilterCache` | `boolean` | No | Cache filter results |

### ITableColumn Interface

| Prop | Type | Description |
|------|------|-------------|
| `header` | `string \| (() => string)` | Header text (th content) |
| `id` | `string \| number` | Column ID (required when using cellRenderer snippet) |
| `getValue` | `(e: T, idx: number) => string \| number` | Callback to get the value to render in the cell |
| `render` | `(e: T, idx: number) => string \| ElementAST \| ElementAST[]` | Render function that can return an HTML string (use with `{@html ...}`) or an ElementAST for complex renderings |
| `headerCss` | `string` | CSS for header cell |
| `cellCss` | `string` | CSS for body cells. Important: If using, the cell drop default styles so it looses padding. Consider add "px-6" |
| `subcols` | `ITableColumn<T>[]` | Divides the header into sub-headers, each becoming a column |
| `onCellEdit` | `(e: T, value: string \| number) => void` | Renders an edit button |
| `onCellSelect` | `(e: T, value: string \| number) => void` | Renders a select button |
| `buttonEditHandler` | `(e: T) => void` | Renders an edit button in the cell |
| `buttonDeleteHandler` | `(e: T) => void` | Renders a delete button in the cell |

### ElementAST Interface

For complex cell rendering without HTML strings:

| Prop | Type | Description |
|------|------|-------------|
| `tagName` | `"DIV" \| "SPAN" \| "BUTTON"` | HTML tag |
| `text` | `string \| number` | Text content |
| `css` | `string` | CSS classes |
| `id` | `number \| string` | Element ID |
| `onClick` | `(id: number \| string) => void` | Click handler |
| `children` | `ElementAST[]` | Child elements |

### CellRendererSnippet

For more control, VTable accepts a `cellRenderer` Svelte 5 snippet. The snippet arguments are:

| Argument | Type | Description |
|----------|------|-------------|
| `record` | `T` | Row data |
| `col` | `ITableColumn<T>` | Column definition |
| `cellContent` | `any` | Default cell content |
| `index` | `number` | Row index |

Important: Use it instead or large HTML strings.

**Example:**
```svelte
<script>
  let users: IUser = $state([{ id: 1, name: "Peter", age: 21 }, { id: 2, name: "Hans", age: 27 }])
  let filterText = $state("")

  let columns: ITableColumn<IUser>[] = [
    { header: "ID", getValue: e => e.ID },
    { header: "Name", 
      render: e => {
        return {
          css: "flex items-center",
          children: [
            { tagName: "i", css: "icon-user" },
            { text: e.Name }
          ]
        }
      }
    },
    { header: "Age", id: "age" }
  ]
</script>

<div class="i-search">
  <div><i class="icon-search"></i></div>
  <input type="text" onkeyup={ev => {
    const value = String((ev.target as any).value||"")
    throttle(() => { filterText = value },150)
  }}>
</div>
<VTable columns={columns} data={almacenStock}
  filterText={filterText}
  useFilterCache={true}
  getFilterContent={e => {
    const producto = productos.productosMap.get(e.ProductoID)
    return [producto?.Nombre, e.SKU, e.Lote].filter(x => x).join(" ").toLowerCase()
  }}
>
  {#snippet cellRenderer(record: IUser, col: ITableColumn<IUser>)}
    {#if col.id === "age"}
      <div class="flex">
        <div class="ff-bold">Age:</div><div>{record.age}</div>
      </div>
    {/if}
  {/snippet}
</VTable>
```

---

## Services and Cache

### GetHandler Class

For fetch requests that can use cache—specifically those without dynamic filters that need data ready for the view to display information—use the `GetHandler` class to construct the service controller.

**Location Rule:** Service files must be named with the `.svelte.ts` extension (e.g., `productos.svelte.ts`) and placed in the **same folder** as the `+page.svelte` file that uses them. This ensures proper colocation of logic and view.

**How it works:** The class makes two fetches:
1. **First fetch (cache):** Always fetches from cache through the Service Worker. If no cache exists, it returns nothing and `handler()` is not called.
2. **Second fetch (server):** Fetches updated records from the server. These records are combined with cached records in the Service Worker, and `handler()` receives all up-to-date records.

The response can be an array of objects or an object where each key is an array.

The `handler()` function is where the raw response is parsed, prepared, and set in the reactive properties of the class declared using the `$state()` rune. This enables reactivity in the view.

**Properties:**
| Property | Type | Description |
|----------|------|-------------|
| `route` | `string` | Backend service route |
| `useCache` | `{ min: number, ver: number }` | Cache config: `min` = cache duration in minutes, `ver` = version (changing this invalidates the cache) |

**Methods:**
- `handler(response)`: Override to process the response and set reactive state properties
- `fetch()`: Executes the fetch. Must be called in the constructor.

**Example:** `productos.svelte.ts`
```typescript
export class ProductosService extends GetHandler {
  route = "productos"
  useCache = { min: 5, ver: 9 }

  productos: IProducto[] = $state([])
  productosMap: Map<number,IProducto> = $state(new Map())

  handler(response: IProducto[]): void {
    this.productos = response
    this.productosMap = new Map(response.map(x => [x.ID, x]))
  }

  constructor(warehouseID: number){
    super()
    this.route = `productos?warehouse=${warehouseID}`
    this.fetch()
  }
}
```

**Example:** `+page.svelte`
```svelte
<script>
  const warehouseID = 1
  const productos = new ProductosService(warehouseID)
</script>

<VTable data={productos.productos}
  ... 
/>
```
