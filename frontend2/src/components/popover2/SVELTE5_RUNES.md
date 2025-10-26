# Svelte 5 Runes Used in Popover2

This document shows all the Svelte 5 runes used in the Popover2 library.

## Overview

The Popover2 library is built using **pure Svelte 5 runes** - no legacy Svelte APIs are used.

## Runes Used

### 1. `$props()` - Component Props
Used to declare and access component props with TypeScript support.

**Example from Popover2.svelte:**
```typescript
let {
  referenceElement,
  open = false,
  placement = 'bottom',
  offset = 8,
  fitViewport = true,
  class: className = '',
  style: customStyle = '',
  children,
  onPositionUpdate
}: Props = $props();
```

### 2. `$state()` - Reactive State
Used to create reactive state variables that trigger re-renders when changed.

**Example from Popover2.svelte:**
```typescript
let floatingElement: HTMLElement | null = $state(null);
let position = $state({ top: 0, left: 0, placement: placement as Placement });
```

**Example from Portal.svelte:**
```typescript
let portalContainer: HTMLElement | null = $state(null);
let mounted = $state(false);
```

### 3. `$derived()` - Computed Values
Used to create derived/computed values that automatically update when dependencies change.

**Example from Popover2.svelte:**
```typescript
const computedStyle = $derived(() => {
  if (!open) return 'display: none;';
  
  return `
    position: absolute;
    top: ${position.top}px;
    left: ${position.left}px;
    z-index: 9999;
    ${customStyle}
  `.trim();
});
```

### 4. `$effect()` - Side Effects
Used to run side effects and cleanup functions. Replaces `onMount`, `onDestroy`, and reactive statements.

**Example from Portal.svelte:**
```typescript
$effect(() => {
  // Create a container div in the body
  const container = document.createElement('div');
  container.style.position = 'absolute';
  container.style.top = '0';
  container.style.left = '0';
  container.style.zIndex = '9999';
  
  const targetElement = target || document.body;
  targetElement.appendChild(container);
  
  portalContainer = container;
  mounted = true;
  
  // Cleanup when effect runs again or component unmounts
  return () => {
    if (container && container.parentNode) {
      container.parentNode.removeChild(container);
    }
    portalContainer = null;
    mounted = false;
  };
});
```

**Example from Popover2.svelte (event listeners):**
```typescript
$effect(() => {
  if (!open) return;
  
  const handleUpdate = () => {
    if (open && referenceElement && floatingElement) {
      updatePosition();
    }
  };
  
  window.addEventListener('scroll', handleUpdate, true);
  window.addEventListener('resize', handleUpdate);
  
  return () => {
    window.removeEventListener('scroll', handleUpdate, true);
    window.removeEventListener('resize', handleUpdate);
  };
});
```

### 5. `{@render}` - Snippet Rendering
Used to render Svelte 5 snippets (replaces slots).

**Example from Portal.svelte:**
```svelte
{#if mounted && portalContainer}
  {@render children?.()}
{/if}
```

**Example from Popover2.svelte:**
```svelte
<div bind:this={floatingElement} class={className} style={computedStyle()}>
  {@render children?.()}
</div>
```

### 6. `bind:this` - Element Binding
Standard Svelte directive (not a rune, but important for the implementation).

**Example from Popover2.svelte:**
```svelte
<div bind:this={floatingElement} ...>
```

**Example from Popover2Example.svelte:**
```svelte
<button bind:this={topButton} onclick={() => showTop = !showTop}>
```

## Key Differences from Legacy Svelte

| Legacy Svelte | Svelte 5 Runes |
|---------------|----------------|
| `export let prop` | `let { prop }: Props = $props()` |
| `let state = value` (reactive by default) | `let state = $state(value)` |
| `$: computed = value` | `const computed = $derived(value)` |
| `$: { /* effect */ }` | `$effect(() => { /* effect */ })` |
| `onMount(() => {...})` | `$effect(() => {...})` |
| `<slot />` | `{@render children?.()}` |

## Benefits of Using Runes

1. **Explicit Reactivity** - Clear what's reactive and what's not
2. **Better TypeScript** - Full type inference for props and state
3. **Cleaner Code** - No magic reactivity, everything is explicit
4. **Better Performance** - More predictable and optimizable
5. **Easier Testing** - Side effects are easier to reason about

## No Legacy APIs Used

The Popover2 library does **NOT** use:
- ❌ `export let` for props
- ❌ `$:` reactive statements
- ❌ `onMount` / `onDestroy`
- ❌ `<slot>` elements
- ❌ Stores for local state

Everything is pure Svelte 5 runes! ✅

