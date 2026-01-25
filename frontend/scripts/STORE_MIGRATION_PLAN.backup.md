# Turborepo & Independent Store Migration Plan

This plan outlines the strategy for transforming the Genix Frontend into a monorepo where the **Store** operates as an independent SvelteKit application, served under the `/store` subpath.

## 🏗️ 1. Proposed Architecture (Monorepo)

The project will be managed using **Turborepo** with npm/bun workspaces.

### Workspace Structure
- **`apps/main`**: The primary administration/ERP application (current root `routes/`).
- **`apps/store`**: The new independent SvelteKit application (derived from `pkg-store/`).
- **`packages/*`**: Shared logic (currently `pkg-core`, `pkg-ui`, `pkg-services`, `pkg-components`).

## 🛠️ 2. Step-by-Step Migration Strategy

### Step 1: Workspace Initialization
- Create a root `package.json` defining the workspaces.
- Define `turbo.json` to manage pipeline tasks (build, dev, lint) across apps.
- Maintain the shared `pkg-*` directories as internal packages that both apps can import.

### Step 2: Transforming `pkg-store` into an App
- Initialize a SvelteKit project structure inside the `pkg-store` directory (or a new `apps/store` directory if relocation is preferred later).
- **Required Files for Store App**:
  - `pkg-store/package.json`: Define as a SvelteKit app.
  - `pkg-store/svelte.config.js`: Configure `adapter-static` and `paths.base = '/store'`.
  - `pkg-store/vite.config.ts`: Define aliases for shared packages (`$core`, `$ui`, etc.).
  - `pkg-store/src/`: Move `pkg-store/components`, `pkg-store/stores`, and `pkg-store/lib` into `pkg-store/src/lib`.

### Step 3: Route Migration
- **Move**: `routes/store/**` (from the main app) $\rightarrow$ `pkg-store/src/routes/`.
- **Refactor**: In the Store app, the routes will no longer be nested under `/store/` in the file system, as the `base` path configuration handles the URL prefix.
  - Example: `routes/store/+page.svelte` $\rightarrow$ `pkg-store/src/routes/+page.svelte`.

### Step 4: Shared Package Consumption
- Update the main app and store app to reference shared packages via workspace aliases.
- Example in `package.json`: `"@genix/core": "workspace:*"`
- This allows `$core` to resolve to the same code in both applications.

## 🚀 3. Build & Deployment (GitHub Pages)

Since GitHub Pages serves from a single directory (usually `docs/` or `gh-pages` branch), we need a **Merge Strategy**.

### The "Subpath" Build Strategy
1.  **Main App Build**: Builds to `apps/main/build/`.
2.  **Store App Build**:
    - Configured with `kit.paths.base = '/store'`.
    - Builds to `pkg-store/build/`.
3.  **Post-Build Merge Script (`scripts/combine-builds.js`)**:
    - Create a clean `dist/` directory.
    - Copy `apps/main/build/*` $\rightarrow$ `dist/`.
    - Copy `pkg-store/build/*` $\rightarrow$ `dist/store/`.
    - **Crucial**: Ensure the Store's `index.html` is at `dist/store/index.html`.

### Asset Handling
- SvelteKit's `paths.base` will ensure that all internal links in the Store app (JS, CSS, Images) are prefixed with `/store/`.
- Shared assets (like fonts or global images) should be copied to the root of `dist/` or duplicated in the store build to ensure 404s are avoided.

## 🧪 4. Local Development Workflow

- **Main App**: `bun run dev` (runs on port 3000).
- **Store App**: `bun run dev` (runs on port 3001, accessible at `localhost:3001/store`).
- **Turbo**: `bun turbo dev` runs both simultaneously.

---

## 📋 Suggested Configuration Snippets

### `pkg-store/svelte.config.js`
```javascript
const config = {
  kit: {
    adapter: adapter({ fallback: '404.html' }),
    paths: {
      base: '/store',
    }
  }
};
```

### `turbo.json`
```json
{
  "pipeline": {
    "build": {
      "outputs": ["build/**"]
    }
  }
}
```