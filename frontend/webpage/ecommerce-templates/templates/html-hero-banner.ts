import type { SectionData } from '../../renderer/section-types';

/**
 * HTML-based hero banner. Demonstrates the HTML -> AST -> Svelte pipeline with
 * native tags only: color attributes become palette CSS variables, layout and
 * typography use Tailwind classes.
 */
export const HtmlHeroBanner: SectionData = {
	id: 'html-hero-banner-v1',
	name: 'Hero Banner (HTML)',
	category: 'hero',
	description: 'Full-width hero with headline, supporting text and a call-to-action button.',
	html: `
		<section background-color="9" class="py-24 px-6 text-center">
			<div class="max-w-3xl mx-auto">
				<h1 data-role="title" color="1" class="text-4xl md:text-6xl font-bold mb-6 leading-tight">
					Fresh Picks, Delivered Fast
				</h1>
				<p data-role="content" color="3" class="text-lg md:text-xl mb-10">
					Discover seasonal favorites and everyday essentials, hand-selected for quality
					and delivered straight to your door.
				</p>
				<a data-role="button" href="/shop" background-color="5" color="1" class="inline-block px-8 py-4 rounded-full font-bold text-lg">
					Shop Now
				</a>
			</div>
		</section>
	`,
}
