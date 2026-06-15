import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Site footer on a dark background: a brand blurb column, three link columns, a small
 * newsletter mock (styled, since <input> isn't renderable), and a bottom copyright row
 * separated by an hr. Lists use list-none with palette text colors.
 */
export const SiteFooter: SectionData = {
	id: 'site-footer-v1',
	name: 'Footer',
	category: 'footer',
	description: 'Multi-column dark footer with links, newsletter and copyright.',
	thumbnail: `
		<div class="flex h-[100px] w-full flex-col justify-center bg-slate-900 px-[14px] py-[12px] text-white">
			<div class="grid grid-cols-4 gap-[12px]">
				<div><div class="h-[8px] w-[40px] rounded bg-white"></div><div class="mt-[6px] h-[4px] w-[48px] rounded bg-slate-600"></div><div class="mt-[3px] h-[4px] w-[40px] rounded bg-slate-600"></div></div>
				<div><div class="h-[4px] w-[32px] rounded bg-slate-500"></div><div class="mt-[6px] h-[4px] w-[40px] rounded bg-slate-600"></div><div class="mt-[3px] h-[4px] w-[32px] rounded bg-slate-600"></div></div>
				<div><div class="h-[4px] w-[32px] rounded bg-slate-500"></div><div class="mt-[6px] h-[4px] w-[40px] rounded bg-slate-600"></div><div class="mt-[3px] h-[4px] w-[32px] rounded bg-slate-600"></div></div>
				<div><div class="h-[4px] w-[40px] rounded bg-slate-500"></div><div class="mt-[6px] h-[16px] w-full rounded bg-slate-700"></div></div>
			</div>
			<div class="mt-[12px] border-t border-slate-700 pt-[8px]"><div class="h-[4px] w-[90px] rounded bg-slate-600"></div></div>
		</div>
	`,
	html: `
		<footer background-color="10" class="px-32 py-64">
			<div class="max-w-7xl mx-auto">
				<div class="grid grid-cols-2 md:grid-cols-4 gap-40">
					<div class="col-span-2 md:col-span-1">
						<div data-role="title" color="#ffffff" class="text-2xl font-black tracking-tight mb-16">ACME&nbsp;Store</div>
						<p data-role="content" color="6" class="text-base leading-relaxed max-w-xs">Thoughtfully designed products, delivered with care to your door since 2014.</p>
					</div>
					<div>
						<div color="5" class="text-sm font-bold uppercase tracking-widest mb-20">Shop</div>
						<ul class="list-none flex flex-col gap-12">
							<li><a href="/new" color="7" class="text-base">New arrivals</a></li>
							<li><a href="/best" color="7" class="text-base">Best sellers</a></li>
							<li><a href="/sale" color="7" class="text-base">Sale</a></li>
						</ul>
					</div>
					<div>
						<div color="5" class="text-sm font-bold uppercase tracking-widest mb-20">Company</div>
						<ul class="list-none flex flex-col gap-12">
							<li><a href="/about" color="7" class="text-base">About us</a></li>
							<li><a href="/careers" color="7" class="text-base">Careers</a></li>
							<li><a href="/contact" color="7" class="text-base">Contact</a></li>
						</ul>
					</div>
					<div>
						<div color="5" class="text-sm font-bold uppercase tracking-widest mb-20">Stay in touch</div>
						<p color="7" class="text-base mb-16">Subscribe for offers & updates.</p>
						<div class="flex gap-8">
							<div background-color="#1e293b" color="5" class="flex-1 rounded-lg px-16 py-12 text-sm">you@email.com</div>
							<a href="/subscribe" background-color="#4f46e5" color="#ffffff" class="inline-block rounded-lg px-20 py-12 text-sm font-bold">Join</a>
						</div>
					</div>
				</div>
				<hr color="8" class="border-t border-slate-700 my-40" />
				<div class="flex flex-col md:flex-row items-center justify-between gap-16">
					<div color="5" class="text-sm">© 2024 ACME Store. All rights reserved.</div>
					<div class="flex gap-24">
						<a href="/privacy" color="6" class="text-sm">Privacy</a>
						<a href="/terms" color="6" class="text-sm">Terms</a>
						<a href="/cookies" color="6" class="text-sm">Cookies</a>
					</div>
				</div>
			</div>
		</footer>
	`,
};
