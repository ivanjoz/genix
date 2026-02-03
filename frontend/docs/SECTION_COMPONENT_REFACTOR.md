# Plan: Native Svelte Section System & Flat Schema Refactor

## Objective
Transition from the current **AST-driven rendering** (deeply nested JSON) to a **Component-driven architecture** (native `.svelte` files) with a **Flat Data Contract**. This simplifies development, improves performance via Svelte 5 runes, and provides a predictable interface for AI agents.

---

## 1. Core Architecture

### 1.1 The Flat Data Contract
Every section will be defined by a single object following this structure:
```typescript
interface SectionData {
  id: string;      // Unique instance ID
  type: string;    // Component name (e.g., 'Hero', 'ProductGrid')
  content: {       // Flat key-value pairs for text and media
    title?: string;
    subTitle?: string;
    image?: string;
    primaryActionLabel?: string;
    primaryActionHref?: string;
    [key: string]: any;
  };
  css: {           // Tailwind classes mapped to specific "slots"
    container?: string;
    heading?: string;
    content?: string;
    [key: string]: string;
  };
}
```

### 1.2 Global Editor Store (`editor.svelte.ts`)
A centralized Svelte 5 state store to manage the builder's lifecycle.
- **`selectedId`**: Tracks which section is currently active in the sidebar.
- **`activeSchema`**: The metadata (fields/slots) of the selected section.
- **`pageData`**: The array of `SectionData` representing the full page.

### 1.3 Metaprogramming Registry
A script will automatically scan the `pkg-store/sections/` directory to generate a `registry.ts`. This ensures SSR compatibility and removes the need for manual registration.

---

## 2. Step-by-Step Implementation

### Step 1: Define Core Types & Shared Store
- Create `pkg-store/renderer/section-types.ts` to house the interfaces.
- Implement `pkg-store/stores/editor.svelte.ts` using `$state`.

### Step 2: Create a Prototype Section
- Create `pkg-store/sections/Hero.svelte`.
- Define the schema in `<script module>`.
- Use native Svelte props for rendering.
```svelte
<script module>
  export const schema = {
    content: ['title', 'subTitle', 'image'],
    css: ['container', 'title', 'subTitle']
  };
</script>
<script>
  let { content, css } = $props();
</script>
<section class={css.container}>...</section>
```

### Step 3: Implement the Registry Generator
- Create `scripts/generate-sections.ts`.
- The script should output a file that imports all sections and exports a `SectionRegistry` object.

### Step 4: Refactor `EcommerceRenderer.svelte`
- Update the renderer to support a hybrid mode.
- If `element.type` exists in `SectionRegistry`, render the native component.
- Otherwise, fallback to the legacy AST renderer (for backward compatibility).

### Step 5: Refactor `BuilderSectionRender.svelte`
- Remove the complex AST field extraction logic.
- Simplify the `onclick` handler to just set the `selectedId` in the `editorStore`.
- The wrapper will now be a thin shell around the native Svelte component.

### Step 6: Update `SectionEditorLayer.svelte`
- Refactor the "Editor Tab" to iterate over the `activeSchema` from the store.
- Create generic input components for the flat `content` and `css` objects.

---

## 3. Benefits

### For Developers
- **Full Svelte Power**: Use transitions, effects, and complex logic inside sections.
- **Type Safety**: Proper TypeScript support for component props.
- **Maintenance**: No more updating a massive `switch` statement in the renderer.

### For AI Agents
- **Predictable API**: The AI always knows that `content.title` is where the text goes.
- **Reduced Token Usage**: No need to pass entire component source code; just the flat schema.
- **Reliability**: The AI cannot "break" the HTML structure; it can only modify the data passed to the component.

### For Performance
- **SSR**: Native components render instantly on the server.
- **Granular Reactivity**: Svelte 5 ensures that changing one CSS class only updates that specific DOM element.
