import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Single large testimonial spotlight: an oversized quotation mark, one centered quote and
 * the author line, on a dark palette background for emphasis.
 */
export const TestimonialSpotlight: SectionData = {
	id: 'testimonial-spotlight-v1',
	name: 'Quote Spotlight',
	category: 'testimonials',
	description: 'A single large customer quote spotlighted on a dark background.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col items-center justify-center bg-slate-900 px-[20px] text-center text-white">
			<div class="text-[28px] leading-none text-indigo-400 font-black">&ldquo;</div>
			<div class="mt-[6px] h-[6px] w-[160px] rounded bg-white/80"></div>
			<div class="mt-[4px] h-[6px] w-[130px] rounded bg-white/80"></div>
			<div class="mt-[10px] h-[5px] w-[70px] rounded bg-indigo-400"></div>
		</div>
	`,
	html: `
		<section background-color="10" class="px-32 py-96 text-center">
			<div class="max-w-3xl mx-auto">
				<div color="#818cf8" class="text-7xl font-black leading-none mb-8">&ldquo;</div>
				<p data-role="title" color="#ffffff" class="text-2xl md:text-4xl font-bold leading-snug mb-32">
					This is the brand I recommend to everyone. The quality, the service, the little
					details — they get it all right.
				</p>
				<div data-role="content" color="#a5b4fc" class="text-lg font-semibold">Camila Ortiz</div>
				<div color="6" class="text-base">Loyal customer since 2021</div>
			</div>
		</section>
	`,
};
