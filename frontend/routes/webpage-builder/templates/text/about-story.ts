import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * About / brand story: two columns — narrative text on the left (eyebrow, title, two
 * paragraphs, signature line) and a photo on the right via ImageEffect box mode.
 */
export const AboutStory: SectionData = {
	id: 'about-story-v1',
	name: 'About / Story',
	category: 'text',
	description: 'A brand story told in two columns with a supporting photo.',
	thumbnail: `
		<div class="flex h-[100px] w-full items-center gap-[12px] bg-white px-[14px]">
			<div class="flex-1">
				<div class="h-[5px] w-[44px] rounded bg-indigo-500"></div>
				<div class="mt-[6px] h-[8px] w-3/4 rounded bg-slate-800"></div>
				<div class="mt-[7px] h-[4px] w-full rounded bg-slate-300"></div>
				<div class="mt-[3px] h-[4px] w-full rounded bg-slate-300"></div>
				<div class="mt-[3px] h-[4px] w-2/3 rounded bg-slate-300"></div>
			</div>
			<div class="h-[80px] w-[80px] rounded-xl bg-slate-300"></div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-56 items-center">
				<div>
					<span color="#4f46e5" class="inline-block text-sm font-bold uppercase tracking-widest mb-16">Our story</span>
					<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-24 leading-tight">Started in a garage, built on a promise</h2>
					<p data-role="content" color="7" class="text-lg leading-relaxed mb-20">
						We began with a simple idea: make products we'd be proud to use ourselves, and treat
						every customer the way we'd want to be treated.
					</p>
					<p color="7" class="text-lg leading-relaxed mb-28">
						A decade later, that promise still guides everything — from how we source materials to
						how we pack each order.
					</p>
					<div color="9" class="text-xl italic font-semibold">— The founding team</div>
				</div>
				<ImageEffect data-role="image"
					src="https://ivanjoz.github.io/genix-assets/images/business-workspace/9.avif"
					fit="cover" aspectRatio="1/1" class="rounded-3xl overflow-hidden shadow-xl" />
			</div>
		</section>
	`,
};
