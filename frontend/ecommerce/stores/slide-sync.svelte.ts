/**
 * Shared "active slide" index between the builder editor and the live preview.
 *
 * Keyed by the slide-set array identity (`node.children`). The SAME array reference
 * is held by AstEditor (the slider node's `children`) and passed to EcommerceSlider
 * (its `childNodes` prop), so both sides agree on the current slide without threading
 * props through AstRenderer. Used only in builder mode; production sliders keep their
 * own internal index (so this map never grows on live pages).
 */
import { SvelteMap } from 'svelte/reactivity';

const active = new SvelteMap<unknown, number>();

export const slideSync = {
	/** Current slide for a slide set (defaults to 0). */
	get(key: unknown): number {
		return active.get(key) ?? 0;
	},
	/** Move a slide set to index `i`. */
	set(key: unknown, i: number): void {
		active.set(key, i);
	},
	has(key: unknown): boolean {
		return active.has(key);
	},
};
