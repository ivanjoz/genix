<script lang="ts">
  import type { SectionData } from '../../pkg-store/renderer/section-types';
  import { SectionRegistry } from '../../pkg-store/templates/registry';
  import { editorStore } from '../../pkg-store/stores/editor.svelte';

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
  const Config = $derived(SectionRegistry[section.type]);

  function handleSelect() {
    editorStore.select(section.id);
  }
</script>

<div
  class="section-wrapper"
  class:section-selected={isSelected}
  
  draggable="true"
  ondragstart={(e) => onDragStart(e, index)}
  ondragend={onDragEnd}
  ondragover={(e) => onDragOver(e, index)}
  
  onclick={handleSelect}
  role="button"
  tabindex="0"
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
