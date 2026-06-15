import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Three testimonial cards in a responsive grid. Star ratings are text glyphs (★), the
 * avatar is an initial inside a colored circle (CSS shape — no <svg> / external avatar).
 */
export const TestimonialsGrid: SectionData = {
	id: 'testimonials-grid-v1',
	name: 'Testimonials Grid',
	category: 'testimonials',
	description: 'Three customer testimonial cards with ratings and author details.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-slate-50 px-[12px] py-[10px]">
			<div class="mx-auto mb-[10px] h-[7px] w-[90px] rounded bg-slate-800"></div>
			<div class="grid grid-cols-3 gap-[8px]">
				<div class="rounded-md bg-white p-[6px] shadow-sm"><div class="text-[8px] text-amber-400">★★★★★</div><div class="mt-[4px] h-[4px] w-full rounded bg-slate-200"></div><div class="mt-[2px] h-[4px] w-4/5 rounded bg-slate-200"></div><div class="mt-[5px] h-[8px] w-[8px] rounded-full bg-indigo-500"></div></div>
				<div class="rounded-md bg-white p-[6px] shadow-sm"><div class="text-[8px] text-amber-400">★★★★★</div><div class="mt-[4px] h-[4px] w-full rounded bg-slate-200"></div><div class="mt-[2px] h-[4px] w-4/5 rounded bg-slate-200"></div><div class="mt-[5px] h-[8px] w-[8px] rounded-full bg-teal-500"></div></div>
				<div class="rounded-md bg-white p-[6px] shadow-sm"><div class="text-[8px] text-amber-400">★★★★★</div><div class="mt-[4px] h-[4px] w-full rounded bg-slate-200"></div><div class="mt-[2px] h-[4px] w-4/5 rounded bg-slate-200"></div><div class="mt-[5px] h-[8px] w-[8px] rounded-full bg-pink-500"></div></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="2" class="px-32 py-72">
			<div class="max-w-7xl mx-auto">
				<div class="text-center mb-56">
					<h2 data-role="title" color="10" class="text-3xl md:text-4xl font-black mb-16">Loved by our customers</h2>
					<p data-role="content" color="6" class="text-lg max-w-2xl mx-auto">Don't just take our word for it — here's what people are saying.</p>
				</div>
				<div class="grid grid-cols-1 md:grid-cols-3 gap-32">
					<div background-color="#ffffff" class="rounded-2xl p-32 shadow-sm border border-slate-100">
						<div color="#f59e0b" class="text-lg mb-16">★★★★★</div>
						<p color="8" class="text-lg leading-relaxed mb-24">"Genuinely the best online shopping experience I've had. Fast delivery and the quality exceeded my expectations."</p>
						<div class="flex items-center gap-16">
							<div background-color="#4f46e5" color="#ffffff" class="w-44 h-44 rounded-full flex items-center justify-center font-bold">A</div>
							<div><div color="9" class="font-bold">Ana Reyes</div><div color="5" class="text-sm">Verified buyer</div></div>
						</div>
					</div>
					<div background-color="#ffffff" class="rounded-2xl p-32 shadow-sm border border-slate-100">
						<div color="#f59e0b" class="text-lg mb-16">★★★★★</div>
						<p color="8" class="text-lg leading-relaxed mb-24">"Beautiful products and effortless checkout. Customer support answered my question within minutes."</p>
						<div class="flex items-center gap-16">
							<div background-color="#0d9488" color="#ffffff" class="w-44 h-44 rounded-full flex items-center justify-center font-bold">M</div>
							<div><div color="9" class="font-bold">Marco Díaz</div><div color="5" class="text-sm">Verified buyer</div></div>
						</div>
					</div>
					<div background-color="#ffffff" class="rounded-2xl p-32 shadow-sm border border-slate-100">
						<div color="#f59e0b" class="text-lg mb-16">★★★★★</div>
						<p color="8" class="text-lg leading-relaxed mb-24">"I keep coming back. Thoughtful packaging, great prices and everything arrives exactly as described."</p>
						<div class="flex items-center gap-16">
							<div background-color="#db2777" color="#ffffff" class="w-44 h-44 rounded-full flex items-center justify-center font-bold">S</div>
							<div><div color="9" class="font-bold">Sofía Luna</div><div color="5" class="text-sm">Verified buyer</div></div>
						</div>
					</div>
				</div>
			</div>
		</section>
	`,
};
