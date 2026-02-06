import { SectionRegistry } from '../templates/registry'; // Note: This will be generated in Step 3/4
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

    // Generate dummy content based on schema
    const dummyContent: any = {};
    if (schema.content) {
      schema.content.forEach(key => {
        if (key === 'title') dummyContent[key] = 'Example Title';
        else if (key === 'subTitle') dummyContent[key] = 'Example Subtitle';
        else if (key === 'description') dummyContent[key] = 'This is a description for your new section. You can edit this text in the editor tab.';
        else if (key === 'primaryActionLabel') dummyContent[key] = 'Get Started';
        else if (key === 'primaryActionHref') dummyContent[key] = '#';
        else if (key === 'textLeft' || key === 'textCenter' || key === 'textRight') dummyContent[key] = 'Sample text for ' + key;
        else if (key === 'image' || key === 'bgImage') dummyContent[key] = 'https://images.unsplash.com/photo-1441986300917-64674bd600d8?q=80&w=2070&auto=format&fit=crop';
        else if (key === 'textLines') dummyContent[key] = [
          { text: 'Feature One: High performance and reliability', css: '' },
          { text: 'Feature Two: User-friendly interface and experience', css: '' },
          { text: 'Feature Three: 24/7 dedicated support team', css: '' }
        ];
        else if (key === 'productosIDs') dummyContent[key] = [1, 2, 3, 4];
      });
    }

    const newSection: SectionData = {
      id: crypto.randomUUID(),
      type: type,
      category: schema.category,
      content: dummyContent,
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
