import type { ITextLine, IGalleryImagen, SectionCategory } from './renderer-types';

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
  productosIDs?: number[];
  categoriasIDs?: number[];
  marcasIDs?: number[];
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
 * The unified data structure for a Section.
 */
export interface SectionData {
  id: string;
  type: string; // The component name, e.g., 'Hero'
  category?: SectionCategory;
  content: StandardContent;
  css: Record<string, string>; // Slot-based CSS, e.g., { container: "...", title: "..." }
  attributes?: Record<string, any>;
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
