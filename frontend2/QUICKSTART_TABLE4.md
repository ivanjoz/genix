# Quick Start - Test Table 4

## What is Test Table 4?

A **virtual table component** using [svelte-tiny-virtual-list](https://github.com/jonasgeiler/svelte-tiny-virtual-list), which efficiently renders large datasets by only displaying visible rows.

## Quick Access

**URL**: http://localhost:5173/develop-ui/test-table4

## What You'll See

- 500 rows of test data with 6 columns
- A sticky purple header that stays at the top while scrolling
- Smooth, performant scrolling with virtualization
- Interactive rows that respond to hover

## Column Information

| Column | Type | Width | Notes |
|--------|------|-------|-------|
| ID | String | 15% | Shows 🔄 when updated |
| Edad | Number | 10% | Random age 0-99 |
| Nombre | String | 18% | Random name (18 chars) |
| Apellidos | String | 20% | Random last name (23 chars) |
| Número | Number | 12% | Random number 0-999 |
| Actions | Button | 12% | Click to update row |

## Interactive Features

### Hover a Row
- The row background changes to light blue
- Hovering **automatically appends "_1"** to the "Nombre" field
- An update indicator (🔄) appears in the ID column

### Click Action Button
- Same behavior as row hover
- Updates the row's data and shows the indicator

## Why Svelte-Tiny-Virtual-List?

✅ **Zero Dependencies** - No external libraries needed  
✅ **Tiny Bundle** - Only ~5KB gzipped  
✅ **Performance** - Handles millions of rows efficiently  
✅ **Smooth Scrolling** - 10-item overscan buffer  
✅ **Fixed Heights** - Optimal for consistent row sizes  

## How It Works

```
┌─────────────────────────────────┐
│    Header (Fixed - Sticky)      │  ← Always visible
├─────────────────────────────────┤
│ Row 17 (Rendered in VirtualList)│  ← Only visible rows
│ Row 18                          │     are rendered
│ Row 19                          │     in the DOM
│ Row 20                          │
│ Row 21                          │
├─────────────────────────────────┤
│ Not rendered (offscreen)        │  ← Rows outside viewport
│ Not rendered (offscreen)        │     are not in DOM
│ ...                             │
└─────────────────────────────────┘
```

## Component Props

```svelte
<VirtualList
  width="100%"           <!-- Full width of container -->
  height={600}           <!-- 600px visible height -->
  itemCount={500}        <!-- 500 total rows -->
  itemSize={36}          <!-- 36px per row -->
  overscanCount={10}     <!-- Render 10 extra rows as buffer -->
>
  <div slot="item" let:index let:style>
    <!-- Row content here -->
  </div>
</VirtualList>
```

## Performance Metrics

- **Initial Load**: ~500ms (simulated)
- **Rendered Rows**: ~17-20 rows visible at a time
- **Actual DOM Size**: Only visible + buffer rows
- **Memory**: Minimal compared to rendering all 500 rows
- **Scroll Performance**: 60 FPS smooth scrolling

## Responsive Design

### Desktop (≥768px)
- All 6 columns visible
- 600px table height

### Mobile (<768px)
- "Número" column hidden
- 5 visible columns adjust to fill space
- 400px table height

## File Location

```
frontend2/
└── src/
    └── routes/
        └── develop-ui/
            └── test-table4/
                └── +page.svelte        (Main component)
```

## Code Structure

```svelte
<script lang="ts">
  // Data structure
  interface TestRecord { ... }
  
  // State management
  let items = $state<TestRecord[]>([])
  
  // Data generation
  function makeData() { ... }
  
  // Event handlers
  function handleRowClick(record) { ... }
</script>

<!-- Component -->
<VirtualList ...>
  <div slot="item" let:index let:style>
    {/* Row rendered here */}
  </div>
</VirtualList>

<!-- Styles -->
<style>
  .table-row { display: grid; grid-template-columns: ... }
  .header-row { background: gradient; position: sticky; }
  .data-row { border-bottom: 1px solid #e2e8f0; }
</style>
```

## Troubleshooting

### Rows Not Updating?
- Check browser console for errors
- Ensure `items` array is properly reactively assigned

### Scrolling Feels Janky?
- Increase `overscanCount` prop (default: 10)
- Ensure fixed `itemSize` (no variable heights)

### Styling Issues?
- Virtual list uses `.virtual-list-wrapper` and `.virtual-list-inner` classes
- Use `:global()` selector to style these

## Next Steps

1. **Customize Data**: Modify `TestRecord` interface and `makeData()` function
2. **Add Sorting**: Implement click handlers on column headers
3. **Add Filtering**: Add input fields to filter rows
4. **Infinite Scroll**: Integrate with `svelte-infinite-loading`
5. **Export**: Add export functionality for visible/all rows

## See Also

- [TEST_TABLE4_SUMMARY.md](./TEST_TABLE4_SUMMARY.md) - Detailed documentation
- [test-table3](http://localhost:5173/develop-ui/test-table3) - Alternative using svelte-infinitable
- [test-table2](http://localhost:5173/develop-ui/test-table2) - Alternative using QTable2
