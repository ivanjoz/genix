import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Full-width call-to-action banner on a gradient background, centered headline, supporting
 * line and a single prominent button.
 */
export const CtaBanner: SectionData = {
	id: 'cta-banner-v1',
	name: 'CTA Banner',
	category: 'cta',
	description: 'Bold full-width call-to-action banner with a single button.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col items-center justify-center bg-gradient-to-r from-indigo-600 to-violet-600 px-[16px] text-center text-white">
			<div class="h-[12px] w-[160px] rounded bg-white/90"></div>
			<div class="mt-[7px] h-[5px] w-[110px] rounded bg-white/50"></div>
			<div class="mt-[12px] h-[18px] w-[80px] rounded-full bg-white"></div>
		</div>
	`,
	html: `
		<section class="px-32 py-80 text-center bg-linear-to-r from-indigo-600 via-indigo-600 to-violet-600">
			<div class="max-w-3xl mx-auto">
				<h2 data-role="title" color="#ffffff" class="text-3xl md:text-5xl font-black leading-tight mb-20">
					Ready to find your new favorite?
				</h2>
				<p data-role="content" color="#e0e7ff" class="text-lg md:text-xl mb-36 max-w-xl mx-auto">
					Join thousands of happy customers and get 10% off your first order today.
				</p>
				<a data-role="button" href="/shop" background-color="#ffffff" color="#4338ca" class="inline-block rounded-full px-44 py-18 text-lg font-bold">
					Start shopping
				</a>
			</div>
		</section>
	`,
};
