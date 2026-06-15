import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Newsletter signup. Since <input> is not a renderable native tag here, the email field is
 * a styled placeholder mock (a bordered div with greyed text) sitting beside a Subscribe
 * button — visually a signup form, edited as static content.
 */
export const CtaNewsletter: SectionData = {
	id: 'cta-newsletter-v1',
	name: 'Newsletter Signup',
	category: 'cta',
	description: 'Email newsletter signup with a headline and subscribe button.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col items-center justify-center bg-slate-50 px-[16px] text-center">
			<div class="h-[10px] w-[150px] rounded bg-slate-800"></div>
			<div class="mt-[7px] h-[5px] w-[120px] rounded bg-slate-300"></div>
			<div class="mt-[12px] flex w-[180px] gap-[6px]">
				<div class="h-[20px] flex-1 rounded-full border border-slate-300 bg-white"></div>
				<div class="h-[20px] w-[58px] rounded-full bg-indigo-600"></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="2" class="px-32 py-80 text-center">
			<div class="max-w-2xl mx-auto">
				<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-16">Get 10% off your first order</h2>
				<p data-role="content" color="7" class="text-lg mb-32">Subscribe for new arrivals, exclusive offers and styling tips — straight to your inbox.</p>
				<div class="flex flex-col sm:flex-row gap-12 max-w-md mx-auto">
					<div background-color="#ffffff" color="5" class="flex-1 text-left rounded-full border border-slate-300 px-24 py-16 text-base">
						you@email.com
					</div>
					<a data-role="button" href="/subscribe" background-color="#4f46e5" color="#ffffff" class="inline-block rounded-full px-32 py-16 text-base font-bold whitespace-nowrap">
						Subscribe
					</a>
				</div>
				<p color="5" class="text-xs mt-16">No spam. Unsubscribe anytime.</p>
			</div>
		</section>
	`,
};
