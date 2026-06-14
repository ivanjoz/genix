import type { SectionCategory, ComponentAST, SectionPreset } from './renderer-types';

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
  Type?: string; // The only supported runtime type is 'HtmlSection'.

  // Authoring/builder-only metadata; NOT persisted (kept lowercase).
  category?: SectionCategory;
  name?: string;
  description?: string;
  /** Trusted miniature HTML used by the builder's template picker. */
  thumbnail?: string;
  presets?: SectionPreset[];

  // HTML sections: `html` is the authoring source (lowercase, not persisted),
  // parsed once at add/load time into `Ast` (the canonical, editable model the
  // renderer/editor/CSS all read). AST node fields stay frontend-owned/lowercase.
  html?: string;
  Ast?: ComponentAST[];

  Css?: Record<string, string>; // Slot-based CSS, e.g., { container: "...", title: "..." }
  Attributes?: Record<string, any>;
}

/**
 * Props received by the HTML section renderer.
 */
export interface SectionProps {
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
  css: string[]; // List of available slots for classes
}
