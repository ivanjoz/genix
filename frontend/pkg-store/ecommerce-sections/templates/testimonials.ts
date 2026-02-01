import type { SectionTemplate } from '../renderer/renderer-types';

export const TestimonialsSection: SectionTemplate = {
    id: 'testimonials-v1',
    name: 'Customer Testimonials',
    category: 'testimonials',
    description: 'Grid of customer reviews with avatars.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-80 bg-[__COLOR:10__] px-20',
        children: [
            {
                tagName: 'h2',
                text: 'What Our Customers Say',
                css: 'text-36 font-bold text-center mb-60 text-[__COLOR:1__]'
            },
            {
                tagName: 'div',
                css: 'max-w-1200 mx-auto grid grid-cols-1 md:grid-cols-3 gap-30',
                children: [
                    {
                        tagName: 'div',
                        css: 'bg-white p-30 rounded-12 shadow-sm border border-gray-100',
                        textLines: [
                            { text: '"Excellent quality and fast shipping. Highly recommend!"', tag: 'p', css: 'text-16 italic text-gray-700 mb-20' },
                            { text: 'Sarah Johnson', tag: 'strong', css: 'block text-14 font-bold text-[__COLOR:1__]' },
                            { text: 'Verified Buyer', tag: 'span', css: 'text-12 text-gray-400' }
                        ]
                    },
                    {
                        tagName: 'div',
                        css: 'bg-white p-30 rounded-12 shadow-sm border border-gray-100',
                        textLines: [
                            { text: '"The design is even better in person. Fits perfectly in my office."', tag: 'p', css: 'text-16 italic text-gray-700 mb-20' },
                            { text: 'Michael Chen', tag: 'strong', css: 'block text-14 font-bold text-[__COLOR:1__]' },
                            { text: 'Interior Designer', tag: 'span', css: 'text-12 text-gray-400' }
                        ]
                    },
                    {
                        tagName: 'div',
                        css: 'bg-white p-30 rounded-12 shadow-sm border border-gray-100',
                        textLines: [
                            { text: '"Amazing customer support. They helped me with every step."', tag: 'p', css: 'text-16 italic text-gray-700 mb-20' },
                            { text: 'Emma Wilson', tag: 'strong', css: 'block text-14 font-bold text-[__COLOR:1__]' },
                            { text: 'Graphic Artist', tag: 'span', css: 'text-12 text-gray-400' }
                        ]
                    }
                ]
            }
        ]
    }
};
