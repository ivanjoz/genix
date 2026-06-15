import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Centered hero on a dark diagonal gradient with a three-up stats row beneath the CTA.
 * Gradient uses Tailwind v4 `bg-linear-to-br`; the stats use a responsive grid that
 * collapses to one column on mobile.
 */
export const HeroGradientStats: SectionData = {
	id: 'hero-gradient-stats-v1',
	name: 'Hero Gradient + Stats',
	category: 'hero',
	description: 'Centered headline on a gradient background with a row of key metrics.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col items-center justify-center bg-gradient-to-br from-indigo-700 to-slate-900 px-[16px] text-center text-white">
			<div class="h-[10px] w-[140px] rounded bg-white/90"></div>
			<div class="mt-[7px] h-[5px] w-[170px] rounded bg-white/40"></div>
			<div class="mt-[12px] flex gap-[18px] text-[10px] font-black text-white/90"><span>12k+</span><span>98%</span><span>24h</span></div>
		</div>
	`,
	html: `
		<section class="relative overflow-hidden px-32 py-96 text-center bg-linear-to-br from-indigo-700 via-indigo-800 to-slate-900">
			<div class="max-w-3xl mx-auto">
				<h1 data-role="title" color="#ffffff" class="text-4xl md:text-6xl font-black leading-tight mb-20">
					Build a store your customers love
				</h1>
				<p data-role="content" color="#e0e7ff" class="text-lg md:text-xl leading-relaxed mb-36 max-w-2xl mx-auto">
					Everything you need to sell online — beautiful storefronts, fast checkout and tools
					that grow with your business.
				</p>
				<a data-role="button" href="/shop" background-color="#ffffff" color="#312e81" class="inline-block rounded-full px-40 py-18 text-lg font-bold mb-56">
					Get started free
				</a>
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-32 max-w-2xl mx-auto">
					<div>
						<div color="#ffffff" class="text-4xl font-black">12k+</div>
						<div color="#c7d2fe" class="text-sm uppercase tracking-wider mt-6">Happy stores</div>
					</div>
					<div>
						<div color="#ffffff" class="text-4xl font-black">98%</div>
						<div color="#c7d2fe" class="text-sm uppercase tracking-wider mt-6">Satisfaction</div>
					</div>
					<div>
						<div color="#ffffff" class="text-4xl font-black">24h</div>
						<div color="#c7d2fe" class="text-sm uppercase tracking-wider mt-6">Support</div>
					</div>
				</div>
			</div>
		</section>
	`,
};
