import type { SectionTemplate } from '../renderer/renderer-types';

export const CategoryGridSection: SectionTemplate = {
    id: 'category-grid-v1',
    name: 'Category Showcase',
    category: 'categories',
    description: 'Grid of product categories for easy navigation.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-16 px-6 bg-[__COLOR:10__]',
        children: [
            {
                tagName: 'div',
                css: 'max-w-7xl mx-auto',
                children: [
                    {
                        tagName: 'h3',
                        text: 'Shop by Category',
                        css: 'text-2xl font-bold mb-8 text-gray-900 border-l-4 border-[__COLOR:5__] pl-4'
                    },
                    {
                        tagName: 'CategoryGrid',
                        css: 'grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4',
                        attributes: { 'categoriasIDs': '1,2,3,4,5,6' } // Using attributes for specialized component props if they are not explicitly in the interface
                    }
                ]
            }
        ]
    }
};
