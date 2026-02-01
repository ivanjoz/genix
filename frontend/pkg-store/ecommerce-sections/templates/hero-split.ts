import type { SectionTemplate } from '../renderer/renderer-types';

export const HeroSplit: SectionTemplate = {
    id: 'hero-split-v1',
    name: 'Hero - Split Layout',
    category: 'hero',
    description: 'Modern two-column hero section with text and image side-by-side.',
    ast: {
        tagName: 'section',
        semanticTag: 'section',
        css: 'bg-[__COLOR:10__] py-[__v1__] px-6 overflow-hidden',
        variables: [
            { key: '__v1__', type: 'py', defaultValue: '64px', label: 'Vertical Padding', min: 32, max: 160 }
        ],
        children: [
            {
                tagName: 'div',
                css: 'max-w-7xl mx-auto grid md:grid-cols-2 gap-12 items-center',
                children: [
                    {
                        tagName: 'div',
                        css: 'order-2 md:order-1',
                        textLines: [
                            { text: 'NEW ARRIVALS', tag: 'span', css: 'text-[__COLOR:5__] font-bold tracking-widest text-sm mb-4 block' },
                            { text: 'Upgrade Your Home Office', tag: 'h2', css: 'text-4xl md:text-5xl lg:text-6xl font-extrabold text-[__COLOR:1__] leading-tight mb-6' },
                            { text: 'Premium furniture designed for comfort and productivity. Handcrafted with sustainable materials.', tag: 'p', css: 'text-lg text-[__COLOR:3__] mb-8 leading-relaxed' }
                        ],
                        children: [
                            {
                                tagName: 'div',
                                css: 'flex flex-wrap gap-4',
                                children: [
                                    {
                                        tagName: 'a',
                                        text: 'Explore Now',
                                        css: 'bg-[__COLOR:1__] text-[__COLOR:10__] px-8 py-3 rounded-md font-semibold hover:opacity-90 transition-opacity',
                                        attributes: { href: '/new-arrivals' }
                                    },
                                    {
                                        tagName: 'a',
                                        text: 'View Catalog',
                                        css: 'border-2 border-[__COLOR:1__] text-[__COLOR:1__] px-8 py-3 rounded-md font-semibold hover:bg-[__COLOR:1__] hover:text-[__COLOR:10__] transition-all',
                                        attributes: { href: '/catalog' }
                                    }
                                ]
                            }
                        ]
                    },
                    {
                        tagName: 'div',
                        css: 'order-1 md:order-2 relative h-[400px] md:h-[500px] bg-[__COLOR:9__] rounded-2xl overflow-hidden shadow-2xl',
                        children: [
                            {
                                tagName: 'img',
                                css: 'w-full h-full object-cover',
                                attributes: { 
                                    src: 'https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?auto=format&fit=crop&q=80&w=1000',
                                    alt: 'Hero Product'
                                }
                            }
                        ]
                    }
                ]
            }
        ]
    }
};
