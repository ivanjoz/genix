import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * FAQ list: a heading and a stack of question/answer pairs separated by hr dividers. Static
 * content (no interactive accordion) — a bold question with a ＋ glyph above each answer.
 */
export const FaqList: SectionData = {
	id: 'faq-list-v1',
	name: 'FAQ List',
	category: 'text',
	description: 'A list of frequently asked questions with answers.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-white px-[18px] py-[10px]">
			<div class="mx-auto mb-[12px] h-[7px] w-[80px] rounded bg-slate-800"></div>
			<div class="flex flex-col gap-[8px]">
				<div class="flex items-center justify-between"><div class="h-[5px] w-3/4 rounded bg-slate-700"></div><span class="text-[10px] text-slate-400">＋</span></div>
				<div class="border-t border-slate-100"></div>
				<div class="flex items-center justify-between"><div class="h-[5px] w-2/3 rounded bg-slate-700"></div><span class="text-[10px] text-slate-400">＋</span></div>
				<div class="border-t border-slate-100"></div>
				<div class="flex items-center justify-between"><div class="h-[5px] w-3/5 rounded bg-slate-700"></div><span class="text-[10px] text-slate-400">＋</span></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-72">
			<div class="max-w-3xl mx-auto">
				<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black text-center mb-48">Frequently asked questions</h2>
				<div class="flex flex-col gap-28">
					<div>
						<h3 color="9" class="text-xl font-bold mb-10">How long does shipping take?</h3>
						<p data-role="content" color="6" class="text-base leading-relaxed">Most orders arrive within 2–5 business days. Express options are available at checkout for next-day delivery.</p>
					</div>
					<hr color="3" class="border-t border-slate-200" />
					<div>
						<h3 color="9" class="text-xl font-bold mb-10">What is your return policy?</h3>
						<p color="6" class="text-base leading-relaxed">You can return any item within 30 days for a full refund — no questions asked. Returns are always free.</p>
					</div>
					<hr color="3" class="border-t border-slate-200" />
					<div>
						<h3 color="9" class="text-xl font-bold mb-10">Do you ship internationally?</h3>
						<p color="6" class="text-base leading-relaxed">Yes, we ship to over 50 countries. Shipping costs and delivery times are calculated at checkout.</p>
					</div>
					<hr color="3" class="border-t border-slate-200" />
					<div>
						<h3 color="9" class="text-xl font-bold mb-10">How can I contact support?</h3>
						<p color="6" class="text-base leading-relaxed">Our team is available 7 days a week via live chat and email, and we typically reply within a few hours.</p>
					</div>
				</div>
			</div>
		</section>
	`,
};
