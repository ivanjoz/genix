import type { SectionData } from '$ecommerce/renderer/section-types';
import { HtmlHeroBanner } from '$ecommerce/ecommerce-templates/templates/html-hero-banner';
import { HtmlCategoryShowcase } from '$ecommerce/ecommerce-templates/templates/html-category-showcase';
import { HtmlImageHero } from '$ecommerce/ecommerce-templates/templates/html-image-hero';
import { HtmlImageFeature } from '$ecommerce/ecommerce-templates/templates/html-image-feature';

export const storeExample: SectionData[] = [
  {
    id: 'html-hero-1',
    Type: 'HtmlSection',
    category: HtmlHeroBanner.category,
    html: HtmlHeroBanner.html,
    Css: {}
  },
  {
    id: 'html-showcase-1',
    Type: 'HtmlSection',
    category: HtmlCategoryShowcase.category,
    html: HtmlCategoryShowcase.html,
    Css: {}
  },
  {
    id: 'html-image-hero-1',
    Type: 'HtmlSection',
    category: HtmlImageHero.category,
    html: HtmlImageHero.html,
    Css: {}
  },
  {
    id: 'html-image-feature-1',
    Type: 'HtmlSection',
    category: HtmlImageFeature.category,
    html: HtmlImageFeature.html,
    Css: {}
  }
];
