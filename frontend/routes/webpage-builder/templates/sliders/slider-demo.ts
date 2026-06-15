import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * HTML-based carousel. Demonstrates the `<Slider>` custom component: each of its
 * direct children becomes one slide. AstRenderer passes the slider its children as
 * raw AST nodes (`childNodes`) plus a `renderChild` snippet, so EcommerceSlider can
 * render each slide subtree independently. Slide 1 reuses ImageEffect in `fill` mode
 * with editable role nodes (title/content/button/image); slides 2-3 are plain blocks.
 */
export const SliderDemo: SectionData = {
	id: 'html-slider-demo-v1',
	name: 'Slider (HTML)',
	category: 'hero',
	description: 'Carousel whose direct children each become a slide, rendered via <Slider>.',
	html: `
		<section class="relative overflow-hidden min-h-[420px] flex items-center">
			<Slider autoplay interval="6000">
				<div class="relative min-h-[420px] flex items-center px-6 py-12">
					<ImageEffect data-role="image" fill fit="cover"
						src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif"
						effect="overlay" tint="#0f172a" intensity="0.9"
					/>
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
				</div>
				<div class="min-h-[420px] w-full flex items-center justify-center bg-slate-800">
					<h1 class="text-4xl font-black text-white">SLIDE 1</h1>
				</div>
				<div class="min-h-[420px] w-full flex items-center justify-center bg-slate-600">
					<h1 class="text-4xl font-black text-white">SLIDE 2</h1>
				</div>
			</Slider>
		</section>
	`,
}
