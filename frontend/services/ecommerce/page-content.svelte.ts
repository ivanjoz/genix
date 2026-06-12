import { GET, POST } from '$libs/http.svelte';
import { collectTokens, generateCss } from '$ecommerce/stores/uno-generator';
import type { SectionData } from '$ecommerce/renderer/section-types';

// The PageID of the page currently open in the builder. The route sets it on
// entry; getPageContent/savePageContent default to it so deeply-nested callers
// (e.g. the editor's Save button) don't need to thread the id down. 0 = the
// default Inicio page (the bare /webpage-builder route), resolved server-side.
let currentPageID = 0;
export const setCurrentPageID = (pageID: number) => {
  currentPageID = pageID > 0 ? pageID : 0;
};

// Build the route, appending ?page-id only for an explicit (non-default) page.
const pageContentRoute = (pageID: number) =>
  pageID > 0 ? `ecommerce-page-content?page-id=${pageID}` : 'ecommerce-page-content';

// A stored section row as returned by GET.ecommerce-page-content. `Content` holds
// the persisted SectionData fields (Type/Ast/Content/Css/Attributes). `Css` is the
// top-level column carrying the whole-page stylesheet (set on section 1 only). The
// runtime `id` is not persisted.
interface IPageContentRow {
  SectionID: number;
  Content: Omit<SectionData, 'id'>;
  Css?: string;
}

// Persist every current section of the Inicio page (page 11, hardcoded server-side).
// The whole page's pre-generated runtime CSS (the UnoCSS output for every section's
// tokens) is stored once in the FIRST section's `PageCss`, so the storefront injects
// a single stylesheet and never runs the UnoCSS engine at view time. The backend
// assigns the 1-based SectionID, hashes each section, and writes only what changed
// (soft-deleting removed positions) — and since the CSS lives in section 1, any
// CSS-affecting change anywhere on the page flips section 1's hash and rewrites it.
export const savePageContent = async (sections: SectionData[], pageID: number = currentPageID) => {
  const pageCss = await generateCss(collectTokens(sections));
  const payload = sections.map((section, index) => ({
    ...section,
    PageCss: index === 0 ? pageCss : '',
  }));

  return POST({
    data: payload,
    route: pageContentRoute(pageID),
    successMessage: 'Cambios guardados',
  });
};

// Load the stored sections of the Inicio page plus the whole-page runtime CSS.
// Sections are rebuilt into editable SectionData (a fresh runtime `id` is assigned
// since it is not persisted). `css` is the single pre-generated stylesheet (stored
// in section 1's Css column) the storefront injects directly. Returns empty when
// the page has no stored content yet.
export const getPageContent = async (pageID: number = currentPageID): Promise<{ sections: SectionData[]; css: string }> => {
  const rows: IPageContentRow[] = await GET({ route: pageContentRoute(pageID) });
  return parsePageContentRows(rows);
};

// Shared parsing for both the authed and public reads. A fresh runtime `id` is
// assigned (it is not persisted) and the single whole-page stylesheet (section 1's
// Css column) is extracted.
const parsePageContentRows = (rows: IPageContentRow[] | null): { sections: SectionData[]; css: string } => {
  const sections = (rows || []).map((row) => ({ id: crypto.randomUUID(), ...row.Content }));
  const css = (rows || []).map((row) => row.Css || '').filter(Boolean).join('\n');
  return { sections, css };
};

// The combined public payload from GET.p-webpage: a page's SEO metatags (Config)
// plus its content rows (Sections).
interface IWebpagePublicResult {
  Config: Record<string, string>;
  Sections: IPageContentRow[];
}

// Storefront loader: ONE public call (GET.p-webpage) returns a page's SEO config +
// content. No auth — makeRoute appends company-id from Env.getCompanyID()
// (VITE_COMPANY_ID at build, window/path at runtime). pageID defaults to the root
// page (resolved server-side). Used by the prerender build and live storefronts.
export const getStoreWebpage = async (
  pageID = 0,
): Promise<{ sections: SectionData[]; css: string; seo: Record<string, string> }> => {
  const route = pageID > 0 ? `p-webpage?id=${pageID}` : 'p-webpage';
	const result: IWebpagePublicResult = await GET({ route });

  const { sections, css } = parsePageContentRows(result?.Sections || []);
  return { sections, css, seo: result?.Config || {} };
};
