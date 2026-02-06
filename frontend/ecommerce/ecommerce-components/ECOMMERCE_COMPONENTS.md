# Ecommerce Components

This document defines the **ready-to-use Svelte components** for building ecommerce stores. Unlike the ComponentAST which defines structure and layout, these components are **self-contained** with their own:

- Internal data fetching
- User interaction handling (add to cart, quantity changes, etc.)
- UI/UX logic (modals, dropdowns, animations)
- State management

These components are referenced in the `ComponentAST` via their `tagName` and configured via props.

---

## Table of Contents

1. [Component Philosophy](#component-philosophy)
2. [Common Props](#common-props)
3. [Product Components](#product-components)
4. [Category Components](#category-components)
5. [Brand Components](#brand-components)
6. [Cart Components](#cart-components)
7. [Search Components](#search-components)
8. [Navigation Components](#navigation-components)
9. [Promotional Components](#promotional-components)
10. [User Components](#user-components)
11. [Component List](#component-list)
12. [Implementation Priority](#implementation-priority)

---

## Component Philosophy

### Self-Contained Components

Each component handles its own complexity internally:

```typescript
// In ComponentAST - simple usage
{
    tagName: 'ProductGrid',
    productosIDs: [1, 2, 3, 4, 5, 6]
}

// The ProductGrid component internally:
// - Fetches product data from store/API
// - Renders product cards with images, prices
// - Handles add-to-cart clicks
// - Shows loading skeletons
// - Manages hover states and animations
```

### Styling Approach

Components accept a `css` prop for **wrapper/container styling**, but internal styles are controlled by the component itself (with theme support via CSS variables).

```typescript
{
    tagName: 'ProductCarousel',
    css: 'my-8 px-4',              // Wrapper styles (margins, padding)
    productosIDs: [1, 2, 3, 4]
    // Internal card styles are handled by the component
}
```

### Theme Integration

Components use CSS variables from the global color palette:

```css
/* Components use these variables internally */
--color-1: #0f172a;   /* Darkest */
--color-2: #1e293b;
/* ... */
--color-10: #f8fafc;  /* Lightest */
```

---

## Common Props

All ecommerce components share these optional props:

| Prop | Type | Description |
|:-----|:-----|:------------|
| `css` | `string` | Additional Tailwind classes for the wrapper |
| `id` | `string` | HTML id attribute |
| `aria` | `AriaAttributes` | Accessibility attributes |

---

## Product Components

### ProductCard

Displays a single product with image, name, price, and add-to-cart button.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productoID` | `number` | Yes | Product ID to display |
| `showPrice` | `boolean` | No | Show price (default: true) |
| `showAddToCart` | `boolean` | No | Show add-to-cart button (default: true) |
| `showRating` | `boolean` | No | Show star rating (default: false) |
| `imageAspect` | `'1:1' \| '4:3' \| '16:9'` | No | Image aspect ratio (default: '1:1') |
| `variant` | `'default' \| 'compact' \| 'detailed'` | No | Card style variant |

**Internal Features:**
- Hover zoom on image
- Quick add-to-cart button
- Sale badge if discounted
- Out-of-stock overlay
- Click navigates to product page

---

### ProductCardHorizontal

Horizontal product card with image on left, details on right.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productoID` | `number` | Yes | Product ID |
| `showDescription` | `boolean` | No | Show product description (default: true) |
| `maxDescriptionLines` | `number` | No | Truncate description (default: 2) |
| `showQuantity` | `boolean` | No | Show quantity selector (default: false) |

---

### ProductGrid

Grid layout of multiple products.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productosIDs` | `number[]` | No* | Specific product IDs |
| `categoriaID` | `number` | No* | Filter by category |
| `marcaID` | `number` | No* | Filter by brand |
| `columns` | `number` | No | Number of columns (default: 4, responsive) |
| `limit` | `number` | No | Max products to show (default: 12) |
| `showPagination` | `boolean` | No | Show pagination (default: false) |
| `gap` | `'sm' \| 'md' \| 'lg'` | No | Gap between cards (default: 'md') |

*At least one data prop required

---

### ProductCarousel

Horizontal scrolling product slider.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productosIDs` | `number[]` | No* | Specific product IDs |
| `categoriaID` | `number` | No* | Filter by category |
| `marcaID` | `number` | No* | Filter by brand |
| `autoplay` | `boolean` | No | Auto-scroll (default: false) |
| `autoplayInterval` | `number` | No | Interval in ms (default: 5000) |
| `showArrows` | `boolean` | No | Show navigation arrows (default: true) |
| `showDots` | `boolean` | No | Show dot indicators (default: false) |
| `slidesToShow` | `number` | No | Visible slides (default: 4, responsive) |

---

### ProductSquare2x2

Featured 2x2 grid for highlighting products.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productosIDs` | `[number, number, number, number]` | Yes | Exactly 4 product IDs |
| `variant` | `'equal' \| 'featured'` | No | Layout style (default: 'equal') |

**Variants:**
- `equal`: All 4 products same size
- `featured`: First product larger, others smaller

---

### ProductList

Vertical list of products (compact view).

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productosIDs` | `number[]` | No* | Specific product IDs |
| `categoriaID` | `number` | No* | Filter by category |
| `limit` | `number` | No | Max products (default: 5) |
| `showImage` | `boolean` | No | Show thumbnail (default: true) |
| `showPrice` | `boolean` | No | Show price (default: true) |

---

### ProductFeatured

Large featured product display.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productoID` | `number` | Yes | Product ID |
| `layout` | `'left' \| 'right' \| 'center'` | No | Image position (default: 'left') |
| `showDescription` | `boolean` | No | Show full description (default: true) |
| `showSpecs` | `boolean` | No | Show specifications (default: false) |

---

## Category Components

### CategoryCard

Single category card with image and name.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `categoriaID` | `number` | Yes | Category ID |
| `showCount` | `boolean` | No | Show product count (default: false) |
| `imageOverlay` | `boolean` | No | Dark overlay on image (default: true) |
| `variant` | `'default' \| 'minimal' \| 'banner'` | No | Card style |

---

### CategoryGrid

Grid of category cards.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `categoriasIDs` | `number[]` | No | Specific category IDs (all if empty) |
| `columns` | `number` | No | Grid columns (default: 4) |
| `limit` | `number` | No | Max categories (default: 8) |
| `showCount` | `boolean` | No | Show product counts (default: false) |

---

### CategoryList

Vertical list of categories (for sidebar).

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `categoriasIDs` | `number[]` | No | Specific IDs (all if empty) |
| `showIcon` | `boolean` | No | Show category icon (default: true) |
| `showCount` | `boolean` | No | Show product count (default: true) |
| `collapsible` | `boolean` | No | Collapsible subcategories (default: true) |

---

### CategoryCarousel

Horizontal scrolling category slider.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `categoriasIDs` | `number[]` | No | Specific IDs (all if empty) |
| `showArrows` | `boolean` | No | Show navigation (default: true) |

---

## Brand Components

### BrandCard

Single brand logo/card.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `marcaID` | `number` | Yes | Brand ID |
| `showName` | `boolean` | No | Show brand name (default: false) |
| `variant` | `'logo' \| 'card'` | No | Display style |

---

### BrandGrid

Grid of brand logos.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `marcasIDs` | `number[]` | No | Specific brand IDs (all if empty) |
| `columns` | `number` | No | Grid columns (default: 6) |
| `limit` | `number` | No | Max brands (default: 12) |

---

### BrandCarousel

Horizontal scrolling brand logos.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `marcasIDs` | `number[]` | No | Specific brand IDs |
| `autoplay` | `boolean` | No | Auto-scroll (default: true) |

---

## Cart Components

### CartWidget

Mini cart icon for header with dropdown preview.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `showCount` | `boolean` | No | Show item count badge (default: true) |
| `showDropdown` | `boolean` | No | Show dropdown on hover (default: true) |
| `dropdownMaxItems` | `number` | No | Items in dropdown (default: 3) |

**Internal Features:**
- Badge with item count
- Dropdown with recent items
- Total price display
- "View Cart" and "Checkout" buttons
- Empty cart state

---

### CartPage

Full cart page component.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `showSummary` | `boolean` | No | Show order summary (default: true) |
| `showCoupon` | `boolean` | No | Show coupon input (default: true) |
| `layout` | `'default' \| 'compact'` | No | Layout style |

**Internal Features:**
- Item list with quantities
- Remove item buttons
- Quantity adjusters
- Subtotal/Total calculation
- Coupon application
- Proceed to checkout button

---

### CartItem

Single cart item row (used internally by CartPage).

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `itemID` | `string` | Yes | Cart item ID |
| `showQuantity` | `boolean` | No | Show quantity controls (default: true) |
| `showRemove` | `boolean` | No | Show remove button (default: true) |

---

## Search Components

### SearchBar

Product search with autocomplete.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `placeholder` | `string` | No | Input placeholder |
| `showIcon` | `boolean` | No | Show search icon (default: true) |
| `autoFocus` | `boolean` | No | Focus on mount (default: false) |
| `maxSuggestions` | `number` | No | Max autocomplete items (default: 5) |
| `showCategories` | `boolean` | No | Show category suggestions (default: true) |

**Internal Features:**
- Debounced search
- Autocomplete dropdown
- Recent searches
- Category quick filters
- Keyboard navigation

---

### SearchResults

Search results page component.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `query` | `string` | Yes | Search query |
| `showFilters` | `boolean` | No | Show filter sidebar (default: true) |
| `showSort` | `boolean` | No | Show sort options (default: true) |
| `columns` | `number` | No | Grid columns (default: 4) |

---

## Navigation Components

### Breadcrumb

Navigation breadcrumb trail.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `items` | `BreadcrumbItem[]` | No | Manual items (auto-generated if empty) |
| `separator` | `string` | No | Separator character (default: '/') |
| `showHome` | `boolean` | No | Show home link (default: true) |

```typescript
interface BreadcrumbItem {
    label: string;
    href?: string;
}
```

---

### Pagination

Page navigation for lists.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `totalPages` | `number` | Yes | Total number of pages |
| `currentPage` | `number` | Yes | Current page (1-indexed) |
| `showFirstLast` | `boolean` | No | Show first/last buttons (default: true) |
| `maxVisible` | `number` | No | Max visible page numbers (default: 5) |

---

### FilterSidebar

Product filtering sidebar.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `showCategories` | `boolean` | No | Show category filter (default: true) |
| `showBrands` | `boolean` | No | Show brand filter (default: true) |
| `showPrice` | `boolean` | No | Show price range (default: true) |
| `showRating` | `boolean` | No | Show rating filter (default: false) |
| `collapsible` | `boolean` | No | Collapsible sections (default: true) |

---

## Promotional Components

### Banner

Full-width promotional banner.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `imageUrl` | `string` | Yes | Banner image URL |
| `href` | `string` | No | Link destination |
| `alt` | `string` | No | Image alt text |
| `height` | `'sm' \| 'md' \| 'lg' \| 'full'` | No | Banner height |
| `overlay` | `boolean` | No | Dark overlay (default: false) |

---

### PromotionCard

Promotional offer card.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `title` | `string` | Yes | Promotion title |
| `subtitle` | `string` | No | Subtitle/description |
| `discount` | `string` | No | Discount text (e.g., "20% OFF") |
| `href` | `string` | No | Link destination |
| `backgroundImage` | `string` | No | Background image URL |
| `backgroundColor` | `string` | No | Background color (uses theme if empty) |

---

### CountdownTimer

Countdown to promotion end.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `endDate` | `string \| Date` | Yes | Countdown end date |
| `title` | `string` | No | Timer title |
| `showDays` | `boolean` | No | Show days (default: true) |
| `variant` | `'default' \| 'compact' \| 'large'` | No | Display style |

---

### AnnouncementBar

Top announcement/notification bar.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `message` | `string` | Yes | Announcement text |
| `href` | `string` | No | Link destination |
| `dismissible` | `boolean` | No | Can be dismissed (default: true) |
| `variant` | `'info' \| 'success' \| 'warning' \| 'promo'` | No | Style variant |

---

## User Components

### UserMenu

User account dropdown menu.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `showAvatar` | `boolean` | No | Show user avatar (default: true) |
| `showName` | `boolean` | No | Show user name (default: false) |

**Internal Features:**
- Login/Register links (if not logged in)
- My Account link
- Order History link
- Wishlist link
- Logout button

---

### WishlistButton

Add to wishlist toggle button.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `productoID` | `number` | Yes | Product ID |
| `variant` | `'icon' \| 'button'` | No | Display style (default: 'icon') |
| `showCount` | `boolean` | No | Show wishlist count (default: false) |

---

### LoginForm

User login form.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `showRegister` | `boolean` | No | Show register link (default: true) |
| `showForgotPassword` | `boolean` | No | Show forgot password (default: true) |
| `redirectTo` | `string` | No | Redirect after login |

---

### RegisterForm

User registration form.

| Prop | Type | Required | Description |
|:-----|:-----|:---------|:------------|
| `showLogin` | `boolean` | No | Show login link (default: true) |
| `redirectTo` | `string` | No | Redirect after register |
| `requirePhone` | `boolean` | No | Require phone number (default: false) |

---

## Component List

### Priority 1: Core (Must Have)

| Component | File | Status |
|:----------|:-----|:-------|
| ProductCard | `ProductCard.svelte` | ⬜ Pending |
| ProductGrid | `ProductGrid.svelte` | ⬜ Pending |
| ProductCarousel | `ProductCarousel.svelte` | ⬜ Pending |
| CategoryCard | `CategoryCard.svelte` | ⬜ Pending |
| CategoryGrid | `CategoryGrid.svelte` | ⬜ Pending |
| CartWidget | `CartWidget.svelte` | ⬜ Pending |
| CartPage | `CartPage.svelte` | ⬜ Pending |
| SearchBar | `SearchBar.svelte` | ⬜ Pending |
| Breadcrumb | `Breadcrumb.svelte` | ⬜ Pending |

### Priority 2: Important

| Component | File | Status |
|:----------|:-----|:-------|
| ProductCardHorizontal | `ProductCardHorizontal.svelte` | ⬜ Pending |
| ProductSquare2x2 | `ProductSquare2x2.svelte` | ⬜ Pending |
| ProductFeatured | `ProductFeatured.svelte` | ⬜ Pending |
| CategoryList | `CategoryList.svelte` | ⬜ Pending |
| BrandGrid | `BrandGrid.svelte` | ⬜ Pending |
| BrandCarousel | `BrandCarousel.svelte` | ⬜ Pending |
| Pagination | `Pagination.svelte` | ⬜ Pending |
| FilterSidebar | `FilterSidebar.svelte` | ⬜ Pending |
| Banner | `Banner.svelte` | ⬜ Pending |

### Priority 3: Nice to Have

| Component | File | Status |
|:----------|:-----|:-------|
| ProductList | `ProductList.svelte` | ⬜ Pending |
| CategoryCarousel | `CategoryCarousel.svelte` | ⬜ Pending |
| BrandCard | `BrandCard.svelte` | ⬜ Pending |
| CartItem | `CartItem.svelte` | ⬜ Pending |
| SearchResults | `SearchResults.svelte` | ⬜ Pending |
| PromotionCard | `PromotionCard.svelte` | ⬜ Pending |
| CountdownTimer | `CountdownTimer.svelte` | ⬜ Pending |
| AnnouncementBar | `AnnouncementBar.svelte` | ⬜ Pending |
| UserMenu | `UserMenu.svelte` | ⬜ Pending |
| WishlistButton | `WishlistButton.svelte` | ⬜ Pending |
| LoginForm | `LoginForm.svelte` | ⬜ Pending |
| RegisterForm | `RegisterForm.svelte` | ⬜ Pending |

---

## Implementation Priority

### Phase 1: MVP Store (Week 1-2)

1. **ProductCard** - Foundation for all product displays
2. **ProductGrid** - Main product listing
3. **CartWidget** - Essential for purchases
4. **CartPage** - Complete cart functionality
5. **SearchBar** - Product discovery

### Phase 2: Enhanced Navigation (Week 2-3)

6. **CategoryCard** - Category navigation
7. **CategoryGrid** - Category browsing
8. **Breadcrumb** - Navigation context
9. **ProductCarousel** - Featured products
10. **Pagination** - Browse large catalogs

### Phase 3: Marketing & UX (Week 3-4)

11. **Banner** - Promotional content
12. **ProductFeatured** - Hero product displays
13. **BrandGrid** - Brand showcase
14. **FilterSidebar** - Advanced filtering
15. **AnnouncementBar** - Store announcements

### Phase 4: Complete Experience (Week 4+)

16. Remaining components as needed
17. User authentication components
18. Wishlist functionality
19. Advanced promotional tools

---

## File Structure

```
pkg-store/ecommerce-components/
├── ECOMMERCE_COMPONENTS.md        # This document
├── index.ts                       # Component exports
├── types.ts                       # Shared TypeScript types
├── registry.ts                    # Component registration for renderer
│
├── products/
│   ├── ProductCard.svelte
│   ├── ProductCardHorizontal.svelte
│   ├── ProductGrid.svelte
│   ├── ProductCarousel.svelte
│   ├── ProductSquare2x2.svelte
│   ├── ProductList.svelte
│   └── ProductFeatured.svelte
│
├── categories/
│   ├── CategoryCard.svelte
│   ├── CategoryGrid.svelte
│   ├── CategoryList.svelte
│   └── CategoryCarousel.svelte
│
├── brands/
│   ├── BrandCard.svelte
│   ├── BrandGrid.svelte
│   └── BrandCarousel.svelte
│
├── cart/
│   ├── CartWidget.svelte
│   ├── CartPage.svelte
│   └── CartItem.svelte
│
├── search/
│   ├── SearchBar.svelte
│   └── SearchResults.svelte
│
├── navigation/
│   ├── Breadcrumb.svelte
│   ├── Pagination.svelte
│   └── FilterSidebar.svelte
│
├── promotional/
│   ├── Banner.svelte
│   ├── PromotionCard.svelte
│   ├── CountdownTimer.svelte
│   └── AnnouncementBar.svelte
│
└── user/
    ├── UserMenu.svelte
    ├── WishlistButton.svelte
    ├── LoginForm.svelte
    └── RegisterForm.svelte
```

---

## Example Usage in ComponentAST

```typescript
// Hero section with featured product carousel
const heroSection: ComponentAST = {
    tagName: 'section',
    css: 'bg-[__COLOR:9__] py-16',
    children: [
        {
            tagName: 'div',
            css: 'max-w-7xl mx-auto px-4',
            children: [
                {
                    tagName: 'h2',
                    text: 'Featured Products',
                    css: 'text-3xl font-bold text-center mb-8'
                },
                {
                    tagName: 'ProductCarousel',
                    productosIDs: [1, 2, 3, 4, 5, 6, 7, 8],
                    showArrows: true,
                    autoplay: true
                }
            ]
        }
    ]
};

// Category showcase
const categoriesSection: ComponentAST = {
    tagName: 'section',
    css: 'py-12',
    children: [
        {
            tagName: 'CategoryGrid',
            css: 'max-w-6xl mx-auto',
            columns: 4,
            showCount: true
        }
    ]
};

// Product grid filtered by category
const productsByCategory: ComponentAST = {
    tagName: 'ProductGrid',
    categoriaID: 5,
    limit: 12,
    showPagination: true
};
```
