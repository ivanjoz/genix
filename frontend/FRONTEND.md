# Genix Frontend Documentation

## Project Overview
Genix Frontend is a modular monorepo using SvelteKit 2 (Svelte 5). It consists of two primary applications: a main Admin backoffice and an independent E-commerce Store. The architecture enforces a strict dependency hierarchy to ensure scalability and maintainability.

### Tech Stack
- **Framework**: SvelteKit 2.x (Svelte 5.x)
- **Runtime**: Bun (Development & Scripts)
- **Build Tool**: Vite with Rolldown (Optimized builds)
- **Styling**: TailwindCSS v4 with CSS Modules
- **Deployment**: Static Site Generation (SSG) targeted at GitHub Pages

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
- `libs/`: (Level 0) Pure technical utilities. No business logic.
    - `http.ts`: Axios-based client with interceptors.
    - `unmarshall.ts`: JSON-to-class/type mapping logic.
- `ui-components/`: (Level 1) "Dumb" UI atoms.
    - Inputs, Buttons, Tables, Virtualized Lists, Modals.
    - Must NOT import from `core`, `services`, or `domain-components`.
- `core/`: (Level 2) Infrastructure and global state.
    - `store.svelte.ts`: Shared reactive state.
    - `env.ts`: Runtime environment variables.
- `services/`: (Level 3) API communication layer.
    - Encapsulates backend endpoints into reusable functions.
- `domain-components/`: (Level 4) Business-aware UI blocks for Admin.
    - `AppHeader.svelte`, `SideMenu.svelte`, `HTMLEditor/`.
    - Allowed to import from `libs`, `ui-components`, `core`, and `services`.

---

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
- `$components`: `./ui-components`
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
- **Circular Dependencies**: Occurs when `ui-components` imports from `core`. Move shared logic to `libs`.
- **Build Mismatch**: If the store doesn't reflect changes, rebuild specifically using `bun run build:store`.

### Best Practices
- **Atomic UI**: Keep `ui-components` generic and reusable.
- **Type Safety**: Define interfaces in `core/types/` for shared entities.
- **Logic Placement**: API calls belong in `services/`, not directly in Svelte components.
- **Hydration**: Use `browser` checks from `$app/environment` when accessing `localStorage` or `window`.
