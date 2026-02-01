import type { SectionTemplate } from '../renderer/renderer-types';

export const FooterBasic: SectionTemplate = {
    id: 'footer-basic-v1',
    name: 'Basic Footer',
    category: 'footer',
    description: 'Standard footer with multi-column links.',
    ast: {
        tagName: 'footer',
        semanticTag: 'footer',
        css: 'bg-[__COLOR:1__] text-white pt-80 pb-40 px-20',
        children: [
            {
                tagName: 'div',
                css: 'max-w-1200 mx-auto grid grid-cols-1 md:grid-cols-4 gap-40 mb-60',
                children: [
                    {
                        tagName: 'div',
                        textLines: [
                            { text: 'STORE NAME', tag: 'h3', css: 'text-20 font-black mb-20 tracking-tighter' },
                            { text: 'The best products for your lifestyle. Handpicked quality and style.', tag: 'p', css: 'text-14 text-[__COLOR:8__] leading-relaxed' }
                        ]
                    },
                    {
                        tagName: 'div',
                        children: [
                            { text: 'Shop', tag: 'h4', css: 'text-16 font-bold mb-20' },
                            { tagName: 'ul', css: 'space-y-10', children: [
                                { tagName: 'li', children: [{ tagName: 'a', text: 'All Products', css: 'text-14 text-[__COLOR:9__] hover:text-white', attributes: { href: '/shop' } }] },
                                { tagName: 'li', children: [{ tagName: 'a', text: 'New Arrivals', css: 'text-14 text-[__COLOR:9__] hover:text-white', attributes: { href: '/new' } }] },
                                { tagName: 'li', children: [{ tagName: 'a', text: 'Best Sellers', css: 'text-14 text-[__COLOR:9__] hover:text-white', attributes: { href: '/best' } }] }
                            ]}
                        ]
                    },
                    {
                        tagName: 'div',
                        children: [
                            { text: 'Support', tag: 'h4', css: 'text-16 font-bold mb-20' },
                            { tagName: 'ul', css: 'space-y-10', children: [
                                { tagName: 'li', children: [{ tagName: 'a', text: 'FAQ', css: 'text-14 text-[__COLOR:9__] hover:text-white', attributes: { href: '/faq' } }] },
                                { tagName: 'li', children: [{ tagName: 'a', text: 'Shipping', css: 'text-14 text-[__COLOR:9__] hover:text-white', attributes: { href: '/shipping' } }] },
                                { tagName: 'li', children: [{ tagName: 'a', text: 'Returns', css: 'text-14 text-[__COLOR:9__] hover:text-white', attributes: { href: '/returns' } }] }
                            ]}
                        ]
                    },
                    {
                        tagName: 'div',
                        children: [
                            { text: 'Contact', tag: 'h4', css: 'text-16 font-bold mb-20' },
                            { tagName: 'ul', css: 'space-y-10', children: [
                                { tagName: 'li', children: [{ tagName: 'span', text: 'Email: support@example.com', css: 'text-14 text-[__COLOR:9__]' }] },
                                { tagName: 'li', children: [{ tagName: 'span', text: 'Phone: +1 234 567 890', css: 'text-14 text-[__COLOR:9__]' }] }
                            ]}
                        ]
                    }
                ]
            },
            {
                tagName: 'div',
                css: 'max-w-1200 mx-auto pt-30 border-t border-white/10 text-center',
                text: '© 2026 Store Name. All rights reserved.',
                attributes: { class: 'text-12 text-[__COLOR:7__]' }
            }
        ]
    }
};
