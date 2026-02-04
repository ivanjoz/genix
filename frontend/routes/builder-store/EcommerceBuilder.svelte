<script lang="ts">
  import type { ColorPalette } from '../../pkg-store/renderer/renderer-types';
  import { generatePaletteStyles } from '../../pkg-store/renderer/token-resolver';
  import { editorStore } from '../../pkg-store/stores/editor.svelte';
  import { liveCSS } from '../../pkg-store/stores/live-css.svelte';
  import SectionEditorLayer from './SectionEditorLayer.svelte';
  import BuilderSectionRender from './BuilderSectionRender.svelte';

  import type { SectionData } from '../../pkg-store/renderer/section-types';

  interface Props {
    elements?: SectionData[];
    values?: Record<string, string>;
    palette?: ColorPalette;
    onUpdate?: (sections: SectionData[]) => void;
  }

  let {
    elements = [],
    values = {},
    palette,
    onUpdate
  }: Props = $props();

  // Initialize editorStore with elements if provided
  $effect(() => {
    if (elements.length > 0 && editorStore.sections.length === 0) {
      editorStore.sections = [...elements];
    }
  });

  // Live Tailwind compilation
  $effect(() => {
    // Track changes to any CSS property in any section
    // We stringify the CSS objects to ensure the effect re-runs on any deep change
    const cssData = JSON.stringify(editorStore.sections.map(s => s.css));
    liveCSS.update();
  });

  let isDraggingOver = $state(false);
  let dropIndex = $state<number | null>(null);
  let dragCounter = 0;

  function handleSectionDragStart(e: DragEvent, idx: number) {
    if (!e.dataTransfer) return;
    e.dataTransfer.setData('application/x-genix-reorder', idx.toString());
    e.dataTransfer.effectAllowed = 'move';
  }

  function handleDragOver(e: DragEvent, idx: number) {
    e.preventDefault();
    e.stopPropagation();
    
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const midpoint = rect.top + rect.height / 2;
    const newDropIndex = e.clientY < midpoint ? idx : idx + 1;
    
    if (dropIndex !== newDropIndex) {
      dropIndex = newDropIndex;
    }
    isDraggingOver = true;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDraggingOver = false;
    dragCounter = 0;
    
    const targetIndex = dropIndex ?? editorStore.sections.length;
    
    if (e.dataTransfer?.getData('application/x-genix-reorder')) {
      const fromIndex = parseInt(e.dataTransfer.getData('application/x-genix-reorder'));
      if (!isNaN(fromIndex)) {
        editorStore.reorder(fromIndex, targetIndex);
      }
    } else if (e.dataTransfer?.getData('application/x-genix-template')) {
      const templateData = JSON.parse(e.dataTransfer.getData('application/x-genix-template'));
      editorStore.addSection(templateData.id, targetIndex);
    }
    
    dropIndex = null;
    onUpdate?.(editorStore.sections);
  }

  const paletteStyles = $derived(generatePaletteStyles(palette));
</script>

<svelte:head>
  {#if liveCSS.css}
    {@html `<style id="live-tailwind-jit">${liveCSS.css}</style>`}
  {/if}
</svelte:head>

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
    class:is-dragging-over={isDraggingOver}
    ondrop={handleDrop}
    ondragenter={(e) => {
      e.preventDefault();
      dragCounter++;
      isDraggingOver = true;
    }}
    ondragover={(e) => {
      e.preventDefault();
      isDraggingOver = true;
      if (e.target === e.currentTarget) {
        const sections = e.currentTarget.querySelectorAll('.section-wrapper');
        if (sections.length > 0) {
          const lastSection = sections[sections.length - 1];
          const rect = lastSection.getBoundingClientRect();
          if (e.clientY > rect.bottom) {
            dropIndex = editorStore.sections.length;
          }
        } else {
          dropIndex = 0;
        }
      }
    }}
    ondragleave={(e) => {
      dragCounter--;
      if (dragCounter <= 0) {
        dragCounter = 0;
        isDraggingOver = false; 
        dropIndex = null;
      }
    }}
  >
    {#each editorStore.sections as section, idx (section.id)}
      {#if isDraggingOver && dropIndex === idx}
        {@render renderPlaceholder()}
      {/if}
      
      <BuilderSectionRender 
        {section}
        index={idx}
        {paletteStyles}
        onDragStart={handleSectionDragStart}
        onDragEnd={() => {
          isDraggingOver = false;
          dropIndex = null;
          dragCounter = 0;
        }}
        onDragOver={handleDragOver}
      />
      
      {#if isDraggingOver && dropIndex === idx + 1 && idx === editorStore.sections.length - 1}
        {@render renderPlaceholder()}
      {/if}
    {/each}

    {#if editorStore.sections.length === 0}
      <div class="empty-canvas">
        <div class="empty-content">
          <h3>Your store is empty</h3>
          <p>Drag a section from the templates tab to start building.</p>
        </div>
      </div>
    {/if}
  </div>

  <SectionEditorLayer
    palette={palette}
    onClose={() => editorStore.select(null)}
  />
</div>

<style>
  .ecommerce-builder {
    position: relative;
    display: flex;
    min-height: 100vh;
    background: #f8fafc;
  }

  .builder-canvas {
    flex: 1;
    padding-bottom: 200px;
  }

  .builder-canvas.is-dragging-over :global(.section-content) {
    pointer-events: none;
  }

  .empty-canvas {
    height: 60vh;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 4px dashed #e2e8f0;
    margin: 40px;
    border-radius: 24px;
  }

  .empty-content {
    text-align: center;
    color: #94a3b8;
  }

  .empty-content h3 {
    font-size: 24px;
    color: #64748b;
    margin-bottom: 8px;
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
    animation: placeholder-pulse 1.5s infinite ease-in-out;
    pointer-events: none;
  }

  .placeholder-content {
    display: flex;
    align-items: center;
    gap: 12px;
    color: #3b82f6;
    font-size: 14px;
    font-weight: 600;
  }

  @keyframes placeholder-pulse {
    0% { opacity: 0.6; transform: scale(0.995); }
    50% { opacity: 1; transform: scale(1); }
    100% { opacity: 0.6; transform: scale(0.995); }
  }
</style>