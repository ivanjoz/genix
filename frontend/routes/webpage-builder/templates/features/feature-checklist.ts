import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Benefits checklist: a title + bulleted list of ✓ items on the left, a clipped photo on
 * the right (ImageEffect curve-left). The check marks are emoji glyphs in list items.
 */
export const FeatureChecklist: SectionData = {
	id: 'feature-checklist-v1',
	name: 'Benefits Checklist',
	category: 'features',
	description: 'A checklist of benefits beside a feature photo with a curved clip.',
	thumbnail: `
		<div class="flex h-[100px] w-full items-center gap-[12px] bg-white px-[14px]">
			<div class="flex-1 flex flex-col gap-[6px]">
				<div class="h-[8px] w-2/3 rounded bg-slate-800"></div>
				<div class="flex items-center gap-[5px]"><span class="text-[10px] leading-none">✅</span><div class="h-[5px] w-full rounded bg-slate-300"></div></div>
				<div class="flex items-center gap-[5px]"><span class="text-[10px] leading-none">✅</span><div class="h-[5px] w-5/6 rounded bg-slate-300"></div></div>
				<div class="flex items-center gap-[5px]"><span class="text-[10px] leading-none">✅</span><div class="h-[5px] w-4/6 rounded bg-slate-300"></div></div>
			</div>
			<div class="h-[84px] w-[72px] rounded-xl bg-slate-300"></div>
		</div>
	`,
	html: `
		<section background-color="2" class="px-32 py-72">
			<div class="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-56 items-center">
				<div>
					<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-24">Everything included, nothing hidden</h2>
					<p data-role="content" color="7" class="text-lg leading-relaxed mb-28">One simple price gets you the whole experience — no surprises at checkout.</p>
					<ul class="list-none flex flex-col gap-16">
						<li class="flex items-start gap-12"><span class="text-xl leading-none">✅</span><span color="8" class="text-lg">Free worldwide shipping & returns</span></li>
						<li class="flex items-start gap-12"><span class="text-xl leading-none">✅</span><span color="8" class="text-lg">2-year warranty on every product</span></li>
						<li class="flex items-start gap-12"><span class="text-xl leading-none">✅</span><span color="8" class="text-lg">Dedicated support, 7 days a week</span></li>
						<li class="flex items-start gap-12"><span class="text-xl leading-none">✅</span><span color="8" class="text-lg">Carbon-neutral delivery</span></li>
					</ul>
				</div>
				<ImageEffect data-role="image"
					src="https://ivanjoz.github.io/genix-assets/images/business-workspace/8.avif"
					fit="cover" aspectRatio="4/5"
					class="rounded-3xl overflow-hidden shadow-xl min-h-[360px]" />
			</div>
		</section>
	`,
};
