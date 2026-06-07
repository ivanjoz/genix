import type { SectionTemplate } from '../../renderer/renderer-types';

/**
 * HTML-based image hero. Demonstrates ImageEffect in `fill` mode: the photo is an
 * absolute background layer of the (relative) section, with the editable
 * title/content/button sitting on top as siblings. The `data-role="image"` node is
 * edited via the builder's ImageBlockEditor (effect, tint, fit, etc.).
 */
export const HtmlImageHero: SectionTemplate = {
	id: 'html-image-hero-v1',
	name: 'Image Hero (HTML)',
	category: 'hero',
	description: 'Full-bleed photo background with an overlaid headline, text and call-to-action.',
	HTML: `
		<section class="relative overflow-hidden min-h-[420px] flex items-center px-6 py-12">
			<ImageEffect data-role="image" fill fit="cover"
				src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif"
				effect="overlay" tint="#0f172a" intensity="0.9" />
			<div class="relative z-10 w-full max-w-6xl mx-auto">
				<div class="max-w-md text-left">
					<h1 data-role="title" color="#ffffff" class="text-4xl md:text-5xl font-black mb-4 leading-tight">
						Style That Speaks
					</h1>
					<p data-role="content" color="#ffffff" class="text-lg mb-8 opacity-90">
						Curated collections for the modern wardrobe, delivered with care.
					</p>
					<a data-role="button" href="/shop" background-color="5" color="1" class="inline-block px-8 py-4 rounded-full font-bold text-lg">
						Shop the Collection
					</a>
				</div>
			</div>
		</section>
	`,
}
