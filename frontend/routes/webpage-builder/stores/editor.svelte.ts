import type { SectionData } from '$ecommerce/renderer/section-types';
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
import { schema as htmlSectionSchema } from '$ecommerce/renderer/HtmlSection.svelte';
import { sectionTemplates } from '../templates';
import { parseHTML } from '$ecommerce/html-ast/parse-html';

// The builder's neutral starting palette (a slate ramp). A page with no saved
// palette starts from this; the agent grows it as it introduces new colors. Returns
// a fresh object/array each call so the reactive store never shares the const.
export const makeDefaultPalette = (): ColorPalette => ({
  id: 'default',
  name: 'Default Palette',
  colors: [
    '#f8fafc', '#f1f5f9', '#e2e8f0', '#cbd5e1', '#94a3b8',
    '#64748b', '#475569', '#334155', '#1e293b', '#0f172a',
  ],
});

// id -> HTML plantilla lookup. HTML templates are an authoring source: their HTML is
// parsed to an AST once at add time and stored as `Ast` (the canonical model).
const htmlTemplatesById = new Map(sectionTemplates.map(t => [t.id, t]));

// FNV-1a: a tiny, fast, synchronous string hash. Used to fingerprint a section's
// persisted content so we can tell whether the page has unsaved changes.
function fnv1a(input: string): string {
  let hash = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0).toString(16);
}

class EditorStore {
  // The current list of sections on the page
  sections = $state<SectionData[]>([]);

  // The page's color palette: a growable list of hex colors referenced 1-based as
  // var(--color-N). Page-global (drives the CSS vars on every section root). Loaded
  // from / saved to section 1's content; grown by the agent (see absorbColors).
  palette = $state<ColorPalette>(makeDefaultPalette());

  // The palette's persisted fingerprint, captured alongside the section baseline.
  // Compared in hasUnsavedChanges so growing the palette counts as an unsaved change.
  private paletteBaseline = '';

  // Per-section content fingerprints captured at load and after each successful save,
  // keyed by the runtime `id`. Compared against the live sections to decide whether
  // there is anything to persist (see hasUnsavedChanges).
  baselineHashes = new Map<string, string>();

  // The ID of the section currently being edited
  selectedId = $state<string | null>(null);

  // Canvas preview mode. 'mobile' renders the sections inside a ~390px iframe so real
  // Tailwind breakpoints (md/lg are min-width media queries) resolve against that
  // narrow viewport. Mobile is a limited view: drag-reorder is off, click-select only.
  viewMode = $state<'desktop' | 'mobile'>('desktop');
  
  // The UI-friendly schema for the currently selected section
  activeSchema = $derived.by(() => {
    if (!this.selectedId) return null;
    const section = this.sections.find(s => s.id === this.selectedId);
    return section?.Type === 'HtmlSection' ? htmlSectionSchema : null;
  });

  // Helper to get the actual data of the selected section
  selectedSection = $derived(
    this.sections.find(s => s.id === this.selectedId) || null
  );

  select(id: string | null) {
    this.selectedId = id;
  }

  updateCss(id: string, slot: string, classes: string) {
    const section = this.sections.find(s => s.id === id);
    if (section) {
      (section.Css ??= {})[slot] = classes;
    }
  }

  addSection(templateId: string, index?: number) {
    // HTML plantilla: parse its HTML to an AST once and store it as the editable model.
    const htmlTemplate = htmlTemplatesById.get(templateId);
    if (!htmlTemplate) return;

    const newSection: SectionData = {
      id: crypto.randomUUID(),
      Type: 'HtmlSection',
      category: htmlTemplate.category,
      Ast: parseHTML(htmlTemplate.html ?? ''),
      Css: {}
    };

    if (typeof index === 'number') {
      this.sections.splice(index, 0, newSection);
    } else {
      this.sections.push(newSection);
    }
  }

  removeSection(id: string) {
    this.sections = this.sections.filter(s => s.id !== id);
    if (this.selectedId === id) this.selectedId = null;
  }

  reorder(from: number, to: number) {
    const item = this.sections.splice(from, 1)[0];
    this.sections.splice(to, 0, item);
  }

  // Hash of a section's PERSISTED content only. Runtime-only / builder-only fields
  // (the uuid `id` and the authoring metadata kept lowercase) are dropped so they
  // never count as a change; what remains mirrors what savePageContent sends.
  private sectionHash(section: SectionData): string {
    const { id, category, name, description, thumbnail, presets, html, ...persisted } = section;
    return fnv1a(JSON.stringify(persisted));
  }

  // Snapshot the current sections as the saved baseline. Called after load and after
  // a successful save so later edits are measured against the last persisted state.
  captureBaseline() {
    this.baselineHashes = new Map(this.sections.map(s => [s.id, this.sectionHash(s)]));
    this.paletteBaseline = JSON.stringify(this.palette.colors);
  }

  // Replace the active palette (e.g. with a page's saved palette on load). Falls
  // back to the default when there are no stored colors.
  setPalette(colors: string[] | undefined) {
    this.palette = colors?.length
      ? { ...makeDefaultPalette(), colors: [...colors] }
      : makeDefaultPalette();
  }

  // True when any section was created, deleted, reordered, or had its content edited
  // since the last baseline. Because the backend SectionID is position-based, order is
  // significant: we compare the sequences position-by-position (the Map preserves the
  // baseline insertion order), so a pure reorder is detected even though no hash changed.
  get hasUnsavedChanges(): boolean {
    if (JSON.stringify(this.palette.colors) !== this.paletteBaseline) return true;
    if (this.sections.length !== this.baselineHashes.size) return true;
    const baseline = [...this.baselineHashes];
    return this.sections.some((s, i) =>
      s.id !== baseline[i][0] || this.sectionHash(s) !== baseline[i][1]
    );
  }
}

export const editorStore = new EditorStore();
