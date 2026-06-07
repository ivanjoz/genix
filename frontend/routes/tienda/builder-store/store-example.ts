import type { SectionData } from '$ecommerce/renderer/section-types';
import { HtmlHeroBanner } from '$ecommerce/ecommerce-templates/templates/html-hero-banner';
import { HtmlCategoryShowcase } from '$ecommerce/ecommerce-templates/templates/html-category-showcase';
import { HtmlImageHero } from '$ecommerce/ecommerce-templates/templates/html-image-hero';
import { HtmlImageFeature } from '$ecommerce/ecommerce-templates/templates/html-image-feature';

export const storeExample: SectionData[] = [
  {
    id: 'hero-1',
    type: 'HeroStandard',
    category: 'hero',
    content: {
      title: 'Welcome to Our Amazing Store',
      subTitle: 'Premium Quality Products',
      description: 'Discover the best products at unbeatable prices. Hand-picked for your lifestyle.',
      primaryActionLabel: 'Shop Now',
      primaryActionHref: '/shop',
      bgImage: 'https://images.unsplash.com/photo-1441986300917-64674bd600d8?q=80&w=2070&auto=format&fit=crop'
    },
    css: {
    }
  },
  {
    id: 'features-1',
    type: 'FeatureListSimple',
    category: 'features',
    content: {
      title: 'Why Choose Us',
      subTitle: 'Our Benefits',
      description: 'We provide the best service in the industry with multiple benefits for our customers.',
      textLeft: 'Fast Delivery',
      textCenter: '24/7 Support',
      textRight: 'Secure Payments'
    },
    css: {

    }
  },
  {
    id: 'products-1',
    type: 'ProductGridSimple',
    category: 'products',
    content: {
      title: 'Featured Products',
      subTitle: 'Best Sellers',
      limit: 4
    },
    css: {

    }
  },
  {
    id: 'html-hero-1',
    type: 'HtmlSection',
    category: HtmlHeroBanner.category,
    content: { html: HtmlHeroBanner.HTML },
    css: {}
  },
  {
    id: 'html-showcase-1',
    type: 'HtmlSection',
    category: HtmlCategoryShowcase.category,
    content: { html: HtmlCategoryShowcase.HTML },
    css: {}
  },
  {
    id: 'html-image-hero-1',
    type: 'HtmlSection',
    category: HtmlImageHero.category,
    content: { html: HtmlImageHero.HTML },
    css: {}
  },
  {
    id: 'html-image-feature-1',
    type: 'HtmlSection',
    category: HtmlImageFeature.category,
    content: { html: HtmlImageFeature.HTML },
    css: {}
  }
];
