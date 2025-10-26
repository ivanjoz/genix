# QTable Component

A high-performance virtual table component for Svelte 5 with support for desktop and mobile card views.

## Features

- ✨ Virtual scrolling for handling large datasets efficiently
- 📱 Responsive design with automatic card view on mobile
- 🔍 Built-in text filtering with highlighting
- 🎨 Customizable styling for headers, cells, and cards
- 🎯 Row selection support
- 📊 Multi-level column headers (subcols)
- ⚡ Powered by TanStack Virtual

## Installation

```bash
pnpm install @tanstack/svelte-virtual
```

## Basic Usage

```svelte
<script lang="ts">
  import QTable, { type ITableColumn } from '$lib/components/QTable.svelte';

  interface Person {
    id: number;
    name: string;
    email: string;
    age: number;
  }

  let data: Person[] = [
    { id: 1, name: 'John Doe', email: 'john@example.com', age: 30 },
    { id: 2, name: 'Jane Smith', email: 'jane@example.com', age: 25 },
    // ... more data
  ];

  const columns: ITableColumn<Person>[] = [
    {
      header: 'ID',
      field: 'id',
      getValue: (record) => record.id,
      css: 'w-20'
    },
    {
      header: 'Name',
      field: 'name',
      getValue: (record) => record.name,
      cardColumn: [0, 1] // Show in mobile card view, row 0, position 1
    },
    {
      header: 'Email',
      field: 'email',
      getValue: (record) => record.email,
      cardColumn: [1, 1]
    },
    {
      header: 'Age',
      field: 'age',
      getValue: (record) => record.age,
      cardColumn: [1, 2]
    }
  ];

  let selectedRow: Person | undefined;
</script>

<QTable
  {columns}
  {data}
  selected={selectedRow}
  isSelected={(record, selected) => record.id === selected?.id}
  onRowClick={(record) => { selectedRow = record; }}
  deviceType={1}
/>
```

## Props

### Required Props

- **`columns`**: `ITableColumn<T>[]` - Array of column definitions
- **`data`**: `T[]` - Array of data records

### Optional Props

- **`maxHeight`**: `string` - Maximum height of the table container (default: `'calc(100vh - 8rem - 12px)'`)
- **`css`**: `string` - Additional CSS classes for the container
- **`style`**: `CSSProperties` - Inline styles for the container
- **`tableStyle`**: `CSSProperties` - Inline styles for the table element
- **`tableCss`**: `string` - Additional CSS classes for the table
- **`styleMobile`**: `CSSProperties` - Additional styles for mobile card view
- **`selected`**: `number | string | T` - Currently selected row identifier
- **`isSelected`**: `(record: T, selected: number | string | T) => boolean` - Function to determine if a row is selected
- **`onRowClick`**: `(record: T) => void` - Callback when a row is clicked
- **`filterText`**: `string` - Text to filter records by
- **`makeFilter`**: `(record: T) => string` - Custom filter function
- **`filterKeys`**: `string[]` - Array of field names to filter on (auto-generates filter function)
- **`deviceType`**: `number` - Device type (1: desktop, 2: tablet, 3: mobile)

## Column Definition (`ITableColumn<T>`)

### Properties

- **`id`**: `number` (optional) - Unique identifier for the column
- **`header`**: `string | (() => string)` - Column header text or function
- **`headerCss`**: `string` (optional) - CSS classes for the header cell
- **`headerStyle`**: `CSSProperties` (optional) - Inline styles for the header cell
- **`cellStyle`**: `CSSProperties` (optional) - Inline styles for data cells
- **`css`**: `string` (optional) - CSS classes for data cells
- **`cardCss`**: `string` (optional) - CSS classes for mobile card view
- **`field`**: `string` (optional) - Field name for filtering
- **`subcols`**: `ITableColumn<T>[]` (optional) - Sub-columns for multi-level headers
- **`cardColumn`**: `[number, (1|2|3)?]` (optional) - Position in mobile card view `[row, position]`
- **`cardRender`**: `(record: T, idx: number, rerender: () => void) => any` (optional) - Custom render for card view
- **`getValue`**: `(record: T, idx: number) => string | number` (optional) - Extract display value
- **`render`**: `(record: T, idx: number, rerender: () => void) => any` (optional) - Custom render function

## Advanced Examples

### Multi-level Headers

```svelte
<script lang="ts">
  const columns: ITableColumn<Person>[] = [
    {
      header: 'Personal Info',
      headerStyle: { 'text-align': 'center' },
      subcols: [
        {
          header: 'Name',
          field: 'name',
          getValue: (r) => r.name
        },
        {
          header: 'Age',
          field: 'age',
          getValue: (r) => r.age
        }
      ]
    },
    {
      header: 'Contact',
      headerStyle: { 'text-align': 'center' },
      subcols: [
        {
          header: 'Email',
          field: 'email',
          getValue: (r) => r.email
        }
      ]
    }
  ];
</script>
```

### Custom Cell Rendering

```svelte
<script lang="ts">
  const columns: ITableColumn<Person>[] = [
    {
      header: 'Name',
      render: (record, idx, rerender) => {
        return `<strong>${record.name}</strong>`;
      }
    },
    {
      header: 'Status',
      render: (record) => {
        const color = record.age > 30 ? 'green' : 'blue';
        return `<span style="color: ${color};">${record.age > 30 ? 'Senior' : 'Junior'}</span>`;
      }
    }
  ];
</script>
```

### Filtering

```svelte
<script lang="ts">
  let filterText = $state('');
  
  // Option 1: Use filterKeys for automatic filtering
  const filterKeys = ['name', 'email'];
  
  // Option 2: Use custom makeFilter function
  const makeFilter = (record: Person) => {
    return `${record.name} ${record.email} ${record.age}`.toLowerCase();
  };
</script>

<input type="text" bind:value={filterText} placeholder="Search..." />

<QTable
  {columns}
  {data}
  {filterText}
  {filterKeys}
/>
```

### Mobile Card View

To enable card view on mobile, add `cardColumn` to your column definitions:

```svelte
<script lang="ts">
  const columns: ITableColumn<Person>[] = [
    {
      header: 'Name',
      field: 'name',
      getValue: (r) => r.name,
      cardColumn: [0, 1], // Row 0, Position 1 in card
      cardCss: 'font-bold text-lg'
    },
    {
      header: 'Email',
      field: 'email',
      getValue: (r) => r.email,
      cardColumn: [1, 1], // Row 1, Position 1
      cardCss: 'text-sm text-gray-600'
    },
    {
      header: 'Age',
      field: 'age',
      getValue: (r) => r.age,
      cardColumn: [1, 2], // Row 1, Position 2
      cardCss: 'text-sm'
    }
  ];
</script>

<QTable {columns} {data} deviceType={3} />
```

## Styling

The component includes default styles but can be customized using:

1. **CSS Classes**: Add custom classes via `css` and `tableCss` props
2. **Inline Styles**: Use `style`, `tableStyle`, and `styleMobile` props
3. **Column Styles**: Set `headerStyle`, `cellStyle`, `headerCss`, `css` on columns
4. **Global CSS**: Override the component's CSS classes

### Available CSS Classes

- `.qtable-c` - Main container for desktop table view
- `.qtable-cards` - Main container for mobile card view
- `.qtable` - The table element
- `.card-ct` - Card container in mobile view
- `.selected` - Applied to selected rows/cards
- `.tr-even` / `.tr-odd` - Alternate row styles
- `._highlight` - Highlighted filter matches

## Performance

The component uses TanStack Virtual for efficient rendering:

- Only visible rows are rendered
- Smooth scrolling for large datasets
- Configurable overscan (default: 5 rows)
- Estimated row height: 34px (desktop), 50px (mobile)

## Browser Support

- Modern browsers with ES6+ support
- Svelte 5+ required
- Works with SvelteKit

## License

Part of the Genix project.

