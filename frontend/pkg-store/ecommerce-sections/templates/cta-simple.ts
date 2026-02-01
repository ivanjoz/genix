import type { SectionTemplate } from '../renderer/renderer-types';

export const CTASimple: SectionTemplate = {
    id: 'cta-simple-v1',
    name: 'Simple Call to Action',
    category: 'cta',
    description: 'Direct CTA with heading and button on a colored background.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-12 px-6',
        children: [
            {
                tagName: 'div',
                css: 'max-w-5xl mx-auto bg-[__COLOR:1__] rounded-3xl p-8 md:p-16 flex flex-col md:flex-row items-center justify-between gap-8',
                children: [
                    {
                        tagName: 'div',
                        css: 'text-left',
                        textLines: [
                            { text: 'Ready to start your journey?', tag: 'h2', css: 'text-3xl md:text-4xl font-bold text-white mb-2' },
                            { text: 'Join over 10,000 customers who love our products.', tag: 'p', css: 'text-[__COLOR:8__] text-lg' }
                        ]
                    },
                    {
                        tagName: 'a',
                        text: 'Get Started',
                        css: 'bg-white text-[__COLOR:1__] px-8 py-4 rounded-xl font-bold hover:bg-[__COLOR:10__] transition-colors whitespace-nowrap',
                        attributes: { href: '/signup' }
                    }
                ]
            }
        ]
    }
};
