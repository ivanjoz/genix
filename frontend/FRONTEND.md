# Genix Frontend Documentation

## 📋 Table of Contents
1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Directory Structure](#directory-structure)
4. [Package System](#package-system)
5. [Development Workflow](#development-workflow)
6. [Dependency Hierarchy](#dependency-hierarchy)
7. [Key Tools](#key-tools)
8. [Deployment](#deployment)
9. [Migration History](#migration-history)

---

## Project Overview

The Genix Frontend is a monorepo-based SvelteKit application that serves both administrative interfaces and an e-commerce store. The project has been migrated from a single application with mixed routes to a modular monorepo architecture where the **Store** operates as an independent SvelteKit application while sharing common packages with the main admin application.

### Key Characteristics
- **Monorepo Architecture**: Multiple applications sharing common packages
- **Independent Store App**: The store runs as a separate SvelteKit app under `/store` subpath
- **Shared Packages**: Core functionality, UI components, and services are shared across apps
- **Single Development Server**: Unified development experience with proxy-based routing
- **Static Site Generation**: Both apps use SvelteKit's static adapter for deployment
- **TypeScript**: Full TypeScript support throughout the project

### Technology Stack
- **Framework**: SvelteKit 2.x with Svelte 5.x
- **Build Tool**: Vite (using Rolldown for faster builds)
- **Styling**: TailwindCSS v4 with CSS Modules
- **Runtime**: Bun for development and scripts
- **Deployment**: Static files to GitHub Pages

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                 Browser                                     │
│  http://localhost:3572 (unified entry point)                │
│    ├── http://localhost:3572/        (Main/Admin App)       │
│    └── http://localhost:3572/store   (Store App)            │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Proxy Server (port 3572)                       │
│         scripts/proxy-server.js                             │
│  Routes: / → Main (3570), /store → Store (3571)           │
└────────┬──────────────────────┬──────────────────────────────┘
         │                      │
         ▼                      ▼
┌──────────────────┐    ┌──────────────────┐
│ Main App         │    │ Store App       │
│ Vite Dev Server  │    │ Vite Dev Server  │
│ Port 3570        │    │ Port 3571        │
└────────┬─────────┘    └────────┬─────────┘
         │                       │
         └───────────┬───────────┘
                     │
         ┌───────────┴───────────┐
         │                       │
    ┌────▼────┐    ┌────────▼─────────┐
    │ pkg-core│    │ pkg-ui           │
    │         │    │ pkg-components   │
    └─────────┘    │ pkg-services     │
                  └──────────────────┘
```

### Design Rationale

#### Why Monorepo?
- **Code Sharing**: Common functionality is shared across main and store apps
- **Consistency**: Same UI components, services, and utilities across apps
- **Efficiency**: Single development workflow, shared dependencies
- **Maintainability**: Easier to update shared code in one place

#### Why Independent Store App?
- **Performance**: Store can be optimized independently without admin code bloat
- **Separation of Concerns**: Store has different requirements (SEO, marketing)
- **Future Flexibility**: Store can be deployed separately or moved to its own repo
- **Build Optimization**: Code splitting prevents admin code from loading on store routes

#### Why Proxy Server for Development?
- **Unified URL**: Both apps accessible from same origin (localhost:3572)
- **No CORS Issues**: Shared cookies and authentication work seamlessly
- **Hot Reloading**: Both apps maintain HMR independently
- **Production Parity**: Matches the production structure where store is at `/store`

---

## Directory Structure

### Root-Level Structure

```
frontend/
├── routes/              # Main SvelteKit application routes (admin)
├── pkg-store/           # Independent store SvelteKit application
├── pkg-core/            # Core utilities, helpers, and types
├── pkg-ui/              # UI layout components (Page, Header, SideMenu)
├── pkg-components/      # Reusable form and display components
├── pkg-services/        # Service layer for API calls
├── static/              # Global static assets (shared)
├── build/               # Combined build output (not committed)
├── scripts/             # Build and development scripts
├── docs/                # Build output for GitHub Pages (not committed)
├── .svelte-kit/         # SvelteKit build artifacts (not committed)
├── package.json         # Root package.json (workspace configuration)
├── svelte.config.js     # Main app SvelteKit configuration
├── vite.config.ts       # Main app Vite configuration
├── tsconfig.json        # TypeScript configuration
├── plugins.js           # Custom build plugins (CSS hashing, SW builder)
├── postbuild.js         # Post-build script for publishing to docs/
└── app.html             # Main app HTML template
```

### Detailed Package Structure

#### `routes/` - Main Application (Admin)
The main SvelteKit application serving administrative interfaces.

```
routes/
├── admin/               # Administrative interfaces
│   ├── empresas/        # Company management
│   ├── usuarios/        # User management
│   └── ...
├── operaciones/         # Operational interfaces
│   ├── productos/       # Product management
│   ├── ventas/          # Sales management
│   └── ...
├── login/               # Authentication routes
├── inicio/              # Dashboard/home
├── develop-ui/          # UI component development and testing
├── components/          # Component development routes
├── +layout.svelte       # Global layout wrapper
├── +page.svelte         # Root page (redirects)
├── +layout.server.js    # Server-side layout logic
├── app.css              # Global styles
└── ...
```

**Key Files**:
- `+layout.svelte`: Wraps all pages with common UI (header, sidebar, etc.)
- `app.css`: Global CSS imports and Tailwind directives

#### `pkg-store/` - Independent Store Application

A complete SvelteKit application serving the e-commerce store under `/store`.

```
pkg-store/
├── routes/              # Store-specific routes (flattened)
│   ├── +page.svelte     # Store home page
│   ├── productos/       # Product listing and details
│   ├── carrito/         # Shopping cart
│   └── ...
├── components/          # Store-specific components
│   ├── ProductCard.svelte
│   ├── MainCarrusel.svelte
│   └── ...
├── stores/              # Svelte stores for state management
│   ├── cart.svelte.ts
│   ├── productos.svelte.ts
│   └── ...
├── lib/                 # Store utilities and helpers
├── static/              # Store-specific static assets
├── workers/             # Store-specific web workers
├── build/               # Store build output (not committed)
├── package.json         # Store app dependencies
├── svelte.config.js     # Store SvelteKit configuration (base: '/store')
├── vite.config.ts       # Store Vite configuration
├── tsconfig.json        # Store TypeScript configuration
└── app.html             # Store HTML template
```

**Key Configuration**:
- `paths.base`: `/store` - Routes the app to `/store` subpath
- `alias`: Points to shared packages in parent directory
- `adapter`: Static adapter for deployment

#### `pkg-core/` - Core Utilities

Foundation package containing shared utilities with no dependencies on other packages.

```
pkg-core/
├── lib/                 # Core utility functions
│   ├── sharedHelpers.ts # Helper functions used across apps
│   ├── http.ts          # HTTP request utilities
│   ├── security.ts      # Security-related functions
│   ├── sw-cache.ts      # Service worker cache utilities
│   ├── unmarshall.ts    # Data unmarshalling helpers
│   └── icons.ts         # Icon mappings
├── types/               # TypeScript type definitions
│   ├── common.ts        # Common types used across apps
│   └── modules.ts       # Module and menu record types
├── workers/             # Shared web workers
│   └── service-worker.ts# Main service worker (built by vite.config.ts)
├── assets/              # Core assets (fonts, images)
├── env.ts               # Environment configuration
├── helpers.ts           # Additional helper functions
├── modules.ts           # Module definitions and menu structure
├── store.svelte.ts      # Global Svelte stores
└── types/               # Additional type definitions
```

**Purpose**: Level 1 dependency - no imports from other packages, provides foundation for all other packages.

#### `pkg-ui/` - UI Layout Components

Shared UI layout components for page structure and navigation.

```
pkg-ui/
├── AppHeader.svelte     # Main application header
├── Header.svelte        # Generic header component
├── HeaderConfig.svelte  # Header configuration
├── MobileMenu.svelte    # Mobile navigation menu
├── Page.svelte          # Page layout wrapper
├── SideMenu.svelte      # Sidebar navigation
├── HTMLEditor/          # HTML editor component directory
├── assets/              # UI-specific assets (fonts, icons)
├── libs/                # Third-party libraries (fontello, etc.)
└── core.module.css      # Core styles for UI components
```

**Purpose**: Provides layout structure (Page, Header, Menu) used across applications. Depends on `pkg-core`.

#### `pkg-components/` - Reusable Form and Display Components

Shared components for forms, data display, and user interactions.

```
pkg-components/
├── Input.svelte         # Text input with validation
├── SearchSelect.svelte  # Select dropdown with search
├── SearchCard.svelte    # Card-based selection component
├── Checkbox.svelte      # Checkbox component
├── CheckboxOptions.svelte # Multiple checkbox selection
├── DateInput.svelte     # Date picker
├── ColorPicker.svelte   # Color selection
├── ImageUploader.svelte # Image upload with preview
├── Imagehash.svelte     # Blurhash/thumbnail display
├── Layer.svelte         # Modal/layer overlay
├── LayerStatic.svelte   # Static layer overlay
├── MobileLayerVertical.svelte # Mobile layer
├── Modal.svelte         # Modal dialog
├── LoginForm.svelte     # Login form
├── OptionsStrip.svelte  # Tab/option strip
├── Renderer.svelte      # Content renderer
├── ArrowSteps.svelte    # Step indicator
├── ButtonLayer.svelte   # Button with layer trigger
├── TopLayerSelector.svelte # Layer selector
├── popover2/            # Popover component implementation
│   ├── Popover2.svelte
│   ├── Portal.svelte
│   ├── positioning.ts
│   └── README.md
├── vTable/              # Virtual table component directory
├── components.module.css # Component styles
└── styles.module.css    # Additional styles
```

**Purpose**: Provides reusable components for forms and data display. Depends on `pkg-core` and `pkg-ui`.

#### `pkg-services/` - Service Layer

API service layer for backend communication.

```
pkg-services/
└── services/
    └── login.ts         # Login/auth service
```

**Purpose**: Handles API calls and backend communication. Depends on `pkg-core`.

---

## Package System

### Workspace Configuration

The project uses npm workspaces to manage dependencies across the monorepo.

**Root `package.json`**:
```json
{
  "name": "genix-frontend",
  "workspaces": [
    "./pkg-store"
  ],
  "scripts": {
    "dev": "node scripts/dev-all.js",
    "dev:main": "vite dev --force --port 3570",
    "dev:store": "cd pkg-store && bun run dev",
    "build": "node scripts/build-all.js",
    "build:main": "vite build",
    "build:store": "cd pkg-store && bun run build",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json"
  }
}
```

### Path Aliases

SvelteKit path aliases provide clean import syntax across the project.

**Main App `svelte.config.js`**:
```javascript
alias: {
  $ui: path.resolve('./pkg-ui'),
  $store: path.resolve('./pkg-store'),
  $stores: path.resolve('./pkg-store/stores'),
  $routes: path.resolve('./routes'),
  $components: path.resolve('./pkg-components'),
  $core: path.resolve('./pkg-core'),
  $services: path.resolve('./pkg-services')
}
```

**Usage Examples**:
```typescript
// Import from core
import { httpGet } from '$core/http';

// Import from UI components
import { Page } from '$ui/Page.svelte';

// Import from shared components
import { Input, SearchSelect } from '$components';

// Import from services
import { loginService } from '$services/services/login';
```

---

## Development Workflow

### Getting Started

1. **Install Dependencies**:
   ```bash
   bun install
   cd pkg-store && bun install
   ```

2. **Start Development Server**:
   ```bash
   bun run dev
   ```
   This starts:
   - Main app on port 3570 (internal)
   - Store app on port 3571 (internal)
   - Proxy server on port 3572 (unified access)

3. **Access Applications**:
   - Main/Admin: http://localhost:3572
   - Store: http://localhost:3572/store

### Development Scripts

| Command | Description |
|---------|-------------|
| `bun run dev` | Start all dev servers (main, store, proxy) |
| `bun run dev:main` | Start only main app (port 3570) |
| `bun run dev:store` | Start only store app (port 3571) |
| `bun run build` | Build both apps and combine output |
| `bun run build:main` | Build only main app |
| `bun run build:store` | Build only store app |
| `bun run check` | Run TypeScript and Svelte checks |
| `bun run check:watch` | Run checks in watch mode |

### Development Server Architecture

The unified development server uses a proxy to route requests to the appropriate application:

```javascript
// scripts/proxy-server.js
const MAIN_PORT = 3570;
const STORE_PORT = 3571;
const PROXY_PORT = 3572;

// Routes / to main app, /store to store app
app.use('/store', proxy({ target: `http://localhost:${STORE_PORT}` }));
app.use('/', proxy({ target: `http://localhost:${MAIN_PORT}` }));
```

**Benefits**:
- Single URL for both applications
- Shared cookies and authentication
- No CORS issues
- Matches production structure

### Building for Production

1. **Build All Applications**:
   ```bash
   bun run build
   ```

   This process:
   - Builds main app to `build/`
   - Builds store app to `pkg-store/build/`
   - Copies store build to `build/store/`
   - Creates `404.html` for SPA routing

2. **Build Output Structure**:
   ```
   build/
   ├── index.html         # Main app entry
   ├── 404.html          # SPA fallback
   ├── _app/             # Main app assets
   ├── store/            # Store app
   │   ├── index.html    # Store entry
   │   └── _app/         # Store assets
   └── sw.js             # Service worker
   ```

3. **Deploy to GitHub Pages**:
   ```bash
   bun run publish
   ```
   This builds and copies output to `../docs/` for GitHub Pages deployment.

### Code Splitting and Optimization

The build system uses chunk splitting to optimize bundle sizes:

```typescript
// vite.config.ts
manualChunks: (id) => {
  if (id.includes('/admin/') || id.includes('/cms/') || id.includes('/operaciones/')) {
    return 'admin';  // Admin-specific code
  }
  if (id.includes('/ecommerce/') || id.includes('/store/')) {
    return 'store';  // Store-specific code
  }
  if (id.includes('/components/') || id.includes('/lib/') || id.includes('/core/')) {
    return 'shared'; // Shared code
  }
  return 'vendor';   // Third-party libraries
}
```

This ensures:
- Admin code doesn't load on store routes
- Store code doesn't load on admin routes
- Shared code is cached independently

---

## Dependency Hierarchy

### Dependency Graph Levels

The project enforces a strict dependency hierarchy to prevent circular dependencies and maintain code organization:

```
Level 1 (No dependencies):
└── pkg-core/
    └── Types, utilities, helpers (no imports from other packages)

Level 2 (Depends on pkg-core only):
├── pkg-ui/              # Layout components
├── pkg-services/        # API services
└── pkg-components/      # Form and display components

Level 3 (Depends on Levels 1-2):
├── routes/              # Main application (admin)
└── pkg-store/           # Store application
```

### Dependency Rules

1. **pkg-core**: Cannot import from any other package
2. **pkg-ui, pkg-services, pkg-components**: Can import from pkg-core only
3. **routes and pkg-store**: Can import from all shared packages

### Checking Dependencies

Use the DAG analyzer to validate dependency hierarchy:

```bash
bun run scripts/analyze-dag.ts
```

This will:
- Visualize the dependency graph
- Detect circular dependencies
- Identify hierarchy violations
- Provide actionable recommendations

### Fixing Import Issues

If imports break after file moves, use the intelligent import fixer:

```bash
# Dry run to see what will be fixed
bun run scripts/intelligent-import-fixer.ts --dry-run

# Apply fixes
bun run scripts/intelligent-import-fixer.ts --fix
```

---

## Key Tools

### 1. Dependency Graph Analyzer (`analyze-dag.ts`)

**Purpose**: Understand and validate dependency relationships between packages.

**When to Run**:
- Before major refactoring
- After moving files
- When build fails with dependency errors
- Periodically to maintain code health

**Output**:
- Dependency graph visualization
- Hierarchy violations
- Circular dependencies
- Actionable recommendations

**Example**:
```bash
bun run scripts/analyze-dag.ts
```

### 2. Intelligent Import Fixer (`intelligent-import-fixer.ts`)

**Purpose**: Automatically fix import errors after file moves or refactoring.

**When to Run**:
- After moving files (routes/store → pkg-store/routes)
- After moving components to resolve DAG violations
- When import paths break
- Before attempting build

**What It Fixes**:
- Missing file extensions
- Incorrect relative paths
- Package alias imports
- Named vs default imports
- Type-only imports

**Example**:
```bash
# Dry run
bun run scripts/intelligent-import-fixer.ts --dry-run

# Apply fixes
bun run scripts/intelligent-import-fixer.ts --fix

# Verify build
bun run build
```

### 3. Thumbhash Prebuild (`thumbhash-prebuild.js`)

**Purpose**: Generate blur placeholders for product images.

**When to Run**:
- Automatically runs before store build (`prebuild:store`)
- After adding new product images

**What It Does**:
- Generates thumbhashes for images in `static/images/`
- Renames images based on thumbhash
- Outputs thumbhash data to `static/images/thumbhash.txt`

**Example**:
```bash
bun run scripts/thumbhash-prebuild.js
```

### 4. Custom Build Plugins (`plugins.js`)

**Purpose**: Enhance build process with custom functionality.

**Features**:
- **CSS Hasher**: Counter-based CSS class hashing for deterministic builds
- **Service Worker Builder**: Builds service worker with Vite

**Usage**:
```javascript
// vite.config.ts
plugins: [
  sveltekit(),
  isBuild && svelteClassHasher(),  // CSS hashing
  tailwindcss(),
  serviceWorkerPlugin()             // SW builder
].filter(x => x)
```

---

## Deployment

### Deployment Structure

The project deploys as a static site to GitHub Pages:

```
docs/                          # GitHub Pages deployment directory
├── index.html                 # Main app entry
├── 404.html                   # SPA fallback
├── _app/                      # Main app assets
│   ├── immutable/
│   ├── version/
│   └── ...
├── store/                     # Store app
│   ├── index.html            # Store entry
│   └── _app/                 # Store assets
├── sw.js                      # Service worker
└── static/                    # Static assets
```

### Deployment Process

1. **Build**:
   ```bash
   bun run build
   ```

2. **Publish** (copies build to docs/):
   ```bash
   bun run publish
   ```

3. **Push to GitHub**:
   ```bash
   git add docs/
   git commit -m "Deploy to GitHub Pages"
   git push
   ```

### Service Worker

The service worker is built by `vite.config.ts` and deployed to `build/sw.js`:

**Source**: `pkg-core/workers/service-worker.ts`
**Build Output**: `static/sw.js`

**Purpose**:
- Cache static assets
- Cache API responses
- Offline support
- Fast page loads

---

## Finding Things in This Project

### Quick Reference

| What You Need | Where to Find It |
|--------------|------------------|
| Admin pages | `routes/admin/`, `routes/operaciones/` |
| Store pages | `pkg-store/routes/` |
| Shared components | `pkg-components/` |
| UI layouts | `pkg-ui/` |
| Utilities | `pkg-core/lib/` |
| Types | `pkg-core/types/` |
| Services | `pkg-services/` |
| Stores (Svelte) | `pkg-store/stores/`, `pkg-core/store.svelte.ts` |
| Static assets | `static/` (global), `pkg-store/static/` (store) |
| Dev scripts | `scripts/` |
| Build config | `vite.config.ts`, `svelte.config.js` |
| Migration docs | `scripts/migration/` |

### Common Tasks

**Add a new admin page**:
1. Create file in `routes/admin/` or `routes/operaciones/`
2. Import components from `$components` or `$ui`
3. Use layout wrapper (handled by `+layout.svelte`)

**Add a new store page**:
1. Create file in `pkg-store/routes/`
2. Import from shared packages (`$ui`, `$components`, `$core`)
3. Use store-specific layout

**Create a reusable component**:
1. Add to `pkg-components/` for form/display components
2. Add to `pkg-ui/` for layout components
3. Import in pages using `$components` or `$ui`

**Add a utility function**:
1. Add to `pkg-core/lib/helpers.ts` or create new file
2. Import using `$core/...`

**Create a new service**:
1. Add to `pkg-services/services/`
2. Import using `$services/services/...`

**Debug dependency issues**:
1. Run `bun run scripts/analyze-dag.ts`
2. Identify circular dependencies
3. Move components to fix hierarchy
4. Run `bun run scripts/intelligent-import-fixer.ts --fix`

---

## Best Practices

### Code Organization

1. **Follow Dependency Hierarchy**: Ensure imports flow from lower to higher levels
2. **Use Path Aliases**: Always use `$core`, `$ui`, `$components`, etc. instead of relative paths
3. **Keep Components Small**: Break down complex components into smaller, reusable pieces
4. **Share Wisely**: Put truly shared code in packages, app-specific code in routes

### Development Workflow

1. **Run DAG Analysis**: Before major changes, analyze dependencies
2. **Use Import Fixer**: After moving files, let the tool fix imports automatically
3. **Build Early and Often**: Don't wait until the end to build
4. **Test Both Apps**: Verify both main and store work after changes

### Performance

1. **Code Splitting**: Leverage automatic chunk splitting for route-specific code
2. **Lazy Load**: Use dynamic imports for large components
3. **Optimize Images**: Use thumbhash for product images
4. **Cache Strategically**: Configure service worker for optimal caching

---

## Troubleshooting

### Common Issues

**Build fails with import errors**:
```bash
# Fix imports
bun run scripts/intelligent-import-fixer.ts --fix
```

**Circular dependency detected**:
```bash
# Analyze DAG
bun run scripts/analyze-dag.ts
# Move problematic components to lower level
```

**Proxy server not working**:
- Ensure both main and store apps are running
- Check ports 3570, 3571, and 3572 are available
- Verify `scripts/proxy-server.js` is configured correctly

**Store styles not loading**:
- Check that `pkg-store/svelte.config.js` has correct aliases
- Verify CSS imports in store components
- Ensure static assets are in `pkg-store/static/`

### Getting Help

1. Check this documentation
2. Review migration plan and status
3. Run DAG analyzer to understand dependencies
4. Check build logs for specific errors

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-18  
**Maintained By**: Engineering Team
