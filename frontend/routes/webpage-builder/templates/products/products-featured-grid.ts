import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Featured products: a section heading + subtitle, the self-fetching <ProductGrid>, and a
 * centered "View all" link below. ProductGrid pulls live catalog data, so in the isolated
 * preview the grid area may be empty — the heading/CTA structure is what's authored here.
 */
export const ProductsFeaturedGrid: SectionData = {
	id: 'products-featured-grid-v1',
	name: 'Featured Products',
	category: 'products',
	description: 'A heading, a grid of featured products and a link to the full catalog.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-white px-[12px] py-[8px]">
			<div class="mx-auto mb-[8px] h-[7px] w-[90px] rounded bg-slate-800"></div>
			<div class="grid grid-cols-4 gap-[7px]">
				<div><div class="aspect-square rounded bg-slate-200"></div><div class="mt-[3px] h-[3px] rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded bg-slate-200"></div><div class="mt-[3px] h-[3px] rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded bg-slate-200"></div><div class="mt-[3px] h-[3px] rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded bg-slate-200"></div><div class="mt-[3px] h-[3px] rounded bg-slate-300"></div></div>
			</div>
			<div class="mx-auto mt-[8px] h-[12px] w-[64px] rounded-full border border-indigo-500"></div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-7xl mx-auto">
				<div class="text-center mb-48">
					<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-16">Featured products</h2>
					<p data-role="content" color="6" class="text-lg max-w-2xl mx-auto">Hand-picked favorites our customers can't get enough of.</p>
				</div>
				<ProductGrid categoryID="22" maxWidth="1200" maxMargin="0" rows="2" rowsMobile="2" />
				<div class="text-center mt-48">
					<a data-role="button" href="/shop" color="#4f46e5" class="inline-block text-lg font-bold border-2 border-indigo-600 rounded-full px-36 py-14">
						View all products →
					</a>
				</div>
			</div>
		</section>
	`,
};
