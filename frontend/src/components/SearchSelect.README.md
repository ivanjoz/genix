# SearchSelect - Svelte 5 Components

This directory contains the Svelte 5 conversion of the SolidJS SearchSelect components.

## Files Created

1. **SearchSelect.svelte** - Main searchable select component
2. **SearchCard.svelte** - Multi-select card component
3. **SearchMobileLayer.svelte** - Mobile layer overlay for selection
4. **searchSelectStore.svelte.ts** - Shared reactive state for mobile layer
5. **searchSelectUtils.ts** - Utility functions for highlighting search terms

## Key Conversion Changes

### SolidJS → Svelte 5 Runes

| SolidJS | Svelte 5 |
|---------|----------|
| `createSignal()` | `$state()` |
| `createMemo()` | `$derived()` |
| `createEffect()` | `$effect()` |
| `props` (interface) | `$props()` |
| two-way binding | `$bindable()` |
| `<Show when={}>` | `{#if}` |
| `<For each={}>` | `{#each}` |
| `ref={variable}` | `bind:this={variable}` |
| `onKeyUp={handler}` | `on:keyup={handler}` |
| `<Portal>` | Direct DOM rendering with `{#if}` |

### Major Changes

1. **Type Assertions**: Removed all inline type assertions (`as Type`) from templates since Svelte doesn't support them in markup. Used `{@const}` blocks and `String()` conversions instead.

2. **Global State**: Moved global state to a separate `.svelte.ts` file (`searchSelectStore.svelte.ts`) since module context scripts can't use runes.

3. **Event Handlers**: Changed from `onEventName` to `on:eventname` (lowercase).

4. **Reactive Statements**: 
   - `createEffect(on(() => [...], () => {...}))` → `$effect(() => {...})`
   - Effects now automatically track dependencies

5. **Props Binding**: Used `$bindable()` for two-way bound props like `saveOn` and `selected`.

## Usage Examples

### Basic Select
```svelte
<SearchSelect
  {options}
  keys="id.name"
  label="Select an option"
  placeholder="Choose..."
  bind:selected={selectedValue}
  onChange={(item) => console.log('Selected:', item)}
/>
```

### Multi-Select Card
```svelte
<SearchCard
  {options}
  keys="id.name"
  label="Categories"
  bind:saveOn={formData}
  save="categoryIds"
/>
```

### With Custom Filtering
```svelte
<SearchSelect
  {options}
  keys="id.name"
  avoidIDs={[1, 2, 3]}
  clearOnSelect={true}
  onChange={(item) => handleSelection(item)}
/>
```

## Props

### SearchSelect Props

- `saveOn?: any` - Object to save selected value to (with `$bindable`)
- `save?: string | keyof T` - Property name in saveOn to update
- `css?: string` - Additional CSS classes
- `options: T[]` - Array of selectable options (required)
- `keys: string` - Format: "idKey.nameKey" (required)
- `label?: string` - Label text
- `placeholder?: string` - Placeholder text
- `max?: number` - Maximum number of options to display (default: 100)
- `onChange?: (e: T) => void` - Callback when selection changes
- `selected?: number | string` - Currently selected ID (with `$bindable`)
- `notEmpty?: boolean` - Prevent clearing selection
- `required?: boolean` - Show validation icon
- `disabled?: boolean` - Disable input
- `clearOnSelect?: boolean` - Clear input after selection
- `avoidIDs?: number[]` - IDs to exclude from options
- `inputCss?: string` - CSS classes for input element
- `icon?: string` - Custom icon class
- `showLoading?: boolean` - Show loading spinner

### SearchCard Props

Similar to SearchSelect but designed for multi-select with visual cards.

## Utility Functions

### `highlString(phrase: string, words: string[])`

Highlights matching words in a phrase. Returns an array of strings and highlight objects.

```typescript
const result = highlString("Hello World", ["wor"]);
// Returns: ["Hello ", { type: 'em', text: "Wor" }, "ld"]
```

## Notes

- The components use generics (`<T>`) for type safety
- Mobile detection automatically switches to layer picker on mobile devices
- Keyboard navigation supported (Arrow Up/Down)
- Debounced search with 120ms throttle
- Accessibility warnings in linter are expected and can be addressed as needed

## Dependencies

- `~/core/main` - For `throttle` function
- `~/app` - For `isMobile()` and `deviceType()`
- `./Cards` - For `Spinner4` loading component
- `~/env` - For `Env.suscribeUrlFlag()` in mobile layer

