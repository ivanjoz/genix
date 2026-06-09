import type { SectionData } from '$ecommerce/renderer/section-types';
import { HtmlHeroBanner } from '$ecommerce/ecommerce-templates/templates/html-hero-banner';
import { HtmlCategoryShowcase } from '$ecommerce/ecommerce-templates/templates/html-category-showcase';
import { HtmlImageHero } from '$ecommerce/ecommerce-templates/templates/html-image-hero';
import { HtmlImageFeature } from '$ecommerce/ecommerce-templates/templates/html-image-feature';

export const storeExample: SectionData[] = [
  {
    id: 'hero-1',
    Type: 'HeroStandard',
    category: 'hero',
    Content: {
      title: 'Welcome to Our Amazing Store',
      subTitle: 'Premium Quality Products',
      description: 'Discover the best products at unbeatable prices. Hand-picked for your lifestyle.',
      primaryActionLabel: 'Shop Now',
      primaryActionHref: '/shop',
      bgImage: 'https://images.unsplash.com/photo-1441986300917-64674bd600d8?q=80&w=2070&auto=format&fit=crop'
    },
    Css: {
    }
  },
  {
    id: 'features-1',
    Type: 'FeatureListSimple',
    category: 'features',
    Content: {
      title: 'Why Choose Us',
      subTitle: 'Our Benefits',
      description: 'We provide the best service in the industry with multiple benefits for our customers.',
      textLeft: 'Fast Delivery',
      textCenter: '24/7 Support',
      textRight: 'Secure Payments'
    },
    Css: {

    }
  },
  {
    id: 'products-1',
    Type: 'ProductGridSimple',
    category: 'products',
    Content: {
      title: 'Featured Products',
      subTitle: 'Best Sellers',
      limit: 4
    },
    Css: {

    }
  },
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
