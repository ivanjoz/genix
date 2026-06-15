import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * "Trusted by" social-proof strip: an eyebrow line above a row of text wordmarks rendered
 * as muted, bold brand names (stand-ins for client logos, since no <svg> logos are loaded).
 */
export const LogosTrustedBy: SectionData = {
	id: 'logos-trusted-by-v1',
	name: 'Trusted By (logos)',
	category: 'testimonials',
	description: 'A row of partner or press wordmarks for social proof.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col items-center justify-center bg-white px-[12px] text-center">
			<div class="h-[5px] w-[110px] rounded bg-slate-300"></div>
			<div class="mt-[16px] flex flex-wrap items-center justify-center gap-x-[14px] gap-y-[8px] text-[11px] font-black text-slate-400">
				<span>NORTH</span><span class="italic">Lumen</span><span class="tracking-widest">VERTEX</span><span class="lowercase">halo</span>
			</div>
		</div>
	`,
	html: `
		<section background-color="1" class="px-32 py-56">
			<div class="max-w-5xl mx-auto text-center">
				<p data-role="title" color="5" class="text-sm font-semibold uppercase tracking-[0.2em] mb-32">Trusted by teams worldwide</p>
				<div class="flex flex-wrap items-center justify-center gap-x-56 gap-y-24">
					<span color="6" class="text-2xl font-black tracking-tight">NORTHWIND</span>
					<span color="6" class="text-2xl font-black tracking-tight italic">Lumen</span>
					<span color="6" class="text-2xl font-black tracking-widest">VERTEX</span>
					<span color="6" class="text-2xl font-black tracking-tight lowercase">halo</span>
					<span color="6" class="text-2xl font-black tracking-tight">Meridian</span>
				</div>
			</div>
		</section>
	`,
};
