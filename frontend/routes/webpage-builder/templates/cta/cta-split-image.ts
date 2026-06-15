import type { SectionData } from '$ecommerce/renderer/section-types';

/**
 * Call-to-action with a duotone photo clipped on the right (ImageEffect curve-right) and a
 * text + button panel filling the box's left side. Box mode: children render inside the photo.
 */
export const CtaSplitImage: SectionData = {
	id: 'cta-split-image-v1',
	name: 'CTA Split w/ Image',
	category: 'cta',
	description: 'Call-to-action overlaid on a duotone photo with a curved clip.',
	thumbnail: `
		<div class="relative h-[100px] w-full overflow-hidden bg-violet-900 text-white">
			<div class="absolute inset-0 bg-gradient-to-r from-violet-900 via-violet-800 to-amber-500/50"></div>
			<div class="relative flex h-full w-3/5 flex-col justify-center px-[16px]">
				<div class="h-[11px] w-[110px] rounded bg-white/90"></div>
				<div class="mt-[7px] h-[5px] w-[90px] rounded bg-white/50"></div>
				<div class="mt-[12px] h-[16px] w-[70px] rounded-full bg-white"></div>
			</div>
		</div>
	`,
	html: `
		<section background-color="2" class="px-32 py-72">
			<div class="max-w-6xl mx-auto">
				<ImageEffect data-role="image"
					src="https://ivanjoz.github.io/genix-assets/images/business-workspace/5.avif"
					layout="curve-right" effect="duotone-matrix" tint="#4c1d95" tint2="#f59e0b"
					intensity="1" aspectRatio="21/9"
					class="rounded-3xl overflow-hidden min-h-[360px] flex items-center p-48">
					<div class="max-w-md text-left">
						<h2 data-role="title" color="#ffffff" class="text-3xl md:text-4xl font-black leading-tight mb-16">
							Members save more
						</h2>
						<p data-role="content" color="#ffffff" class="text-lg mb-28 opacity-90">
							Sign up for early access to drops, members-only pricing and free express shipping.
						</p>
						<a data-role="button" href="/join" background-color="#ffffff" color="#4c1d95" class="inline-block rounded-full px-36 py-16 text-lg font-bold">
							Become a member
						</a>
					</div>
				</ImageEffect>
			</div>
		</section>
	`,
};
