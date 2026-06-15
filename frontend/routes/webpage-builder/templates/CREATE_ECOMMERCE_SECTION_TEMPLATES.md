# Creating Ecommerce Section Templates

## Quick Start

A section template is a reusable component definition that combines structure, content, and e-commerce functionality.

```typescript
import type { SectionData } from '../../renderer/section-types';

export const MySection: SectionData = {
    id: 'my-section-v1',
    name: 'My Section',
    category: 'products',
    description: 'Description of what this section does',
    html: `<!-- raw HTML with native tags + custom component tags -->`,
    presets: [ /* Optional presets */ ]
};
```

A template is just a `SectionData` (the single section interface) used in its authoring
role. Author the markup as the `html` string — when the template is added it is parsed once
into `ast` (a `ComponentAST[]`), which becomes the canonical, editable model the
renderer/editor/CSS all read. Don't hand-author `ast`.

## Template Structure

### Required Properties

- **id**: Unique identifier (use kebab-case with version suffix)
- **name**: Display name for the UI
- **category**: One of: `hero`, `products`, `categories`, `testimonials`, `features`, `cta`, `footer`, `header`, `gallery`, `text`
- **description**: Brief explanation of the section's purpose
- **html**: The raw HTML source defining the section (parsed to `ast` on add)

### Optional Properties

- **thumbnail**: Trusted miniature HTML rendered in the builder's template picker
- **presets**: Pre-configured variations

## Using ComponentProps

All ComponentAST nodes inherit these e-commerce properties:

```typescript
{
    title: string,              // Section/component heading
    productIDs?: number[],    // Specific products to display
    categoriaID?: number,       // Filter by category
    marcaID?: number,          // Filter by brand
    secondaryImagen?: string,   // Secondary image (banners)
    iconImagen: string,         // Icon for decoration
    gallery?: IGalleryImagen[], // Image gallery
    limit?: number             // Max items to show
}
```

### Example: Product Grid with Title and Limit

```typescript
{
    tagName: 'ProductGrid',
    title: 'Featured Products',
    productIDs: [1, 2, 3, 4],
    limit: 4,
    css: 'grid grid-cols-4 gap-6'
}
```

### Example: Gallery Component

```typescript
{
    tagName: 'Gallery',
    title: 'Product Views',
    iconImagen: '/icons/camera.svg',
    gallery: [
        { image: '/img1.jpg', title: 'Front', description: 'Front view' },
        { image: '/img2.jpg', title: 'Side', description: 'Side view' }
    ]
}
```

## Specialized Components

### Product Components

| Component | Required Props | Optional Props |
|-----------|---------------|----------------|
| ProductCard | `productIDs` | `title`, `limit` |
| ProductGrid | `productIDs` or `categoriaID` or `marcaID` | `title`, `limit`, `iconImagen` |
| ProductCarousel | `productIDs` | `title`, `limit` |

### Category Components

| Component | Required Props | Optional Props |
|-----------|---------------|----------------|
| CategoryCard | `categoriaID` | `title`, `secondaryImagen` |
| CategoryGrid | `categoriaID` | `title`, `iconImagen` |

### Media Components

| Component | Required Props | Optional Props |
|-----------|---------------|----------------|
| Gallery | `gallery` | `title`, `iconImagen` |
| Banner | `title` | `secondaryImagen`, `iconImagen` |

## Defining Variables

Variables allow users to customize section properties:

```typescript
variables: [
    {
        key: '__v1__',
        type: 'padding',
        defaultValue: '64px',
        label: 'Section Padding',
        min: 32,
        max: 128,
        step: 8
    },
    {
        key: '__COLOR:2__',
        type: 'color',
        defaultValue: '#FF5733',
        label: 'Primary Color'
    }
]
```

Use variables in CSS with the `__` prefix:

```typescript
css: 'py-[__v1__] bg-[__COLOR:2__]'
```

## Creating Presets

Presets provide pre-configured variations:

```typescript
presets: [
    {
        id: 'minimal',
        name: 'Minimal',
        variables: {
            '__v1__': '40px',
            '__COLOR:2__': '#000000'
        }
    },
    {
        id: 'bold',
        name: 'Bold',
        variables: {
            '__v1__': '120px',
            '__COLOR:2__': '#FF0000'
        }
    }
]
```

## Complete Example

```typescript
import type { SectionData } from '../../renderer/section-types';

export const ProductShowcase: SectionData = {
    id: 'product-showcase-v1',
    name: 'Product Showcase',
    category: 'products',
    description: 'Featured products with title and optional category filter',
    // Author the markup as HTML. `data-role` marks editable nodes for the builder;
    // `color`/`background-color` attributes map to palette CSS vars; custom component
    // tags (e.g. <ProductsByCategory>) coerce their attributes into props.
    html: `
        <section class="py-16 px-4 bg-white">
            <div class="max-w-7xl mx-auto">
                <h2 data-role="title" color="9" class="text-3xl font-bold text-center mb-10">
                    Featured Products
                </h2>
                <ProductsByCategory categoryID="1" maxWidth="1680" rows="2" rowsMobile="3" />
            </div>
        </section>
    `,
};
```

## Best Practices

1. **Use kebab-case with version** for IDs: `hero-centered-v1`
2. **Always provide defaults** for variables
3. **Limit to 3-5 presets** to avoid choice paralysis
4. **Use semantic HTML** for accessibility (header, nav, section)
5. **Make sections responsive** with Tailwind breakpoints
6. **Provide clear descriptions** for library previews
7. **Use color tokens** for consistent theming: `__COLOR:1__`
8. **Test with data** - ensure sections work with real products/categories
9. **Keep AST flat** when possible - avoid unnecessary nesting
10. **Document intent** with descriptive `tagName` values

## Common Patterns

### Hero Section with Banner

```typescript
{
    tagName: 'Banner',
    title: 'Summer Sale',
    secondaryImagen: '/banner.jpg',
    iconImagen: '/icons/sun.svg',
    css: 'h-96 bg-cover bg-center'
}
```

### Category Section with Grid

```typescript
{
    tagName: 'section',
    children: [
        {
            tagName: 'CategoryGrid',
            title: 'Shop by Category',
            categoriaID: 0, // 0 = all categories
            limit: 6,
            css: 'grid grid-cols-2 md:grid-cols-3 gap-4'
        }
    ]
}
```

### Product Carousel with Title

```typescript
{
    tagName: 'ProductCarousel',
    title: 'New Arrivals',
    productIDs: [101, 102, 103, 104, 105],
    limit: 5,
    iconImagen: '/icons/new.svg',
    css: 'overflow-x-auto'
}
```
