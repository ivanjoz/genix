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
  // Runtime-only identity (a fresh uuid is assigned on load); NOT persisted.
  id: string;
  // PascalCase fields below mirror the backend SectionContent struct
  // (backend/ecommerce/types/page_content.go) and are persisted as-is.
  Type?: string; // Component name, e.g. 'HeroStandard'. HTML templates resolve to 'HtmlSection' when added.

  // Authoring/builder-only metadata; NOT persisted (kept lowercase).
  category?: SectionCategory;
  name?: string;
  description?: string;
  thumbnail?: string;
  presets?: SectionPreset[];

  // HTML sections: `html` is the authoring source (lowercase, not persisted),
  // parsed once at add/load time into `Ast` (the canonical, editable model the
  // renderer/editor/CSS all read). AST node fields stay frontend-owned/lowercase.
  html?: string;
  Ast?: ComponentAST[];

  // Component sections: flat content fields.
  Content?: StandardContent;

  Css?: Record<string, string>; // Slot-based CSS, e.g., { container: "...", title: "..." }
  Attributes?: Record<string, any>;
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
