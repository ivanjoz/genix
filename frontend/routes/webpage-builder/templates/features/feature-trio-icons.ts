import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Three feature cards, each with an emoji glyph inside a rounded color tile (CSS shape,
 * since no <svg> is available), a sub-heading and a short blurb. Responsive 1->3 columns.
 */
export const FeatureTrioIcons: SectionData = {
	id: 'feature-trio-icons-v1',
	name: 'Feature Trio',
	category: 'features',
	description: 'Three icon cards highlighting key benefits or services.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-white px-[14px] py-[10px]">
			<div class="mx-auto mb-[10px] h-[7px] w-[90px] rounded bg-slate-800"></div>
			<div class="grid grid-cols-3 gap-[8px]">
				<div class="rounded-md border border-slate-200 p-[6px] text-center"><div class="text-[14px] leading-none">🚚</div><div class="mt-[5px] h-[4px] w-full rounded bg-slate-300"></div><div class="mt-[2px] h-[4px] w-4/5 rounded bg-slate-200"></div></div>
				<div class="rounded-md border border-slate-200 p-[6px] text-center"><div class="text-[14px] leading-none">🔒</div><div class="mt-[5px] h-[4px] w-full rounded bg-slate-300"></div><div class="mt-[2px] h-[4px] w-4/5 rounded bg-slate-200"></div></div>
				<div class="rounded-md border border-slate-200 p-[6px] text-center"><div class="text-[14px] leading-none">↩️</div><div class="mt-[5px] h-[4px] w-full rounded bg-slate-300"></div><div class="mt-[2px] h-[4px] w-4/5 rounded bg-slate-200"></div></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-7xl mx-auto">
				<div class="text-center mb-56">
					<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-16">Why shop with us</h2>
					<p data-role="content" color="6" class="text-lg max-w-2xl mx-auto">Everything we do is built around making your experience effortless.</p>
				</div>
				<div class="grid grid-cols-1 md:grid-cols-3 gap-32">
					<div background-color="#ffffff" class="rounded-2xl p-32 shadow-sm border border-slate-100">
						<div background-color="#eef2ff" class="w-56 h-56 rounded-2xl flex items-center justify-center text-3xl mb-20">🚚</div>
						<h3 color="9" class="text-xl font-bold mb-10">Fast, free shipping</h3>
						<p color="6" class="text-base leading-relaxed">Free delivery on every order over $50, dispatched within 24 hours.</p>
					</div>
					<div background-color="#ffffff" class="rounded-2xl p-32 shadow-sm border border-slate-100">
						<div background-color="#ecfdf5" class="w-56 h-56 rounded-2xl flex items-center justify-center text-3xl mb-20">🔒</div>
						<h3 color="9" class="text-xl font-bold mb-10">Secure checkout</h3>
						<p color="6" class="text-base leading-relaxed">Your payment is protected with bank-grade encryption, every time.</p>
					</div>
					<div background-color="#ffffff" class="rounded-2xl p-32 shadow-sm border border-slate-100">
						<div background-color="#fef3c7" class="w-56 h-56 rounded-2xl flex items-center justify-center text-3xl mb-20">↩️</div>
						<h3 color="9" class="text-xl font-bold mb-10">Easy 30-day returns</h3>
						<p color="6" class="text-base leading-relaxed">Changed your mind? Return anything within 30 days, no questions asked.</p>
					</div>
				</div>
			</div>
		</section>
	`,
};
