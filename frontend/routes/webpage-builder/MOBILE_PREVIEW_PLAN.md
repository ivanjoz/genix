# Mobile-view toggle for the Ecommerce builder

## Goal
Add a button to switch the builder canvas between desktop and a ~390px mobile
preview. Mobile preview must honor real Tailwind breakpoints (`md/lg/xl/2xl` are
**media queries** at 749/1139/1379/1539px — see `webpage/stores/uno-generator.ts`),
so a plain narrow `<div>` is wrong: media queries evaluate against the browser
viewport, not element width.

## Strategy: same-origin iframe + Svelte `mount()`
Render the sections into an **empty same-origin iframe** whose width is 390px.
Media queries then resolve against the iframe's own viewport → true mobile layout.
Because it's the same JS runtime, the mounted tree **shares the same `editorStore`
instance**, so click-to-select works by calling `editorStore.select(id)` directly
— no `postMessage`, and `SectionEditorLayer` (in the parent panel) keeps reacting
to `editorStore.selectedId` unchanged.

Mobile preview is a **limited view**: drag-and-drop disabled, click-to-select only.

## State
Add `viewMode: 'desktop' | 'mobile'` to `editorStore` (shared hub already imported
by both `EcommerceBuilder` and `SectionEditorLayer` — no prop threading).
Default `'desktop'`.

## Changes

### 1. `stores/editor.svelte` — `viewMode` state
- `viewMode = $state<'desktop' | 'mobile'>('desktop')`

### 2. `builder/BuilderSectionRender.svelte` — `selectOnly` mode
- New prop `interaction: 'full' | 'selectOnly'` (default `'full'`).
- `selectOnly`: omit `draggable` and the `ondragstart/ondragend/ondragover`
  handlers; keep `onclick`/`onkeydown` select. Hide the "Drag to move" hint.

### 3. `builder/MobilePreviewFrame.svelte` (new)
- Renders `<iframe>` (390px wide, centered, full height, no border).
- On mount: clone parent `<head>` stylesheets (`link[rel=stylesheet]`, build
  `<style>`) into `iframeDoc.head` so build-time Tailwind chrome is present.
- `mount(MobileCanvas, { target: iframeDoc.body })`; `unmount` on teardown.
- `$effect` mirrors `liveCSS.css` into an iframe-head `<style id="live-tailwind-jit">`
  so runtime utilities apply at the iframe's viewport width.

### 4. `builder/MobileCanvas.svelte` (new — the iframe root)
- `{#each editorStore.sections}` → `BuilderSectionRender interaction="selectOnly"`.
- Re-establish any context the AST components need (check `EC_BUILDER_MODE` /
  `setContext` in `BuilderSectionRender` — context does NOT cross a fresh `mount()`
  root, so this new root must set what those components read above them).

### 5. `builder/EcommerceBuilder.svelte` — branch canvas by mode
- `editorStore.viewMode === 'mobile'` → render `<MobilePreviewFrame />`
  (no canvas-level `ondrop`/`ondragover` wiring).
- else → existing `.builder-canvas`.

### 6. `builder/SectionEditorLayer.svelte` — the toggle button
- Replace the `.section-title` ("HTML Section") span with a desktop/mobile toggle
  that flips `editorStore.viewMode`.
- Make the `layer-actions` bar render even with no selection (slim bar = just the
  toggle); with a selection = toggle + Save/Delete/Close as today.

## Open question (answer before coding)
- Toggle visibility when nothing is selected: plan assumes **always visible**
  (needed to return from mobile→desktop after deselect). Confirm.

## Verify-before-code check
- Context flow across `mount()`: confirm which `setContext` keys
  `HtmlSection`/`EcommerceSlider` rely on so `MobileCanvas` re-provides them.
