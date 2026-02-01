import type { SectionTemplate } from '../renderer/renderer-types';

export const HeroCentered: SectionTemplate = {
    id: 'hero-centered-v1',
    name: 'Hero - Centered Text',
    category: 'hero',
    description: 'Full-width hero section with centered text and CTA button.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'bg-[__COLOR:2__] py-[__v1__] px-4 flex items-center justify-center min-h-[400px]',
        variables: [
            { key: '__v1__', type: 'py', defaultValue: '80px', label: 'Vertical Padding', min: 40, max: 200 }
        ],
        children: [
            {
                tagName: 'div',
                css: 'max-w-4xl mx-auto text-center',
                textLines: [
                    { text: 'Discover Your Style', tag: 'h1', css: 'text-4xl md:text-6xl font-bold text-[__COLOR:10__] mb-6' },
                    { text: 'Exclusive collections for the modern shopper. Quality meets elegance in every piece.', tag: 'p', css: 'text-lg md:text-xl text-[__COLOR:8__] mb-10 max-w-2xl mx-auto' }
                ],
                children: [
                    {
                        tagName: 'a',
                        text: 'Shop Collection',
                        css: 'inline-block bg-[__COLOR:6__] text-[__COLOR:1__] px-10 py-4 rounded-full font-bold text-lg hover:bg-[__COLOR:7__] transition-all transform hover:scale-105 shadow-lg',
                        attributes: { href: '/shop' }
                    }
                ]
            }
        ]
    },
    presets: [
        {
            id: 'compact',
            name: 'Compact',
            variables: { '__v1__': '40px' }
        },
        {
            id: 'spacious',
            name: 'Spacious',
            variables: { '__v1__': '120px' }
        }
    ]
};
