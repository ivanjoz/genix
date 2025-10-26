# Genix Frontend 2 - Svelte 5 Migration

Modern frontend implementation using Svelte 5, SvelteKit, and Tailwind CSS.

## 🚀 Overview

This is a complete migration of the Genix frontend from Solid.js to Svelte 5, featuring:

- **Svelte 5** with modern runes ($state, $derived, $effect)
- **SvelteKit** for routing and SSR
- **Tailwind CSS** for styling
- **TypeScript** for type safety
- **Responsive design** for mobile, tablet, and desktop
- **Dark mode** support
- **Accessibility** features (ARIA labels, keyboard navigation)

## 📁 Project Structure

```
frontend2/
├── src/
│   ├── components/
│   │   ├── SideMenu.svelte      # Main navigation menu
│   │   └── Header.svelte        # Top header bar
│   ├── types/
│   │   └── menu.ts              # Menu types and configuration
│   ├── routes/
│   │   ├── +layout.svelte       # Main layout
│   │   ├── +page.svelte         # Home page
│   │   └── admin/
│   │       └── empresas/
│   │           └── +page.svelte # Example page
│   ├── lib/
│   │   └── assets/              # Static assets
│   └── app.css                  # Global styles
├── static/                      # Public files
├── MENU_MIGRATION.md           # Detailed migration guide
├── COMPONENT_SUMMARY.md        # Component documentation
├── CSS_CONVERSION_GUIDE.md     # CSS to Tailwind conversion
└── README.md                   # This file
```

## 🛠️ Installation

```bash
# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## 📦 Key Dependencies

```json
{
  "svelte": "^5.0.0",
  "@sveltejs/kit": "^2.0.0",
  "tailwindcss": "^3.4.0",
  "typescript": "^5.0.0"
}
```

## 🎨 Components

### SideMenu.svelte

Comprehensive side navigation menu with:
- Collapsible sections
- Desktop minimization
- Mobile slide-in menu
- Route highlighting
- Smooth animations
- Dark mode support

**Usage:**
```svelte
<script>
  import SideMenu from './components/SideMenu.svelte';
  
  let isMobileOpen = $state(false);
  let isMinimized = $state(false);
</script>

<SideMenu bind:isMobileOpen bind:isMinimized />
```

### Header.svelte

Top navigation bar with:
- Theme switcher
- Settings dropdown
- Mobile menu toggle
- Reload functionality
- Logout action

**Usage:**
```svelte
<script>
  import Header from './components/Header.svelte';
</script>

<Header 
  showMenuButton={true}
  onMenuToggle={() => toggleMobileMenu()}
  title="Mi App"
/>
```

## 🎯 Features

### Desktop Experience
- **Fixed sidebar** with hover-to-expand when minimized
- **Smooth transitions** between states
- **Keyboard navigation** support
- **Custom scrollbar** styling

### Mobile Experience
- **Slide-in menu** from right
- **Backdrop blur** effect
- **Touch-optimized** interactions
- **Bottom-sheet** style on small screens

### Theme Support
- **Light mode** (default)
- **Dark mode** toggle in header
- **Persists** to localStorage
- **Smooth transitions** between themes

### Accessibility
- **ARIA labels** on all interactive elements
- **Keyboard navigation** (Tab, Enter, Escape)
- **Focus indicators** visible
- **Screen reader** friendly

## 🔧 Configuration

### Menu Structure

Edit `src/types/menu.ts` to customize the menu:

```typescript
import type { IModule } from './menu';

export const myMenu: IModule = {
  id: 1,
  name: 'Mi Sistema',
  menus: [
    {
      id: 1,
      name: 'Dashboard',
      minName: 'DASH',
      options: [
        {
          name: 'Overview',
          minName: 'OVR',
          route: '/dashboard',
          icon: 'chart'
        }
      ]
    }
  ]
};
```

### Styling

Global styles in `src/app.css`:

```css
:root {
  --header-height: 3rem;
  --menu-width: 14rem;
  --primary: #4042a3;
}
```

Component-specific styles use Tailwind utilities:

```svelte
<div class="bg-blue-600 hover:bg-blue-700 transition-colors">
  Button
</div>
```

## 📱 Responsive Design

Breakpoints:
- **Mobile**: < 768px (Tailwind `md`)
- **Tablet**: 768px - 1024px
- **Desktop**: > 1024px

```svelte
<!-- Hide on mobile, show on desktop -->
<div class="hidden md:block">Desktop only</div>

<!-- Show on mobile, hide on desktop -->
<div class="block md:hidden">Mobile only</div>
```

## 🎨 Theming

Toggle theme:

```typescript
// In Header.svelte
function toggleTheme(theme: 'light' | 'dark') {
  document.body.classList.remove('light', 'dark');
  document.body.classList.add(theme);
  localStorage.setItem('ui-color', theme);
}
```

Add theme-specific styles:

```svelte
<div class="bg-white dark:bg-gray-900 text-gray-900 dark:text-white">
  Content
</div>
```

## 🚀 Development

### Adding a New Page

1. Create a new route file:
```bash
src/routes/my-page/+page.svelte
```

2. Add content:
```svelte
<script lang="ts">
  // Your logic
</script>

<h1>My Page</h1>
```

3. Add to menu in `src/types/menu.ts`:
```typescript
{
  name: 'My Page',
  route: '/my-page',
  icon: 'icon'
}
```

### Adding a New Component

1. Create component file:
```bash
src/components/MyComponent.svelte
```

2. Define props:
```svelte
<script lang="ts">
  let { title, data = [] } = $props<{
    title: string;
    data?: string[];
  }>();
</script>
```

3. Use component:
```svelte
<script>
  import MyComponent from './components/MyComponent.svelte';
</script>

<MyComponent title="Hello" data={['a', 'b']} />
```

## 🧪 Testing

```bash
# Run tests
npm test

# Type checking
npm run check

# Linting
npm run lint

# Format code
npm run format
```

## 📚 Documentation

- **[MENU_MIGRATION.md](./MENU_MIGRATION.md)** - Detailed migration guide
- **[COMPONENT_SUMMARY.md](./COMPONENT_SUMMARY.md)** - Component documentation
- **[CSS_CONVERSION_GUIDE.md](./CSS_CONVERSION_GUIDE.md)** - CSS to Tailwind guide

## 🔄 Migration from Solid.js

Key differences:

1. **Reactivity**: Svelte runes instead of signals
   ```typescript
   // Solid.js
   const [value, setValue] = createSignal(0);
   
   // Svelte 5
   let value = $state(0);
   ```

2. **Computed Values**: $derived instead of createMemo
   ```typescript
   // Solid.js
   const double = createMemo(() => value() * 2);
   
   // Svelte 5
   let double = $derived(value * 2);
   ```

3. **Effects**: $effect instead of createEffect
   ```typescript
   // Solid.js
   createEffect(() => {
     console.log(value());
   });
   
   // Svelte 5
   $effect(() => {
     console.log(value);
   });
   ```

4. **Styling**: Tailwind instead of CSS modules
   ```svelte
   <!-- Solid.js -->
   <div class={styles.container}>Content</div>
   
   <!-- Svelte 5 -->
   <div class="bg-white p-4 rounded-lg">Content</div>
   ```

## 🐛 Known Issues

None at the moment! 🎉

## 📝 TODO

- [ ] Add icon library (lucide-svelte)
- [ ] Add menu search functionality
- [ ] Add user favorites system
- [ ] Add breadcrumb navigation
- [ ] Add notification badges
- [ ] Add keyboard shortcuts
- [ ] Add loading states
- [ ] Add error boundaries
- [ ] Add unit tests
- [ ] Add E2E tests

## 🤝 Contributing

1. Create a feature branch
2. Make changes
3. Run linter and type check
4. Test on mobile and desktop
5. Update documentation
6. Submit pull request

## 📄 License

Same as parent project.

## 🙏 Acknowledgments

- Original Solid.js implementation: `../frontend/src/core/menu.tsx`
- Svelte team for Svelte 5
- Tailwind CSS team
- SvelteKit team

## 📞 Support

For issues or questions, please refer to the project documentation or contact the development team.

---

**Version**: 1.0.0  
**Last Updated**: 2024  
**Status**: ✅ Ready for use
