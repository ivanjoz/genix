import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Dark metrics band: four large numbers with labels on a dark palette background.
 * Pure native markup, responsive 2->4 columns.
 */
export const StatsBand: SectionData = {
	id: 'stats-band-v1',
	name: 'Stats / Metrics Band',
	category: 'features',
	description: 'A bold band of key metrics on a dark background.',
	thumbnail: `
		<div class="flex h-[100px] w-full items-center justify-around bg-slate-900 px-[12px] text-center text-white">
			<div><div class="text-[16px] font-black">10k+</div><div class="text-[7px] text-slate-400">orders</div></div>
			<div><div class="text-[16px] font-black">4.9★</div><div class="text-[7px] text-slate-400">rating</div></div>
			<div><div class="text-[16px] font-black">50+</div><div class="text-[7px] text-slate-400">countries</div></div>
			<div><div class="text-[16px] font-black">24h</div><div class="text-[7px] text-slate-400">support</div></div>
		</div>
	`,
	html: `
		<section background-color="10" class="px-32 py-64">
			<div class="max-w-6xl mx-auto grid grid-cols-2 md:grid-cols-4 gap-40 text-center">
				<div>
					<div data-role="title" color="#ffffff" class="text-4xl md:text-5xl font-black">10k+</div>
					<div color="6" class="text-sm uppercase tracking-widest mt-8">Orders shipped</div>
				</div>
				<div>
					<div color="#ffffff" class="text-4xl md:text-5xl font-black">4.9★</div>
					<div color="6" class="text-sm uppercase tracking-widest mt-8">Average rating</div>
				</div>
				<div>
					<div color="#ffffff" class="text-4xl md:text-5xl font-black">50+</div>
					<div color="6" class="text-sm uppercase tracking-widest mt-8">Countries served</div>
				</div>
				<div>
					<div color="#ffffff" class="text-4xl md:text-5xl font-black">24h</div>
					<div color="6" class="text-sm uppercase tracking-widest mt-8">Support response</div>
				</div>
			</div>
		</section>
	`,
};
