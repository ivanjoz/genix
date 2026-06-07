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
			categoryIDs: { type: 'number[]' },
		},
	},

	ImageEffect: {
		tagName: 'ImageEffect',
		description:
			'Image with a layout (composition/clip shape) and a visual effect (duotone, glass, vignette...). Overlay markup goes inside as children.',
		props: {
			src: { type: 'string' },
			layout: { type: 'string' },
			effect: { type: 'string' },
			// `color`/`color2` props drive the duotone/overlay tints. We expose them as
			// `tint`/`tint2` because the bare `color` attribute is already consumed by the
			// style-attribute compiler (text color), so it never reaches props.
			tint: { type: 'string', prop: 'color' },
			tint2: { type: 'string', prop: 'color2' },
			intensity: { type: 'number' },
			blur: { type: 'number' },
			aspectRatio: { type: 'string' },
			// `fill` turns the image into an absolute background layer of its parent
			// (which must be positioned); `fit` controls object-fit/position.
			fill: { type: 'boolean' },
			fit: { type: 'string', enum: ['cover', 'contain', 'contain-left', 'contain-right'] },
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
