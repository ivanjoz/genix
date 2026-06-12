import type { SectionData } from '$ecommerce/renderer/section-types';
import { HeroBanner } from '$ecommerce/templates/hero/hero-banner';
import { CategoryShowcase } from '$ecommerce/templates/products/category-showcase';
import { ImageHero } from '$ecommerce/templates/hero/image-hero';
import { ImageFeature } from '$ecommerce/templates/images/image-feature';

export const storeExample: SectionData[] = [
  {
    id: 'html-hero-1',
    Type: 'HtmlSection',
    category: HeroBanner.category,
    html: HeroBanner.html,
    Css: {}
  },
  {
    id: 'html-showcase-1',
    Type: 'HtmlSection',
    category: CategoryShowcase.category,
    html: CategoryShowcase.html,
    Css: {}
  },
  {
    id: 'html-image-hero-1',
    Type: 'HtmlSection',
    category: ImageHero.category,
    html: ImageHero.html,
    Css: {}
  },
  {
    id: 'html-image-feature-1',
    Type: 'HtmlSection',
    category: ImageFeature.category,
    html: ImageFeature.html,
    Css: {}
  }
];
