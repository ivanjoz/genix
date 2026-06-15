import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Site header / navigation bar. Pure native markup: a flex row with a text logo,
 * centered nav links and a call-to-action button. Spacing is px-accurate because the
 * project sets Tailwind --spacing to 1px (so px-32 = 32px).
 */
export const SiteHeaderNav: SectionData = {
	id: 'site-header-nav-v1',
	name: 'Header / Nav Bar',
	category: 'header',
	description: 'Top navigation bar with a logo, menu links and a call-to-action button.',
	thumbnail: `
		<div class="flex h-[100px] w-full items-center justify-between bg-white px-[16px]">
			<strong class="text-[12px] text-slate-900">ACME</strong>
			<div class="flex gap-[10px] text-[8px] text-slate-500"><span>Shop</span><span>About</span><span>Contact</span></div>
			<span class="rounded bg-indigo-600 px-[10px] py-[5px] text-[8px] text-white">Sign in</span>
		</div>
	`,
	html: `
		<header background-color="1" class="w-full px-32 py-20 shadow-sm">
			<nav class="max-w-7xl mx-auto flex items-center justify-between gap-24">
				<a data-role="title" href="/" color="10" class="text-2xl font-black tracking-tight">ACME&nbsp;Store</a>
				<ul class="hidden md:flex items-center gap-32 list-none">
					<li><a href="/shop" color="8" class="text-base font-medium">Shop</a></li>
					<li><a href="/collections" color="8" class="text-base font-medium">Collections</a></li>
					<li><a href="/about" color="8" class="text-base font-medium">About</a></li>
					<li><a href="/contact" color="8" class="text-base font-medium">Contact</a></li>
				</ul>
				<a data-role="button" href="/account" background-color="#4f46e5" color="#ffffff" class="inline-block rounded-lg px-24 py-12 text-base font-bold">
					Sign in
				</a>
			</nav>
		</header>
	`,
};
