import type { SectionTemplate } from '../../renderer/renderer-types';

/**
 * HTML-based category showcase. Demonstrates custom components inside the HTML
 * (CategoryDescription + ProductsByCategory) alongside native markup, with
 * attribute props coerced via the component schemas.
 */
export const HtmlCategoryShowcase: SectionTemplate = {
	id: 'html-category-showcase-v1',
	name: 'Category Showcase (HTML)',
	category: 'products',
	description: 'A category heading and description followed by a grid of its products.',
	HTML: `
		<section background-color="2" class="py-16 px-6">
			<div class="max-w-7xl mx-auto">
				<h2 data-role="title" color="9" class="text-3xl font-bold text-center mb-3">
					Shop by Category
				</h2>
				<CategoryDescription categoryIDs="[22]" class="text-center text-gray-600 max-w-2xl mx-auto mb-10" />
				<ProductsByCategory categoryID="22" limit="8" />
			</div>
		</section>
	`,
};
