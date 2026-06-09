/**
 * Maps custom component tag names (as used in HTML templates) to the real
 * Svelte components that render them. Seeded with the components that exist
 * today; extend as more from ECOMMERCE_COMPONENTS.md get built.
 */
import type { Component } from 'svelte';
import ProductsByCategory from '$ecommerce/ecommerce-components/ProductsByCategory.svelte';
import CategoryDescription from '$ecommerce/ecommerce-components/ecommerce-attributes/CategoryDescription.svelte';
import ProductCard from '$ecommerce/components/ProductCard.svelte';
import ImageEffect from '$ecommerce/components/ImageEffect.svelte';
import EcommerceSlider from '$ecommerce/components/EcommerceSlider.svelte';
import EcommerceTabs from '$ecommerce/components/EcommerceTabs.svelte';

export const astComponentRegistry: Record<string, Component<any>> = {
	ProductsByCategory,
	CategoryDescription,
	ProductCard,
	ImageEffect,
	Slider: EcommerceSlider,
	TabbedLayer: EcommerceTabs,
};

export function isCustomComponentTag(tagName: string): boolean {
	return tagName in astComponentRegistry;
}
