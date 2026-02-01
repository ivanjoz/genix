import type { IProducto } from '$services/services/productos.svelte';

export type AriaRole = 
    | 'button' | 'link' | 'navigation' | 'main' | 'banner'
    | 'contentinfo' | 'complementary' | 'region' | 'list'
    | 'listitem' | 'img' | 'dialog' | 'alert';

export interface AriaAttributes {
    label?: string;
    labelledBy?: string;
    describedBy?: string;
    role?: AriaRole;
    hidden?: boolean;
    live?: 'polite' | 'assertive' | 'off';
    expanded?: boolean;
    controls?: string;
}

export type SemanticTag = 'header' | 'main' | 'footer' | 'nav' | 'article' | 'aside' | 'section' | 'div' | 'span' | 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' | 'a' | 'button' | 'img';

export type TextTag = 'span' | 'p' | 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6' | 'strong' | 'em';

export interface ITextLine {
    text: string;
    css: string;
    tag?: TextTag;
}

export interface ComponentVariable {
    key: string;
    defaultValue: string;
    type: string;
    units?: string[];
    min?: number;
    max?: number;
    step?: number;
    label?: string;
    group?: string;
    description?: string;
}

export type StructuredDataType = 
    | 'Product'
    | 'ProductList'
    | 'BreadcrumbList'
    | 'Organization'
    | 'FAQPage'
    | 'Review';

export interface SectionSEO {
    headingLevel?: 1 | 2 | 3 | 4 | 5 | 6;
    structuredData?: StructuredDataType;
    priority?: number;
    indexable?: boolean;
}

export interface ComponentAST {
    id?: string | number;
    tagName: string;
    css?: string;
    style?: string;
    text?: string;
    textLines?: ITextLine[];
    backgroudImage?: string; // Typo preserved for compatibility if needed, or check EcommerceRenderer.svelte
    children?: ComponentAST[];
    slot?: string;
    variables?: ComponentVariable[];
    description?: string;
    onClick?: (id: number | string) => void;
    attributes?: Record<string, string>;
    productosIDs?: number[];
    productos?: IProducto[];
    categoriaID?: number;
    marcaID?: number;
    aria?: AriaAttributes;
    semanticTag?: SemanticTag;
    seo?: SectionSEO;
}

export interface ColorPalette {
    id: string;
    name: string;
    colors: [string, string, string, string, string, string, string, string, string, string];
}
