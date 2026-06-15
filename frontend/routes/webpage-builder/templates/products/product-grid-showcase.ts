import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Product grid used directly in the AST. Unlike CategoryShowcase (which wraps the grid in
 * ProductsByCategory), here <ProductGrid> self-fetches the catalog. It exercises the builder's
 * grid-layout detection: the editor surfaces maxWidth / maxMargin / rows / rowsMobile for it.
 */
export const ProductGridShowcase: SectionData = {
	id: 'html-product-grid-showcase-v1',
	name: 'Product Grid (HTML)',
	category: 'products',
	description: 'A standalone product grid that self-fetches its products, with editable layout.',
	thumbnail: `
		<div class="h-[112px] bg-white px-6 py-7 text-slate-800">
			<strong class="block text-center text-[7px]">Our Products</strong>
			<div class="mt-7 grid grid-cols-3 gap-4">
				<div><div class="aspect-square rounded-sm bg-slate-200"></div><div class="mt-2 h-2 rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded-sm bg-slate-200"></div><div class="mt-2 h-2 rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded-sm bg-slate-200"></div><div class="mt-2 h-2 rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded-sm bg-slate-200"></div><div class="mt-2 h-2 rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded-sm bg-slate-200"></div><div class="mt-2 h-2 rounded bg-slate-300"></div></div>
				<div><div class="aspect-square rounded-sm bg-slate-200"></div><div class="mt-2 h-2 rounded bg-slate-300"></div></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="1" class="py-16 px-6">
			<div class="max-w-7xl mx-auto">
				<h2 data-role="title" color="9" class="text-3xl font-bold text-center mb-10">
					Our Products
				</h2>
				<ProductGrid categoryID="22" maxWidth="1200" maxMargin="80" rows="2" rowsMobile="3" />
			</div>
		</section>
	`,
};
