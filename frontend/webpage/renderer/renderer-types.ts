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

export interface IGalleryImagen {
	image: string
	title: string
	description: string
}

/**
 * A node in the parsed HTML section AST. Structural only — section content
 * fields (products, titles, etc.) live on `StandardContent` in section-types.ts.
 */
export interface ComponentAST {
	tagName: string;
	css?: string;
	/** Inline style compiled from color attributes (palette tokens -> var(--color-N)). */
	style?: string;
	text?: string;
	children?: ComponentAST[];
	/** Editable role for the builder (from `data-role`), e.g. 'title' | 'content' | 'button'. */
	role?: string;
	/** Coerced props for custom components (tagName starting uppercase). */
	props?: Record<string, any>;
	attributes?: Record<string, string>;
}

export interface ColorPalette {
	id: string;
	name: string;
	colors: [string, string, string, string, string, string, string, string, string, string];
}

export type SectionCategory =
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

export interface SectionPreset {
	id: string;
	name: string;
	variables: Record<string, string>;
}

