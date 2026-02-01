import type { SectionTemplate } from '../renderer/renderer-types';

export const FeaturesSection: SectionTemplate = {
    id: 'features-v1',
    name: 'Key Features',
    category: 'features',
    description: 'Highlight the benefits of your store/products.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-100 px-20 bg-white',
        children: [
            {
                tagName: 'div',
                css: 'max-w-1200 mx-auto grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-40',
                children: [
                    {
                        tagName: 'div',
                        css: 'text-center',
                        children: [
                            { tagName: 'div', css: 'w-60 h-60 bg-[__COLOR:9__] rounded-full mx-auto mb-20 flex items-center justify-center text-24', text: '🚚' },
                            { text: 'Free Shipping', tag: 'h3', css: 'text-18 font-bold mb-10' },
                            { text: 'On all orders over $100', tag: 'p', css: 'text-14 text-gray-500' }
                        ]
                    },
                    {
                        tagName: 'div',
                        css: 'text-center',
                        children: [
                            { tagName: 'div', css: 'w-60 h-60 bg-[__COLOR:9__] rounded-full mx-auto mb-20 flex items-center justify-center text-24', text: '💳' },
                            { text: 'Secure Payment', tag: 'h3', css: 'text-18 font-bold mb-10' },
                            { text: '100% secure payment processing', tag: 'p', css: 'text-14 text-gray-500' }
                        ]
                    },
                    {
                        tagName: 'div',
                        css: 'text-center',
                        children: [
                            { tagName: 'div', css: 'w-60 h-60 bg-[__COLOR:9__] rounded-full mx-auto mb-20 flex items-center justify-center text-24', text: '🔄' },
                            { text: 'Easy Returns', tag: 'h3', css: 'text-18 font-bold mb-10' },
                            { text: '30-day money back guarantee', tag: 'p', css: 'text-14 text-gray-500' }
                        ]
                    },
                    {
                        tagName: 'div',
                        css: 'text-center',
                        children: [
                            { tagName: 'div', css: 'w-60 h-60 bg-[__COLOR:9__] rounded-full mx-auto mb-20 flex items-center justify-center text-24', text: '🎧' },
                            { text: '24/7 Support', tag: 'h3', css: 'text-18 font-bold mb-10' },
                            { text: 'Dedicated support at any time', tag: 'p', css: 'text-14 text-gray-500' }
                        ]
                    }
                ]
            }
        ]
    }
};
