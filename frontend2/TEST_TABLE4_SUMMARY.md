# Test Table 4 - Svelte Tiny Virtual List Implementation

## Overview

**test-table4** is a virtual table component using the `svelte-tiny-virtual-list` library. This component demonstrates efficient rendering of large datasets using virtualization.

## Key Features

- **Virtual Rendering**: Only renders visible items using `svelte-tiny-virtual-list`
- **Fixed Header**: Sticky header that remains visible while scrolling
- **Grid Layout**: CSS Grid-based table layout for better performance
- **Interactive Rows**: Rows respond to mouse hover with visual feedback
- **Data Mutation**: Click or hover on rows to trigger updates
- **Responsive Design**: Adapts to mobile devices
- **Performance**: Can handle 500+ rows smoothly with virtualization

## Library Information

- **Package**: `svelte-tiny-virtual-list` v3.0.1
- **Repository**: [https://github.com/jonasgeiler/svelte-tiny-virtual-list](https://github.com/jonasgeiler/svelte-tiny-virtual-list)
- **Dependencies**: Zero dependencies (tiny & efficient)
- **License**: MIT

## Component Structure

### Data Structure
```typescript
interface TestRecord {
  id: string;           // 12-character random ID
  edad: number;         // Age (0-99)
  nombre: string;       // Name (18 characters)
  apellidos: string;    // Last name (23 characters)
  numero: number;       // Random number (0-999)
  _updated?: boolean;   // Update indicator flag
}
```

### Columns
1. **ID** - 15% width (displays update indicator if modified)
2. **Edad** - 10% width (age, centered)
3. **Nombre** - 18% width (name)
4. **Apellidos** - 20% width (last name)
5. **Número** - 12% width (number, centered)
6. **Actions** - 12% width (action button, centered)

## Layout & Styling

### CSS Grid
The table uses CSS Grid with the following column distribution:
```css
grid-template-columns: 15% 10% 18% 20% 12% 12%;
```

### Visual Design
- **Header**: Gradient background (purple to violet)
- **Row Height**: 36px fixed
- **Alternating Rows**: Subtle background color alternation for readability
- **Hover Effect**: Light blue background on row hover
- **Responsive**: Mobile layout hides the "Número" column on screens < 768px

## Performance Characteristics

### Initial Load
- **Rows**: 500 items
- **Load Time**: ~500ms simulated delay
- **Overscan Count**: 10 (buffer items for smooth scrolling)

### Virtualization Benefits
- Only renders ~17-20 rows at a time (depending on viewport)
- Smooth scrolling with minimal reflow/repaint
- Handles large datasets without performance degradation

## User Interactions

### Row Hover
- Triggers `handleRowClick(record)` function
- Appends "_1" to the record's "nombre" field
- Toggles the `_updated` flag to show indicator

### Action Button Click
- Same handler as row hover
- Visual feedback through color change on hover

## Responsive Behavior

### Desktop (≥768px)
- Full table with all 6 columns visible
- 600px table height

### Mobile (<768px)
- "Número" column hidden
- Columns adjust to fill space:
  - ID: 18%
  - Nombre: 22%
  - Apellidos: 25%
  - Actions: 15%
- Table height reduces to 400px

## Comparison with Other Tables

| Aspect | test-table2 | test-table3 | test-table4 |
|--------|-----------|-----------|-----------|
| Library | QTable2 | svelte-infinitable | svelte-tiny-virtual-list |
| Virtualization | Yes | Yes | Yes |
| Infinite Scroll | No | Yes | No |
| Rows | 50,000 | 100+ (dynamic) | 500 |
| Fixed Header | Yes | Yes | Yes |
| Dependencies | Multiple | Multiple | Zero |
| Grid/Table | Hybrid | Table | Grid |

## Installation & Setup

### 1. Add Package
```bash
pnpm install svelte-tiny-virtual-list
```

### 2. Import Component
```svelte
import VirtualList from 'svelte-tiny-virtual-list';
```

### 3. Basic Usage
```svelte
<VirtualList
  width="100%"
  height={600}
  itemCount={items.length}
  itemSize={36}
  overscanCount={10}
>
  <div slot="item" let:index let:style {style}>
    <!-- Your content here -->
  </div>
</VirtualList>
```

## Key Props Used

- **width**: "100%" (full container width)
- **height**: 600 (viewport height in pixels)
- **itemCount**: items.length (total items to virtualize)
- **itemSize**: 36 (fixed height per row)
- **overscanCount**: 10 (buffer items for smooth scrolling)

## Future Enhancements

1. Infinite scroll integration with `svelte-infinite-loading`
2. Variable height rows
3. Sticky first/last items
4. Custom item rendering
5. Scroll-to-index functionality
6. Sorting/filtering capabilities
7. Column resizing
8. Export functionality

## File Location

`/src/routes/develop-ui/test-table4/+page.svelte`

## Access

Navigate to: `/develop-ui/test-table4`
