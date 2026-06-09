<script lang="ts">
import { setContext } from 'svelte';
import type { SectionData } from '$ecommerce/renderer/section-types';
import { SectionRegistry } from '$ecommerce/templates/registry';
import { editorStore } from '$ecommerce/stores/editor.svelte';
import { EC_BUILDER_MODE } from '$ecommerce/renderer/builder-context';

// Mark everything rendered below as builder-mode so AST components (e.g. EcommerceSlider)
// can sync with the editor and disable production-only behaviour like autoplay.
setContext(EC_BUILDER_MODE, true);

  interface Props {
    section: SectionData;
    index: number;
    paletteStyles: string;
    onDragStart: (e: DragEvent, index: number) => void;
    onDragEnd: (e: DragEvent) => void;
    onDragOver: (e: DragEvent, index: number) => void;
  }

  let {
    section,
    index,
    paletteStyles,
    onDragStart,
    onDragEnd,
    onDragOver
  }: Props = $props();

  const isSelected = $derived(editorStore.selectedId === section.id);
  const Config = $derived(section.type ? SectionRegistry[section.type] : undefined);

  function handleSelect() {
    editorStore.select(section.id);
  }

  // The browser's default drag image is a raster snapshot of the dragged element. When
  // a section is only partially in view (scrolled under the sticky store header), that
  // snapshot captures the full element plus whatever overlaps it on screen — so the
  // ghost shows content that isn't the section. Replace it with a small controlled chip.
  function setReorderDragImage(e: DragEvent, label: string) {
    if (!e.dataTransfer) return;
    const chip = document.createElement('div');
    chip.textContent = label;
    Object.assign(chip.style, {
      position: 'fixed',
      top: '-1000px',
      left: '-1000px',
      padding: '6px 12px',
      background: '#2563eb',
      color: '#fff',
      fontSize: '12px',
      fontWeight: '700',
      letterSpacing: '0.5px',
      textTransform: 'uppercase',
      borderRadius: '6px',
      fontFamily: 'sans-serif',
      boxShadow: '0 4px 12px rgba(37, 99, 235, 0.4)',
      whiteSpace: 'nowrap',
      pointerEvents: 'none',
      zIndex: '99999',
    } satisfies Partial<CSSStyleDeclaration>);
    document.body.appendChild(chip);
    e.dataTransfer.setDragImage(chip, 12, 12);
    // The browser snapshots the chip synchronously during dragstart; remove it after.
    setTimeout(() => chip.remove(), 0);
  }
</script>

<div
  class="section-wrapper"
  class:section-selected={isSelected}
  
  draggable="true"
  ondragstart={(e) => {
    setReorderDragImage(e, Config?.schema.name || section.type || 'Section');
    onDragStart(e, index);
  }}
  ondragend={onDragEnd}
  ondragover={(e) => onDragOver(e, index)}
  
  onclick={handleSelect}
  role="button"
  tabindex="0"
  aria-label={`Edit ${Config?.schema.name || section.type} section`}
  onkeydown={(e) => e.key === 'Enter' && handleSelect()}
>
  <div class="section-outline"></div>
  <div class="section-label">
    <span class="section-label-icon">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <path d="M3 9h18M9 21V9"/>
      </svg>
    </span>
    <span>{Config?.schema.name || section.type}</span>
    <span class="section-label-hint">Click to edit • Drag to move</span>
  </div>
  
  <div class="section-content" style={paletteStyles}>
    {#if Config}
      <Config.component
        content={section.content}
        ast={section.ast}
        css={section.css}
        {...section.attributes}
      />
    {:else}
      <div class="p-20 bg-slate-100 text-slate-400 text-center border-2 border-dashed border-slate-200">
        Component "{section.type}" not found in registry.
      </div>
    {/if}
  </div>
</div>

<style>
  .section-wrapper {
    position: relative;
    cursor: pointer;
    transition: all 0.2s ease;
    /* Bound the section to the container width and clip inner overflow. Without this,
       a section whose content is wider than the viewport (e.g. a Slider's 300%-wide
       track) makes the wrapper's painted box that wide, so the native drag snapshot
       captures the full inner content width instead of just the container. */
    width: 100%;
    max-width: 100%;
    overflow: hidden;
  }

  .section-content {
    overflow: hidden;
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
    z-index: 51;
  }

  .section-wrapper:hover .section-outline {
    opacity: 1;
    box-shadow: inset 0 0 0 6px rgb(37 99 235 / 15%);
  }

  .section-wrapper.section-selected .section-outline {
    border: 2px solid #2563eb;
    opacity: 1;
  }

  .section-label {
    position: absolute;
    top: 8px;
    left: 8px;
    display: flex;
    align-items: center;
    gap: 8px;
    background: #2563eb;
    color: white;
    font-size: 11px;
    font-weight: 700;
    padding: 4px 10px;
    border-radius: 4px;
    z-index: 10000;
    opacity: 0;
    transform: translateY(4px);
    transition: all 0.2s ease;
    pointer-events: none;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
  }

  .section-wrapper:hover .section-label,
  .section-wrapper.section-selected .section-label {
    opacity: 1;
    transform: translateY(0);
  }

  .section-wrapper.section-selected .section-label {
    background: #1d4ed8;
  }

  .section-label-hint {
    font-size: 9px;
    font-weight: 400;
    text-transform: none;
    opacity: 0.8;
  }

  .section-wrapper.section-selected .section-label-hint {
    display: none;
  }
</style>
