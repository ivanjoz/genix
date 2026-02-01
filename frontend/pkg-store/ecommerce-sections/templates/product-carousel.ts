import type { SectionTemplate } from '../renderer/renderer-types';

export const ProductCarouselSection: SectionTemplate = {
    id: 'product-carousel-v1',
    name: 'Product Slider',
    category: 'products',
    description: 'Horizontal scrolling slider for products.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-20 bg-gray-50 overflow-hidden',
        children: [
            {
                tagName: 'div',
                css: 'max-w-7xl mx-auto px-4 mb-12',
                textLines: [
                    { text: 'Trending Now', tag: 'h2', css: 'text-4xl font-black text-center text-gray-800' }
                ]
            },
            {
                tagName: 'ProductCarousel',
                css: 'px-4',
                productosIDs: [10, 11, 12, 13, 14, 15, 16]
            }
        ]
    }
};
