# Reusable Ecommerce Section System

This system enables the creation, management, and rendering of dynamic and highly customizable User Interface (UI) sections for building ecommerce stores. The goal is to provide a component library that users can select, configure, and save to generate their own store.

## System Architecture

The system is divided into three main layers:

1.  **Definition (ComponentAST):** A data structure that describes the content, behavior, and styling of a section.
2.  **Rendering (EcommerceRenderer.svelte):** The engine responsible for interpreting the `ComponentAST` and converting it into DOM elements or Svelte components.
3.  **Component Library:** A collection of `.ts` files that serve as base templates (Blueprints) for the sections.

---

## 1. Data Structure (ComponentAST)

Each section is defined using the `ComponentAST` interface. This structure is recursive, allowing for complex compositions.

### Main Properties:
- `id`: Unique identifier for the instance.
- `tagName`: Defines the HTML element (div, section, button) or the name of a custom component (e.g., `ProductCard`).
- `css`: Tailwind CSS classes that define the style. Can contain variables and color tokens.
- `description`: Meta-description of the layout (e.g., "Hero banner with image on the left and CTA on the right"). Used to generate previews in the library.
- `text` / `textLines`: Simple or structured text content (via lines with independent styles).
- `variables`: A list of `ComponentVariable` objects for CSS class parameterization.
- `children`: An array of `ComponentAST` for nesting.

---

## 2. Parameterization and Variables

To make a section "editable," we use the `ComponentVariable` object.

### ComponentVariable
```typescript
interface ComponentVariable {
    key: string;          // Token to replace, e.g.: "__v1__"
    defaultValue: string; // Initial value
    type: string;         // Tailwind prefix, e.g.: "w", "h", "bg", "text"
    units?: string[];     // Allowed units (px, rem, %, etc.)
}
```

**Resolution Mechanism:**
The `EcommerceRenderer` processes the `css` field by replacing variable keys with their current values.
*Example:* If `css` is `w-[__v1__] h-10` and `__v1__` is `500px`, the result will be `w-[500px] h-10`.

---

## 3. Global Colors and Theming

The system supports a global color palette of 10 colors defined by the user (from darkest to lightest).

- **Tokens:** Used via the format `__COLOR:1__` through `__COLOR:10__`.
- **Processing:** Before rendering, a Regex process searches for these tokens in style properties and replaces them with the hexadecimal values or CSS variables of the active palette.

---

## 4. Specialized Components (E-commerce Ready)

If the `tagName` matches a registered component, the renderer delegates the logic to that component instead of using a standard HTML element.

| TagName | Description | Required Data |
| :--- | :--- | :--- |
| `ProductCard` | Standard product card. | `productos` or `productosIDs` |
| `ProductCardHorizontal` | Horizontal version of the card. | `productos` or `productosIDs` |
| `ProductGrid` | Automatic product grid. | `categoriaID` or `marcaID` |

---

## 5. Content Editing: ITextLine[]

For components requiring mixed typography (e.g., a Hero with a bold title and a thin subtitle), `ITextLine` is used.

```typescript
interface ITextLine {
    text: string;
    css: string; // Specific styles for this line
}
```

**Line Editor:**
A form component should be implemented to allow the user to:
1. Add or remove text blocks.
2. Edit the content of each block.
3. Configure basic styles (size, color, alignment) that will be translated into Tailwind classes.

---

## 6. Component Library

Each section template must reside in an independent `.ts` file within the library folder.

### Definition Example (`hero-section.ts`):
```typescript
export const HeroSection: ComponentAST = {
    tagName: 'section',
    description: 'Hero section with solid background, centered text, and buy button.',
    css: 'bg-[__COLOR:2__] p-20 text-center',
    variables: [
        { key: '__v1__', type: 'p', defaultValue: '20' }
    ],
    children: [
        { tagName: 'h1', text: 'Welcome to our store', css: 'text-4xl font-bold' },
        { tagName: 'ProductCard', css: 'mt-10', productosIDs: [1, 2, 3] }
    ]
};
```

### Library Registration
The `libraryAddSectionComponent(e: ComponentAST)` function is used to register these templates, making them available in the user's selection panel.

---

## Proposed Improvements and Roadmap

1.  **Snapshot Generation:** Use the `description` property along with a headless rendering engine to generate automatic thumbnails for each section.
2.  **Style Presets:** Allow the same section (`ComponentAST`) to have multiple "skins" or predefined variable presets.
3.  **Animations:** Integrate an `animation` property in `ComponentAST` to handle smooth transitions using libraries like `framer-motion` or `svelte-transitions`.
4.  **Drag & Drop:** Implement the ability to reorder sections within the `EcommerceRenderer` via a visual interface.
