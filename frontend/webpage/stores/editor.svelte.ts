import type { SectionData } from '../renderer/section-types';
import { schema as htmlSectionSchema } from '../renderer/HtmlSection.svelte';
import { sectionTemplates } from '../templates';
import { parseHTML } from '../html-ast/parse-html';

// id -> HTML plantilla lookup. HTML templates are an authoring source: their HTML is
// parsed to an AST once at add time and stored as `Ast` (the canonical model).
const htmlTemplatesById = new Map(sectionTemplates.map(t => [t.id, t]));

class EditorStore {
  // The current list of sections on the page
  sections = $state<SectionData[]>([]);
  
  // The ID of the section currently being edited
  selectedId = $state<string | null>(null);
  
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
}

export const editorStore = new EditorStore();
