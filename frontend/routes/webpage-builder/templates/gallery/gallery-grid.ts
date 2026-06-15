import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Photo gallery: a heading above a responsive mosaic of native <img> tiles. The first tile
 * spans two rows on desktop for a magazine feel. Uses the six available CDN photos.
 */
export const GalleryGrid: SectionData = {
	id: 'gallery-grid-v1',
	name: 'Photo Gallery',
	category: 'gallery',
	description: 'A responsive mosaic gallery of lifestyle photos.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-white px-[12px] py-[8px]">
			<div class="mx-auto mb-[8px] h-[7px] w-[80px] rounded bg-slate-800"></div>
			<div class="grid grid-cols-4 grid-rows-2 gap-[5px] h-[58px]">
				<div class="row-span-2 rounded bg-slate-300"></div>
				<div class="rounded bg-slate-200"></div>
				<div class="rounded bg-slate-300"></div>
				<div class="row-span-2 rounded bg-slate-300"></div>
				<div class="rounded bg-slate-200"></div>
				<div class="rounded bg-slate-300"></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-7xl mx-auto">
				<div class="text-center mb-48">
					<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-16">From our community</h2>
					<p data-role="content" color="6" class="text-lg max-w-2xl mx-auto">Real moments from people who love what we make. Tag us to be featured.</p>
				</div>
				<div class="grid grid-cols-2 md:grid-cols-4 auto-rows-[200px] gap-16">
					<img src="https://ivanjoz.github.io/genix-assets/images/business-workspace/4.avif" alt="Gallery photo" class="w-full h-full object-cover rounded-2xl row-span-2" />
					<img src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif" alt="Gallery photo" class="w-full h-full object-cover rounded-2xl" />
					<img src="https://ivanjoz.github.io/genix-assets/images/business-workspace/6.avif" alt="Gallery photo" class="w-full h-full object-cover rounded-2xl" />
					<img src="https://ivanjoz.github.io/genix-assets/images/business-workspace/7.avif" alt="Gallery photo" class="w-full h-full object-cover rounded-2xl row-span-2" />
					<img src="https://ivanjoz.github.io/genix-assets/images/business-workspace/8.avif" alt="Gallery photo" class="w-full h-full object-cover rounded-2xl" />
					<img src="https://ivanjoz.github.io/genix-assets/images/business-workspace/9.avif" alt="Gallery photo" class="w-full h-full object-cover rounded-2xl" />
				</div>
			</div>
		</section>
	`,
};
