# SideMenu Component Update

## Summary
Updated the SideMenu component to match the original Solid.js implementation with the following key changes:

## Key Changes

### 1. **Narrower Default Width**
- **Before**: Menu was 14rem (w-56) by default
- **After**: Menu is now 4.5rem (w-18) by default, matching the original
- Expands to 14rem (w-56) on hover

### 2. **Fontello Icons**
- **Before**: Using emoji icons (🏢, 📦, 📄, etc.)
- **After**: Using fontello icon classes (`icon-cube`, `icon-cog`, `icon-home-1`, etc.)
- Added fontello CSS import in `app.css`

### 3. **Hover Expansion Behavior**
- **Before**: Menu was controlled by `isMinimized` prop
- **After**: Menu is always minimized by default, expands automatically on hover
- Removed `isMinimized` prop completely

### 4. **Visual Improvements**
- Minimized view shows only abbreviated menu names (minName)
- Arrow icons only visible when menu is expanded
- Submenu items show only icons when minimized, full text when expanded
- Better alignment and spacing to match original design

### 5. **Code Structure**
- Removed toggle button for minimizing/expanding
- Simplified state management
- Better responsive behavior for mobile view

## Files Changed

### `/frontend2/src/components/SideMenu.svelte`
- Complete rewrite to match Solid.js behavior
- Always minimized by default (w-18 / 4.5rem)
- Expands on hover to w-56 (14rem)
- Uses fontello icon classes
- Removed `isMinimized` prop

### `/frontend2/src/app.css`
```css
@import './lib/fontello-embedded.css';
```
Added import for fontello icon fonts

### `/frontend2/src/routes/+layout.svelte`
- Removed `isMenuMinimized` state
- Fixed main content margin to always use `ml-18` (matches 4.5rem menu width)
- Removed `bind:isMinimized` from SideMenu component

## Fontello Icons Used in Menus
Based on the modules configuration:
- `icon-flow-merge` - Gestión
- `icon-black-tie` - Empresas
- `icon-cog` - Parámetros
- `icon-adult` - Usuarios
- `icon-shield` - Perfiles & Accesos
- `icon-database` - Backups
- `icon-cube` - Operaciones, Productos
- `icon-home-1` - Sedes & Almacenes
- `icon-chart-bar` - Productos Stock
- `icon-truck` - Almacén Movimientos
- `icon-suitcase` - Cajas & Bancos
- `icon-exchange` - Cajas Movimientos
- `icon-flash` - Ventas
- `icon-buffer` - UI Components, CMS
- `icon-tasks` - Comercial, Reportes

## Behavior Differences from Original

### Kept from Original:
- ✅ Narrow default width (4.5rem)
- ✅ Hover expansion to full width
- ✅ Fontello icons
- ✅ Minimized text labels (minName)
- ✅ Smooth transitions
- ✅ Active route highlighting

### Modern Improvements:
- Better mobile menu animation (slide from left)
- Cleaner Tailwind CSS styling
- Svelte 5 runes for reactivity
- Better accessibility (ARIA labels)
- Consistent with modern design patterns

## Usage

```svelte
<!-- In +layout.svelte -->
<SideMenu bind:isMobileOpen={isMobileMenuOpen} />
```

The menu will:
1. Start at 4.5rem width (minimized)
2. Expand to 14rem on hover
3. Show full menu names when expanded
4. Show abbreviated names (minName) when minimized
5. Icons remain visible in both states

