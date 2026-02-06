import type { SectionTemplate } from '../../renderer/renderer-types';

export const CategoryWithProducts: SectionTemplate = {
    id: 'category-with-products-v1',
    name: 'Category with Products',
    category: 'categories',
    description: 'Display a category description followed by products from that category.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-16 px-4 bg-gray-50',
        children: [
            {
                tagName: 'div',
                css: 'max-w-7xl mx-auto',
                children: [
                    {
                        tagName: 'CategoryDescription',
                        css: 'text-lg text-gray-700 leading-relaxed mb-12 text-center',
                        categoriasIDs: [13]
                    },
                    {
                        tagName: 'ProductsByCategory',
                        css: 'w-full',
                        categoriasIDs: [13],
                        limit: 8
                    }
                ]
            }
        ]
    }
};