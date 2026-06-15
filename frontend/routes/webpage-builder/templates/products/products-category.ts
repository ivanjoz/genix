import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Category showcase: a heading, the category's own description via <CategoryDescription>,
 * and its products via <ProductsByCategory>. Both custom components self-fetch live data,
 * so in the isolated preview only the heading renders concretely.
 */
export const ProductsCategory: SectionData = {
	id: 'products-category-v1',
	name: 'Category Showcase',
	category: 'products',
	description: 'A category heading and description above a grid of its products.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-slate-50 px-[14px] py-[8px] text-center">
			<div class="mx-auto h-[7px] w-[100px] rounded bg-slate-800"></div>
			<div class="mx-auto mt-[5px] h-[4px] w-[150px] rounded bg-slate-300"></div>
			<div class="mt-[10px] grid grid-cols-3 gap-[8px]">
				<div class="aspect-square rounded bg-slate-200"></div>
				<div class="aspect-square rounded bg-slate-200"></div>
				<div class="aspect-square rounded bg-slate-200"></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="2" class="px-32 py-72">
			<div class="max-w-7xl mx-auto">
				<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black text-center mb-16">Shop by category</h2>
				<CategoryDescription categoryIDs="[22]" class="text-center text-lg max-w-2xl mx-auto mb-48" />
				<ProductsByCategory categoryID="22" maxWidth="1200" rows="2" rowsMobile="2" />
			</div>
		</section>
	`,
};
