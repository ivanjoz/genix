# Iconify Migration Plan

Replace the current Fontello `icon-*` CSS class system with Iconify Tailwind CSS icons in the Svelte frontend.
The migration keeps the existing string-based icon API while replacing old Fontello classes with concrete Iconify classes.

## Short Answer: Does Iconify Download All Icons?

No. The default Iconify Svelte component loads only the icon data requested by the rendered icons at runtime through the Iconify API.

For this project, runtime API loading is not ideal because the admin and storefront should not depend on a third-party icon API for UI controls. Use a local/offline strategy instead:

- Install only the Iconify package needed by the chosen strategy.
- Install only the icon set package or icon JSON collection we decide to use, not every Iconify library.
- Bundle only the icons referenced by Genix, or a small curated set that covers the existing Fontello names.

Project decision: use Tailwind Iconify CSS classes with Font Awesome 4 as the primary ERP/app icon family, and keep Lucide available for WYSIWYG/editor toolbar icons. Replace old `icon-*` names with explicit classes such as `icon-[fa--plus]`, `icon-[fa--trash]`, and `icon-[fa--floppy-o]`.

Official references:

- Iconify Svelte component: https://iconify.design/docs/icon-components/svelte/
- Iconify icon components overview: https://iconify.design/docs/icon-components/

## Current State

Fontello is loaded globally from:

- `frontend/app.html`
- `frontend/webpage/app.html`
- `frontend/static/libs/fontello-embedded.css`
- `frontend/domain-components/libs/fontello-embedded.css`
- `frontend/domain-components/libs/fontello-prerender.css`

The codebase uses icon names in two forms:

- Direct markup: `<i class="icon-plus"></i>`
- Component props: `<Button icon="icon-plus" />`, `saveIcon="icon-floppy"`, `iconOnShow="icon-cancel"`

Important shared render points:

- `frontend/ui-components/buttons/Button.svelte`
- `frontend/ui-components/buttons/ButtonLayer.svelte`
- `frontend/ui-components/layers/Layer.svelte`
- `frontend/ui-components/layers/Modal.svelte`
- `frontend/ui-components/form/FilterInput.svelte`
- `frontend/ui-components/form/SearchSelect.svelte`
- `frontend/ui-components/vTable/*`

The current icon inventory includes about 140 unique `icon-*` names. Some are already missing from the Fontello bundle, for example `icon-mobile`, `icon-desktop`, `icon-bell`, `icon-cart`, `icon-chip`, `icon-message`, `icon-rotate-right`, and `icon-spin5`.

## Migration Strategy

### 1. Choose Offline Iconify Mode

Use a local/offline bundle instead of runtime API loading.

Preferred implementation:

- Add `@iconify/tailwind4`.
- Add selected local icon collection packages, primarily `@iconify-json/fa`.
- Configure the Tailwind v4 plugin in both Tailwind entry files.
- Use generated CSS classes directly: `icon-[fa--plus]`.

Alternative implementation:

- Use the newer per-icon Svelte component packages, such as `@iconify-svelte/tabler`.
- Import individual icon components directly.
- This gives strong tree-shaking, but it is less convenient for the current project because many icons are passed as strings through shared components.

Decision for this migration: keep string props such as `Button icon="..."`, but rewrite their values to Tailwind Iconify classes. This avoids a runtime wrapper and keeps the existing component API.

### 2. Configure Tailwind Iconify

Install local dependencies:

```sh
cd frontend
bun add -d @iconify/tailwind4 @iconify-json/fa @iconify-json/lucide
```

Add to:

- `frontend/routes/tailwind.css`
- `frontend/webpage/routes/tailwind.css`

```css
@plugin "@iconify/tailwind4" {
  prefix: "icon";
  scale: 1;
}
```

### 3. Replace Legacy Classes Mechanically

Do not keep `iconNameByLegacyClass` if the codebase is rewritten directly. Use an explicit replacement map plus a script/regex pass so every old class becomes a concrete Tailwind Iconify class.

Example replacement:

```svelte
<Button color="green" icon="icon-[fa--plus]" />
<i class="icon-[fa--trash] text-red-600"></i>
```

Mapping rules:

- Prefer one icon family for visual consistency.
- Use `fa` first for ERP, navigation, finance, logistics, security, and general app actions.
- Use `lucide` for WYSIWYG/editor toolbar icons where lighter tool glyphs are clearer.
- Treat missing Fontello names as first-class replacements instead of preserving the broken behavior.

### 4. Verify Shared Components

Shared components can keep rendering icon strings with `<i class={icon}></i>` because the string is now a concrete Iconify Tailwind class.
Verify the main icon render paths still pass icon class names through unchanged:

- `Button.svelte`
- `ButtonLayer.svelte`
- `Layer.svelte`
- `Modal.svelte`
- `FilterInput.svelte`
- `SearchSelect.svelte`
- `Checkbox.svelte`
- `CheckboxOptions.svelte`
- `VTable.svelte`
- `TableGrid.svelte`
- `MobileCardsVirtualList.svelte`

This removes most icon usage without touching every page at once.

### 5. Replace Direct `<i>` Usage

Replace direct markup in pages and domain components:

```svelte
<!-- Before -->
<i class="icon-trash"></i>

<!-- After -->
<i class="icon-[fa--trash]"></i>
```

For dynamic classes:

```svelte
<!-- Before -->
<i class={total === 0 ? 'icon-plus' : 'icon-pencil'}></i>

<!-- After -->
<i class={total === 0 ? 'icon-[fa--plus]' : 'icon-[fa--pencil]'}></i>
```

If the old `<i>` included Tailwind classes, keep those classes on the same element:

```svelte
<i class="icon-[fa--exclamation-triangle] text-red-500"></i>
```

### 6. Handle HTML String Icons Separately

Some components return strings that include icon HTML, for example table render callbacks.

Do not introduce a runtime Iconify component into strings. Instead:

- Prefer structured render data where practical.
- If the current component expects HTML strings, use concrete local CSS classes such as `<i class="icon-[fa--refresh]"></i>`.
- Keep the change local to the component that owns the render callback.

### 7. Remove Fontello Imports And Assets

Only after `rg "icon-[a-z0-9_-]+" frontend` shows no remaining Fontello-style render usage outside docs and the mapping file:

- Remove Fontello `<link>` tags from `frontend/app.html`.
- Remove Fontello `<link>` tags from `frontend/webpage/app.html`.
- Remove `frontend/static/libs/fontello-embedded.css`.
- Remove `frontend/domain-components/libs/fontello-embedded.css`.
- Remove or replace `frontend/domain-components/libs/fontello-prerender.css`.
- Update `frontend/core/agent/screenshot.ts` so it no longer treats `fontello` as a pseudo font family.

Keep the legacy mapping file until all string props use real Iconify names.

### 8. Update Documentation

Update:

- `frontend/FRONTEND.md`
- `frontend/docs/UI_COMPONENTS.md`

Document the new icon convention:

```svelte
<Button color="green" icon="icon-[fa--plus]" label="Shows the create form." />
```

New code should use Iconify Tailwind classes directly.

## Validation Checklist

Run these checks after each phase:

```sh
cd frontend
bun run check
bun run build
```

Search checks:

```sh
rg "fontello|fontello-embedded|fontello-prerender" frontend
rg "<i[^>]+icon-|class=\\{[^}]*icon-" frontend
rg "icon-[a-z0-9_-]+" frontend
```

Manual UI checks:

- Header icons render.
- Side menu icons render.
- Buttons with `icon`, `saveIcon`, and `iconOnShow` render.
- Tables render edit/delete icons.
- Form validation icons render.
- Storefront prerender route still renders icons.
- Icons inherit text color and button sizing.

## Suggested Phases

1. Add Iconify dependency, wrapper, mapping file, and minimal tests by running `bun run check`.
2. Migrate shared UI components.
3. Migrate direct page/domain `<i>` usage.
4. Switch new examples to direct Iconify names.
5. Remove Fontello links and CSS assets.
6. Build admin and storefront, then visually verify the main workflows.

## Open Decision

Pick the icon family before implementation.

Recommended: `tabler`, because it has a broad ERP-friendly outline set and maps cleanly to common actions like save, delete, edit, search, warehouse, truck, chart, calendar, and settings.
