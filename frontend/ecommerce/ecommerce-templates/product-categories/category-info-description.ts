import type { SectionTemplate } from '../../renderer/renderer-types';

export const CategoryInfoDescription: SectionTemplate = {
    id: 'category-info-description-v1',
    name: 'Category Description',
    category: 'categories',
    description: 'Display a category description in a styled section.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-12 px-4 bg-white',
        children: [
            {
                tagName: 'div',
                css: 'max-w-4xl mx-auto text-center',
                children: [
                    {
                        tagName: 'CategoryDescription',
                        css: 'text-lg text-gray-700 leading-relaxed',
                        categoriasIDs: [13]
                    }
                ]
            }
        ]
    }
};