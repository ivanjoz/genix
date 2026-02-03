import { SectionRegistry } from '../sections/registry'; // Note: This will be generated in Step 3/4
import type { SectionData, SectionSchema } from '../renderer/section-types';

class EditorStore {
  // The current list of sections on the page
  sections = $state<SectionData[]>([]);
  
  // The ID of the section currently being edited
  selectedId = $state<string | null>(null);
  
  // The UI-friendly schema for the currently selected section
  activeSchema = $derived.by(() => {
    if (!this.selectedId) return null;
    const section = this.sections.find(s => s.id === this.selectedId);
    if (!section) return null;
    
    // We get the schema from the registry based on the section type
    // If the registry isn't generated yet or doesn't have it, we return a fallback
    return SectionRegistry?.[section.type]?.schema || null;
  });

  // Helper to get the actual data of the selected section
  selectedSection = $derived(
    this.sections.find(s => s.id === this.selectedId) || null
  );

  select(id: string | null) {
    this.selectedId = id;
  }

  updateContent(id: string, key: string, value: any) {
    const section = this.sections.find(s => s.id === id);
    if (section) {
      section.content[key] = value;
    }
  }

  updateCss(id: string, slot: string, classes: string) {
    const section = this.sections.find(s => s.id === id);
    if (section) {
      section.css[slot] = classes;
    }
  }

  addSection(type: string, index?: number) {
    const schema = SectionRegistry?.[type]?.schema;
    if (!schema) return;

    const newSection: SectionData = {
      id: crypto.randomUUID(),
      type: type,
      category: schema.category,
      content: {},
      css: {}
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
