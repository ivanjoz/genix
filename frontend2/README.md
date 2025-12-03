# Genix Frontend 2

Frontend using Svelte 5, SvelteKit, and Tailwind CSS.

Tailwind Configuration:
- Important: The --spacing is set to 1px

## 🎨 UI Components

### Page and layers

All non-public pages need to be inside the Page component.

If the page is subdivided in sections, you can use the prop "options" and "selected" in the Page component. This options are rendered in the the top main menu so is recomended a maximun of 3 options because they are stacked horizontally.

Example:
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

The component OptionsStrip can also be used to render options menu to create sections in the page itself or inside layer or small components.

Example:
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

There are two ways to render overlayed information: Modals and Layers. A Layer is a container element that appear on the side or on the bottom of the escreen.

The Layer and Modal component can be shown buttons like: Save o Delete using the prop handlers. For example using the onSave() handler shows automatically a button. There is also an onDelete() and onClose() handlers. The Layer and Modal always display a close button.

The Layer can show a horizontal menu bar for conditional rendering using de "options" prop.

Example:
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

### Form Components

The base components are in the src/components/ folder.
The most important components for making forms are: Input.svelte, SearchSelect and SearchCard.svelte. Use SearchSelect instead of select, it provides autocompletion.

All the form components has this logic. There is a prop called "saveOn" where is reference of the form object is passed a props called "save" thats the name of the field that will be use to save the content of the input or select.

In components that needs an "options" array. The component render the value of the option based on the "keyName" props, so the value to be rendered is object[keyName]. The component set the ID in the form getted with object[keyId]

The SearchCard component renders an input with a space above or to the side where the selected options are render as cards. The options are saved in the form as an array of ids.

Example:
```svelte
<script>
  const countries = [{ id: 1, value: "UK", id: 2, value: "Canada", id: 3: value: "Germany" }], { id: 2, name: "" }
  const roles = [{ id: 1, name: "Costumer" }, { id: 2, name: "Manager" }, { id: 3, name: "Supervisor" }]

  let userForm = $state({ name: "", countryID: 0, active: false, roleIDs: [] })
</script>

<div class="grid grid-cols-12 md:flex md:flex-row gap-10">
  <Input label="Nombre" bind:saveOn={userForm} save="name" css="col-span-12"/>
  <SearchSelect label="Nombre" bind:saveOn={userForm} save="countryID" 
    options={countries} keyId="id" keyName="value" css="col-span-12"
  />
  <Checkbox label="Is Admin" bind:saveOn={userForm} save="active" 
    css="col-span-12" 
  />
  <SearchCard label="Nombre" bind:saveOn={userForm} save="roleIDs" 
    options={countries} keyId="id" keyName="name" css="col-span-12"
  />
</div>
```

### Tables: VTable

The VTable table component is very versatile. It need a column definition, an array of ITableColumn<T> elements. 

The VTable component have the following props:
- columns: ITableColumn<T>[]
- data: T[]
- maxHeight: string
- css: string. Of the div container
- tableCss: string. Of the actual <table> element
- selected: T. The selected element
- isSelected: (row: T, selected: T | number) => boolean. A function to identify the selected row.
- cellRenderer: CellRendererSnippet<T>
- filterText: string
- getFilterContent: (row: T) => string. It constructs the string to be used to make the comparison with the filterText. For example: {e => e.Name +" "+ e.Email}

The props of the ITableColumn object are:
- header: string. The name to render in the header of the table
- getValue?: (e: T, idx: number) => string. A callback to get the value to render in the cell.
- render: (e: T, idx: number) => string | ElementAST | ElementAST[]. A render function that can return a HTML string using the {@html ...} snippet. It also can return a AST of the element to render, for complex renderings.
- headerCss: string
- header: string. The th content
- cellCss: string
- subcols: ITableColumn<T>[]. It divide the header into sub-headers, and each is a column.
- onCellEdit?: (e:T, value: string|number) => void. It renders a edit buttom.
- onCellSelect?: (e:T, value: string|number) => void. It renders a delete buttom.
- id: string. Necesary is using the cellRenderer snippet.

The ElementAST interface:
  - id?: number | string
  - css?: string
  - tagName?: "DIV" | "SPAN" | "BUTTON",
  - text?: string | number
  - onClick?: (id: number | string) => void
  - children?: ElementAST[]

For more control, the VTable can include a cellRenderer Svente 5 snippet. The arguments of the snippet are: 
  - record: T
  - col: ITableColumn<T>
  - cellContent?: string 
  - index?: number

Example:
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

### Services and cache

The fetch requests than can use cache, the ones that don't use dynamic filters and need to be ready for the view to show information, use the GetHandler class to construct the service controller.

The arguments in he constructor() are optional. The way it works it makes two fetches. The first one is always to the cache through the Service Worker. If there is no cache it does not return anything so the handler() function does not execute. The second fetch is done to the server to fetch the updated records. There records are combined with the cache records in the Service Worker and it returns to the handler() function all the records up to date. It can return an array of object or an object where each key is an array. 

The handler() function is where the raw response is parsed, prepared and setted in the reactive properties of the class declared using the $state() rune. This enables the reactivity on the view.

The main properties are:
 - route: string. The route of the backend service.
 - useCache: { min: number, ver: number }. The "min" property are the minutes of the caché. If the cache has not expired it will not fetch the server. The "ver" property is the version, if ti changes the cache is invalidated.

The constructor needs to execute the this.fetch() function.

Example: productos.svelte.ts
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

Example: +page.svelte
```svelte
<script>
  const warehouseID = 1
  const productos = new ProductosService(warehouseID)
</script>

<VTable data={productos.productos}
  ... 
/>
```
