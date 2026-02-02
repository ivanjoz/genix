import type { ComponentAST } from '../../pkg-store/renderer/renderer-types';
import { FooterBasic } from '$store/ecommerce-templates/templates/footer-basic'
import { CategoryGridSection } from '$store/ecommerce-templates/templates/category-grid'
import { CategoryInfoDescription } from '$store/ecommerce-templates/product-categories/category-info-description'
import { CategoryWithProducts } from '$store/ecommerce-templates/product-categories/category-with-products'

export const storeExample: ComponentAST[] = [
	{
		tagName: 'section',
		semanticTag: 'section',
		css: 'bg-primary-500 py-20 px-4 text-center',
		children: [
			{
				tagName: 'h1',
				semanticTag: 'h1',
				css: 'text-5xl font-bold mb-4',
				text: 'Welcome to Our Amazing Store'
			},
			{
				tagName: 'p',
				semanticTag: 'p',
				css: 'text-xl mb-8',
				text: 'Discover the best products at unbeatable prices.'
			},
			{
				tagName: 'a',
				semanticTag: 'a',
				css: 'bg-white text-primary-600 px-8 py-3 rounded-full font-semibold hover:bg-gray-100 transition-colors',
				text: 'Shop Now',
				attributes: {
					href: '/shop'
				}
			}
		]
	},
	{
		tagName: 'section',
		semanticTag: 'section',
		css: 'py-16 px-4 max-w-7xl mx-auto',
		children: [
			{
				tagName: 'h2',
				semanticTag: 'h2',
				css: 'text-3xl font-bold mb-10 text-center',
				text: 'Featured Products'
			},
			{
				tagName: 'div',
				css: 'grid grid-cols-1 md:grid-cols-3 gap-8',
				children: [
					{
						tagName: 'ProductCard',
						productosIDs: [1],
						css: 'border rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow'
					},
					{
						tagName: 'ProductCard',
						productosIDs: [4],
						css: 'border rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow'
					},
					{
						tagName: 'ProductCard',
						productosIDs: [5],
						css: 'border rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow'
					}
				]
			}
		]
	},
	{
		tagName: 'section',
		css: 'bg-primary-500 py-20 px-4 text-center',
		children: [
			{
				tagName: 'h2',
				text: 'This are our amazing categories',
				css: 'text-3xl font-bold mb-8'
			},
			{
				tagName: 'CategoryDescription',
				css: 'text-lg mb-8',
				categoriasIDs: [13]
			},
			{
				tagName: 'ProductsByCategory',
				limit: 4,
				categoriasIDs: [13]
			}
		]
	},
	CategoryInfoDescription.ast,
	CategoryWithProducts.ast,
	CategoryGridSection.ast,
	FooterBasic.ast
];
