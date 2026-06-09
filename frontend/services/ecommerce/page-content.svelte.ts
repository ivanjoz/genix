import { GET, POST } from '$libs/http.svelte';
import { collectTokens, generateCss } from '$ecommerce/stores/uno-generator';
import type { SectionData } from '$ecommerce/renderer/section-types';

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
export const savePageContent = async (sections: SectionData[]) => {
  const pageCss = await generateCss(collectTokens(sections));
  const payload = sections.map((section, index) => ({
    ...section,
    PageCss: index === 0 ? pageCss : '',
  }));

  return POST({
    data: payload,
    route: 'ecommerce-page-content',
    successMessage: 'Cambios guardados',
  });
};

// Load the stored sections of the Inicio page plus the whole-page runtime CSS.
// Sections are rebuilt into editable SectionData (a fresh runtime `id` is assigned
// since it is not persisted). `css` is the single pre-generated stylesheet (stored
// in section 1's Css column) the storefront injects directly. Returns empty when
// the page has no stored content yet.
export const getPageContent = async (): Promise<{ sections: SectionData[]; css: string }> => {
  const rows: IPageContentRow[] = await GET({ route: 'ecommerce-page-content' });
  const sections = (rows || []).map((row) => ({ id: crypto.randomUUID(), ...row.Content }));
  // Only section 1 carries the Css column (the whole-page stylesheet); the rest are
  // empty, so this resolves to that single stylesheet.
  const css = (rows || []).map((row) => row.Css || '').filter(Boolean).join('\n');
  return { sections, css };
};
