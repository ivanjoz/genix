# Turborepo & Independent Store Migration Plan - SIMPLIFIED

## 📋 Executive Summary

This plan outlines a strategy for transforming the Genix Frontend into a monorepo where the **Store** operates as an independent SvelteKit application within `pkg-store/`, served under the `/store` subpath. **This simplified approach preserves the current directory structure** and only moves store routes into `pkg-store/`, minimizing disruption and risk.

## 🔍 Current Architecture Analysis

### Current Project Structure
```
frontend/
├── routes/              # Main SvelteKit routes (admin + store mixed)
│   ├── store/          # Store-specific routes (TO BE MOVED)
│   ├── admin/          # Admin routes
│   ├── inicio/         # Home routes
│   ├── login/          # Authentication routes
│   ├── operaciones/    # Operations routes
│   ├── +layout.svelte  # Global layout
│   └── +page.svelte    # Root page
├── pkg-store/          # Store-specific code (TO BECOME APP)
│   ├── components/     # Store components
│   ├── stores/         # Store state management
│   ├── lib/            # Store utilities (currently empty)
│   ├── routes/         # Empty (will receive store routes)
│   ├── static/         # Store-specific static files
│   ├── workers/        # Store workers
│   └── scripts/        # Store-specific scripts
├── pkg-core/           # Core utilities and helpers ✅ STAYS
├── pkg-ui/             # UI components ✅ STAYS
├── pkg-components/     # Shared components ✅ STAYS
├── pkg-services/       # Services ✅ STAYS
├── static/             # Global static files ✅ STAYS
├── svelte.config.js    # Single app configuration (will be replaced)
├── vite.config.ts      # Custom plugins and build config
├── plugins.js          # Custom build plugins (class hasher, service worker)
└── postbuild.js        # Post-build script for GitHub Pages
```

### Current Dependencies & Complexity
- **Shared Dependencies**: Both admin and store share `$core`, `$ui`, `$components`, `$services`
- **Circular References**: Some store components import from admin, admin imports from store
- **Service Worker**: Global service worker built from `pkg-core/workers/service-worker.ts`
- **Custom Build Plugins**: Complex class hashing and service worker building
- **Port Configuration**: Dev server runs on port 3570
- **GitHub Pages**: Build outputs to `build/`, copied to `../docs/` via postbuild.js
- **Asset Management**: Shared assets in `static/`, store-specific in `pkg-store/static/`

### DAG (Dependency Graph) Issues
```
routes/store/ ← imports → pkg-components/
    ↑                            ↓
pkg-store/              imports pkg-ui/
    ↓                            ↑
pkg-core/ ←─────────────────────┘
```

**Problem**: Circular dependencies between `routes/store/`, `pkg-components/`, and `pkg-ui/`.

**Solution**: Move problematic components to `pkg-core/` to create a clear hierarchy where `pkg-core` has no dependencies on other packages.

## 🏗️ Target Architecture

### Final Project Structure After Migration
```
frontend/
├── routes/              # Main app (admin only)
│   ├── admin/
│   ├── inicio/
│   ├── login/
│   ├── operaciones/
│   ├── +layout.svelte
│   └── +page.svelte
├── pkg-store/           # ✨ Independent SvelteKit app
│   ├── routes/          # Store routes (flattened from routes/store/)
│   ├── components/
│   ├── stores/
│   ├── lib/
│   ├── static/
│   ├── workers/
│   ├── package.json     # App package.json
│   ├── svelte.config.js # App config with base: '/store'
│   └── vite.config.ts  # App config
├── pkg-core/           # ✅ STAYS (may have moved components)
├── pkg-ui/             # ✅ STAYS
├── pkg-components/     # ✅ STAYS
├── pkg-services/       # ✅ STAYS
├── static/             # ✅ STAYS
├── svelte.config.js    # Main app config (admin only)
├── vite.config.ts      # Main app config (admin only)
├── package.json        # Workspace root config
└── scripts/
    ├── dev-all.js      # Single dev server orchestrator
    ├── build-all.js    # Combined build script
    └── proxy-server.js # Development proxy server
```

### Dependency Hierarchy After Migration
```
Level 1 (No dependencies):
└── pkg-core/

Level 2 (Depends on pkg-core):
├── pkg-ui/
├── pkg-services/
└── pkg-components/      # After moving circular components to pkg-core

Level 3 (Depends on Level 1-2):
├── routes/ (admin app)
└── pkg-store/ (store app)
```

## ⚠️ Critical Issues & Solutions

### 1. Circular Dependencies
**Issue**: Store routes import components that import from store routes.

**Solution**: Move problematic components to `pkg-core/` in Phase 1.

### 2. Service Worker Scope
**Issue**: Current service worker serves both apps.

**Solution**: Create separate service workers:
- Main app: `sw.js` in root
- Store app: `sw.js` in `pkg-store/static/`

### 3. Shared State
**Issue**: Admin and store share state via `$core/store.svelte`.

**Solution**: Keep shared state in `pkg-core/`, accessible by both apps.

## 🛠️ Phased Migration Strategy

### Phase 1: Dependency Analysis & DAG Validation (Days 1-2)
**Goal**: Understand current dependency structure and identify violations before moving files

#### Step 1.1: Run Initial DAG Analysis
Use the automated `analyze-dag.ts` script to understand current state:
```bash
bun run scripts/analyze-dag.ts
```

**What this script does:**
- Analyzes all imports across packages
- Detects circular dependencies
- Identifies hierarchy violations
- Visualizes the dependency graph
- Provides actionable recommendations

**Expected Output:**
- Dependency graph showing current package relationships
- List of hierarchy violations (e.g., pkg-ui importing from pkg-components)
- List of circular dependencies
- Files and symbols affected by violations

#### Step 1.2: Refine analyze-dag.ts if Needed
Review the output and check if the script accurately captures all dependencies. If needed, refine the script:

**Potential refinements needed:**
- Update alias mappings if new aliases exist
- Add missing packages to the PACKAGES array
- Adjust ALLOWED_DEPENDENCIES rules
- Improve import pattern matching
- Add better visualization of complex cycles

**Test refinements:**
```bash
bun run scripts/analyze-dag.ts
# Verify output is accurate and complete
```

#### Step 1.3: Document Current State
Create a baseline documentation:
```bash
# Save current DAG analysis
bun run scripts/analyze-dag.ts > dag-analysis-before.txt

# Count current violations
echo "Violations found: $(grep -c 'HIERARCHY\|CIRCULAR' dag-analysis-before.txt)"
```

#### Step 1.4: Plan Component Migration
Based on DAG analysis, create a migration plan:

**Identify components to move:**
```bash
# Find all files in pkg-components that pkg-ui depends on
grep -r "from '\$components" pkg-ui/ | cut -d: -f2 | sort -u > components-to-move.txt
```

**Prioritize moves:**
1. Utilities and helpers (no dependencies on other packages)
2. Shared state management
3. Base components (no UI dependencies)
4. Type definitions
5. Everything else that causes cycles

### Phase 2: Move Files & Stabilize Structure (Days 3-5)
**Goal**: Move store routes and components to pkg-store, then fix imports automatically

#### Step 2.1: Move Store Routes to pkg-store
```bash
# Copy store routes to pkg-store (preserve original for now)
cp -r routes/store/* pkg-store/routes/

# Verify structure
tree pkg-store/routes/
```

**Expected structure:**
```
pkg-store/routes/
├── +page.svelte
├── +page.svelte.json
├── +layout.svelte
├── +layout.ts
├── components/
│   └── +page.svelte
└── store.css
```

#### Step 2.2: Create pkg-store App Configuration
**Create `pkg-store/package.json`:**
```json
{
  "name": "@genix/store",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  	"scripts": {
  		"dev": "vite dev --force --port 3571",
  		"build": "vite build",
  		"check": "svelte-check --tsconfig ./tsconfig.json",
  		"preview": "vite preview"
  	},
  "dependencies": {
    "svelte": "^5.39.5",
    "@sveltejs/kit": "^2.43.2"
  },
  "devDependencies": {
    "@sveltejs/adapter-static": "^3.0.10",
    "@sveltejs/vite-plugin-svelte": "^6.2.0",
    "svelte-check": "^4.3.2",
    "vite": "npm:rolldown-vite@latest",
    "rolldown": "^1.0.0-beta.58"
  }
}
```

**Create `pkg-store/svelte.config.js`:**
```javascript
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { getCounter, getCounterFomFile } from '../plugins.js';

const isBuild = process.argv.includes('build');
const componentMap = new Map();

if (isBuild) {
  getCounterFomFile();
}

export default {
  preprocess: vitePreprocess(),
  compilerOptions: {
    hmr: false,
    cssHash: ({ hash, css, name, filename }) => {
      if (isBuild) {
        const key = filename || hash(css);
        if (!componentMap.has(key)) {
          componentMap.set(key, getCounter());
        }
        return componentMap.get(key);
      }
      if (!filename) {
        return `svelte-${hash(css).substring(0, 8)}`;
      }
      const fileNamePart = filename.split(/[\\/]/).pop();
      if (!fileNamePart) {
        return `svelte-${hash(css).substring(0, 8)}`;
      }
      const componentName = fileNamePart
        .split('.')[0]
        .replace(/^\+/, '')
        .replace(/[^a-zA-Z0-9_-]/g, '_')
        .replace(/^[0-9]/, '_$&');
      const safeName = componentName || 'comp';
      return `${safeName}_${hash(css).substring(0, 8)}`;
    }
  },
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: false,
      strict: true
    }),
    paths: {
      base: '/store'
    },
    files: {
      assets: 'static',
      lib: 'lib',
      routes: 'routes',
      appTemplate: 'app.html'
    },
    alias: {
      '$ui': '../pkg-ui',
      '$store': './',
      '$routes': './routes',
      '$components': '../pkg-components',
      '$core': '../pkg-core',
      '$services': '../pkg-services',
      '$lib': './lib'
    },
    prerender: {
      handleHttpError: 'warn'
    }
  }
};
```

**Create `pkg-store/vite.config.ts`:**
```typescript
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import path from 'path';
import { svelteClassHasher, getCounter, getCounterFomFile } from '../plugins.js';

const isBuild = process.argv.includes('build');
const cssModuleMap = new Map();

if (isBuild) {
  getCounterFomFile();
}

export default defineConfig({
  root: path.resolve(__dirname),
  publicDir: './static',
  server: {
    port: 3571,
    fs: {
      strict: false,
      allow: [path.resolve(__dirname), path.resolve(__dirname, '..')]
    }
  },
  css: {
    modules: {
      generateScopedName: (name, filename, _css) => {
        if (isBuild) {
          const key = `${filename}:${name}`;
          if (!cssModuleMap.has(key)) {
            cssModuleMap.set(key, getCounter());
          }
          return cssModuleMap.get(key)!;
        }
        return `m-${name}_${Math.random().toString(36).substring(2, 6)}`;
      }
    }
  },
  build: {
    minify: false,
    cssMinify: false,
    rollupOptions: {
      output: {
        hashCharacters: 'base64',
        manualChunks: (id) => {
          if (id.includes('/components/') || id.includes('/lib/') || id.includes('/core/')) {
            return 'shared';
          }
          return 'vendor';
        }
      }
    }
  },
  plugins: [
    sveltekit(),
    isBuild && svelteClassHasher(),
    tailwindcss()
  ].filter(x => x)
});
```

**Create `pkg-store/app.html`:**
```html
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<link rel="icon" href="%sveltekit.assets%/favicon.ico" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		%sveltekit.head%
	</head>
	<body data-sveltekit-preload-data="hover">
		<div style="display: contents">%sveltekit.body%</div>
	</body>
</html>
```

**Create `pkg-store/tsconfig.json`:**
```json
{
  "extends": "../tsconfig.json",
  "compilerOptions": {
    "types": ["vite/client"]
  },
  "include": ["**/*.d.ts", "**/*.ts", "**/*.js", "**/*.svelte"]
}
```



#### Step 2.4: Run Intelligent Import Fixer
After moving files, use the automated `intelligent-import-fixer.ts` to resolve all import errors:

```bash
# Analyze and fix imports (dry run first)
bun run scripts/intelligent-import-fixer.ts --dry-run

# Review the report to ensure fixes are correct
cat import-fixer-report.txt

# Apply fixes
bun run scripts/intelligent-import-fixer.ts --fix
```

**What this script does:**
- Scans all files in the project
- Identifies broken imports after file moves
- Suggests and applies automatic fixes for:
  - Missing file extensions
  - Incorrect import paths
  - Type-only imports
  - Named vs default imports
- Provides detailed report of all changes

**Fix types applied:**
1. Updates relative import paths after file moves
2. Corrects package alias imports
3. Adds missing `.svelte` extensions
4. Converts between named and default imports
5. Resolves type import issues

#### Step 2.5: Verify Build After Import Fixes
```bash
# Try to build store app
cd pkg-store
bun run build

# If build fails, check errors
bun run check

# Review intelligent-import-fixer report
cat import-fixer-report.txt
```

**Expected outcome:**
- All import paths updated correctly
- No "Module not found" errors
- TypeScript errors resolved
- Build succeeds

#### Step 2.6: Iterative Fixing (If Needed)
If build still fails:

```bash
# 1. Run DAG analysis again to see new violations
bun run scripts/analyze-dag.ts > dag-analysis-after-move.txt

# 2. Compare with baseline
diff dag-analysis-before.txt dag-analysis-after-move.txt

# 3. Identify remaining issues
grep "HIERARCHY\|CIRCULAR" dag-analysis-after-move.txt

# 4. Run intelligent-import-fixer again if there are import errors
bun run scripts/intelligent-import-fixer.ts --fix

# 5. Repeat until build succeeds
```

**Goal**: Achieve stable dev build without DAG violations before proceeding.

### Phase 3: Resolve DAG Violations (Days 6-8)
**Goal**: Fix dependency graph issues identified by analyze-dag.ts

#### Step 3.1: Run Full DAG Analysis
```bash
bun run scripts/analyze-dag.ts > current-dag.txt
```

#### Step 3.2: Analyze Hierarchy Violations
Focus on violations that prevent the desired structure:

**Common violations and fixes:**

1. **pkg-ui importing from pkg-components**
   - Fix: Move the component to pkg-ui or pkg-core
   
2. **pkg-components importing from pkg-store**
   - Fix: This shouldn't exist after moving store to pkg-store
   
3. **pkg-core importing from any other package**
   - Fix: This is NEVER allowed - move the code to the dependent package

#### Step 3.3: Move Components to Fix DAG
Based on DAG analysis, move components to resolve violations:

```bash
# Example: pkg-ui depends on pkg-components/SomeUtility.svelte
mv pkg-components/SomeUtility.svelte pkg-core/lib/

# Example: pkg-components depends on pkg-ui/SomeButton.svelte  
mv pkg-ui/SomeButton.svelte pkg-components/lib/

# Example: pkg-store depends on routes (shouldn't happen)
# This means store still has import from old location
# Use intelligent-import-fixer to fix
bun run scripts/intelligent-import-fixer.ts --fix
```

#### Step 3.4: Re-run DAG Analysis After Each Move
```bash
# After each component move, verify DAG improved
bun run scripts/analyze-dag.ts

# Check if violations decreased
echo "Violations before: $(grep -c 'HIERARCHY\|CIRCULAR' previous-dag.txt)"
echo "Violations after: $(grep -c 'HIERARCHY\|CIRCULAR' current-dag.txt)"
```

#### Step 3.5: Use intelligent-import-fixer After Moves
After moving components, fix import paths:

```bash
bun run scripts/intelligent-import-fixer.ts --fix
```

#### Step 3.6: Verify No DAG Violations
Continue the process until:

```bash
bun run scripts/analyze-dag.ts
# Output: ✅ No violations detected!
```

**Success criteria:**
- Zero hierarchy violations
- Zero circular dependencies
- All packages follow allowed dependency rules
- Store app builds successfully

### Phase 4: Configure Main App (Day 9)
**Goal**: Update main app configuration for admin-only

#### Step 3.3: Update Store Layout
Check and update `pkg-store/routes/+layout.svelte`:
```svelte
<script>
  import "./store.css";
  import "$ui/libs/fontello-prerender.css";
  import blurhashScript from '$ui/libs/blurhash?raw';
  import { productosServiceState } from '$services/services/productos.svelte';
  
  let { children, data } = $props();

  productosServiceState.categorias = data.productos.categorias
  productosServiceState.productos = data.productos.productos
</script>

<svelte:head>
  {@html '<script>' + blurhashScript + '</script>'}
  {@html `<link rel="stylesheet" href="/store/libs/fontello-embedded.css">`}
</svelte:head>

{@render children()}
```

#### Step 3.4: Update Store Page
Update `pkg-store/routes/+page.svelte`:
```svelte
<script lang="ts">
import Header from '$components/Header.svelte';
import MainCarrusel from '$components/MainCarrusel.svelte';
import ProductCards from '$components/ProductCards.svelte';

let categorias = [
  { Name: "Perfumes", Image: "/store/images/categoria_1.webp" },
  { Name: "Casacas", Image: "/store/images/casaca_icon_sm3.webp" },
  { Name: "Zapatos", Image: "/store/images/categoria_1.webp" },
  { Name: "Relojes", Image: "/store/images/categoria_1.webp" },
  { Name: "Decoración", Image: "/store/images/categoria_1.webp" },
];
</script>

<Header />
<MainCarrusel {categorias} />
<div class="h-800 _1">
  <ProductCards />
</div>
```

### Phase 4: Configure Main App (Day 7)
**Goal**: Update main app configuration for admin-only

#### Step 4.1: Update Root package.json
```json
{
  "name": "genix-frontend",
  "version": "2.0.0",
  "private": true,
  "type": "module",
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
    "check": "svelte-check --tsconfig ./tsconfig.json"
  },
  "devDependencies": {
    "@sveltejs/adapter-static": "^3.0.10",
    "@sveltejs/kit": "^2.43.2",
    "@sveltejs/vite-plugin-svelte": "^6.2.0",
    "svelte": "^5.39.5",
    "svelte-check": "^4.3.2",
    "vite": "npm:rolldown-vite@latest",
    "rolldown": "^1.0.0-beta.58"
  }
}
```

#### Step 4.2: Update Main svelte.config.js
No major changes needed, just ensure it doesn't include store routes.

#### Step 4.3: Update Main Service Worker
Update `static/sw.js` to only cache admin routes:
```javascript
const CACHE_NAME = 'genix-admin-v1';

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll([
        '/',
        '/admin/',
        // Add admin-specific assets
      ]);
    })
  );
});

self.addEventListener('fetch', (event) => {
  // Skip /store routes (handled by store app)
  if (event.request.url.includes('/store/')) {
    return fetch(event.request);
  }
  
  event.respondWith(
    caches.match(event.request).then((response) => {
      return response || fetch(event.request);
    })
  );
});
```

### Phase 5: Single Development Server (Day 10)
**Goal**: Create unified development experience

#### Step 5.1: Install Proxy Dependencies
```bash
bun add -D http-proxy-middleware
```

#### Step 5.2: Create Proxy Server
```javascript
// scripts/proxy-server.js
import http from 'http';
import { createProxyMiddleware } from 'http-proxy-middleware';

const MAIN_PORT = 3570;
const STORE_PORT = 3571;
const PROXY_PORT = 3570; // Use same port as before

// Create proxy middleware for store
const storeProxy = createProxyMiddleware({
  target: `http://localhost:${STORE_PORT}`,
  changeOrigin: true,
  pathRewrite: {
    '^/store': '' // Remove /store prefix when proxying to store app
  },
  ws: true // Enable WebSocket proxying
});

// Create proxy middleware for main
const mainProxy = createProxyMiddleware({
  target: `http://localhost:${MAIN_PORT}`,
  changeOrigin: true,
  ws: true
});

// Create main server
const server = http.createServer((req, res) => {
  console.log(`${req.method} ${req.url} → ${req.url.startsWith('/store') ? 'store' : 'main'}`);
  
  if (req.url.startsWith('/store')) {
    storeProxy(req, res);
  } else {
    mainProxy(req, res);
  }
});

// Handle WebSocket upgrades
server.on('upgrade', (req, socket, head) => {
  if (req.url.startsWith('/store')) {
    storeProxy.upgrade(req, socket, head);
  } else {
    mainProxy.upgrade(req, socket, head);
  }
});

server.listen(PROXY_PORT, () => {
  console.log(`🚀 Development proxy server running on http://localhost:${PROXY_PORT}`);
  console.log(`📋 Main (Admin): http://localhost:${PROXY_PORT}`);
  console.log(`🛒 Store: http://localhost:${PROXY_PORT}/store`);
});
```

#### Step 5.3: Create Development Orchestrator
```javascript
// scripts/dev-all.js
import { spawn } from 'child_process';
import path from 'path';

const startApp = (app, command, cwd) => {
  return new Promise((resolve, reject) => {
    const server = spawn('bun', ['run', command], {
      cwd: path.resolve(process.cwd(), cwd),
      stdio: 'inherit',
      shell: true
    });

    server.on('error', reject);

    // Wait for server to be ready
    setTimeout(() => resolve(server), 5000);
  });
};

const main = async () => {
  console.log('🚀 Starting development environment...');

  // Start main app (without store routes)
  console.log('📋 Starting main app...');
  const mainApp = await startApp('main', 'dev:main', '.');

  // Wait a bit
  await new Promise(resolve => setTimeout(resolve, 3000));

  // Start store app
  console.log('🛒 Starting store app...');
  const storeApp = await startApp('store', 'dev', 'pkg-store');

  // Wait a bit
  await new Promise(resolve => setTimeout(resolve, 2000));

  // Start proxy server
  console.log('🔗 Starting proxy server...');
  const proxy = await startApp('proxy', 'node', 'scripts/proxy-server.js');

  console.log('✅ All services started successfully!');
  console.log('📋 Main (Admin): http://localhost:3570');
  console.log('🛒 Store: http://localhost:3570/store');
  console.log('\n💡 Tips:');
  console.log('   - Main app runs internally on port 3570');
  console.log('   - Store app runs internally on port 3571');
  console.log('   - Proxy server routes requests appropriately');
  console.log('   - Ctrl+C to stop all services');

  // Handle cleanup
  const cleanup = () => {
    console.log('\n🛑 Shutting down services...');
    mainApp.kill();
    storeApp.kill();
    proxy.kill();
    process.exit(0);
  };

  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);
};

main().catch((error) => {
  console.error('❌ Failed to start development environment:', error);
  process.exit(1);
});
```

#### Step 5.4: Test Development Environment
```bash
# Test single dev server
bun run dev

# Verify:
# - Admin works at http://localhost:3570
# - Store works at http://localhost:3570/store
# - Hot module replacement works for both
# - No CORS errors
```

### Phase 6: Build & Deployment (Day 11)
**Goal**: Set up production build pipeline

#### Step 6.1: Create Combined Build Script
```javascript
// scripts/build-all.js
import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const BUILD_DIR = 'build';

console.log('🚀 Starting build process...');

// Clean previous builds
if (fs.existsSync(BUILD_DIR)) {
  fs.rmSync(BUILD_DIR, { recursive: true });
}
fs.mkdirSync(BUILD_DIR, { recursive: true });

// Build main app
console.log('📦 Building main app...');
execSync('bun run build:main', { stdio: 'inherit' });

// Copy main build
console.log('📋 Copying main build...');
const copyDirectory = (src, dest) => {
  if (!fs.existsSync(src)) {
    console.warn(`⚠️  Source directory ${src} does not exist`);
    return;
  }
  
  fs.mkdirSync(dest, { recursive: true });
  const entries = fs.readdirSync(src, { withFileTypes: true });

  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);

    if (entry.isDirectory()) {
      copyDirectory(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
    }
  }
};

copyDirectory('build', BUILD_DIR);

// Remove the temporary build directory
fs.rmSync('build', { recursive: true });

// Build store app
console.log('📦 Building store app...');
execSync('bun run build:store', { stdio: 'inherit' });

// Copy store build to /store/ subdirectory
console.log('📋 Copying store build...');
copyDirectory('pkg-store/build', path.join(BUILD_DIR, 'store'));

// Remove the temporary store build directory
fs.rmSync('pkg-store/build', { recursive: true });

// Copy static files
console.log('📋 Copying static files...');
copyDirectory('static', BUILD_DIR);

// Create 404.html for SPA routing
console.log('📋 Creating 404.html...');
const indexPath = path.join(BUILD_DIR, 'index.html');
const notFoundPath = path.join(BUILD_DIR, '404.html');

if (fs.existsSync(indexPath)) {
  fs.copyFileSync(indexPath, notFoundPath);
  console.log('✅ Created 404.html from index.html');
} else {
  console.warn('⚠️  index.html not found, skipping 404.html creation');
}

console.log('✅ Build completed successfully!');
console.log(`📁 Output directory: ${BUILD_DIR}`);
```

#### Step 6.2: Update Post-Build Script
Update existing `postbuild.js` to work with new structure:
```javascript
// postbuild.js
import fs from 'fs';
import path from 'path';

const BUILD_DIR = 'build';
const DOCS_DIR = path.join('..', 'docs');

const publishToDocs = () => {
  console.log('--- Starting publish to docs folder ---');
  
  try {
    fs.accessSync(BUILD_DIR);
  } catch {
    console.error(`❌ Build directory '${BUILD_DIR}' not found. Please build the project first.`);
    return;
  }

  try {
    // Clean docs directory
    if (fs.existsSync(DOCS_DIR)) {
      const entries = fs.readdirSync(DOCS_DIR, { withFileTypes: true });
      for (const entry of entries) {
        if (entry.isDirectory()) {
          fs.rmSync(path.join(DOCS_DIR, entry.name), { recursive: true, force: true });
          console.log(`🗑️  Removed directory '${entry.name}' from '${DOCS_DIR}'`);
        }
      }
    } else {
      fs.mkdirSync(DOCS_DIR, { recursive: true });
      console.log(`📁 Created '${DOCS_DIR}' directory`);
    }

    // Copy build folder
    const copyDirectory = (src, dest) => {
      fs.mkdirSync(dest, { recursive: true });
      const entries = fs.readdirSync(src, { withFileTypes: true });
      for (const entry of entries) {
        const srcPath = path.join(src, entry.name);
        const destPath = path.join(dest, entry.name);
        if (entry.isDirectory()) {
          copyDirectory(srcPath, destPath);
        } else {
          fs.copyFileSync(srcPath, destPath);
        }
      }
    };

    copyDirectory(BUILD_DIR, DOCS_DIR);
    console.log(`✅ Copied contents from '${BUILD_DIR}' to '${DOCS_DIR}'`);

    // Create 404.html
    const indexPath = path.join(DOCS_DIR, 'index.html');
    const notFoundPath = path.join(DOCS_DIR, '404.html');
    
    if (fs.existsSync(indexPath)) {
      fs.copyFileSync(indexPath, notFoundPath);
      console.log(`✅ Created 404.html from index.html`);
    }

    console.log('--- Publish to docs folder completed ---');
  } catch (error) {
    console.error('❌ An error occurred during publishing:', error);
  }
};

const args = process.argv.slice(2);
if (args.includes('--publish')) {
  publishToDocs();
}
```

#### Step 6.3: Test Build Pipeline
```bash
# Test combined build
bun run build

# Verify output structure
ls -lh build/

# Test post-build
bun run build --publish

# Verify docs folder
ls -lh ../docs/
```

### Phase 7: Testing & Validation (Days 12-14)
**Goal**: Comprehensive testing of both apps

#### Step 7.1: Test Main App
```bash
# Run main app in isolation
bun run dev:main

# Test:
# - Admin routes work
# - Admin features work
# - No store routes accessible
```

#### Step 7.2: Test Store App
```bash
# Run store app in isolation
bun run dev:store

# Test:
# - Store routes work at /store/
# - Store features work
# - Store-specific static files load
```

#### Step 7.3: Test Combined Dev
```bash
# Test combined dev
bun run dev

# Test:
# - Admin at http://localhost:3570 works
# - Store at http://localhost:3570/store works
# - Navigation between apps works
# - Hot reload works for both
# - No CORS errors
# - Service workers register correctly
```

#### Step 7.4: Test Production Build
```bash
# Test production build
bun run build

# Preview build
bun run preview

# Test:
# - All routes work in production
# - Static files load correctly
# - No 404 errors
# - Service workers cache correctly
```

### Phase 8: Cleanup (Day 15)
**Goal**: Remove old structure

#### Step 8.1: Final DAG Verification
Before cleanup, verify one last time:
```bash
# Run final DAG analysis
bun run scripts/analyze-dag.ts > final-dag.txt

# Confirm no violations
grep "HIERARCHY\|CIRCULAR" final-dag.txt
# Should return nothing

# Compare with initial analysis
diff dag-analysis-before.txt final-dag.txt
```

#### Step 8.2: Remove Old Store Routes
```bash
# Only do this after confirming everything works!
rm -rf routes/store/
```

#### Step 8.3: Update Documentation
- Update README.md with new structure
- Document development workflow
- Document build process
- Update deployment instructions

## 🚀 Single Development Server Architecture

### Development Server Setup

```
┌─────────────────────────────────────────────────────────────┐
│                 Browser                                     │
│  http://localhost:3570 (main/admin)                        │
│  http://localhost:3570/store (store)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Proxy Server (port 3570)                       │
│         scripts/proxy-server.js                             │
│  Routes: / → Main, /store → Store                          │
└────────┬──────────────────────┬──────────────────────────────┘
         │                      │
         ▼                      ▼
┌──────────────────┐    ┌──────────────────┐
│ Main App         │    │ Store App       │
│ Vite Dev Server  │    │ Vite Dev Server  │
│ Port 3570        │    │ Port 3571        │
└──────────────────┘    └──────────────────┘
```

### Benefits of This Approach
✅ **Minimal Disruption**: Only move store routes to pkg-store  
✅ **Preserved Structure**: All pkg-* directories stay in place  
✅ **Unified URL**: Both apps accessible from `localhost:3570`  
✅ **No CORS**: Same origin for all requests  
✅ **Shared Cookies**: Session management works seamlessly  
✅ **Clear Separation**: Store is truly independent  
✅ **Easy Rollback**: Just move routes back if it fails

## 📊 Success Criteria

### Technical Metrics
- ✅ Both apps build independently
- ✅ Combined build succeeds
- ✅ Build time ≤ 1.5x original build time
- ✅ Bundle sizes within 10% of original
- ✅ All existing routes work
- ✅ Service workers function correctly
- ✅ Single dev server works
- ✅ **analyze-dag.ts shows zero violations**
- ✅ **intelligent-import-fixer.ts runs without errors**
- ✅ **No manual import fixes needed**

### Business Metrics
- ✅ Zero downtime during migration
- ✅ No user-visible bugs
- ✅ Performance maintained or improved
- ✅ Development velocity maintained

## ⚠️ Risk Assessment

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| Import path errors after move | High | High | Automated fix with intelligent-import-fixer.ts |
| DAG violations not fully resolved | High | Medium | Iterative analysis with analyze-dag.ts until clean |
| Build failures after moves | High | Medium | Test builds after each phase, use automated tools |
| intelligent-import-fixer makes mistakes | Medium | Medium | Dry run first, review report, verify build |
| Dev server proxy issues | Medium | Low | Test proxy thoroughly, have fallback to individual servers |
| Service worker conflicts | High | Low | Separate service workers with unique scopes |
| Asset loading issues | Medium | Low | Test static files, verify paths |
| Deployment issues | High | Low | Test deployment on staging, have rollback plan |

### Additional Mitigation Strategies

**For DAG Issues:**
- Use analyze-dag.ts iteratively - run after each change
- Document each move and its effect on violations
- If stuck, consider creating new shared packages instead of moving
- Keep detailed log of all changes made

**For Import Issues:**
- Always run intelligent-import-fixer.ts with --dry-run first
- Review the generated report carefully
- Apply fixes in small batches, verify each batch
- Keep backup before applying fixes
- Test build after each batch of fixes

**For Build Stability:**
- Never make multiple changes without testing build
- Use incremental approach: one change → test → next change
- Maintain working state at all times
- Use git to commit after each successful phase

## 🔄 Rollback Strategy

### Pre-Migration Backup
```bash
git checkout -b pre-store-migration-backup
git tag v-before-store-migration
cp -r frontend/ frontend-backup/
```

### Incremental Backups
After each major phase, create checkpoints:

```bash
# After Phase 1 (DAG analysis)
git commit -am "Phase 1: Initial DAG analysis complete"
git tag phase-1-dag-analysis

# After Phase 2 (Files moved)
git commit -am "Phase 2: Files moved to pkg-store"
git tag phase-2-files-moved

# After Phase 3 (DAG resolved)
git commit -am "Phase 3: DAG violations resolved"
git tag phase-3-dag-clean

# After Phase 7 (Testing complete)
git commit -am "Phase 7: All tests passing"
git tag phase-7-tested
```

### Rollback Procedure

#### Rollback to Specific Phase
```bash
# Rollback to after Phase 1
git checkout phase-1-dag-analysis

# Rollback to after Phase 2
git checkout phase-2-files-moved

# Rollback to before any changes
git checkout v-before-store-migration
```

#### Full Rollback Procedure
1. Stop all services
2. Identify last known good state: `git log --oneline --tags`
3. Revert to that state: `git checkout <tag-name>`
4. Restore from backup if needed: `cp -r frontend-backup/* frontend/`
5. Deploy previous version
6. Investigate issues
7. Schedule new migration attempt with lessons learned

### Rollback Triggers
- analyze-dag.ts shows violations after multiple attempts
- intelligent-import-fixer.ts causes build failures
- Build fails consistently for > 2 hours despite fixes
- DAG violations cannot be resolved
- Critical bugs in production
- Performance degradation > 30%
- Security vulnerabilities
- Store app doesn't work after migration

### When to Rollback vs. Iterate
**Rollback if:**
- Cannot achieve clean DAG after 5 attempts
- intelligent-import-fixer.ts makes breaking changes
- Import errors cannot be resolved
- Build fails catastrophically

**Iterate if:**
- Minor import path errors (fixable manually)
- Small number of DAG violations (< 5)
- Build fails with clear fixable errors
- Type errors only (not structural issues)

## 🎯 Conclusion

This simplified migration plan provides a practical approach that:
- ✅ Preserves current directory structure
- ✅ Only moves store routes into pkg-store
- ✅ Uses a single development server for unified experience
- ✅ Maintains all existing functionality
- ✅ Provides clear rollback path
- ✅ Minimizes risk and disruption
- ✅ **Uses automated tools (analyze-dag.ts, intelligent-import-fixer.ts)**
- ✅ **Achieves clean DAG through iterative analysis**
- ✅ **Reduces manual error-prone work**

## 🛠️ Key Tools & Their Usage

### 1. analyze-dag.ts
**Purpose**: Understand and validate dependency graph

**When to run:**
- Before migration (baseline)
- After moving files
- After each component move
- Before final cleanup
- Whenever build fails due to dependencies

**Expected output:**
- Dependency graph visualization
- Hierarchy violations (e.g., pkg-ui importing pkg-components)
- Circular dependencies
- Actionable recommendations

**Example workflow:**
```bash
# Initial analysis
bun run scripts/analyze-dag.ts > baseline.txt

# Make changes

# Verify improvement
bun run scripts/analyze-dag.ts > after.txt
diff baseline.txt after.txt
```

### 2. intelligent-import-fixer.ts
**Purpose**: Automatically fix import errors after file moves

**When to run:**
- After moving files (routes/store → pkg-store/routes)
- After moving components to resolve DAG
- Anytime import paths break
- Before attempting build

**What it fixes:**
- Missing file extensions
- Incorrect relative paths
- Package alias imports
- Named vs default imports
- Type-only imports

**Example workflow:**
```bash
# First, dry run to see what will be fixed
bun run scripts/intelligent-import-fixer.ts --dry-run

# Review the report
cat import-fixer-report.txt

# If looks good, apply fixes
bun run scripts/intelligent-import-fixer.ts --fix

# Verify build
cd pkg-store && bun run build
```

## 📋 Workflow Summary

```
Phase 1: DAG Analysis (Days 1-2)
  ├─ analyze-dag.ts → understand current state
  ├─ Document violations
  └─ Plan component moves

Phase 2: Move Files & Fix Imports (Days 3-5)
  ├─ Move routes/store → pkg-store/routes
  ├─ intelligent-import-fixer.ts --fix
  ├─ Test build
  └─ Iterate until build succeeds

Phase 3: Resolve DAG Violations (Days 6-8)
  ├─ analyze-dag.ts (after moves)
  ├─ Move components to fix violations
  ├─ intelligent-import-fixer.ts --fix
  ├─ analyze-dag.ts (verify)
  └─ Repeat until zero violations

Phase 4-8: Configuration, Testing, Cleanup (Days 9-15)
  └─ Standard development workflow
```

The key advantage of this approach is its use of automated tools to handle the complex and error-prone work of managing dependencies and imports, allowing the team to focus on business logic rather than manual file path updates.

---

**Document Version**: 4.0 - Simplified  
**Last Updated**: 2025-01-18  
**Author**: Engineering Team  
**Status**: Ready for Implementation