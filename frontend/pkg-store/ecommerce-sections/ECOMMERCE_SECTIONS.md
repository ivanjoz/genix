# Reusable Ecommerce Section System

This system enables the creation, management, and rendering of dynamic and highly customizable User Interface (UI) sections for building ecommerce stores. The goal is to provide a component library that users can select, configure, and save to generate their own store.

---

## Table of Contents

1. [System Architecture](#1-system-architecture)
2. [Core Interfaces](#2-core-interfaces)
3. [Parameterization and Variables](#3-parameterization-and-variables)
4. [Global Colors and Theming](#4-global-colors-and-theming)
5. [Responsive Design Strategy](#5-responsive-design-strategy)
6. [Specialized Components](#6-specialized-components)
7. [Content Editing: ITextLine](#7-content-editing-itextline)
8. [Component Library](#8-component-library)
9. [Accessibility (A11Y)](#9-accessibility-a11y)
10. [SEO Considerations](#10-seo-considerations)
11. [Implementation Phases](#11-implementation-phases)
12. [Critical Considerations](#12-critical-considerations)
13. [Alternative Strategies](#13-alternative-strategies)
14. [Open Questions](#14-open-questions)

---

## 1. System Architecture

The system is divided into three main layers:

1. **Definition (ComponentAST):** A data structure that describes the content, behavior, and styling of a section.
2. **Rendering (EcommerceRenderer.svelte):** The engine responsible for interpreting the `ComponentAST` and converting it into DOM elements or Svelte components.
3. **Component Library:** A collection of `.ts` files that serve as base templates (Blueprints) for the sections.

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Interface                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ Section      │  │ Variable     │  │ Preview              │  │
│  │ Picker       │  │ Editor       │  │ Panel                │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ComponentAST (JSON)                         │
│  - Structure, styles, variables, product IDs                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   EcommerceRenderer.svelte                      │
│  ┌─────────────────────┐  ┌─────────────────────────────────┐  │
│  │ Token Resolver      │  │ Component Factory               │  │
│  │ (variables, colors) │  │ (HTML or Specialized)           │  │
│  └─────────────────────┘  └─────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Rendered DOM                               │
│  HTML elements + Specialized Svelte Components                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Core Interfaces

### ComponentProps (E-commerce Properties)

This interface contains the non-generic properties used by e-commerce components (logic-enabled components). These properties are inherited by all ComponentAST nodes.

```typescript
interface IGalleryImagen {
    image: string;
    title: string;
    description: string;
}

export interface ComponentProps {
    title: string;                        // Component title or heading text
    productosIDs?: number[];              // Specific product IDs to display
    categoriaID?: number;                 // Category ID for filtering products
    marcaID?: number;                     // Brand ID for filtering products
    secondaryImagen?: string;             // Secondary image URL (banners, backgrounds)
    iconImagen: string;                   // Icon image URL for decorative elements
    gallery?: IGalleryImagen[];           // Array of gallery images with metadata
    limit?: number;                       // Limit of elements to show (e.g., products in grid)
}
```

### ComponentAST (Complete Definition)

`ComponentAST` extends `ComponentProps`, inheriting all e-commerce properties while providing structure for declarative components.

```typescript
interface ComponentAST extends ComponentProps {
    // === Identity ===
    id?: string | number;                 // Unique identifier for the instance
    tagName: string;                      // HTML element or custom component name
    
    // === Styling ===
    css?: string;                         // Tailwind CSS classes (supports variables and color tokens)
    style?: string;                       // Inline styles (use sparingly)
    
    // === Content ===
    text?: string;                        // Simple text content
    textLines?: ITextLine[];              // Structured text with independent styles
    backgroudImage?: string;              // Background image URL (note: preserved for compatibility)
    
    // === Structure ===
    children?: ComponentAST[];            // Nested components
    slot?: string;                        // Named slot for composition
    
    // === Customization ===
    variables?: ComponentVariable[];      // Editable parameters
    description?: string;                 // Meta-description for library previews
    
    // === HTML Attributes ===
    attributes?: Record<string, string>;  // href, src, alt, target, etc.
    
    // === Interactivity ===
    onClick?: (id: number | string) => void; // Click handler function
    
    // === Accessibility ===
    aria?: AriaAttributes;                // ARIA properties
    semanticTag?: SemanticTag;            // Semantic HTML hint
    
    // === SEO ===
    seo?: SectionSEO;                     // SEO metadata
}

type SemanticTag = 'header' | 'main' | 'footer' | 'nav' | 'article' | 'aside' | 'section' | 'div' | 'span' | 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' | 'a' | 'button' | 'img';
```

### ITextLine

```typescript
interface ITextLine {
    text: string;
    css: string;                          // Specific styles for this line
    tag?: 'span' | 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6';
}
```

---

## 3. Parameterization and Variables

To make a section "editable," we use the `ComponentVariable` object.

### ComponentVariable

```typescript
interface ComponentVariable {
    key: string;              // Token to replace, e.g.: "__v1__"
    defaultValue: string;     // Initial value
    type: string;             // Tailwind prefix, e.g.: "w", "h", "bg", "text", "p", "m", "gap"
    units?: string[];         // Allowed units (px, rem, %, vw, vh, etc.)
    
    // === Validation ===
    min?: number;             // Minimum value (for numeric types)
    max?: number;             // Maximum value (for numeric types)
    
    // === UI Hints ===
    label?: string;           // Human-readable label for editor
    group?: string;           // Group related variables in editor UI
    description?: string;     // Help text for the editor
}
```

### Resolution Mechanism

The `EcommerceRenderer` processes the `css` field by replacing variable keys with their current values.

**Example:**
- Input: `css: "w-[__v1__] h-[__v2__] p-[__v3__]"`
- Variables: `__v1__ = 500px`, `__v2__ = 300px`, `__v3__ = 20px`
- Output: `w-[500px] h-[300px] p-[20px]`

### Variable Types Reference

| Type | Description | Example Values |
|:-----|:------------|:---------------|
| `w` | Width | `100px`, `50%`, `full`, `screen` |
| `h` | Height | `200px`, `auto`, `screen` |
| `p` | Padding | `4`, `20px`, `2rem` |
| `m` | Margin | `4`, `auto`, `-20px` |
| `gap` | Gap | `4`, `16px` |
| `bg` | Background color | `red-500`, `[#ff0000]` |
| `text` | Text color/size | `white`, `xl`, `2xl` |
| `rounded` | Border radius | `lg`, `full`, `[20px]` |
| `shadow` | Box shadow | `md`, `lg`, `2xl` |

---

## 4. Global Colors and Theming

The system supports a global color palette of 10 colors defined by the user (from darkest to lightest).

### Color Tokens

- **Format:** `__COLOR:1__` through `__COLOR:10__`
- **Convention:**
  - `__COLOR:1__` - `__COLOR:3__`: Dark shades (backgrounds, footers)
  - `__COLOR:4__` - `__COLOR:6__`: Mid tones (accents, borders)
  - `__COLOR:7__` - `__COLOR:10__`: Light shades (backgrounds, text on dark)

### Color Palette Interface

```typescript
interface ColorPalette {
    id: string;
    name: string;
    colors: [string, string, string, string, string, string, string, string, string, string];
    // Index 0 = __COLOR:1__, Index 9 = __COLOR:10__
}

// Example palette
const defaultPalette: ColorPalette = {
    id: 'default',
    name: 'Ocean Blue',
    colors: [
        '#0f172a', // 1 - Darkest
        '#1e293b', // 2
        '#334155', // 3
        '#475569', // 4
        '#64748b', // 5
        '#94a3b8', // 6
        '#cbd5e1', // 7
        '#e2e8f0', // 8
        '#f1f5f9', // 9
        '#f8fafc', // 10 - Lightest
    ]
};
```

### Processing

Before rendering, a Regex process searches for these tokens in style properties and replaces them with CSS variables:

```typescript
function resolveColorTokens(css: string, palette: ColorPalette): string {
    return css.replace(/__COLOR:(\d+)__/g, (_, index) => {
        return palette.colors[parseInt(index) - 1] || '#000000';
    });
}
```

---

## 5. Responsive Design Strategy

Responsive design is handled entirely through **Tailwind's built-in responsive prefixes**. No additional abstraction is needed.

### Tailwind Breakpoints

| Prefix | Min Width | Usage |
|:-------|:----------|:------|
| `sm:` | 640px | Small tablets |
| `md:` | 768px | Tablets |
| `lg:` | 1024px | Laptops |
| `xl:` | 1280px | Desktops |
| `2xl:` | 1536px | Large screens |

### Usage in ComponentAST

Simply include responsive prefixes in the `css` field:

```typescript
{
    tagName: 'div',
    css: 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 p-4 md:p-8 lg:p-12'
}

// Text that changes size
{
    tagName: 'h1',
    text: 'Welcome',
    css: 'text-2xl md:text-4xl lg:text-6xl font-bold'
}

// Layout that switches from column to row
{
    tagName: 'div',
    css: 'flex flex-col md:flex-row gap-4 md:gap-8'
}
```

### Mobile-First Approach

Always design mobile-first, then add larger breakpoints:

```typescript
// ✅ Good: Mobile-first
css: 'p-4 md:p-8 lg:p-16'

// ❌ Avoid: Desktop-first (harder to maintain)
css: 'p-16 md:p-8 sm:p-4'
```

---

## 6. Specialized Components (E-commerce Ready)

If the `tagName` matches a registered component, the renderer delegates the logic to that component instead of using a standard HTML element.

### Component Registry

These are pre-built Svelte components that handle their own internal logic (buttons, modals, add-to-cart, etc.). The `ComponentAST` only needs to provide the data properties inherited from `ComponentProps`.

| TagName | Description | Data Props |
|:--------|:------------|:-----------|
| `ProductCard` | Product card with image, price, and add-to-cart button | `productosIDs` |
| `ProductCardHorizontal` | Horizontal product card with description | `productosIDs` |
| `ProductGrid` | Grid of products with optional pagination | `productosIDs`, `categoriaID`, `marcaID`, `limit` |
| `ProductCarousel` | Horizontal scrolling product slider | `productosIDs`, `limit` |
| `CategoryCard` | Category card with image and product count | `categoriaID`, `title` |
| `CategoryGrid` | Grid of category cards | `categoriaID`, `title` |
| `SearchBar` | Product search with autocomplete | - |
| `CartWidget` | Mini cart icon with count and dropdown | - |
| `CartPage` | Full cart page with items list | - |
| `Breadcrumb` | Navigation breadcrumb | - |
| `Gallery` | Image gallery with lightbox | `gallery`, `title`, `iconImagen` |
| `Banner` | Promotional banner with image | `title`, `secondaryImagen`, `iconImagen` |

### How Specialized Components Work

Specialized components are self-contained. They:
- Fetch their own data internally using the provided properties from `ComponentProps`
- Handle all user interactions (add to cart, quantity changes, etc.)
- Manage their own modals and popups
- Apply their own internal styling (can be customized via `css` prop)

```typescript
// Usage in ComponentAST - provide properties from ComponentProps
{
    tagName: 'ProductGrid',
    title: 'Featured Products',     // Section heading
    css: 'my-8',                    // Additional wrapper styles
    productosIDs: [101, 102, 103, 104, 105, 106],
    limit: 8                        // Limit to 8 products
}

// Or filter by category with custom title
{
    tagName: 'ProductGrid',
    title: 'Summer Collection',     // Custom section title
    categoriaID: 5,                 // Shows all products from category 5
    limit: 12,                      // Show up to 12 products
    iconImagen: '/icons/sun.svg'    // Optional icon for decoration
}

// Gallery component with images
{
    tagName: 'Gallery',
    title: 'Product Showcase',
    gallery: [
        { image: '/img1.jpg', title: 'View 1', description: 'Front view' },
        { image: '/img2.jpg', title: 'View 2', description: 'Side view' },
        { image: '/img3.jpg', title: 'View 3', description: 'Detail view' }
    ],
    iconImagen: '/icons/camera.svg'
}
```

---

## 7. Content Editing: ITextLine

For components requiring mixed typography (e.g., a Hero with a bold title and a thin subtitle), `ITextLine` is used.

### ITextLine Interface

```typescript
interface ITextLine {
    text: string;
    css: string;                    // Tailwind classes for this line
    tag?: TextTag;                  // HTML tag to use
}

type TextTag = 'span' | 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' | 'strong' | 'em';
```

> **Note**: For links within text content, use a separate `<a>` element as a child instead of `textLines`.

### Example Usage

```typescript
{
    tagName: 'div',
    css: 'text-center space-y-4',
    textLines: [
        { text: 'Summer Sale', tag: 'span', css: 'text-sm uppercase tracking-widest text-[__COLOR:5__]' },
        { text: 'Up to 50% Off', tag: 'h1', css: 'text-5xl font-bold text-[__COLOR:1__]' },
        { text: 'Limited time offer on selected items', tag: 'p', css: 'text-lg text-[__COLOR:4__]' }
    ]
}
```

### Line Editor Requirements

The editor component should allow users to:
1. Add, remove, and reorder text blocks
2. Edit the content of each block
3. Select the HTML tag (h1-h6, p, span)
4. Configure styles via UI controls:
   - Font size (text-sm, text-base, text-lg, text-xl, etc.)
   - Font weight (font-normal, font-medium, font-bold)
   - Color (from palette or custom)
   - Alignment (text-left, text-center, text-right)
   - Spacing (tracking, leading)

---

## 8. Component Library

Each section template must reside in an independent `.ts` file within the library folder.

### Section Template Structure

```typescript
interface SectionTemplate {
    // === Metadata ===
    id: string;                       // Unique template ID
    name: string;                     // Display name
    category: SectionCategory;        // For organization
    description: string;              // For library preview
    thumbnail?: string;               // Preview image URL
    
    // === Template ===
    ast: ComponentAST;                // The actual component tree
    
    // === Variants ===
    presets?: SectionPreset[];        // Predefined variable combinations
}

type SectionCategory = 
    | 'hero'
    | 'products'
    | 'categories'
    | 'testimonials'
    | 'features'
    | 'cta'
    | 'footer'
    | 'header'
    | 'gallery'
    | 'text';

interface SectionPreset {
    id: string;
    name: string;                     // e.g., "Dark Mode", "Minimal", "Bold"
    variables: Record<string, string>;
}
```

### Example Template (`hero-centered.ts`)

```typescript
import type { SectionTemplate } from './types';

export const HeroCentered: SectionTemplate = {
    id: 'hero-centered-v1',
    name: 'Hero - Centered Text',
    category: 'hero',
    description: 'Full-width hero section with centered text and CTA button.',
    
    ast: {
        tagName: 'section',
        css: 'bg-[__COLOR:2__] py-[__v1__] px-4',
        variables: [
            { key: '__v1__', type: 'py', defaultValue: '80px', label: 'Vertical Padding', min: 40, max: 200 }
        ],
        children: [
            {
                tagName: 'div',
                css: 'max-w-4xl mx-auto text-center',
                textLines: [
                    { text: 'Welcome to Our Store', tag: 'h1', css: 'text-5xl font-bold text-[__COLOR:10__] mb-4' },
                    { text: 'Discover amazing products at great prices', tag: 'p', css: 'text-xl text-[__COLOR:7__] mb-8' }
                ],
                children: [
                    {
                        tagName: 'a',
                        text: 'Shop Now',
                        css: 'inline-block bg-[__COLOR:6__] text-[__COLOR:1__] px-8 py-3 rounded-lg font-semibold hover:bg-[__COLOR:7__] transition-colors',
                        attributes: { href: '/productos' }
                    }
                ]
            }
        ]
    },
    
    presets: [
        {
            id: 'minimal',
            name: 'Minimal',
            variables: { '__v1__': '60px' }
        },
        {
            id: 'bold',
            name: 'Bold & Spacious',
            variables: { '__v1__': '120px' }
        }
    ]
};
```

### Library Registration

```typescript
import { HeroCentered } from './templates/hero-centered';
import { ProductGridFeatured } from './templates/product-grid-featured';

// Register all templates
const sectionLibrary: SectionTemplate[] = [
    HeroCentered,
    ProductGridFeatured,
    // ... more templates
];

export function getSectionsByCategory(category: SectionCategory): SectionTemplate[] {
    return sectionLibrary.filter(s => s.category === category);
}

export function getSectionById(id: string): SectionTemplate | undefined {
    return sectionLibrary.find(s => s.id === id);
}
```

---

## 9. Accessibility (A11Y)

### AriaAttributes Interface

```typescript
interface AriaAttributes {
    label?: string;           // aria-label
    labelledBy?: string;      // aria-labelledby (ID reference)
    describedBy?: string;     // aria-describedby (ID reference)
    role?: AriaRole;
    hidden?: boolean;         // aria-hidden
    live?: 'polite' | 'assertive' | 'off';
    expanded?: boolean;       // For toggleable elements
    controls?: string;        // ID of controlled element
}

type AriaRole = 
    | 'button' | 'link' | 'navigation' | 'main' | 'banner'
    | 'contentinfo' | 'complementary' | 'region' | 'list'
    | 'listitem' | 'img' | 'dialog' | 'alert';
```

### Accessibility Guidelines

1. **Semantic HTML**: Use appropriate `semanticTag` values
2. **Heading Hierarchy**: Ensure proper H1 → H6 order
3. **Alt Text**: All images must have `attributes.alt`
4. **Focus Management**: Interactive elements must be keyboard accessible
5. **Color Contrast**: Ensure sufficient contrast between color tokens
6. **Screen Reader**: Use `aria` properties for non-obvious interactions

### Example with Accessibility

```typescript
{
    tagName: 'section',
    semanticTag: 'section',
    aria: { label: 'Featured Products', role: 'region' },
    css: 'py-16',
    children: [
        {
            tagName: 'h2',
            text: 'Featured Products',
            css: 'text-3xl font-bold mb-8'
        },
        {
            tagName: 'ProductGrid',
            aria: { role: 'list' },
            productosIDs: [1, 2, 3, 4]
        }
    ]
}
```

---

## 10. SEO Considerations

### SectionSEO Interface

```typescript
interface SectionSEO {
    headingLevel?: 1 | 2 | 3 | 4 | 5 | 6;  // Ensure proper hierarchy
    structuredData?: StructuredDataType;
    priority?: number;                       // For internal importance ranking
    indexable?: boolean;                     // Should content be indexed
}

type StructuredDataType = 
    | 'Product'
    | 'ProductList'
    | 'BreadcrumbList'
    | 'Organization'
    | 'FAQPage'
    | 'Review';
```

### SEO Best Practices

1. **Single H1**: Only one H1 per page (usually in hero)
2. **Structured Data**: Add JSON-LD for product sections
3. **Image Optimization**: Use WebP, lazy loading, proper dimensions
4. **Core Web Vitals**: Minimize layout shift, optimize LCP

### Structured Data Generation

```typescript
function generateProductListSchema(products: Product[]): object {
    return {
        '@context': 'https://schema.org',
        '@type': 'ItemList',
        itemListElement: products.map((p, i) => ({
            '@type': 'ListItem',
            position: i + 1,
            item: {
                '@type': 'Product',
                name: p.name,
                image: p.image,
                offers: {
                    '@type': 'Offer',
                    price: p.price,
                    priceCurrency: 'PEN'
                }
            }
        }))
    };
}
```

---

## 11. Implementation Phases

### Phase 1: Core Foundation (Week 1-2)
- [ ] Define complete TypeScript interfaces in `types.ts`
- [ ] Implement `EcommerceRenderer.svelte` with recursive rendering
- [ ] Create token replacement engine (variables + colors)
- [ ] Implement CSS variable system for theming
- [ ] Basic error handling and validation

### Phase 2: Specialized Components (Week 2-3)
- [ ] Create `ProductCard.svelte` with add-to-cart functionality
- [ ] Create `ProductGrid.svelte` with layout options
- [ ] Create `CategoryCard.svelte` and `CategoryGrid.svelte`
- [ ] Create `CartWidget.svelte` and `SearchBar.svelte`
- [ ] Create 5-10 starter section templates

### Phase 3: Editor System (Week 3-4)
- [ ] Build variable editor UI (sliders, inputs, dropdowns)
- [ ] Implement `ITextLine` editor with WYSIWYG controls
- [ ] Create color palette picker component
- [ ] Add live preview system
- [ ] Implement section reordering (drag-and-drop)

### Phase 4: Persistence & State (Week 4-5)
- [ ] Define JSON serialization format
- [ ] Implement save/load functionality
- [ ] Create undo/redo system
- [ ] Add section versioning and migration
- [ ] Implement template duplication

### Phase 5: Polish & Optimization (Week 5-6)
- [ ] Add responsive preview modes (mobile/tablet/desktop)
- [ ] Generate thumbnails for library
- [ ] Implement lazy loading for sections
- [ ] Add animation support
- [ ] Performance optimization and testing

---

## 12. Critical Considerations

### Performance

1. **Memoization**: Cache rendered sections that haven't changed
2. **Virtualization**: For pages with many sections, render only visible ones
3. **Lazy Loading**: Load below-fold sections on scroll
4. **Image Optimization**: Use srcset, lazy loading, WebP format
5. **Bundle Size**: Code-split specialized components

```typescript
// Example: Lazy component loading
const componentMap = {
    ProductCard: () => import('./components/ProductCard.svelte'),
    ProductGrid: () => import('./components/ProductGrid.svelte'),
};
```

### Security

1. **Input Sanitization**: Never trust user-provided CSS/HTML
2. **Whitelist Classes**: Only allow known Tailwind classes
3. **No `{@html}`**: Avoid raw HTML rendering without sanitization
4. **URL Validation**: Validate all URLs in `href` attributes

```typescript
const ALLOWED_CSS_PATTERNS = [
    /^(w|h|p|m|gap|text|bg|border|rounded|shadow|flex|grid)-/,
    /^(hover|focus|active|md|lg|xl):/,
];

function sanitizeCss(css: string): string {
    return css.split(' ')
        .filter(cls => ALLOWED_CSS_PATTERNS.some(p => p.test(cls)))
        .join(' ');
}
```

### Migration & Versioning

1. **Schema Version**: Include version in saved data
2. **Migration Functions**: Create upgraders for each version change
3. **Backward Compatibility**: Keep deprecated fields for grace period

```typescript
interface SavedSection {
    schemaVersion: number;
    ast: ComponentAST;
    metadata: {
        createdAt: string;
        updatedAt: string;
    };
}

const migrations: Record<number, (data: any) => any> = {
    1: (data) => ({ ...data, schemaVersion: 2, ast: migrateV1toV2(data.ast) }),
    2: (data) => ({ ...data, schemaVersion: 3, ast: migrateV2toV3(data.ast) }),
};
```

### Testing Strategy

1. **Unit Tests**: Test token resolution, validation, sanitization
2. **Component Tests**: Test each specialized component in isolation
3. **Snapshot Tests**: Capture rendered output for regression detection
4. **E2E Tests**: Test full save → render → edit cycle
5. **Visual Regression**: Compare screenshots across changes

---

## 13. Alternative Strategies

### Strategy A: Slot-Based Composition

Instead of pure AST, use predefined slots:

```typescript
interface SlotBasedTemplate {
    id: string;
    layout: string;  // CSS grid template
    slots: {
        [name: string]: {
            accepts: string[];
            maxItems?: number;
            required?: boolean;
        };
    };
}

// Example
const heroTemplate: SlotBasedTemplate = {
    id: 'hero-split',
    layout: 'grid md:grid-cols-2 gap-8',
    slots: {
        media: { accepts: ['Image', 'Video'], maxItems: 1, required: true },
        content: { accepts: ['TextBlock', 'Button'], maxItems: 5 },
    }
};
```

**Pros**: More intuitive drag-drop, clearer constraints, less error-prone  
**Cons**: Less flexible, more templates needed

### Strategy B: Design Tokens System

Replace inline variables with semantic tokens:

```typescript
const designTokens = {
    spacing: {
        'section-y': 'py-16 md:py-24',
        'content-gap': 'gap-8',
    },
    typography: {
        'heading-hero': 'text-4xl md:text-6xl font-bold',
        'body-large': 'text-lg leading-relaxed',
    }
};

// Usage
{
    css: '$spacing.section-y $typography.heading-hero'
}
```

**Pros**: More semantic, easier global updates, better theming  
**Cons**: Less granular control, steeper learning curve

### Strategy C: Visual Builder First

Build visual editor before AST system:

1. Users drag pre-built blocks
2. Configure via visual controls only
3. System generates AST internally
4. Advanced users can edit AST directly

**Pros**: Better UX for non-technical users  
**Cons**: More initial development, harder to debug

---

## 14. Open Questions

These decisions should be made before implementation:

1. **State Management**: Svelte stores vs context vs props drilling?
2. **Persistence**: Local storage vs backend API vs both?
3. **Versioning**: How to handle template updates for existing stores?
4. **Multi-language**: Support for i18n in text content?
5. **Animations**: CSS-only vs Svelte transitions?
6. **Mobile Editor**: Will users edit on mobile devices?
7. **Preview**: Separate preview mode or inline editing?
8. **Export**: Can users export their store as static HTML?

---

## Appendix: Example Complete Section

```typescript
const featuredProductsSection: ComponentAST = {
    id: 'section-featured-products',
    tagName: 'section',
    semanticTag: 'section',
    css: 'bg-[__COLOR:9__] py-[__v1__] px-4',
    aria: { label: 'Featured Products' },
    seo: { headingLevel: 2, structuredData: 'ProductList' },
    
    variables: [
        { 
            key: '__v1__', 
            type: 'py', 
            defaultValue: '64px',
            label: 'Section Padding',
            min: 32,
            max: 128,
            step: 8
        }
    ],
    
    children: [
        {
            tagName: 'div',
            css: 'max-w-7xl mx-auto',
            children: [
                {
                    tagName: 'ProductGrid',
                    title: 'Featured Products',  // Using title from ComponentProps
                    css: 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6',
                    productosIDs: [101, 102, 103, 104, 105, 106, 107, 108],
                    limit: 8                      // Limit displayed products
                },
                {
                    tagName: 'div',
                    css: 'text-center mt-12',
                    children: [
                        {
                            tagName: 'a',
                            text: 'View All Products →',
                            css: 'inline-block text-[__COLOR:3__] font-medium hover:text-[__COLOR:2__] transition-colors',
                            attributes: { href: '/productos' }
                        }
                    ]
                }
            ]
        }
    ]
};

// Example with Gallery component
const productGallerySection: ComponentAST = {
    id: 'section-product-gallery',
    tagName: 'section',
    title: 'Product Gallery',  // Section title
    css: 'py-16 px-4 bg-white',
    
    children: [
        {
            tagName: 'Gallery',
            title: 'Product Views',
            iconImagen: '/icons/gallery.svg',
            gallery: [
                { 
                    image: '/products/1/front.jpg', 
                    title: 'Front View', 
                    description: 'Product from the front' 
                },
                { 
                    image: '/products/1/side.jpg', 
                    title: 'Side View', 
                    description: 'Product from the side' 
                },
                { 
                    image: '/products/1/detail.jpg', 
                    title: 'Detail View', 
                    description: 'Close-up detail' 
                }
            ]
        }
    ]
};
```
