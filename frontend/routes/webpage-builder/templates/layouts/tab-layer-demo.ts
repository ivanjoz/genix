import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * HTML-based tabbed layer. Demonstrates the `<TabbedLayer>` custom component: each of its
 * direct children becomes one tab panel, and an OptionsStrip (labels from the `options`
 * attribute) selects which single panel is shown. AstRenderer passes the layer its
 * children as raw AST nodes (`childNodes`) plus a `renderChild` snippet, so EcommerceTabs
 * renders the active panel's subtree independently. The builder edits one panel at a time
 * via the same OptionsStrip, synced with the live preview.
 */
export const TabbedLayerDemo: SectionData = {
	id: 'tab-layer-demo-v1',
	name: 'Tab Layer (HTML)',
	category: 'hero',
	description: 'Tabbed layer whose direct children each become a selectable panel, rendered via <TabbedLayer>.',
	html: `
		<section class="relative overflow-hidden min-h-[420px] flex items-center">
			<TabbedLayer options="My Style|Option 2|Option 3">
				<div class="relative min-h-[420px] flex items-center px-6 py-12">
					<div class="relative z-10 w-full max-w-6xl mx-auto">
						<div class="max-w-md text-left">
							<h1 data-role="title" color="#ffffff" class="text-4xl md:text-5xl font-black mb-4 leading-tight">
								Style That Speaks
							</h1>
							<p data-role="content" color="#ffffff" class="text-lg mb-8 opacity-90">
								Curated collections for the modern wardrobe, delivered with care.
							</p>
							<a data-role="button" href="/shop" background-color="5" color="1" class="inline-block px-8 py-4 rounded-full font-bold text-lg">
								Shop the Collection
							</a>
						</div>
					</div>
				</div>
				<div class="min-h-[420px] w-full flex items-center justify-center bg-slate-800">
					<h1 class="text-4xl font-black text-white">SLIDE 1</h1>
				</div>
				<div class="min-h-[420px] w-full flex items-center justify-center bg-slate-600">
					<h1 class="text-4xl font-black text-white">SLIDE 2</h1>
				</div>
			</TabbedLayer>
		</section>
	`,
}
