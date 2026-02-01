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

  let hoveredSectionIdx = $state<number | null>(null);
  let selectedSection = $state<SelectedSection | null>(null);
  let editorOpen = $state(false);
  let renderKey = $state(0);

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
    {#if element.productos}
      {#each element.productos as producto}
        <ProductCard productoID={producto.ID} css={getResolvedCss(element)}/>
      {/each}
    {/if}
  {:else if element.tagName === 'ProductCardHorizonal'}
    {#if element.productos}
      {#each element.productos as producto}
        <ProductCardHorizonal producto={producto} css={getResolvedCss(element)}/>
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
    {@const isHovered = hoveredSectionIdx === sectionIdx}
    {@const isSelected = selectedSection?.path[0] === sectionIdx.toString()}
    {#key renderKey}
    <div
      class="section-wrapper"
      class:section-hovered={isHovered}
      class:section-selected={isSelected}
      onmouseenter={() => hoveredSectionIdx = sectionIdx}
      onmouseleave={() => hoveredSectionIdx = null}
      onclick={() => handleSectionClick(element, sectionIdx)}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && handleSectionClick(element, sectionIdx)}
    >
      <div class="section-label">
        <span class="section-label-icon">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <path d="M3 9h18M9 21V9"/>
          </svg>
        </span>
        <span>{element.tagName || 'section'}</span>
        <span class="section-label-hint">Click to edit</span>
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
    {/key}
  {/if}
{/snippet}

<div class="ecommerce-builder">
  <div class="builder-canvas" class:editor-open={editorOpen}>
    {#if Array.isArray(elements)}
      {#each elements as element, idx}
        {@render renderSection(element, idx)}
      {/each}
    {:else}
      {@render renderSection(elements, 0)}
    {/if}
  </div>

  <SectionEditorLayer
    bind:open={editorOpen}
    section={selectedSection}
    {palette}
    onFieldUpdate={handleFieldUpdate}
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
    transition: margin-right 0.3s ease;
  }

  .builder-canvas.editor-open {
    margin-right: 400px;
  }

  .section-wrapper {
    position: relative;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .section-wrapper.section-hovered {
    outline: 2px dashed #3b82f6;
    outline-offset: 4px;
  }

  .section-wrapper.section-selected {
    outline: 3px solid #2563eb;
    outline-offset: 4px;
    box-shadow: 0 0 0 8px rgba(37, 99, 235, 0.1);
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

  .section-wrapper.section-hovered .section-label,
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
