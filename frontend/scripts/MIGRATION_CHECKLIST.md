# Store Migration - Remaining Work Checklist

**Created:** January 25, 2025  
**Migration Plan:** `scripts/STORE_MIGRATION_PLAN.md`  
**Status Report:** `scripts/MIGRATION_STATUS.md`  

---

## 🎯 Overview

**Completed:** Phases 1-6 (✅ 75% complete)  
**Remaining:** Phases 7-8 (❌ 25% remaining)  

This checklist provides actionable steps to complete the migration through testing, validation, and cleanup.

---

## 📋 Phase 7: Testing & Validation (Days 12-14)

### 7.1 Test Main App (In Isolation)

**Objective:** Verify main admin app works correctly without store routes

**Run Command:**
```bash
bun run dev:main
```

**Expected Port:** 3570

**Test Checklist:**
- [ ] Admin homepage loads at `http://localhost:3570`
- [ ] Admin login page works
- [ ] Admin dashboard loads
- [ ] Admin navigation works
- [ ] Admin features work (add, edit, delete)
- [ ] Service worker registers (check DevTools)
- [ ] No store routes accessible (404 on `/store`)
- [ ] No console errors
- [ ] Hot module replacement works
- [ ] Static files load (fonts, images, icons)

**Success Criteria:**
- All admin routes load correctly
- No broken functionality
- Performance feels snappy

**Troubleshooting:**
- If build fails: Run `bun run build:main` and check errors
- If routes 404: Check `routes/` directory structure
- If service worker fails: Check browser DevTools Console

---

### 7.2 Test Store App (In Isolation)

**Objective:** Verify store app works correctly as standalone app

**Run Command:**
```bash
cd pkg-store
bun run dev
```

**Expected Port:** 3571

**Test Checklist:**
- [ ] Store homepage loads at `http://localhost:3571`
- [ ] Store layout loads correctly
- [ ] Header component displays
- [ ] Product carousel works
- [ ] Product cards display
- [ ] Category images load
- [ ] Mobile menu works
- [ ] Blurhash images load
- [ ] Service worker registers
- [ ] No console errors
- [ ] Hot module replacement works

**Test Routes:**
```bash
# Test main route
curl http://localhost:3571/store/ | grep -o "<title>.*</title>"

# Check for 404s (should return index.html for SPA)
curl -I http://localhost:3571/store/ | grep "200 OK"
```

**Success Criteria:**
- All store pages load correctly
- All images and assets load
- Interactive features work
- Performance feels smooth

**Troubleshooting:**
- If images 404: Check `pkg-store/static/` directory
- If styles broken: Check Tailwind CSS is loading
- If HMR fails: Restart dev server

---

### 7.3 Test Combined Development Environment

**Objective:** Verify unified development experience works

**Run Command:**
```bash
bun run dev
```

**Expected Proxy Port:** 3572

**URLs to Test:**
- Main (Admin): `http://localhost:3572`
- Store: `http://localhost:3572/store`

**Test Checklist:**

**Admin Functionality:**
- [ ] Admin loads at `http://localhost:3572`
- [ ] All admin features work
- [ ] No CORS errors in console
- [ ] Service worker registers
- [ ] Admin navigation works

**Store Functionality:**
- [ ] Store loads at `http://localhost:3572/store`
- [ ] Store assets load from `/store/static/`
- [ ] No CORS errors in console
- [ ] Store service worker registers
- [ ] Store navigation works

**Integration:**
- [ ] Can navigate between apps
- [ ] Links from admin work
- [ ] Links from store work
- [ ] Shared state works (if applicable)
- [ ] Hot module replacement works for both
- [ ] WebSocket connections work (if applicable)

**CORS Testing:**
```bash
# Check for CORS errors in browser DevTools Console
# Open: http://localhost:3572
# Open: http://localhost:3572/store
# Look for red errors in Console tab
```

**Success Criteria:**
- Both apps accessible through single port
- No CORS errors
- Navigation between apps seamless
- HMR works independently for each app

**Troubleshooting:**
- If proxy fails: Check `scripts/proxy-server.js` is running
- If CORS errors: Check proxy configuration and origins
- If one app not accessible: Check that app's dev server is running

**Expected Logs:**
```
🚀 Starting development environment...
📋 Starting main app...
🛒 Starting store app...
🔗 Starting proxy server...
✅ All services started successfully!
📋 Main (Admin): http://localhost:3572
🛒 Store: http://localhost:3572/store
```

---

### 7.4 Test Production Build

**Objective:** Verify production build works correctly

**Build Commands:**
```bash
# Build both apps
bun run build

# Preview production build
bun run preview
```

**Expected Preview Port:** 3000 (or check your config)

**Test Checklist:**

**Main App (Production):**
- [ ] Admin loads at `http://localhost:3000`
- [ ] All admin features work
- [ ] Admin service worker registers
- [ ] Assets are minified (check Network tab)
- [ ] No console errors
- [ ] Performance is good (LCP < 2s)

**Store App (Production):**
- [ ] Store loads at `http://localhost:3000/store`
- [ ] Store service worker registers
- [ ] Store assets load from `/store/`
- [ ] Assets are minified
- [ ] No console errors
- [ ] Performance is good (LCP < 2s)

**Build Output Verification:**
```bash
# Check build structure
ls -lh build/
ls -lh build/store/

# Verify 404.html exists
test -f build/404.html && echo "✅ 404.html exists" || echo "❌ 404.html missing"

# Verify store index.html exists
test -f build/store/index.html && echo "✅ Store index.html exists" || echo "❌ Store index.html missing"
```

**Performance Testing:**
```bash
# Use Chrome DevTools Lighthouse
# Open: http://localhost:3000
# Lighthouse > Performance > Generate Report
# Check: LCP < 2s, FID < 100ms, CLS < 0.1
```

**Success Criteria:**
- All routes work in production
- Assets properly minified and optimized
- Performance meets targets (LCP < 2s)
- No console errors
- Service workers register correctly
- Build size is reasonable

**Troubleshooting:**
- If build fails: Check logs for errors
- If assets 404: Check build output structure
- If performance is poor: Check bundle sizes and optimize
- If service worker fails: Check sw.js registration

**Expected Build Output:**
```
✅ Build completed successfully!
📁 Output directory: build

File sizes:
- build/_app/immutable/assets/*.css: ~30-50KB
- build/_app/immutable/chunks/*.js: ~300-400KB total
- build/store/_app/immutable/assets/*.css: ~30-50KB
- build/store/_app/immutable/chunks/*.js: ~350KB total
```

---

## 🧹 Phase 8: Cleanup (Day 15)

### 8.1 Final DAG Verification

**Objective:** Confirm zero violations remain

**Run Command:**
```bash
bun run scripts/analyze-dag.ts > final-dag-analysis.txt
```

**Check Checklist:**
- [ ] Zero hierarchy violations
- [ ] Zero circular dependencies
- [ ] All packages follow allowed dependency rules
- [ ] Store app at correct level (Level 4)
- [ ] Main app at correct level (Level 5)

**Verify Clean Graph:**
```bash
# Check for violations
grep -i "violation\|circular\|hierarchy" final-dag-analysis.txt

# Should return nothing if clean
```

**Success Criteria:**
- Output shows "No violations detected!"
- Dependency graph matches expected structure

**Expected Output:**
```
✅ No violations detected! The dependency graph is valid.
```

---

### 8.2 Remove Old Store Routes

**Objective:** Clean up any remaining old files

**Check and Remove:**
```bash
# Check if old store routes exist
ls -la routes/store/ 2>/dev/null && echo "⚠️ Old store routes exist" || echo "✅ No old store routes"

# If they exist, remove them
rm -rf routes/store/ && echo "✅ Old store routes removed"

# Verify removal
ls routes/ | grep -i store && echo "❌ Still exists" || echo "✅ Clean"
```

**Check for Old Imports:**
```bash
# Search for old store imports in main app
grep -r "from.*routes/store" routes/ --include="*.svelte" --include="*.ts" --include="*.js" && echo "⚠️ Found old imports" || echo "✅ No old imports"

# Search for old store imports in components
grep -r "from.*routes/store" pkg-components/ pkg-ui/ --include="*.svelte" --include="*.ts" --include="*.js" && echo "⚠️ Found old imports" || echo "✅ No old imports"
```

**Success Criteria:**
- No `routes/store/` directory exists
- No imports from old location in any files

---

### 8.3 Update Documentation

**Objective:** Document new architecture for team

#### 8.3.1 Update README.md

**Add New Section:**
```markdown
## Project Structure

This project uses a multi-app architecture with separate admin and store applications.

### Applications

- **Main App (Admin):** Admin interface for managing products and orders
  - Dev: `bun run dev:main` (port 3570)
  - Build: `bun run build:main`
  
- **Store App:** Public-facing e-commerce store
  - Dev: `bun run dev:store` (port 3571)
  - Build: `bun run build:store`
  - Path: `/store`

### Combined Development

For unified development experience:
```bash
bun run dev
```

This starts both apps and a proxy server on port 3572:
- Admin: `http://localhost:3572`
- Store: `http://localhost:3572/store`

### Package Structure

```
pkg-core/          # Base utilities (Level 0)
pkg-services/      # API services (Level 1)
pkg-components/    # Shared components (Level 2)
pkg-ui/           # UI components (Level 3)
pkg-store/        # Store application (Level 4)
routes/           # Admin routes (Level 5)
```

### Build & Deploy

```bash
# Build both apps
bun run build

# Preview production build
bun run preview

# Deploy to docs
bun run publish
```
```

**Update Existing Sections:**
- [ ] Update installation instructions
- [ ] Update development setup instructions
- [ ] Update deployment instructions
- [ ] Add troubleshooting section

---

#### 8.3.2 Create Architecture Diagram

**Create File:** `docs/ARCHITECTURE.md`

```markdown
# System Architecture

## Application Overview

The Genix Frontend consists of two independent SvelteKit applications:

### 1. Admin Application
- **Purpose:** Content management and administration
- **Access:** Protected, authenticated users only
- **Route:** `/` (root)
- **Port:** 3570 (development), 3572 (via proxy)

### 2. Store Application
- **Purpose:** Public-facing e-commerce store
- **Access:** Public, no authentication required
- **Route:** `/store`
- **Port:** 3571 (development), 3572 (via proxy)

## Package Dependency Hierarchy

```
Level 0: pkg-core
├─ Level 1: pkg-services
│  ├─ Level 2: pkg-components
│  │  └─ Level 3: pkg-ui
│  │     └─ Level 4: pkg-store
│  └─ Level 3: pkg-ui
│     └─ Level 4: pkg-store
├─ Level 2: pkg-components
│  └─ Level 3: pkg-ui
│     └─ Level 4: pkg-store
└─ Level 3: pkg-ui
   └─ Level 4: pkg-store

Level 5: routes (Admin)
├─ Level 3: pkg-ui
├─ Level 2: pkg-components
├─ Level 1: pkg-services
└─ Level 0: pkg-core
```

## Development Workflow

### Local Development

```bash
# Start both apps with proxy
bun run dev

# Start only admin app
bun run dev:main

# Start only store app
bun run dev:store
```

### Proxy Configuration

The proxy server (`scripts/proxy-server.js`) routes requests:
- `/store/*` → Store app (port 3571)
- `/*` → Admin app (port 3570)

### Build Process

```bash
# Build both apps
bun run build

# Output structure:
build/
├── _app/           # Admin app bundles
├── index.html      # Admin entry
├── 404.html       # SPA fallback
├── sw.js          # Admin service worker
└── store/         # Store app (subdirectory)
    ├── _app/      # Store bundles
    ├── index.html # Store entry
    └── sw.js      # Store service worker
```

## Service Worker Strategy

### Admin Service Worker
- **File:** `static/sw.js`
- **Scope:** `/` (root)
- **Behavior:** Skips `/store/*` routes
- **Cache:** Admin assets only

### Store Service Worker
- **File:** `pkg-store/static/sw.js`
- **Scope:** `/store/`
- **Behavior:** Caches store assets only
- **Independence:** Separate from admin service worker

## State Management

### Admin Application
- Uses Svelte 5 stores (`$state`, `$derived`)
- Service worker for API caching
- Shared state via pkg-core

### Store Application
- Uses Svelte 5 stores (`$state`, `$derived`)
- Separate service worker
- Independent state management

## Static Assets

### Admin Assets
- **Location:** `static/`
- **Build Output:** `build/`
- **Access:** `/files/*`, `/fonts/*`, etc.

### Store Assets
- **Location:** `pkg-store/static/`
- **Build Output:** `build/store/`
- **Access:** `/store/files/*`, `/store/fonts/*`, etc.

## Deployment Strategy

### Production Deployment

1. Build both apps: `bun run build`
2. Deploy `build/` directory to server
3. Configure server for SPA routing
4. Both apps accessible via same domain

### CDN Configuration

- Admin assets: `https://domain.com/_app/`
- Store assets: `https://domain.com/store/_app/`

### Caching Strategy

- Admin: Long-term cache for assets, short for HTML
- Store: Long-term cache for assets, short for HTML
- Service workers: Version-specific cache names

## Monitoring

### Key Metrics
- LCP (Largest Contentful Paint): < 2s
- FID (First Input Delay): < 100ms
- CLS (Cumulative Layout Shift): < 0.1

### Error Tracking
- Sentry integration for error monitoring
- Custom error logging in service workers
- Performance metrics collection
```

---

#### 8.3.3 Create Troubleshooting Guide

**Create File:** `docs/TROUBLESHOOTING.md`

```markdown
# Troubleshooting Guide

## Development Issues

### Dev Server Won't Start

**Symptom:** `bun run dev` fails or hangs

**Solutions:**
1. Check port conflicts:
```bash
lsof -i :3570  # Admin
lsof -i :3571  # Store
lsof -i :3572  # Proxy
```

2. Kill existing processes:
```bash
pkill -f "vite dev"
pkill -f "proxy-server"
```

3. Clear cache:
```bash
rm -rf node_modules/.vite
rm -rf pkg-store/node_modules/.vite
```

### Proxy Server Not Routing

**Symptom:** Only one app accessible through proxy

**Solutions:**
1. Check proxy server is running:
```bash
ps aux | grep proxy-server
```

2. Check proxy logs for errors
3. Verify port configuration in `scripts/proxy-server.js`
4. Test individual apps first:
```bash
bun run dev:main   # Test on 3570
bun run dev:store  # Test on 3571
```

### Hot Module Replacement Not Working

**Symptom:** Changes don't reflect in browser without refresh

**Solutions:**
1. Check WebSocket connections in DevTools
2. Verify Vite HMR configuration
3. Restart dev server
4. Check browser console for HMR errors

### CORS Errors

**Symptom:** CORS errors in browser console

**Solutions:**
1. Check proxy configuration in `scripts/proxy-server.js`
2. Verify `changeOrigin: true` is set
3. Check for same-origin policy violations
4. Ensure both apps use correct origins

## Build Issues

### Build Fails

**Symptom:** `bun run build` fails with errors

**Solutions:**
1. Check for TypeScript errors:
```bash
bun run check
```

2. Check for missing dependencies:
```bash
bun install
cd pkg-store && bun install
```

3. Clear build cache:
```bash
rm -rf .svelte-kit
rm -rf pkg-store/.svelte-kit
rm -rf build
```

4. Check import paths:
```bash
bun run scripts/intelligent-import-fixer.ts --dry-run
```

### Build Produces Empty Output

**Symptom:** `build/` directory exists but mostly empty

**Solutions:**
1. Check adapter configuration in `svelte.config.js`
2. Verify `adapter-static` is installed
3. Check build logs for warnings
4. Ensure prerendering is configured correctly

### Service Worker Issues

**Symptom:** Service worker doesn't register or cache

**Solutions:**

**Admin Service Worker:**
```bash
# Check sw.js exists
test -f build/sw.js || echo "Missing admin sw.js"

# Check sw.js is accessible
curl -I http://localhost:3000/sw.js
```

**Store Service Worker:**
```bash
# Check sw.js exists
test -f build/store/sw.js || echo "Missing store sw.js"

# Check sw.js is accessible
curl -I http://localhost:3000/store/sw.js
```

**Debug Service Worker:**
1. Open Chrome DevTools > Application > Service Workers
2. Check for errors in console
3. Unregister and re-register service worker
4. Clear site data and refresh

### Static Asset 404s

**Symptom:** Images, fonts, or other assets not loading

**Solutions:**

**Admin Assets:**
```bash
# Check assets in build
ls -lh build/static/
ls -lh build/_app/immutable/assets/

# Verify file exists
test -f build/images/example.jpg || echo "Missing asset"
```

**Store Assets:**
```bash
# Check assets in store build
ls -lh build/store/static/
ls -lh build/store/_app/immutable/assets/

# Verify file exists
test -f build/store/images/example.jpg || echo "Missing asset"
```

**Fix Asset Paths:**
1. Check import paths in components
2. Verify assets are in `pkg-store/static/` directory
3. Check build output structure
4. Use correct asset paths (`/store/images/...`)

## Production Issues

### Build Works in Dev, Fails in Production

**Symptom:** Local dev works, production build has issues

**Solutions:**
1. Test production build locally:
```bash
bun run build
bun run preview
```

2. Check environment variables
3. Verify API endpoints are correct
4. Check for hardcoded localhost URLs
5. Review browser console for errors

### Performance Issues

**Symptom:** Pages load slowly, poor LCP scores

**Solutions:**
1. Run Lighthouse audit
2. Check bundle sizes
3. Optimize images and assets
4. Implement lazy loading
5. Check for blocking JavaScript
6. Review server response times

### 404 Errors on Refresh

**Symptom:** Direct URLs 404 on page refresh

**Solutions:**
1. Check 404.html exists:
```bash
test -f build/404.html && echo "✅ Exists" || echo "❌ Missing"
```

2. Verify server SPA routing configuration
3. Check Nginx/Apache configuration
4. Ensure fallback to index.html works

## Deployment Issues

### Build Deployment Fails

**Symptom:** Deployment process fails

**Solutions:**
1. Verify build output locally:
```bash
bun run build
ls -lh build/
```

2. Check deployment logs for errors
3. Verify server permissions
4. Ensure enough disk space
5. Check network connectivity

### Mixed Content Warnings

**Symptom:** Browser blocks insecure content

**Solutions:**
1. Ensure HTTPS is enabled
2. Check all asset URLs use HTTPS
3. Update API endpoints to use HTTPS
4. Verify CDN URLs use HTTPS

## State Management Issues

### State Not Updating

**Symptom:** Changes to state don't reflect in UI

**Solutions:**
1. Check Svelte 5 runes usage (`$state`, `$derived`)
2. Verify component reactivity
3. Check for state mutations outside of runes
4. Use DevTools to inspect state

### Shared State Issues

**Symptom:** State not shared between components

**Solutions:**
1. Verify state is exported from store file
2. Check import paths are correct
3. Ensure state is initialized once
4. Check for multiple store instances

## Getting Help

### Log Files
- Dev server logs: Terminal output
- Build logs: Terminal output
- Browser logs: DevTools Console
- Service worker logs: DevTools > Application > Service Workers

### Debug Commands
```bash
# Check for TypeScript errors
bun run check

# Check DAG violations
bun run scripts/analyze-dag.ts

# Check for import issues
bun run scripts/intelligent-import-fixer.ts --dry-run

# Clean build artifacts
rm -rf .svelte-kit pkg-store/.svelte-kit build pkg-store/build

# Reinstall dependencies
rm -rf node_modules pkg-store/node_modules
bun install
cd pkg-store && bun install
```

### Useful Resources
- SvelteKit Docs: https://kit.svelte.dev/docs
- Svelte 5 Docs: https://svelte.dev/docs/svelte-v5-migration-guide
- Vite Docs: https://vitejs.dev/guide/
- Migration Plan: `scripts/STORE_MIGRATION_PLAN.md`
- Status Report: `scripts/MIGRATION_STATUS.md`
```

---

#### 8.3.4 Update Team Onboarding

**Add to Onboarding Docs:**

```markdown
## Development Environment Setup

### Project Structure

This is a multi-app SvelteKit project with two main applications:

1. **Admin Application** (`/`) - Content management
2. **Store Application** (`/store`) - Public e-commerce store

### Quick Start

```bash
# Install dependencies
bun install

# Start development server (both apps)
bun run dev

# Open in browser
# Admin: http://localhost:3572
# Store: http://localhost:3572/store
```

### Individual App Development

```bash
# Admin only
bun run dev:main

# Store only
cd pkg-store && bun run dev
```

### Building

```bash
# Build both apps
bun run build

# Preview production build
bun run preview

# Deploy to docs
bun run publish
```

### Key Directories

- `pkg-core/` - Shared utilities and helpers
- `pkg-services/` - API service layer
- `pkg-components/` - Shared UI components
- `pkg-ui/` - UI component library
- `pkg-store/` - Store application
- `routes/` - Admin application routes

### Development Tips

1. Use `bun run dev` for unified development
2. Check port 3572 for both apps
3. Hot module replacement works independently for each app
4. Service workers are isolated per app
5. Static assets are separated per app

### Common Issues

- **Port conflicts:** Check ports 3570, 3571, 3572
- **CORS errors:** Use `bun run dev` (proxy server)
- **Build fails:** Run `bun run check` first
- **HMR not working:** Restart dev server

### Getting Help

- See `docs/TROUBLESHOOTING.md` for common issues
- Check `scripts/STORE_MIGRATION_PLAN.md` for architecture
- Review `scripts/MIGRATION_STATUS.md` for current state
```

---

## 🚀 Pre-Deployment Checklist

### Before deploying to production:

- [ ] All tests in Phase 7 passed
- [ ] Both apps work in isolation
- [ ] Combined dev environment works
- [ ] Production build works locally
- [ ] Lighthouse scores > 90
- [ ] No console errors
- [ ] Service workers register correctly
- [ ] All assets load without 404s
- [ ] Hot module replacement works
- [ ] Zero DAG violations
- [ ] Documentation updated
- [ ] Team notified of changes
- [ ] Rollback plan documented
- [ ] Monitoring configured

### Deployment Steps:

1. **Backup current production**
```bash
# Backup production build
cp -r /path/to/production /path/to/backup-$(date +%Y%m%d)
```

2. **Run final build**
```bash
bun run build
```

3. **Test production build locally**
```bash
bun run preview
# Test all functionality
```

4. **Deploy to staging**
```bash
# Deploy to staging environment
# Run full test suite
# Monitor for errors
```

5. **Deploy to production**
```bash
# Deploy to production
# Monitor logs closely for 1 hour
# Check error rates
# Monitor performance metrics
```

6. **Verify deployment**
- [ ] Admin loads at `https://domain.com/`
- [ ] Store loads at `https://domain.com/store`
- [ ] No errors in logs
- [ ] Performance metrics are good
- [ ] User activity is normal

### Post-Deployment Monitoring:

- Monitor logs for 24 hours
- Check error rates
- Monitor performance metrics
- Watch for user complaints
- Be ready to rollback if needed

---

## 📞 Emergency Contacts

### Technical Issues:
- **DevOps:** [Contact information]
- **Lead Developer:** [Contact information]
- **System Admin:** [Contact information]

### Business Issues:
- **Product Manager:** [Contact information]
- **Stakeholders:** [Contact information]

### Rollback Emergency:
- **Procedure:** Documented in `docs/ROLLBACK.md`
- **Authorization:** [Who can authorize]
- **Timeline:** [Time limit for rollback]

---

## ✅ Final Verification

Before marking migration complete:

- [ ] All Phase 7 tests passed
- [ ] All Phase 8 documentation complete
- [ ] Pre-deployment checklist complete
- [ ] Team trained on new architecture
- [ ] Monitoring and alerting configured
- [ ] Rollback procedure tested
- [ ] Documentation published
- [ ] Stakeholders notified

---

**Checklist Version:** 1.0  
**Last Updated:** January 25, 2025  
**Next Review:** After Phase 7 completion