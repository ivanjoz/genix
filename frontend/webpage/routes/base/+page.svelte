<script lang="ts">
	import { Env } from '$core/env';
	import MobileMenu from '$domain/MobileMenu.svelte';
	import Header from '$ecommerce/components/Header.svelte';
	import EcommerceRenderer from '$ecommerce/renderer/EcommerceRenderer.svelte';
	import { getStoreSnapshotName, parseStoreWebpageSnapshot } from '$services/ecommerce/page-content.svelte';
	import type { SectionData } from '$ecommerce/renderer/section-types';
	import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
	import { onMount } from 'svelte';

	// This route is the runtime ("live") storefront: instead of prerendered per-company
	// content, it reads the published CDN snapshot (live/pages/<companyID>-<pageID>.json)
	// at view time. The inline <head> script below starts that fetch before the JS bundle
	// even loads (parking it on window._pageContentPromise); onMount just awaits it.
	let isLoading = $state(true);
	let sections = $state<SectionData[]>([]);
	let runtimeCss = $state('');
	let seo = $state<Record<string, string>>({});

	// The CDN snapshot file name + CDN host, baked into <head> metas so the early inline
	// fetch and the onMount fallback resolve the exact same URL.
	const snapshotName = getStoreSnapshotName();

	// Matches the builder's default palette so `--color-N` vars resolve identically.
	const defaultPalette: ColorPalette = {
		id: 'default',
		name: 'Default Palette',
		colors: [
			'#f8fafc', '#f1f5f9', '#e2e8f0', '#cbd5e1', '#94a3b8',
			'#64748b', '#475569', '#334155', '#1e293b', '#0f172a'
		]
	};

	onMount(async () => {
		try {
			// The inline <head> script normally starts this fetch before the bundle loads.
			// If it didn't run (e.g. dev/CSR), rebuild the same request HERE from the
			// `page-id` meta header first — same source of truth as the inline script —
			// and only synthesize the snapshot name from the company id as a last resort.
			if (!window._pageContentPromise) {
				console.log("No se encontró el: pageContentPromise")
				const file = document.getElementById('page-id')?.getAttribute('content') || snapshotName;
				const url = Env.makeCDNRoute('live', 'pages', file);
				window._pageContentPromise = fetch(url).then((response) => response.json());
			}

			const content = parseStoreWebpageSnapshot(await window._pageContentPromise);
			sections = content.sections;
			runtimeCss = content.css;
			seo = content.seo;
		} catch (contentLoadError) {
			console.error('[LiveStorePage] content load failed', contentLoadError);
		} finally {
			isLoading = false;
		}
	});
</script>

<svelte:head>
	{@html `<meta id="cdn-url" content="${Env.CDN_URL}">`}
	{@html `<meta id="page-id" content="${snapshotName}">`}
	{@html `<script>
		(function(){
			var cdn = document.getElementById("cdn-url")?.getAttribute("content") || "";
			var file = document.getElementById("page-id")?.getAttribute("content") || "";
			if (cdn && file) {
				cdn = cdn.replace(/^https?:\\/\\//, "").replace(/\\/+$/, "");
				window._pageContentPromise = fetch("https://" + cdn + "/live/pages/" + file)
					.then(function(response){ return response.json(); });
			}
		})();
	</script>`}

	<title>{seo.title || "-"}</title>
	<meta name="description" content={seo.description||"-"} />
	<meta name="keywords" content={seo.keywords || "-"} />
	<meta property="og:title" content={seo.ogTitle || "-"} />
	<meta property="og:description" content={seo.ogDescription || "-"} />
	<meta property="og:image" content={seo.ogImage || "-"} />
	<link rel="icon" href={seo.favicon || "-"} />
	{#if runtimeCss}
		{@html `<style id="store-runtime-css">${runtimeCss}</style>`}
	{/if}
</svelte:head>

{#if isLoading}
	<div class="live-page-loader" role="status" aria-label="Cargando">
		<span class="live-page-spinner mr-8"></span>
		<span>Cargando Página...</span>
	</div>
{:else}
	<Header />
	<MobileMenu />
	<EcommerceRenderer elements={sections} palette={defaultPalette} />
{/if}

<style>
	.live-page-loader {
		position: fixed;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #fff;
	}

	.live-page-spinner {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		border: 4px solid #e2e8f0;
		border-top-color: #475569;
		animation: live-page-spin 0.8s linear infinite;
	}

	@keyframes live-page-spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
