# Genix Frontend Refactor Plan

## 🎯 Goal
Reorganize the frontend structure to improve clarity, separate business logic from generic utilities, and follow a more semantic naming convention.

## 📁 New Directory Structure
The `pkg-*` prefix will be removed from all folders.

| Current/Old Name | New Name | Responsibility |
| :--- | :--- | :--- |
| `pkg-components` | `ui-components` | "Dumb" UI atoms (Inputs, Buttons, Tables). No business logic. |
| `pkg-ui` | `domain-components` | Business-aware UI blocks (Headers, SideMenus, specific editors). |
| `pkg-store` | `ecommerce` | The independent store application logic and routes. |
| `pkg-core` | `core` | App infrastructure (State, Security, Env, Modules). |
| `pkg-services` | `services` | API communication layer. |
| (New) | `libs` | Pure technical utilities (HTTP client, string/math helpers, unmarshall). |

## 🏗️ Dependency Hierarchy (DAG)
1. **`libs`** (Level 0): No dependencies.
2. **`ui-components`** (Level 1): Depends on `libs`.
3. **`core`** (Level 2): Depends on `libs`.
4. **`services`** (Level 3): Depends on `libs`, `core`.
5. **`domain-components`** (Level 4): Depends on `libs`, `core`, `services`.
6. **`ecommerce` & `routes`** (Level 5): Consumers of all above.

## 🛠️ Step-by-Step Execution

### Step 1: Directory Reorganization
- [ ] Rename `business-components` to `domain-components` (since it was renamed in a previous attempt).
- [ ] Rename `pkg-core` to `core`.
- [ ] Rename `pkg-services` to `services`.
- [ ] Ensure `ui-components`, `ecommerce`, and `libs` exist.

### Step 2: Logic Split (`core` -> `libs`)
- [ ] Move `core/lib/http.ts` to `libs/http.ts`.
- [ ] Move `core/lib/unmarshall.ts` to `libs/unmarshall.ts`.
- [ ] Move `core/lib/sharedHelpers.ts` to `libs/sharedHelpers.ts`.
- [ ] Move generic assets (like `angle.svg`) from `core/assets` to `libs/assets`.

### Step 3: Configuration Updates
- [ ] Update root `package.json` workspaces (point to `./ecommerce`).
- [ ] Update root `svelte.config.js` aliases:
    - `$ui` -> `$domain` (`domain-components`)
    - `$components` -> `ui-components`
    - `$core` -> `core`
    - `$services` -> `services`
    - `$libs` -> `libs`
- [ ] Update root `vite.config.ts` alias resolver logic.
- [ ] Update root `tsconfig.json` paths.
- [ ] Repeat configuration updates inside `ecommerce/` app configs.

### Step 4: Import Correction
- [ ] Run a global search and replace for path aliases.
- [ ] Run `intelligent-import-fixer.ts` to resolve broken paths.
- [ ] Manually verify critical entry points (`+layout.svelte`, `store.svelte.ts`).

### Step 5: Verification
- [ ] Run `bun run check` for type safety.
- [ ] Run `bun run build` to ensure the pipeline is intact.
- [ ] Run `analyze-dag.ts` to verify the new hierarchy.
