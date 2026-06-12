import type { SectionData } from '../../renderer/section-types';

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
