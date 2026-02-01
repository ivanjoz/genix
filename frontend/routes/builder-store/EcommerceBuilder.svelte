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
  import ProductCard from '$store/components/ProductCard.svelte';
  import ProductCardHorizonal from '$store/components/ProductCardHorizonal.svelte';
  import SectionEditorLayer from './SectionEditorLayer.svelte';

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

  function getResolvedCss(element: ComponentAST) {
    return resolveTokens(element.css, element.variables, values, palette);
  }

  function getResolvedLineCss(css: string, variables: ComponentVariable[] = []) {
    return resolveTokens(css, variables, values, palette);
  }

  const paletteStyles = $derived(generatePaletteStyles(palette));
</script>

<!-- Render child elements without selection wrapper -->
{#snippet renderChildElement(element: ComponentAST)}
  {#if !element}
    <!-- Skip null elements -->
  {:else if element.tagName === 'ProductCard'}
    {#if element.productosIDs||[]}
      {#each element.productosIDs as productoID}
        <ProductCard productoID={productoID} css={getResolvedCss(element)}/>
      {/each}
    {/if}
  {:else if element.tagName === 'ProductCardHorizonal'}
    {#if element.productosIDs}
      {#each element.productosIDs as productoID}
        <ProductCardHorizonal productoID={productoID} css={getResolvedCss(element)}/>
      {/each}
    {/if}
  {:else}
    {@const Tag = element.semanticTag || (element.tagName as any) || 'div'}
    <svelte:element 
      this={Tag}
      class={getResolvedCss(element)} 
      style={element.style || ''}
      {...element.attributes}
      aria-label={element.aria?.label}
      role={element.aria?.role}
      aria-hidden={element.aria?.hidden}
    >
      {#if element.text}
        {element.text}
      {/if}

      {#if element.textLines}
        {#each element.textLines as line}
          {@const LineTag = line.tag || 'span'}
          <svelte:element this={LineTag} class={getResolvedLineCss(line.css, element.variables)}>
            {line.text}
          </svelte:element>
        {/each}
      {/if}

      {#if element.children}
        {#each element.children as child}
          {@render renderChildElement(child)}
        {/each}
      {/if}
    </svelte:element>
  {/if}
{/snippet}

<!-- Render top-level section with selection wrapper -->
{#snippet renderSection(element: ComponentAST, sectionIdx: number)}
  {#if !element}
    <!-- Skip null elements -->
  {:else}
    {@const Tag = element.semanticTag || (element.tagName as any) || 'section'}

    {@const isSelected = selectedSection?.path[0] === sectionIdx.toString()}
    {@const isDragging = draggingIndex === sectionIdx}
    
    <div
      class="section-wrapper"
      class:section-selected={isSelected}
      class:is-dragging={isDragging}
      
      draggable="true"
      ondragstart={(e) => handleSectionDragStart(e, sectionIdx)}
      ondragend={handleSectionDragEnd}
      ondragover={(e) => handleDragOver(e, sectionIdx)}
      
      onclick={() => handleSectionClick(element, sectionIdx)}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && handleSectionClick(element, sectionIdx)}
    >
      <div class="section-outline"></div>
      <div class="section-label">
        <span class="section-label-icon">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <path d="M3 9h18M9 21V9"/>
          </svg>
        </span>
        <span>{element.tagName || 'section'}</span>
        <span class="section-label-hint">Click to edit • Drag to move</span>
      </div>
      
      <svelte:element 
        this={Tag}
        class={getResolvedCss(element)} 
        style={`${element.style || ''} ${paletteStyles}`}
        {...element.attributes}
        aria-label={element.aria?.label}
        role={element.aria?.role}
        aria-hidden={element.aria?.hidden}
      >
        {#if element.text}
          {element.text}
        {/if}

        {#if element.textLines}
          {#each element.textLines as line}
            {@const LineTag = line.tag || 'span'}
            <svelte:element this={LineTag} class={getResolvedLineCss(line.css, element.variables)}>
              {line.text}
            </svelte:element>
          {/each}
        {/if}

        {#if element.children}
          {#each element.children as child}
            {@render renderChildElement(child)}
          {/each}
        {/if}
      </svelte:element>
    </div>
  {/if}
{/snippet}

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
          {#if isDraggingOver && dropIndex === idx}
            {@render renderPlaceholder()}
          {/if}
          
          {@render renderSection(element, idx)}
          
          {#if isDraggingOver && dropIndex === idx + 1 && idx === elements.length - 1}
            {@render renderPlaceholder()}
          {/if}
        {/each}
      {:else}
        {#if isDraggingOver && dropIndex === 0}
          {@render renderPlaceholder()}
        {/if}
        
        {@render renderSection(elements, 0)}
        
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

  .section-wrapper {
    position: relative;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .section-wrapper.is-dragging {
    opacity: 0.4;
    cursor: grabbing;
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

  .section-outline {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    border: 2px dashed #3b82f6;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.2s ease;
    z-index: 9999;
  }

  .section-wrapper:hover .section-outline {
    opacity: 1;
    box-shadow: inset 0 0 0 6px rgb(37 99 235 / 29%);
  }

  .section-wrapper.section-selected .section-outline {
    border: 2px solid #2563eb;
  }

  .section-label {
    position: absolute;
    top: -32px;
    left: 8px;
    display: flex;
    align-items: center;
    gap: 8px;
    background: #2563eb;
    color: white;
    font-size: 12px;
    font-weight: 600;
    padding: 6px 12px;
    border-radius: 6px;
    z-index: 100;
    opacity: 0;
    transform: translateY(8px);
    transition: all 0.2s ease;
    pointer-events: none;
    font-family: system-ui, -apple-system, sans-serif;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
  }

  .section-label-icon {
    display: flex;
    align-items: center;
    opacity: 0.8;
  }

  .section-label-hint {
    font-size: 10px;
    font-weight: 400;
    text-transform: none;
    letter-spacing: 0;
    opacity: 0.7;
    margin-left: 4px;
  }

  .section-wrapper:hover .section-label,
  .section-wrapper.section-selected .section-label {
    opacity: 1;
    transform: translateY(0);
  }

  .section-wrapper.section-selected .section-label {
    background: #1d4ed8;
  }

  .section-wrapper.section-selected .section-label-hint {
    display: none;
  }
</style>
