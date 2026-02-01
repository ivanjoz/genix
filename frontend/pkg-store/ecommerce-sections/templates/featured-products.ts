import type { SectionTemplate } from '../renderer/renderer-types';

export const FeaturedProducts: SectionTemplate = {
    id: 'featured-products-v1',
    name: 'Featured Products Grid',
    category: 'products',
    description: 'A responsive grid displaying featured products.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-16 px-4 bg-white',
        children: [
            {
                tagName: 'div',
                css: 'max-w-7xl mx-auto',
                children: [
                    {
                        tagName: 'div',
                        css: 'flex justify-between items-end mb-10',
                        children: [
                            {
                                tagName: 'div',
                                children: [
                                    { text: 'Top Sellers', tag: 'h2', css: 'text-3xl font-bold text-gray-900' },
                                    { text: 'Check out our most popular items', tag: 'p', css: 'text-gray-500 mt-2' }
                                ]
                            },
                            {
                                tagName: 'a',
                                text: 'View All →',
                                css: 'text-[__COLOR:5__] font-semibold hover:underline',
                                attributes: { href: '/all-products' }
                            }
                        ]
                    },
                    {
                        tagName: 'ProductGrid',
                        css: 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-x-6 gap-y-10',
                        productosIDs: [1, 2, 3, 4, 5, 6, 7, 8]
                    }
                ]
            }
        ]
    }
};
