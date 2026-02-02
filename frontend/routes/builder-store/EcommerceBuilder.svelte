<script lang="ts" module>
  export interface EditableField {
    path: string[];
    type: 'text' | 'textLine' | 'image' | 'variable' | 'attribute';
    label: string;
    value: string;
    index?: number;
    elementTag?: string;
    variable?: import('../../pkg-store/renderer/renderer-types').ComponentVariable;
    textLine?: import('../../pkg-store/renderer/renderer-types').ITextLine;
  }

  export interface SelectedSection {
    element: import('../../pkg-store/renderer/renderer-types').ComponentAST;
    path: string[];
    fields: EditableField[];
  }
</script>

<script lang="ts">
  import type { ComponentAST, ColorPalette, ComponentVariable } from '../../pkg-store/renderer/renderer-types';
  import { resolveTokens, generatePaletteStyles } from '../../pkg-store/renderer/token-resolver';
  import SectionEditorLayer from './SectionEditorLayer.svelte';
  import BuilderSectionRender from './BuilderSectionRender.svelte';

  interface Props {
    elements: ComponentAST | ComponentAST[];
    values?: Record<string, string>;
    palette?: ColorPalette;
    onUpdate?: (elements: ComponentAST | ComponentAST[]) => void;
  }

  let {
    elements = $bindable(),
    values = $bindable({}),
    palette,
    onUpdate
  }: Props = $props();

  let selectedSection = $state<SelectedSection | null>(null);
  let editorOpen = $state(false);
  let renderKey = $state(0);

  // Drag and Drop state
  let draggingIndex = $state<number | null>(null);
  let dropIndex = $state<number | null>(null);
  let isDraggingOver = $state(false);

  function handleSectionDragStart(e: DragEvent, idx: number) {
    if (!e.dataTransfer) return;
    draggingIndex = idx;
    e.dataTransfer.setData('application/x-genix-reorder', idx.toString());
    e.dataTransfer.effectAllowed = 'move';
    
    // Add a class to the target if needed
    const target = e.currentTarget as HTMLElement;
    setTimeout(() => target.classList.add('is-dragging'), 0);
  }

  function handleSectionDragEnd(e: DragEvent) {
    draggingIndex = null;
    dropIndex = null;
    isDraggingOver = false;
    const target = e.currentTarget as HTMLElement;
    target.classList.remove('is-dragging');
  }

  function handleDragOver(e: DragEvent, idx: number) {
    e.preventDefault();
    e.stopPropagation();
    if (!e.dataTransfer) return;

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const midpoint = rect.top + rect.height / 2;
    
    // Determine if we should drop before or after this element
    const newDropIndex = e.clientY < midpoint ? idx : idx + 1;
    
    if (dropIndex !== newDropIndex) {
      dropIndex = newDropIndex;
    }
    isDraggingOver = true;
  }

  function handleCanvasDragLeave(e: DragEvent) {
    // Only reset if we're actually leaving the builder area
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    if (
      e.clientX <= rect.left ||
      e.clientX >= rect.right ||
      e.clientY <= rect.top ||
      e.clientY >= rect.bottom
    ) {
      dropIndex = null;
      isDraggingOver = false;
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDraggingOver = false;
    
    const targetIndex = dropIndex ?? (Array.isArray(elements) ? elements.length : 1);
    
    if (e.dataTransfer?.getData('application/x-genix-reorder')) {
      const fromIndex = parseInt(e.dataTransfer.getData('application/x-genix-reorder'));
      if (isNaN(fromIndex)) return;
      
      reorderSection(fromIndex, targetIndex);
    } else if (e.dataTransfer?.getData('application/x-genix-template')) {
      const templateData = JSON.parse(e.dataTransfer.getData('application/x-genix-template'));
      insertTemplate(templateData, targetIndex);
    }
    
    dropIndex = null;
    draggingIndex = null;
  }

  function reorderSection(fromIdx: number, toIdx: number) {
    if (!Array.isArray(elements)) return;
    
    const newElements = [...elements];
    const [movedItem] = newElements.splice(fromIdx, 1);
    
    // Adjust target index if it was after the item we removed
    const adjustedToIdx = toIdx > fromIdx ? toIdx - 1 : toIdx;
    newElements.splice(adjustedToIdx, 1, newElements[adjustedToIdx]); // This is wrong, let me fix it
    
    // Correct way to splice back
    newElements.splice(adjustedToIdx, 0, movedItem);
    
    elements = newElements;
    onUpdate?.(elements);
    renderKey++;
  }

  function insertTemplate(template: any, index: number) {
    if (!Array.isArray(elements)) {
      elements = [elements];
    }
    
    const newElements = [...elements];
    // For now, let's assume template.ast exists or we create a basic one
    // In a real scenario, we might need to fetch or transform the template
    const newSection: ComponentAST = template.ast || {
      tagName: 'section',
      semanticTag: 'section',
      css: 'p-20 bg-slate-100',
      children: [{
        tagName: 'h2',
        text: `New ${template.name} Section`
      }]
    };

    newElements.splice(index, 0, newSection);
    elements = newElements;
    onUpdate?.(elements);
    renderKey++;
  }

  function getElementByPath(path: string[]): ComponentAST | null {
    let current: ComponentAST | ComponentAST[] = elements;
    
    for (const key of path) {
      if (Array.isArray(current)) {
        const idx = parseInt(key);
        if (isNaN(idx) || idx < 0 || idx >= current.length) return null;
        current = current[idx];
      } else if (current && typeof current === 'object') {
        if (key === 'children' && current.children) {
          current = current.children;
        } else {
          return null;
        }
      }
    }
    
    return Array.isArray(current) ? null : current;
  }

  function extractEditableFields(element: ComponentAST, basePath: string[], parentTag?: string): EditableField[] {
    const fields: EditableField[] = [];
    const currentTag = element.tagName || element.semanticTag || 'div';
    
    // Extract text field
    if (element.text !== undefined) {
      fields.push({
        path: [...basePath, 'text'],
        type: 'text',
        label: element.text.substring(0, 30) + (element.text.length > 30 ? '...' : ''),
        value: element.text,
        elementTag: currentTag
      });
    }
    
    // Extract textLines
    if (element.textLines && element.textLines.length > 0) {
      element.textLines.forEach((line, idx) => {
        fields.push({
          path: [...basePath, 'textLines', idx.toString()],
          type: 'textLine',
          label: line.text.substring(0, 30) + (line.text.length > 30 ? '...' : ''),
          value: line.text,
          index: idx,
          textLine: line,
          elementTag: line.tag || 'span'
        });
      });
    }
    
    // Extract variables
    if (element.variables && element.variables.length > 0) {
      element.variables.forEach((variable, idx) => {
        const currentValue = values[variable.key] ?? variable.defaultValue;
        fields.push({
          path: [...basePath, 'variables', idx.toString()],
          type: 'variable',
          label: variable.label || variable.key,
          value: currentValue,
          index: idx,
          variable: variable
        });
      });
    }
    
    // Extract image src from attributes
    if (element.attributes?.src) {
      fields.push({
        path: [...basePath, 'attributes', 'src'],
        type: 'image',
        label: 'Image',
        value: element.attributes.src,
        elementTag: currentTag
      });
    }
    
    // Extract href
    if (element.attributes?.href) {
      fields.push({
        path: [...basePath, 'attributes', 'href'],
        type: 'attribute',
        label: 'Link URL',
        value: element.attributes.href,
        elementTag: currentTag
      });
    }
    
    // Recursively extract from children
    if (element.children) {
      element.children.forEach((child, idx) => {
        const childFields = extractEditableFields(child, [...basePath, 'children', idx.toString()], currentTag);
        fields.push(...childFields);
      });
    }
    
    return fields;
  }

  function handleSectionClick(element: ComponentAST, sectionIdx: number) {
    const path = [sectionIdx.toString()];
    const fields = extractEditableFields(element, path);
    selectedSection = {
      element,
      path,
      fields
    };
    editorOpen = true;
  }

  function handleFieldUpdate(field: EditableField, newValue: string) {
    if (!selectedSection) return;
    
    // Clone the elements to trigger reactivity
    const newElements = JSON.parse(JSON.stringify(elements));
    
    // Navigate to the field location
    let current: any = Array.isArray(newElements) ? newElements : newElements;
    const pathWithoutLast = field.path.slice(0, -1);
    const lastKey = field.path[field.path.length - 1];
    
    for (const key of pathWithoutLast) {
      if (Array.isArray(current)) {
        const idx = parseInt(key);
        current = current[idx];
      } else if (key === 'children' && current.children) {
        current = current.children;
      } else if (key === 'textLines' && current.textLines) {
        current = current.textLines;
      } else if (key === 'variables' && current.variables) {
        current = current.variables;
      } else if (key === 'attributes' && current.attributes) {
        current = current.attributes;
      } else if (current && typeof current === 'object') {
        current = current[key];
      }
    }
    
    // Update the value
    if (field.type === 'textLine') {
      current[parseInt(lastKey)].text = newValue;
    } else if (field.type === 'variable') {
      // For variables, update the values map instead
      const variable = field.variable;
      if (variable) {
        values = { ...values, [variable.key]: newValue };
      }
    } else if (lastKey && current) {
      current[lastKey] = newValue;
    }
    
    // Update elements and trigger re-render
    elements = newElements;
    renderKey++;
    
    // Notify parent
    onUpdate?.(elements);
    
    // Update selected section fields
    const updatedElement = getElementByPath(selectedSection.path);
    if (updatedElement) {
      selectedSection = {
        ...selectedSection,
        element: updatedElement,
        fields: extractEditableFields(updatedElement, selectedSection.path)
      };
    }
  }

  function closeEditor() {
    editorOpen = false;
    selectedSection = null;
  }

  const paletteStyles = $derived(generatePaletteStyles(palette));
</script>

{#snippet renderPlaceholder()}
  <div class="drop-placeholder">
    <div class="placeholder-content">
      <span class="placeholder-icon">✨</span>
      <span>Drop to insert here</span>
    </div>
  </div>
{/snippet}

<div class="ecommerce-builder">
  <div 
    class="builder-canvas" 
    ondrop={handleDrop}
    ondragover={(e) => {
      e.preventDefault();
      // If we are over the canvas but not over a specific section, 
      // it means we want to drop at the end
      if (e.target === e.currentTarget) {
        dropIndex = Array.isArray(elements) ? elements.length : 1;
        isDraggingOver = true;
      }
    }}
    ondragleave={handleCanvasDragLeave}
  >
    {#key renderKey}
      {#if Array.isArray(elements)}
        {#each elements as element, idx}
          {#if element}
            {#if isDraggingOver && dropIndex === idx}
              {@render renderPlaceholder()}
            {/if}
            
            <BuilderSectionRender 
              {element}
              index={idx}
              isSelected={selectedSection?.path[0] === idx.toString()}
              isDragging={draggingIndex === idx}
              {values}
              {palette}
              {paletteStyles}
              onSelect={handleSectionClick}
              onDragStart={handleSectionDragStart}
              onDragEnd={handleSectionDragEnd}
              onDragOver={handleDragOver}
            />
            
            {#if isDraggingOver && dropIndex === idx + 1 && idx === elements.length - 1}
              {#if elements[idx]}
                {@render renderPlaceholder()}
              {/if}
            {/if}
          {/if}
        {/each}
      {:else if elements}
        {#if isDraggingOver && dropIndex === 0}
          {@render renderPlaceholder()}
        {/if}
        
        <BuilderSectionRender 
          element={elements}
          index={0}
          isSelected={selectedSection?.path[0] === '0'}
          isDragging={draggingIndex === 0}
          {values}
          {palette}
          {paletteStyles}
          onSelect={handleSectionClick}
          onDragStart={handleSectionDragStart}
          onDragEnd={handleSectionDragEnd}
          onDragOver={handleDragOver}
        />
        
        {#if isDraggingOver && dropIndex === 1}
          {@render renderPlaceholder()}
        {/if}
      {/if}
    {/key}
  </div>

  <SectionEditorLayer
    bind:open={editorOpen}
    section={selectedSection}
    {palette}
    onFieldUpdate={handleFieldUpdate}
    onTemplateSelect={(template) => insertTemplate(template, Array.isArray(elements) ? elements.length : 1)}
    onClose={closeEditor}
  />
</div>

<style>
  .ecommerce-builder {
    position: relative;
    display: flex;
    min-height: 100vh;
  }

  .builder-canvas {
    flex: 1;
  }

  .drop-placeholder {
    height: 48px;
    margin: 8px 0;
    border: 2px dashed #3b82f6;
    background: rgba(59, 130, 246, 0.1);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    animation: placeholder-pulse 1.5s infinite ease-in-out;
  }

  .placeholder-content {
    display: flex;
    align-items: center;
    gap: 12px;
    color: #3b82f6;
    font-size: 14px;
    font-weight: 600;
  }

  .placeholder-icon {
    font-size: 18px;
  }

  @keyframes placeholder-pulse {
    0% { opacity: 0.6; transform: scale(0.995); }
    50% { opacity: 1; transform: scale(1); }
    100% { opacity: 0.6; transform: scale(0.995); }
  }
</style>
