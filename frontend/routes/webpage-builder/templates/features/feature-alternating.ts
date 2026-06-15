import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Two alternating feature rows: photo + text, with the image on opposite sides each row
 * (md:order utilities flip the second row). Uses ImageEffect box mode with rounded photos.
 */
export const FeatureAlternating: SectionData = {
	id: 'feature-alternating-v1',
	name: 'Feature Alternating Rows',
	category: 'features',
	description: 'Alternating image-and-text rows for storytelling product features.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center gap-[10px] bg-white px-[14px]">
			<div class="flex items-center gap-[10px]"><div class="h-[36px] w-[56px] rounded bg-slate-300"></div><div class="flex-1"><div class="h-[6px] w-3/4 rounded bg-slate-800"></div><div class="mt-[4px] h-[4px] w-full rounded bg-slate-300"></div><div class="mt-[2px] h-[4px] w-5/6 rounded bg-slate-200"></div></div></div>
			<div class="flex items-center gap-[10px]"><div class="flex-1"><div class="h-[6px] w-3/4 rounded bg-slate-800"></div><div class="mt-[4px] h-[4px] w-full rounded bg-slate-300"></div><div class="mt-[2px] h-[4px] w-5/6 rounded bg-slate-200"></div></div><div class="h-[36px] w-[56px] rounded bg-slate-300"></div></div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-6xl mx-auto flex flex-col gap-72">
				<div class="grid grid-cols-1 md:grid-cols-2 gap-48 items-center">
					<ImageEffect data-role="image"
						src="https://ivanjoz.github.io/genix-assets/images/business-workspace/6.avif"
						fit="cover" aspectRatio="4/3" class="rounded-3xl overflow-hidden shadow-lg" />
					<div>
						<span color="#4f46e5" class="inline-block text-sm font-bold uppercase tracking-widest mb-12">Quality first</span>
						<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-16">Materials you can feel</h2>
						<p data-role="content" color="7" class="text-lg leading-relaxed">Every product starts with responsibly sourced materials, chosen for how they look, feel and last over time.</p>
					</div>
				</div>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-48 items-center">
					<div class="md:order-2">
						<ImageEffect data-role="image"
							src="https://ivanjoz.github.io/genix-assets/images/business-workspace/7.avif"
							fit="cover" aspectRatio="4/3" class="rounded-3xl overflow-hidden shadow-lg" />
					</div>
					<div class="md:order-1">
						<span color="#0d9488" class="inline-block text-sm font-bold uppercase tracking-widest mb-12">Made to last</span>
						<h2 color="10" class="text-3xl md:text-4xl font-black mb-16">Designed for everyday</h2>
						<p color="7" class="text-lg leading-relaxed">Functional details and timeless shapes mean our pieces fit seamlessly into your routine — and stay there.</p>
					</div>
				</div>
			</div>
		</section>
	`,
};
