# Plan: unify `SectionData` + `SectionTemplate`, hoist `html`/`ast` to the section root

## Goal
One interface (`SectionData`) describes both a placed section and an authoring template.
The HTML-section fields `html`/`ast` move **out of `content`** to the section **root**, so
`content` is reserved for the flat component-section fields (`StandardContent`).

## Unified `SectionData` (section-types.ts)
```ts
export interface SectionData {
  id: string;
  type?: string;                 // component name; HTML templates resolve to 'HtmlSection' on add
  name?: string;                 // template: display name
  category?: SectionCategory;
  description?: string;          // template: description
  thumbnail?: string;            // template: thumbnail
  presets?: SectionPreset[];     // template: presets
  html?: string;                 // HTML sections: authoring source (parsed once -> ast)
  ast?: ComponentAST[];          // HTML sections: canonical editable model
  content?: StandardContent;     // component sections: flat fields
  css?: Record<string, string>;  // slot-based classes
  attributes?: Record<string, any>;
}
```
- `SectionTemplate` is **deleted** from renderer-types.ts (it becomes `SectionData`).
  `SectionPreset` stays in renderer-types; section-types imports it.
- Template-only and instance-only fields are optional → a few guards needed downstream.

## Shared component props (section-types.ts)
The two renderers pass the same props to every section component, so introduce one shape:
```ts
export interface SectionProps {
  content?: StandardContent;     // component sections
  ast?: ComponentAST[];          // HTML section
  css?: Record<string, string>;
  [key: string]: any;            // attributes passthrough
}
```

## Edits

1. **section-types.ts** — add `SectionProps`; rewrite `SectionData` as above (import
   `ComponentAST`, `SectionPreset` from renderer-types). Remove `html`/`ast` from
   `StandardContent`.
2. **renderer-types.ts** — delete `SectionTemplate`.
3. **Both renderers** (`BuilderSectionRender.svelte`, `EcommerceRenderer.svelte`) — pass
   `content`/`ast`/`css`/`{...attributes}`; guard the registry lookup on optional `type`:
   `const Config = section.type ? SectionRegistry[section.type] : undefined`.
4. **4 section components** — type props as `SectionProps`:
   - `HtmlSection.svelte`: `let { ast = [], css = {} }: SectionProps` → `<AstRenderer nodes={ast} />`.
   - `HeroStandard` / `FeatureListSimple` / `ProductGridSimple`:
     `let { content = {}, css = {} }: SectionProps` (StandardContent fields already optional).
5. **editor.svelte.ts** — addSection HTML path stores root `ast` (`ast: parseHTML(t.html)`);
   lookup uses `t.html`. `updateContent`/`updateCss` init the optional object:
   `(section.content ??= {})[key] = value`.
6. **EcommerceBuilder.svelte** — normalize root `html` → root `ast`.
7. **uno-generator.ts** — `collectTokens` reads `section.ast` (root), not `content.ast`.
8. **EditorTab.svelte** — role/category nodes read `section?.ast`; guard optional
   `content?.`/`css?.` accesses.
9. **4 ecommerce-templates** + **parse-html.test.ts** + **store-example.ts** — rename
   template field `HTML` → `html`; typed as `SectionData`. store-example HTML entries use
   root `html` (no `content`).
10. **Docs** — update `CREATE_ECOMMERCE_SECTION_TEMPLATES.md` / `ECOMMERCE_SECTIONS.md`
    interface references (`SectionTemplate`→`SectionData`, `HTML`→`html`).

## Verify
`npm run check` clean; add each plantilla → renders; GUARDAR prints root `ast`; editing works.
```
