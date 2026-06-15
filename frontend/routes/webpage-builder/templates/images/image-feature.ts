import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * HTML-based image feature. Demonstrates a clipped curve layout combined with a
 * duotone-matrix effect: the photo curves in from the right while text sits on
 * the tinted left side. The `data-role="image"` node is edited via ImageBlockEditor.
 */
export const ImageFeature: SectionData = {
	id: 'html-image-feature-v1',
	name: 'Image Feature (HTML)',
	category: 'features',
	description: 'Curved photo split with a duotone treatment and a text panel beside it.',
	html: `
		<section background-color="2" class="px-6 py-16">
			<div class="max-w-6xl mx-auto">
				<ImageEffect data-role="image"
					src="https://ivanjoz.github.io/genix-assets/images/business-workspace/6.avif"
					layout="curve-right" effect="duotone-matrix" tint="#4c2d82" tint2="#f59e0b"
					intensity="1" aspectRatio="21/9"
					class="rounded-2xl overflow-hidden min-h-[420px] flex items-center p-12">
					<div class="max-w-sm mr-auto text-left">
						<h2 data-role="title" color="#ffffff" class="text-3xl md:text-4xl font-black mb-4 leading-tight">
							Crafted for Comfort
						</h2>
						<p data-role="content" color="#ffffff" class="text-lg mb-8 opacity-90">
							Premium materials and timeless design, made to last season after season.
						</p>
						<a data-role="button" href="/shop" background-color="#ffffff" color="2" class="inline-block px-8 py-4 rounded-full font-bold text-lg">
							Explore Now
						</a>
					</div>
				</ImageEffect>
			</div>
		</section>
	`,
}
