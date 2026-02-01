import type { SectionTemplate } from '../renderer/renderer-types';

export const NewsletterSection: SectionTemplate = {
    id: 'newsletter-v1',
    name: 'Newsletter Signup',
    category: 'cta',
    description: 'Email subscription section to capture leads.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'py-100 bg-[__COLOR:9__] px-20',
        children: [
            {
                tagName: 'div',
                css: 'max-w-800 mx-auto text-center',
                textLines: [
                    { text: 'Stay in the Loop', tag: 'h2', css: 'text-32 font-bold mb-20' },
                    { text: 'Subscribe to our newsletter and get 10% off your first order!', tag: 'p', css: 'text-16 text-gray-600 mb-40' }
                ],
                children: [
                    {
                        tagName: 'form',
                        css: 'flex flex-col sm:flex-row gap-10 justify-center',
                        children: [
                            {
                                tagName: 'input',
                                css: 'w-full sm:w-400 px-20 py-12 rounded-8 border border-gray-300 focus:ring-2 focus:ring-[__COLOR:5__] outline-none',
                                attributes: { type: 'email', placeholder: 'Enter your email', required: 'true' }
                            },
                            {
                                tagName: 'button',
                                text: 'Subscribe',
                                css: 'bg-[__COLOR:1__] text-white px-30 py-12 rounded-8 font-bold hover:opacity-90 transition-opacity',
                                attributes: { type: 'submit' }
                            }
                        ]
                    }
                ]
            }
        ]
    }
};
