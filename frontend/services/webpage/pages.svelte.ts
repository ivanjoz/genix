import { GET, GetHandler, POST } from '$libs/http.svelte';
import { Env } from '$core/env';
import { fileToImage } from '$libs/helpers';
import { tr } from '$core/store.svelte';
import { addProcess, updateProcess } from '$core/notifications.svelte';

// One storefront page. ID is reused as the PageID of the builder content.
// Status: 0 removed, 1 active, 2 published. IDs 10-14 are injected system pages
// (returned by the server, not stored) and are read-only in the UI.
export interface IWebpage {
  ID: number;
  Name: string;
  Route: string;
  // Numeric ID of the page thumbnail image (0/undefined = none). Upload comes later.
  Image?: number;
  UpdatedBy?: number;
  ss: number;
  upd: number;
}

// The highest reserved/system page ID. Pages with ID <= this are the fixed
// system pages: read-only and never stored in the table.
export const LAST_SYSTEM_PAGE_ID = 14;

// The always-present system pages. They are NOT persisted — the service injects
// them at the top of every list so the store always has its fixed routes. Their
// IDs are reused as the builder PageID. Keep in sync with the Website menu and the
// backend systemPages() route reservation.
export const SYSTEM_PAGES: IWebpage[] = [
  { ID: 10, Name: 'Home|Inicio', Route: '/', ss: 1, upd: 0 },
  { ID: 11, Name: 'About Us|Nosotros', Route: '/about', ss: 1, upd: 0 },
  { ID: 12, Name: 'Store|Tienda', Route: '/store', ss: 1, upd: 0 },
];

export class WebpagesService extends GetHandler<IWebpage> {
  route = 'webpage-pages';
  routePost = 'webpage-page';
  useCache = { min: 5, ver: 3 };
  inferRemoveFromStatus = true;
  prependOnSave = true;

  // The cache returns the full set of stored (user) pages on every call; we reset
  // and re-inject the fixed system pages at the top so they are always present and
  // never depend on a server round-trip. Any cached row in the system ID range
  // (<= 14) is stale (user pages are always >= 15) and is dropped to avoid
  // duplicate keys with SYSTEM_PAGES.
  handler(result: IWebpage[]): void {
    this.records = [];
    this.recordsMap = new Map();
    const rows = result || [];
    // Stored rows in the system range (<= 14) are not user pages — they exist only
    // to hold a system page's showcase thumbnail (Image). Merge their Image/upd into
    // the injected SYSTEM_PAGES instead of showing them as separate (duplicate) rows.
    const systemRows = new Map(rows.filter((p) => p.ID <= LAST_SYSTEM_PAGE_ID).map((p) => [p.ID, p]));
    const systemPages = SYSTEM_PAGES.map((page) => {
      const stored = systemRows.get(page.ID);
      return stored ? { ...page, Image: stored.Image, upd: stored.upd, UpdatedBy: stored.UpdatedBy } : page;
    });
    const userPages = rows.filter((page) => page.ID > LAST_SYSTEM_PAGE_ID);
    this.addSavedRecords(...systemPages, ...userPages);
  }

  constructor(init: boolean = false) {
    super();
    // Seed the system pages up-front so they show immediately, even before (or
    // without) a successful fetch. fetch() re-runs handler() to merge user pages.
    this.handler([]);
    if (init) this.fetch();
  }
}

// Known SEO metatag keys, mirrored from the backend seoMetatagKeys.
export const SEO_METATAG_KEYS = [
  'title', 'description', 'keywords', 'ogTitle', 'ogDescription', 'ogImage', 'favicon',
] as const;
export type SeoMetatagKey = (typeof SEO_METATAG_KEYS)[number];

// Storefront config read from parameters group 10: the domain plus every SEO key.
export type IWebsiteConfig = Partial<Record<SeoMetatagKey, string>> & { domain?: string };

export const getWebsiteConfig = (): Promise<IWebsiteConfig> =>
  GET({ route: 'website-config' }) as Promise<IWebsiteConfig>;

export const postWebsiteSeo = (metatags: Partial<Record<SeoMetatagKey, string>>) =>
  POST({ data: metatags, route: 'website-seo', successMessage: 'SEO guardado' });

export const postWebsiteDomain = (Domain: string) =>
  POST({ data: { Domain }, route: 'website-domain', successMessage: 'Dominio guardado' });

// CDN folder for page showcase thumbnails (mirrors the backend showcaseImageFolder).
const SHOWCASE_FOLDER = 'img-webpage';

// CDN URL of a page's showcase thumbnail, or '' if it has none. The imageID changes
// on every save, so the URL is self-busting (no extra cache-buster query needed).
export const showcaseImageSrc = (page: IWebpage): string =>
  page.Image && page.Image > 0
    ? Env.makeCDNRoute(SHOWCASE_FOLDER, `${Env.getCompanyID()}_${page.Image}`) + '.avif'
    : '';

// Background pipeline: convert the captured screenshot blob to a single ~0.4 Mpx AVIF
// (632² ≈ 400 000 px) and upload it to POST.webpage-showcase-image. Shown as a header
// process so the user isn't blocked. Errors are surfaced via the process, not thrown.
export const uploadShowcaseImage = async (pageID: number, blob: Blob): Promise<void> => {
  const processID = addProcess(tr('Page preview|Vista previa'), tr('Converting image|Convirtiendo imagen') + '...', 1);
  try {
    const avif = await fileToImage(blob, 632, 'avif');
    updateProcess(processID, '', tr('Saving preview...|Guardando vista previa...'), 1);
		await POST({
			data: { Content: avif },
			route: `webpage-showcase-image?page-id=${pageID}`,
			refreshRoutes: ["webpage-page"]
		});
    updateProcess(processID, '', tr('Preview saved|Vista previa guardada'), 2);
  } catch (error) {
    updateProcess(processID, '', tr('Preview failed|Error en vista previa') + `: ${String(error)}`, 0);
    console.error('showcase image upload failed::', pageID, error);
  }
};
