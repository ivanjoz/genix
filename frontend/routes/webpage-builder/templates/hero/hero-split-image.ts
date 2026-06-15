import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Split hero: editable text column on the left, an ImageEffect photo box on the right.
 * Uses the box mode of ImageEffect (aspectRatio + rounded). Collapses to a single column
 * on mobile via the grid-cols-1 md:grid-cols-2 switch.
 */
export const HeroSplitImage: SectionData = {
	id: 'hero-split-image-v1',
	name: 'Hero Split (text + photo)',
	category: 'hero',
	description: 'Two-column hero with a headline and call-to-action beside a feature photo.',
	thumbnail: `
		<div class="flex h-[100px] w-full items-center gap-[12px] bg-white px-[16px]">
			<div class="flex-1">
				<div class="h-[8px] w-2/3 rounded bg-slate-800"></div>
				<div class="mt-[6px] h-[5px] w-full rounded bg-slate-300"></div>
				<div class="mt-[4px] h-[5px] w-5/6 rounded bg-slate-300"></div>
				<div class="mt-[10px] h-[14px] w-[48px] rounded-full bg-indigo-600"></div>
			</div>
			<div class="h-[72px] w-[84px] rounded-md bg-slate-300"></div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-56 items-center">
				<div class="text-left">
					<span color="#4f46e5" class="inline-block text-sm font-bold uppercase tracking-widest mb-16">New Season</span>
					<h1 data-role="title" color="10" class="text-4xl md:text-5xl font-black leading-tight mb-20">
						Everyday essentials, beautifully made
					</h1>
					<p data-role="content" color="7" class="text-lg leading-relaxed mb-32 max-w-md">
						Thoughtfully designed products built to last. Discover the pieces our customers
						come back for, season after season.
					</p>
					<div class="flex flex-wrap gap-16">
						<a data-role="button" href="/shop" background-color="#4f46e5" color="#ffffff" class="inline-block rounded-full px-32 py-16 text-lg font-bold">
							Shop the collection
						</a>
						<a href="/about" color="9" class="inline-block rounded-full px-32 py-16 text-lg font-bold border-2 border-slate-300">
							Learn more
						</a>
					</div>
				</div>
				<ImageEffect data-role="image"
					src="https://ivanjoz.github.io/genix-assets/images/business-workspace/4.avif"
					fit="cover" aspectRatio="4/3"
					class="rounded-3xl overflow-hidden shadow-xl" />
			</div>
		</section>
	`,
};
