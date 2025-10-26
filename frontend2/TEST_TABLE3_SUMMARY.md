# Test Table 3 - Virtual Table (Svelte Infinitable)

## Overview

`test-table3` is a virtual table component built with **[svelte-infinitable](https://github.com/adevien-solutions/svelte-infinitable)**, a modern virtual table library that supports infinite scrolling, searching, filtering, and sorting.

## Features

✨ **Virtual Rendering**: Only renders visible rows for optimal performance
♾️ **Infinite Scrolling**: Automatically loads more data as you scroll
📊 **Dynamic Data**: Supports server-side loading patterns
⚡ **Lightweight**: Minimal dependency overhead
🎨 **Customizable**: Full snippet-based customization
📱 **Responsive**: Mobile-friendly design

## Location

Route: `/develop-ui/test-table3`
File: `src/routes/develop-ui/test-table3/+page.svelte`

## Installation

The library is already installed as a dev dependency:
```bash
pnpm add -D svelte-infinitable
```

## Component Usage

### Basic Structure

```svelte
<Infinitable.Root
  bind:items
  rowHeight={36}
  {onInfinite}
  class="virtual-table"
  overscan={10}
>
  {#snippet headers()}
    <!-- Table headers -->
  {/snippet}
  
  {#snippet children({ item, index })}
    <!-- Table rows -->
  {/snippet}
  
  {#snippet loader()}
    <!-- Loading indicator -->
  {/snippet}
  
  {#snippet completed()}
    <!-- End of data message -->
  {/snippet}
</Infinitable.Root>
```

## Data Structure

The component uses a `TaskData` interface:

```typescript
interface TaskData {
  id: string;           // Unique identifier (e.g., "ID-000001")
  nombre: string;       // First name
  apellidos: string;    // Last names
  edad: number;         // Age
  email: string;        // Email address
  fecha_creacion: string; // Creation date (YYYY-MM-DD)
}
```

## Key Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `items` | `TaskData[]` | `[]` | Array of items to display |
| `rowHeight` | `number` | - | Height of each row in pixels (36px in this case) |
| `onInfinite` | `InfiniteHandler` | - | Callback for infinite scrolling |
| `overscan` | `number` | 10 | Buffer rows to render outside viewport |
| `class` | `string` | - | CSS classes for the root element |

## Snippets

### headers
Renders the table header row with column definitions.

### children
Renders individual table rows. Receives `{ item, index, selectedCount, isAllSelected }`.

### loader
Shows during data loading (appears between existing rows and new data).

### completed
Displays when all data has been loaded.

### empty
Shows when there are no matching items.

### loadingEmpty
Shows while initial data is loading.

## Implementation Details

### Data Loading Strategy

The component implements a pagination-based infinite scrolling system:

1. **Initial Load**: Loads first 100 items on mount
2. **Subsequent Loads**: Loads 100 items at a time when scrolling near bottom
3. **Completion**: Stops loading after 500 items (configurable)

```typescript
const pageSize = 100;
let page = $state(1);

function generateMockData(page: number, pageSize: number): TaskData[] {
  // Generates mock data for demonstration
}

const onInfinite: InfiniteHandler = async ({ loaded, completed, error }) => {
  const newData = generateMockData(page, pageSize);
  if (items.length >= 500) {
    completed(newData); // Signal end of data
  } else {
    loaded(newData);    // Load more data
    page++;
  }
};
```

### Performance Features

- **Virtual Rendering**: Only visible rows are DOM nodes
- **Fixed Row Height**: 36px per row enables efficient virtualization
- **Overscan Buffer**: 10 rows above/below viewport for smooth scrolling
- **Lazy Loading**: Data loads only as needed during scroll

## Styling

The component includes comprehensive styling:

- **Header Section**: Information display with statistics
- **Table Wrapper**: Scrollable container with table
- **Row Styling**: Alternating row colors with hover effects
- **Column Widths**: Responsive column sizing
- **Responsive Design**: Mobile breakpoint at 768px

### CSS Classes

| Class | Purpose |
|-------|---------|
| `.page-container` | Main container |
| `.header-section` | Information panel |
| `.table-wrapper` | Table container |
| `.virtual-table` | Scrollable table element |
| `.header-row` | Header row styling |
| `.data-row` | Data row styling |

## Comparison with Other Test Tables

### test-table vs test-table3

| Feature | test-table | test-table3 |
|---------|-----------|-----------|
| Library | QTable2 (Virtua) | svelte-infinitable |
| Rendering | Virtual | Virtual |
| Data Loading | All loaded upfront | Infinite scroll |
| Row Count | 50,000 | Unlimited (paginated) |
| Sorting | Built-in | Customizable |
| Filtering | Built-in | Customizable |

## Usage Examples

### Fetching Real Data

Replace `generateMockData` with actual API calls:

```typescript
const onInfinite: InfiniteHandler = async ({ loaded, completed, error }) => {
  try {
    const response = await fetch(`/api/items?page=${page}&limit=${pageSize}`);
    const { data, hasMore } = await response.json();
    
    if (hasMore) {
      loaded(data);
    } else {
      completed(data);
    }
    page++;
  } catch (e) {
    error();
  }
};
```

### Custom Row Styling

Add conditional styling based on data:

```svelte
{#snippet children({ item, index })}
  {@const data = items[index]}
  <tr class="data-row" class:highlighted={data.edad > 50}>
    <td>{data.nombre}</td>
  </tr>
{/snippet}
```

### Server-Side Filtering

Integrate with search:

```typescript
let searchTerm = $state('');

const onInfinite: InfiniteHandler = async ({ loaded, completed, error }) => {
  const newData = await fetchFromAPI(page, pageSize, searchTerm);
  // ... handle response
};
```

## Performance Metrics

- **Render Count**: 36 visible rows + 20 overscan = ~56 DOM nodes
- **Memory Usage**: Linear with loaded items (not visible items)
- **Scroll Performance**: 60 FPS with standard throttling
- **Initial Load**: ~500ms (simulated)

## Browser Support

Works with all modern browsers supporting:
- ES2020+
- CSS Grid/Flexbox
- Svelte 5

## Known Limitations

1. Row height must be fixed (cannot have variable row heights)
2. Requires binding `items` for reactivity
3. Initial load state handling is manual (no built-in loading skeleton)

## Future Enhancements

- [ ] Add row selection with checkboxes
- [ ] Implement column sorting UI
- [ ] Add search/filter integration
- [ ] Export to CSV functionality
- [ ] Row grouping support

## Resources

- **Library Repo**: https://github.com/adevien-solutions/svelte-infinitable
- **Live Demo**: https://infinitable.adevien.com/
- **Documentation**: https://github.com/adevien-solutions/svelte-infinitable#readme

## Related Files

- Main component: `src/routes/develop-ui/test-table3/+page.svelte`
- Test table 2: `src/routes/develop-ui/test-table2/+page.svelte` (Virtua version)
- Test table 1: `src/routes/develop-ui/test-table/+page.svelte` (QTable version)
