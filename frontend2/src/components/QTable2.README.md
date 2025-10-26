# QTable2 Component

A high-performance virtual table component for Svelte 5 using the **[virtua](https://github.com/inokawa/virtua)** library.

## Overview

QTable2 is an alternative to QTable that uses the `virtua/svelte` library instead of TanStack Virtual. Virtua is optimized for performance and provides excellent virtual scrolling capabilities.

## Library Used

- **virtua** - [GitHub Repository](https://github.com/inokawa/virtua/tree/main/src/svelte)
- **Version**: 0.46.3
- **License**: MIT
- Supports: React, Vue, Svelte, and Solid

## Key Features

- ✨ Virtual scrolling with virtua library
- 📱 Responsive design with automatic card view on mobile
- 🔍 Built-in text filtering with highlighting
- 🎨 Customizable styling for headers, cells, and cards
- 🎯 Row selection support
- 📊 Multi-level column headers (subcols)
- ⚡ Extremely fast performance

## Installation

```bash
pnpm install virtua
```

## Basic Usage

```svelte
<script lang="ts">
  import QTable2, { type ITableColumn } from '$lib/components/QTable2.svelte';

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
      cardColumn: [0, 1] // Show in mobile card view
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
      getValue: (record) => record.age
    }
  ];
</script>

<QTable2 {columns} {data} />
```

## API

### Props

All props are identical to QTable:

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `columns` | `ITableColumn<T>[]` | Yes | Column definitions |
| `data` | `T[]` | Yes | Array of data records |
| `maxHeight` | `string` | No | Maximum height (default: `'calc(100vh - 8rem - 12px)'`) |
| `filterText` | `string` | No | Text to filter by |
| `selected` | `T \| number \| string` | No | Selected row |
| `onRowClick` | `(record: T) => void` | No | Click handler |
| `deviceType` | `number` | No | 1=desktop, 3=mobile |
| ...and all other QTable props |

### Column Definition

Same as QTable - see `QTable.README.md` for full documentation.

## Differences from QTable

| Feature | QTable (TanStack) | QTable2 (virtua) |
|---------|-------------------|------------------|
| **Library** | @tanstack/svelte-virtual | virtua |
| **Bundle Size** | ~20KB | ~15KB (smaller!) |
| **Performance** | Excellent | Excellent |
| **API** | Store-based | Component-based |
| **Complexity** | Moderate | Simpler |
| **Maintenance** | Active | Very Active |

## Performance Comparison

Both libraries provide excellent virtual scrolling, but virtua has some advantages:

- **Smaller bundle size**: ~25% smaller than TanStack
- **Simpler API**: Component-based API is more intuitive
- **Better SSR support**: Works seamlessly with SvelteKit
- **Active development**: Regular updates and improvements

## Example Page

Test the component at `/develop-ui/test-table2`:

```bash
cd frontend2
pnpm dev
# Navigate to /develop-ui/test-table2
```

## Why Use QTable2 over QTable?

### Choose QTable2 (virtua) if:
- ✅ You want a smaller bundle size
- ✅ You prefer simpler APIs
- ✅ You need better SSR/SvelteKit integration
- ✅ You're starting a new project

### Choose QTable (TanStack) if:
- ✅ You're already using TanStack libraries
- ✅ You need consistency with React/Vue versions
- ✅ You're migrating from existing TanStack code

## Technical Details

### Virtual Scrolling

QTable2 uses virtua's `VList` component:

```svelte
<VList data={records} itemSize={34}>
  {#snippet children(item, index)}
    <!-- Row content -->
  {/snippet}
</VList>
```

### Key Features of virtua

1. **Automatic item sizing**: Can handle dynamic heights
2. **Horizontal scrolling**: Supports horizontal lists
3. **Window virtualization**: Can virtualize entire page
4. **SSR friendly**: Works with server-side rendering
5. **TypeScript first**: Excellent type support

## Migration from QTable

QTable2 has the **exact same API** as QTable, so migration is simple:

```svelte
<!-- Before -->
<script>
  import QTable from '$lib/components/QTable.svelte';
</script>
<QTable {columns} {data} />

<!-- After -->
<script>
  import QTable2 from '$lib/components/QTable2.svelte';
</script>
<QTable2 {columns} {data} />
```

That's it! Everything else works the same.

## Browser Support

- Modern browsers (ES6+)
- Svelte 5+ required
- Works with SvelteKit

## Credits

- **virtua library**: [https://github.com/inokawa/virtua](https://github.com/inokawa/virtua)
- **Author**: inokawa
- **Svelte support**: [https://github.com/inokawa/virtua/tree/main/src/svelte](https://github.com/inokawa/virtua/tree/main/src/svelte)

## License

Part of the Genix project.

