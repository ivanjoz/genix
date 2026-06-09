import type { ITextLine, IGalleryImagen, SectionCategory, ComponentAST, SectionPreset } from './renderer-types';

/**
 * Standard content properties for AI-friendly flat schema.
 * Components pick which of these they use via their exported schema.
 */
export interface StandardContent {
  title?: string;
  subTitle?: string;
  description?: string;
  
  // Specific text placements
  textLeft?: string;
  textCenter?: string;
  textRight?: string;
  
  // Rich content
  textLines?: ITextLine[];
  
  // Media
  image?: string;
  secondaryImagen?: string;
  iconImagen?: string;
  bgImage?: string;
  videoUrl?: string;
  
  // E-commerce & Lists
  productIDs?: number[];
  categoryIDs?: number[];
  brandIDs?: number[];
  gallery?: IGalleryImagen[];
  limit?: number;
  
  // Actions
  primaryActionLabel?: string;
  primaryActionHref?: string;
  secondaryActionLabel?: string;
  secondaryActionHref?: string;
  
  // Custom catch-all for flexible components
  [key: string]: any;
}

/**
 * The single data structure for a Section — describes both an authoring template
 * (HTML source + metadata) and a placed section instance (content/ast + css).
 * Fields that apply to only one of those roles are optional.
 */
export interface SectionData {
  id: string;
  type?: string; // Component name, e.g. 'HeroStandard'. HTML templates resolve to 'HtmlSection' when added.
  category?: SectionCategory;

  // Template metadata (authoring side).
  name?: string;
  description?: string;
  thumbnail?: string;
  presets?: SectionPreset[];

  // HTML sections: `html` is the authoring source, parsed once at add/load time into
  // `ast` (the canonical, editable model the renderer/editor/CSS all read).
  html?: string;
  ast?: ComponentAST[];

  // Component sections: flat content fields.
  content?: StandardContent;

  css?: Record<string, string>; // Slot-based CSS, e.g., { container: "...", title: "..." }
  attributes?: Record<string, any>;
}

/**
 * Props every section component receives from the renderer. A component reads only
 * the fields it needs (`content` for component sections, `ast` for the HTML section).
 */
export interface SectionProps {
  content?: StandardContent;
  ast?: ComponentAST[];
  css?: Record<string, string>;
  [key: string]: any; // attributes passthrough
}

/**
 * Metadata exported by each component to define its editable interface.
 */
export interface SectionSchema {
  name: string;
  description?: string;
  category: SectionCategory;
  content: (keyof StandardContent)[];
  css: string[]; // List of available slots for classes
}
