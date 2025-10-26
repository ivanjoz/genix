# Menu Component Migration Guide

This document describes the migration of the menu system from Solid.js to Svelte 5.

## Components

### 1. SideMenu.svelte

A comprehensive side navigation menu component migrated from the Solid.js `MainMenu` component.

#### Features

- **Collapsible Menu Sections**: Menu items can be expanded/collapsed
- **Desktop & Mobile Support**: Responsive design with different layouts
- **Minimize Mode**: Desktop menu can be minimized to show only icons
- **Hover Expansion**: Minimized menu expands on hover
- **Route Highlighting**: Active routes are visually highlighted
- **Smooth Animations**: Transitions for opening/closing menus
- **Dark Mode Support**: Built-in dark mode styling
- **Accessibility**: ARIA labels and keyboard navigation support

#### Props

```typescript
interface SideMenuProps {
	module?: IModule;           // Menu structure (optional, has defaults)
	isMinimized?: boolean;      // Desktop menu minimized state (bindable)
	isMobileOpen?: boolean;     // Mobile menu open state (bindable)
}
```

#### Menu Structure

```typescript
interface IMenuRecord {
	name: string;           // Full menu name
	minName?: string;       // Abbreviated name (for minimized mode)
	id?: number;            // Unique identifier
	route?: string;         // Navigation route
	options?: IMenuRecord[]; // Sub-menu items
	icon?: string;          // Icon identifier
}

interface IModule {
	id: number;
	name: string;
	menus: IMenuRecord[];
}
```

#### Usage

```svelte
<script>
	import SideMenu from './components/SideMenu.svelte';
	
	let isMobileOpen = $state(false);
	let isMinimized = $state(false);
</script>

<SideMenu 
	bind:isMobileOpen 
	bind:isMinimized 
/>
```

### 2. Header.svelte

Top navigation header with settings and actions.

#### Features

- **Theme Switcher**: Toggle between light and dark modes
- **Settings Dropdown**: Collapsible settings panel
- **Reload Action**: Force reload with cache clearing
- **Logout Action**: Clear session and redirect
- **Mobile Menu Toggle**: Show/hide mobile menu
- **Loading Indicator**: Ready for async operations

#### Props

```typescript
interface HeaderProps {
	onMenuToggle?: () => void;  // Mobile menu toggle callback
	title?: string;             // Header title
	showMenuButton?: boolean;   // Show mobile menu button
}
```

#### Usage

```svelte
<script>
	import Header from './components/Header.svelte';
	
	let showMobileMenu = $state(false);
</script>

<Header
	showMenuButton={true}
	onMenuToggle={() => (showMobileMenu = !showMobileMenu)}
	title="Sistema Genix"
/>
```

## CSS Migration

### Original CSS Classes → Tailwind Equivalents

| Original Class | Tailwind Equivalent | Notes |
|---------------|---------------------|-------|
| `.main-menu-c` | `fixed left-0 top-0 h-screen` | Fixed sidebar |
| `.menu-main-c1` | `overflow-y-auto flex-1` | Scrollable menu container |
| `.menus-c1` | `flex flex-col` | Menu section wrapper |
| `.menu-main-label` | `h-12 px-3 flex items-center` | Menu header |
| `.menu-option-c1` | `h-9 px-2 flex items-center` | Menu option |
| `.icon-c1` | `transition-transform` | Animated icon |
| `.main-menu-mob` | `fixed right-0 top-0` | Mobile menu |

### Color Scheme

The color scheme has been converted to Tailwind's color system:

- **Background**: `gray-900` → `gray-950` gradient
- **Hover**: `gray-800` / `gray-800/50`
- **Active**: `blue-600` / `blue-900/30`
- **Border**: `gray-800`
- **Text**: `white`, `gray-400`, `blue-300`

### Key Styling Features

1. **Gradient Backgrounds**: `bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950`
2. **Transitions**: All interactive elements have smooth transitions
3. **Custom Scrollbar**: Thin, styled scrollbar for menu overflow
4. **Hover Effects**: Distinct hover states for all interactive elements
5. **Border Accents**: Left border highlights for active states

## Svelte 5 Features Used

### 1. Runes

```typescript
// State management
let menuOpen = $state<[number, string]>([0, '']);
let isMenuHover = $state(false);

// Derived state
let menuWidth = $derived(isMinimized && !isMenuHover ? 'w-18' : 'w-56');

// Props with defaults
let { module = $bindable(), isMinimized = $bindable(false) } = $props();
```

### 2. Effects

```typescript
// React to route changes
$effect(() => {
	if ($page?.url?.pathname) {
		currentPathname = $page.url.pathname;
		menuOpen = getMenuOpenFromRoute(module, currentPathname);
	}
});
```

### 3. Event Handlers

```svelte
<!-- Inline event handlers -->
<button onclick={() => toggleMenu(menu.id || 0)}>
	Toggle Menu
</button>

<!-- With parameters -->
<a onclick={() => handleMenuItemClick(option.route || '/', menu.id || 0)}>
	Link
</a>
```

## Differences from Solid.js Version

### Improvements

1. **Better Type Safety**: Full TypeScript support with Svelte 5
2. **Simpler State Management**: No separate signals, just reactive variables
3. **Better Accessibility**: Added ARIA labels and roles
4. **Responsive Design**: Better mobile/tablet support
5. **Dark Mode**: Built-in dark mode support
6. **Icon System**: Simplified icon system (ready for icon library)

### Changes

1. **Route Management**: Using SvelteKit's `$page` store instead of custom routing
2. **Menu Modes**: Simplified mode system (removed mode 2/3 split views)
3. **Styling**: Converted to Tailwind CSS instead of CSS modules
4. **Mobile Menu**: Slide-in from right instead of full overlay
5. **Logo**: Emoji placeholder (can be replaced with actual logo)

## Integration with Layout

The components are integrated in the main layout:

```svelte
<!-- +layout.svelte -->
<script lang="ts">
	import Header from '../components/Header.svelte';
	import SideMenu from '../components/SideMenu.svelte';

	let isMobileMenuOpen = $state(false);
	let isMenuMinimized = $state(false);
</script>

<Header
	showMenuButton={true}
	onMenuToggle={() => (isMobileMenuOpen = !isMobileMenuOpen)}
/>

<SideMenu 
	bind:isMobileOpen={isMobileMenuOpen} 
	bind:isMinimized={isMenuMinimized} 
/>

<main class="ml-56 pt-12">
	{@render children?.()}
</main>
```

## Customization

### Adding New Menu Items

```typescript
const customModule: IModule = {
	id: 1,
	name: 'Mi Sistema',
	menus: [
		{
			id: 1,
			name: 'Administración',
			minName: 'ADM',
			options: [
				{
					name: 'Usuarios',
					minName: 'USR',
					route: '/admin/users',
					icon: 'icon-users'
				}
			]
		}
	]
};
```

### Styling Modifications

All styles are in the component files using:
- Tailwind utility classes
- Custom CSS in `<style>` blocks for complex animations
- CSS variables in `app.css` for theme values

### Icon System

Currently using emoji placeholders. To integrate an icon library:

1. Install icon library (e.g., `lucide-svelte`, `@iconify/svelte`)
2. Import icon components
3. Replace emoji with icon components:

```svelte
<script>
	import { Building, Users, Settings } from 'lucide-svelte';
</script>

<Building size={16} />
```

## Testing

The components can be tested with:

1. **Route Navigation**: Test all menu links navigate correctly
2. **Responsive Design**: Test on mobile, tablet, and desktop
3. **Theme Switching**: Test light/dark mode transitions
4. **Minimize/Expand**: Test menu minimize functionality
5. **Mobile Menu**: Test mobile menu open/close
6. **Keyboard Navigation**: Test tab navigation and keyboard shortcuts

## Future Enhancements

1. **Search Functionality**: Add menu search feature
2. **Favorites**: Allow users to favorite menu items
3. **Breadcrumbs**: Add breadcrumb navigation
4. **Icon Library**: Integrate proper icon library
5. **Animations**: Add more sophisticated animations
6. **Keyboard Shortcuts**: Add keyboard shortcuts for common actions
7. **Multi-level Menus**: Support more than 2 levels of nesting
8. **Drag to Reorder**: Allow users to customize menu order
9. **Notifications**: Add notification badges to menu items
10. **User Preferences**: Save menu state to user preferences

## Performance

- **Lazy Loading**: Menu items can be lazy loaded
- **Virtual Scrolling**: For very long menus, implement virtual scrolling
- **Code Splitting**: Menu components are code-split by default with SvelteKit
- **Minimal Re-renders**: Svelte's reactivity minimizes unnecessary updates

## Browser Support

- Modern browsers (Chrome, Firefox, Safari, Edge)
- Mobile browsers (iOS Safari, Chrome Android)
- Requires ES2020+ support
- CSS Grid and Flexbox support required

