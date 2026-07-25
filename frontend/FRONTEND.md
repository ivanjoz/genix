# Genix Frontend Documentation

## Project Overview
Genix Frontend is a modular monorepo using SvelteKit 2 (Svelte 5). It consists of two primary applications: a main Admin backoffice and an independent E-commerce Store. The architecture enforces a strict dependency hierarchy to ensure scalability and maintainability.

### Tech Stack
- **Framework**: SvelteKit 2.x (Svelte 5.x)
- **Runtime**: Bun (Development & Scripts)
- **Styling**: TailwindCSS v4 with CSS Modules
---

## Architecture & Development Workflow

### Development Server & Proxy
The project uses a unified development entry point to simulate production routing and avoid CORS issues.
- **Main Admin Port**: 3570 (Internal)
- **Store Port**: 3571 (Internal)
- **Proxy Port**: 3572 (Public Entry) - Managed by `scripts/proxy-server.js`.
- **Unified URL**: Access both apps via `http://localhost:3572` (Admin at `/`, Store at `/store`).

### Build & Deployment Structure
The build process merges both applications into a single static directory structure.
- `build/`: Final combined output.
    - `index.html`: Admin entry point.
    - `404.html`: SPA fallback for Admin routes.
    - `store/`: Independent store application.
        - `index.html`: Store entry point.
- `publish`: Syncs the `build/` folder to the root `docs/` directory for GitHub Pages.

---

## Directory Structure & Responsibilities

### Applications
- `routes/`: Main Admin/Backoffice application logic and pages.
- `ecommerce/`: Customer-facing store. Built as a separate SvelteKit project.
    - `ecommerce/routes/`: Store-specific pages.
    - `ecommerce/stores/`: Local Svelte state (cart, products).

### Shared Packages (Level-based Hierarchy)
- `libs/`: (Level 0) Thin Genix adapters and remaining technical utilities.
    - `ui-runtime.svelte.ts`: The single configuration entry for `@genix/ui`. One
      `createUiRuntime` call sets routing, tenant, translation, notifications, request
      reporting, and session/access policy (`security`), and exports `genixUiRuntime`
      plus `security`, and binds that runtime's transport to `GET`/`POST`/`GetHandler`
      and its image conversion helpers. The admin layout registers the access catalog
      on top of it.
- `packages/genix-ui/`: (Level 1) reusable UI atoms in root-level source folders.
    - Inputs, tables, modals, Excel, HTTP transport, caches, assets, and UI automation.
    - Excel owns its WASM asset; Svelte/Vite emits the fingerprinted file automatically.
    - HTTP, cache, UI state, images, conversion, and persistence are composed once through
      `createUiRuntime`.
- `core/`: (Level 2) Infrastructure and global state.
    - `store.svelte.ts`: Shared reactive state.
    - `env.ts`: Runtime environment variables.
- `services/`: (Level 3) API communication layer.
    - Encapsulates backend endpoints into reusable functions.
- `domain-components/`: (Level 4) Business-aware UI blocks for Admin.
    - `AppHeader.svelte` and thin host adapters for reusable package components.
    - Allowed to import from `libs`, `@genix/ui`, `core`, and `services`.

---

### Directory structure
Each page is a folder, for example  frontend/routes/logistica/purchase-orders

In the case of page with multiple views (The view options are rendered in the top menu) each view must be a component like this

```svelte
<script lang="ts">
  import { useUI } from '@genix/ui'

  const ui = useUI()
</script>

<Page title="Órdenes de Compra" options={pageOptions}>
  {#if ui.state.pageOptionSelected === 1}
    <PurchaseOrderCreate />
  {/if}
  {#if ui.state.pageOptionSelected === 2}
    <PurchaseOrderReport />
  {/if}
</Page>
```

Example of directory files:
 - +page.svelte
 - PurchaseOrderCreate.svelte
 - PurchaseOrderReport.svelte
 - Component.svelte
 - Component2.svelte
 - purchase_order.svelte.ts (where the service is declared for fetching)
 - purchase_order.md

 The .md file must start with a 2 - 4 line description of what you can do in the page, including the views. This will serve as an index for the Agent to determine if is necesary to navigate to this page.

### Agentic Componentes

The resulting HTML MUST help the Agent to navigate, so using aria-label for example in form container to describe the form content, or ussing label property on Button element to describe the action is mandatory, example:

```svelte
<Button name="Nuevo" label="Shows the create a product form in a side layer."
  color="green" icon="icon-plus"
/>
<div aria-label="Product Form">
	...
</div>
```


## Dependency Hierarchy (DAG) Rules
Strict rules prevent circular dependencies and ensure the `ecommerce` app remains lightweight.

1. **Hierarchy Violation**: A lower-level package (e.g., `libs`) cannot import from a higher-level one (e.g., `core`).
2. **Ecommerce Isolation**: The Store CANNOT import from `domain-components` or `routes`. It must remain independent.
3. **Cross-App Imports**: Never import directly between `routes/` and `ecommerce/`.
4. **Validation**: Always run `bun run scripts/analyze-dag.ts` after structural changes.

---

## Path Aliases
Aliases are configured in `svelte.config.js` and `tsconfig.json`.
- `$core`: `./core`
- `$libs`: `./libs`
- `$components`: `./packages/genix-ui`
- `$domain`: `./domain-components` (Admin Only)
- `$services`: `./services`
- `$ecommerce`: `./ecommerce`
- `$routes`: `./routes` (Admin Only)

---

## Agent Guidance: Common Tasks

- **Adding an Admin Page**: Create a file in the appropriate module directory within `routes/` (e.g., `routes/configuracion/`, `routes/negocio/`, `routes/comercial/`, etc.). Use `$domain` for layout and `$components` for forms.
- **Adding a Store Page**: Create a file in `ecommerce/routes/`. Only use `$components`, `$core`, and `$services`.
- **Modifying Global Logic**: Edit `core/store.svelte.ts` or `core/modules.ts`.
- **Fixing Styles**: Most components use local CSS modules (`[name].module.css`). Check `app.css` for global Tailwind variables.

---

## Troubleshooting & Best Practices

### Common Issues
- **CORS/Proxy Errors**: Ensure both dev servers are running before starting the proxy.
- **Circular Dependencies**: Package source must not import app infrastructure
  (`services`, `domain-components`, or route modules). Inject host capabilities
  through `libs/ui-runtime.svelte.ts`.
- **Build Mismatch**: If the store doesn't reflect changes, rebuild specifically using `bun run build:store`.

### Best Practices
- **Atomic UI**: Keep root-level source folders in `packages/genix-ui` generic and reusable.
- **Hydration**: Use `browser` checks from `$app/environment` when accessing `localStorage` or `window`.
