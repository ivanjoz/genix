import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Minimal centered hero: small uppercase eyebrow, large headline, a single supporting
 * line and a primary + ghost button pair. Generous whitespace, no imagery.
 */
export const HeroMinimalEyebrow: SectionData = {
	id: 'hero-minimal-eyebrow-v1',
	name: 'Hero Minimal',
	category: 'hero',
	description: 'Clean centered hero with an eyebrow label, headline and dual buttons.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col items-center justify-center bg-white px-[16px] text-center">
			<div class="h-[5px] w-[60px] rounded bg-indigo-500"></div>
			<div class="mt-[10px] h-[14px] w-[150px] rounded bg-slate-900"></div>
			<div class="mt-[8px] h-[5px] w-[120px] rounded bg-slate-300"></div>
			<div class="mt-[12px] flex gap-[10px]"><span class="h-[14px] w-[52px] rounded-full bg-slate-900"></span><span class="h-[14px] w-[52px] rounded-full border border-slate-300"></span></div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-96 text-center">
			<div class="max-w-3xl mx-auto">
				<span color="#4f46e5" class="inline-block text-sm font-bold uppercase tracking-[0.2em] mb-20">Handcrafted goods</span>
				<h1 data-role="title" color="10" class="text-4xl md:text-6xl font-black leading-tight mb-20">
					Less, but better
				</h1>
				<p data-role="content" color="7" class="text-lg md:text-xl leading-relaxed mb-40 max-w-xl mx-auto">
					A curated edit of timeless pieces, made with care and built to be kept.
				</p>
				<div class="flex flex-wrap justify-center gap-16">
					<a data-role="button" href="/shop" background-color="10" color="1" class="inline-block rounded-full px-36 py-16 text-lg font-bold">
						Browse the edit
					</a>
					<a href="/story" color="9" class="inline-block rounded-full px-36 py-16 text-lg font-bold border-2 border-slate-300">
						Our story
					</a>
				</div>
			</div>
		</section>
	`,
};
