/**
 * Prop schemas for custom components usable inside HTML templates.
 *
 * Each schema declares the component's HTML attributes and how to coerce them
 * into typed Svelte props. This doubles as the spec an agent authors against.
 *
 * Keyed by `tagName` exactly as written in the HTML (case preserved).
 */
import type { ComponentSchema } from './coerce';

export const componentSchemas: Record<string, ComponentSchema> = {
	ProductsByCategory: {
		tagName: 'ProductsByCategory',
		description: 'Grid of products belonging to a single category.',
		props: {
			categoryID: { type: 'number' },
			limit: { type: 'number', default: 8 },
		},
	},

	CategoryDescription: {
		tagName: 'CategoryDescription',
		description: 'Renders the description text of a category.',
		props: {
			categoriasIDs: { type: 'number[]' },
		},
	},

	ProductCard: {
		tagName: 'ProductCard',
		description: 'A single product card.',
		props: {
			productoID: { type: 'number' },
			mode: { type: 'string', enum: ['vertical', 'horizontal'] },
			useQuantityControls: { type: 'boolean' },
			hideCloseButton: { type: 'boolean' },
		},
	},
};

export function getComponentSchema(tagName: string): ComponentSchema | undefined {
	return componentSchemas[tagName];
}
