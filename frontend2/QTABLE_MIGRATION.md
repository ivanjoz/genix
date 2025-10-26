# QTable Migration Guide: SolidJS to Svelte 5

This document explains the migration of the QTable component from SolidJS (TanStack Solid) to Svelte 5 (TanStack Svelte).

## Overview

The QTable component has been successfully migrated from SolidJS to Svelte 5, maintaining all core features while adapting to Svelte's reactive paradigm using the new runes API.

## Key Changes

### 1. Reactive Primitives

#### SolidJS Approach
```typescript
// SolidJS uses signals and effects
const [records, setRecords] = createSignal(props.data);
const [recordRows, setRecordRows] = createSignal(Virtualizer.getVirtualItems());

createEffect(on(() => [props.data, props.filterText||""], () => {
  const recordsFiltered = filter(props, props.filterText);
  setRecords(recordsFiltered);
}));
```

#### Svelte 5 Approach
```typescript
// Svelte 5 uses runes ($state, $derived, $effect)
let records = $state<T[]>([]);

const filteredRecords = $derived.by(() => {
  // Filtering logic
  return filteredData;
});

$effect(() => {
  records = filteredRecords;
});
```

**Key Differences:**
- `createSignal()` → `$state()`
- `createMemo()` → `$derived` or `$derived.by()`
- `createEffect()` → `$effect()`
- No explicit getters/setters in Svelte, just direct access

### 2. Component Props

#### SolidJS Approach
```typescript
export function QTable<T>(props: IQTable<T>) {
  // Access via props.data, props.columns, etc.
}
```

#### Svelte 5 Approach
```typescript
let {
  columns = $bindable(),
  data = $bindable(),
  maxHeight = 'calc(100vh - 8rem - 12px)',
  // ... other props with defaults
}: IQTableProps<T> = $props();
```

**Key Differences:**
- Destructured props with `$props()`
- Built-in default values
- `$bindable()` for two-way binding (if needed)
- Type-safe with generics

### 3. Virtual Scrolling API

#### SolidJS Approach
```typescript
import { createVirtualizer } from "@tanstack/solid-virtual"

const Virtualizer = createVirtualizer({
  count: props.data.length || 1,
  estimateSize: () => 34,
  getScrollElement: () => containerRef,
});

const recordRows = Virtualizer.getVirtualItems();
```

#### Svelte 5 Approach
```typescript
import { createVirtualizer } from '@tanstack/svelte-virtual';

let virtualizerStore = $state<ReturnType<typeof createVirtualizer>>();
let virtualizer = $state<any>();

$effect(() => {
  if (containerRef) {
    virtualizerStore = createVirtualizer({
      count: records.length || 1,
      getScrollElement: () => containerRef!,
      estimateSize: () => 34,
      overscan: 5
    });

    const unsubscribe = virtualizerStore.subscribe((value) => {
      virtualizer = value;
    });

    return () => unsubscribe();
  }
});

const virtualItems = $derived(virtualizer?.getVirtualItems() || []);
```

**Key Differences:**
- Svelte version returns a `Readable` store
- Need to subscribe to the store to get virtualizer instance
- Cleanup handled with effect return function

### 4. JSX to Svelte Template Syntax

#### SolidJS JSX
```tsx
<Show when={!isCardView()}>
  <table class="qtable">
    <For each={recordRows()}>
      {(row, i) => {
        const record = records()[row.index];
        return <tr>{/* ... */}</tr>
      }}
    </For>
  </table>
</Show>
```

#### Svelte 5 Template
```svelte
{#if !isCardView}
  <table class="qtable">
    {#each virtualItems as row, i (row.index)}
      {@const record = records[row.index]}
      <tr>{/* ... */}</tr>
    {/each}
  </table>
{/if}
```

**Key Differences:**
- `<Show when={}>` → `{#if}`
- `<For each={}>` → `{#each}`
- `{@const}` for inline constants
- Key tracking with `(key)` syntax in `#each`

### 5. Event Handlers

#### SolidJS
```tsx
<tr onClick={ev => {
  ev.stopPropagation();
  if(props.onRowCLick){ props.onRowCLick(record) }
}}>
```

#### Svelte 5
```svelte
<tr onclick={(ev) => {
  ev.stopPropagation();
  onRowClick?.(record);
}}>
```

**Key Differences:**
- Event handlers use lowercase (`onclick` vs `onClick`)
- Optional chaining works the same way
- No need for conditional checks, just use `?.`

### 6. Styling

Both frameworks support similar styling approaches:

#### Inline Styles
```typescript
// Helper function to convert style objects to strings
function styleToString(styleObj?: CSSProperties): string {
  if (!styleObj) return '';
  return Object.entries(styleObj)
    .map(([key, value]) => `${key}: ${value}`)
    .join('; ');
}
```

#### CSS Classes
```svelte
<style>
  .qtable {
    width: 100%;
    border-collapse: collapse;
  }
  
  /* Component-scoped styles */
</style>
```

### 7. Refs / Element Binding

#### SolidJS
```typescript
let containerRef: HTMLDivElement = undefined as unknown as HTMLDivElement;

<div ref={containerRef}>
```

#### Svelte 5
```typescript
let containerRef = $state<HTMLDivElement>();

<div bind:this={containerRef}>
```

**Key Differences:**
- Svelte uses `bind:this` directive
- Type-safe with optional chaining

## Features Maintained

All original features have been maintained:

1. ✅ Virtual scrolling with TanStack Virtual
2. ✅ Desktop table view and mobile card view
3. ✅ Text filtering with highlighting
4. ✅ Row selection
5. ✅ Multi-level headers (subcols)
6. ✅ Custom cell rendering
7. ✅ Custom card rendering for mobile
8. ✅ Customizable styling
9. ✅ Type safety with TypeScript generics

## Performance Comparison

Both implementations offer similar performance:

- **Virtual Scrolling**: Both use TanStack Virtual (same core library)
- **Reactivity**: Svelte's compiler-based reactivity is generally more efficient
- **Bundle Size**: Svelte typically produces smaller bundles
- **Runtime**: Svelte has less runtime overhead

## Migration Checklist

When migrating other components from SolidJS to Svelte 5:

- [ ] Replace reactive primitives (signals → runes)
- [ ] Convert JSX to Svelte template syntax
- [ ] Update component props pattern
- [ ] Handle store subscriptions properly (if using stores)
- [ ] Convert event handlers to lowercase
- [ ] Update refs to use `bind:this`
- [ ] Convert inline styles helper functions
- [ ] Test all features thoroughly
- [ ] Check for accessibility warnings
- [ ] Verify TypeScript types

## Benefits of Svelte 5 Version

1. **Simpler Syntax**: Less boilerplate with runes
2. **Better DX**: More intuitive template syntax
3. **Type Safety**: Better TypeScript integration
4. **Performance**: Compiler-based reactivity
5. **Smaller Bundle**: Less runtime code
6. **Accessibility**: Built-in a11y warnings

## Usage Example

See `example-table.svelte` for a complete working example with:
- 1000 rows of sample data
- Filtering functionality
- Row selection
- Device type switching
- Custom cell rendering

## Next Steps

1. Test the component with your actual data
2. Customize styling to match your design system
3. Add any additional features you need
4. Consider creating specialized variants for specific use cases

## References

- [Svelte 5 Runes Documentation](https://svelte.dev/docs/svelte/runes)
- [TanStack Virtual Documentation](https://tanstack.com/virtual/latest)
- [Svelte 5 Migration Guide](https://svelte.dev/docs/svelte/v5-migration-guide)

