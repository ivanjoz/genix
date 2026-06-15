import { editorStore } from './editor.svelte';
import { collectTokens, generateCss } from '$ecommerce/stores/uno-generator';

/**
 * Live CSS for runtime-authored Tailwind classes in the builder.
 *
 * Drives one injected <style> tag. Instead of a global MutationObserver, we
 * regenerate on demand from the editor state we already track: `update()`
 * collects the class tokens (slot CSS + HTML section AST + text lines) and runs
 * the real UnoCSS engine over them (see uno-generator.ts).
 *
 * `update()` reads `editorStore.sections` synchronously (via collectTokens)
 * before awaiting, so a Svelte `$effect` that calls it tracks those reads as
 * dependencies and re-runs on any change. Generation is async, so a sequence
 * counter guards against a slow run overwriting a newer one (latest wins).
 */
class LiveCSSStore {
	css = $state('');
	private seq = 0;

	async update() {
		// Synchronous read -> registers reactive dependencies for the caller's $effect.
		const tokens = collectTokens(editorStore.sections);
		const current = ++this.seq;

		const css = await generateCss(tokens);

		// Drop stale results from a run superseded by a newer edit.
		if (current !== this.seq) return;
		this.css = css;
	}
}

export const liveCSS = new LiveCSSStore();
