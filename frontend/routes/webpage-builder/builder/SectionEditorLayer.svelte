<script lang="ts">
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
import { editorStore } from '../stores/editor.svelte';
  import { savePageContent, getCurrentPageID } from '$services/ecommerce/page-content.svelte';
  import { uploadShowcaseImage } from '$services/webpage/pages.svelte';
  import { captureShowcaseBlob } from './showcase-capture';
  import { Loading, Notify } from '$libs/helpers';
  import { Core, tr } from '$core/store.svelte';
  import T from '$components/misc/T.svelte';
  import EditorTab from '../components/EditorTab.svelte';
  import TemplatesTab from '../components/TemplatesTab.svelte';
  import GalleryTab from '../components/GalleryTab.svelte';
  import ConfigTab from '../components/ConfigTab.svelte';

  interface Props {
    palette?: ColorPalette;
    onClose: () => void;
  }

  let {
    palette,
    onClose
  }: Props = $props();

  let activeTabId = $state('editor');

  const tabs = [
    { id: 'editor', name: 'Editor|Editor', icon: '✏️' },
    { id: 'templates', name: 'Templates|Plantillas', icon: '🧩' },
    { id: 'gallery', name: 'Gallery|Galería', icon: '🖼️' },
    { id: 'config', name: 'Config|Config', icon: '⚙️' }
  ];

  // Auto-switch to editor tab when a section is selected
  $effect(() => {
    if (editorStore.selectedId) {
      activeTabId = 'editor';
    }
  });

  function handleClose() {
    editorStore.select(null);
    onClose();
  }

  async function handleSave() {
    // Nothing to persist if no section was created, deleted, reordered, or edited
    // since the last load/save — avoid a pointless round-trip and tell the user.
    if (!editorStore.hasUnsavedChanges) {
      Notify.info('No hay cambios a enviar.');
      return;
    }

    // Capture the showcase thumbnail first — the only step the user waits on (the
    // DOM must be in its current, unmodified state). Conversion + upload run in the
    // background afterwards so saving isn't blocked.
    Loading.standard(tr('Generating preview...|Generando vista previa...', Core.languaje));
    let thumbnail: Blob | null = null;
    try {
      thumbnail = await captureShowcaseBlob();
      // Persist every section of the page. The backend assigns each section's
      // position-based id, hashes its content, and writes only what changed.
      await savePageContent(editorStore.sections);
      // Re-snapshot the now-saved state so a subsequent Save with no further edits
      // correctly reports "no changes".
      editorStore.captureBaseline();
    } finally {
      Loading.remove();
    }

    // Fire-and-forget: convert to AVIF and upload as the page thumbnail. Skipped for
    // the bare /webpage-builder route (pageID 0), which has no concrete page to attach to.
    const pageID = getCurrentPageID();
    if (thumbnail && pageID > 0) uploadShowcaseImage(pageID, thumbnail);
  }

  function handleDelete() {
    if (editorStore.selectedId) {
      editorStore.removeSection(editorStore.selectedId);
      onClose();
    }
  }
</script>

<div class="editor-layer">
  <div class="layer-inner">
    <div class="layer-header">
      <div class="tab-nav">
        {#each tabs as tab}
          <button
            class="tab-btn"
            class:active={activeTabId === tab.id}
            onclick={() => activeTabId = tab.id}
            title={tr(tab.name, Core.languaje)}
            aria-label={tr(tab.name, Core.languaje)}
          >
            <span class="tab-icon">{tab.icon}</span>
            <span class="tab-name"><T text={tab.name} /></span>
          </button>
        {/each}
      </div>
    </div>

    {#if editorStore.selectedId && activeTabId === 'editor'}
      <div class="layer-actions">
        {#if editorStore.activeSchema}
          <span class="section-title" title={editorStore.activeSchema.name}>{editorStore.activeSchema.name}</span>
        {/if}
        <button class="action-btn save-btn" onclick={handleSave} title={tr('Save|Guardar', Core.languaje)}>
          <T text="Save|Guardar" />
        </button>
        <button class="action-btn delete-btn" onclick={handleDelete} title={tr('Delete|Eliminar', Core.languaje)} aria-label="Delete the section"><i class="icon-trash"></i></button>
        <button class="action-btn close-btn icon-btn" onclick={handleClose} title={tr('Close|Cerrar', Core.languaje)} aria-label="Close the section editor"><i class="icon-cancel"></i></button>
      </div>
    {/if}

    <div class="layer-content">
      {#if activeTabId === 'editor'}
        {#if editorStore.selectedSection && editorStore.activeSchema}
          <EditorTab {palette} />
        {:else}
          <div class="empty-state">
            <span class="empty-icon">👆</span>
            <p><T text="Select a section to edit|Selecciona una sección para editar" /></p>
          </div>
        {/if}
      {:else if activeTabId === 'templates'}
        <TemplatesTab onSelect={(template) => editorStore.addSection(template.id)} />
      {:else if activeTabId === 'gallery'}
        <GalleryTab />
      {:else if activeTabId === 'config'}
        <ConfigTab {palette} />
      {/if}
    </div>
  </div>
</div>

<style>
  .editor-layer {
    position: fixed;
    top: var(--header-height);
    right: 0;
    width: 280px;
    height: calc(100vh - var(--header-height));
    background: #0f172a;
    border-left: 1px solid #1e293b;
    z-index: 205;
    overflow: hidden;
    transition: width 0.24s cubic-bezier(0.4, 0, 0.2, 1);
    color: #e2e8f0;
  }

  .editor-layer:hover, .layer-inner {
    width: 410px;
  }

  .editor-layer:hover {
    box-shadow: -10px 0 30px rgba(0, 0, 0, 0.5);
  }

  .layer-inner {
    height: 100%;
    display: flex;
    flex-direction: column;
    position: absolute;
    left: 0;
    top: 0;
  }

  .layer-header {
    background: #1e293b;
    border-bottom: 1px solid #334155;
    flex-shrink: 0;
    display: flex;
    align-items: center;
  }

  .tab-nav {
    display: flex;
    padding: 6px;
    gap: 6px;
    flex: 1;
  }

  .tab-btn {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 5px 4px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.2s;
  }

  .tab-btn:hover {
    background: rgba(255, 255, 255, 0.05);
    color: #f1f5f9;
  }

  .tab-btn.active {
    background: #334155;
    color: #3b82f6;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  }

  .tab-icon {
    font-size: 17px;
    line-height: 1;
  }

  .tab-name {
    font-size: 12px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.8px;
    line-height: 1;
  }

  .layer-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    padding: 8px 12px;
    flex-shrink: 0;
    border-bottom: 1px solid #1e293b;
  }

  .section-title {
    flex: 1;
    min-width: 0;
    font-size: 14px;
    font-weight: 700;
    color: #94a3b8;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .action-btn {
    border: none;
    border-radius: 6px;
    padding: 6px 12px;
    font-size: 13px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .save-btn {
    background: #3b82f6;
    color: white;
  }

  .save-btn:hover {
    background: #2563eb;
  }

  .delete-btn {
    background: #7f1d1d;
    color: #fecaca;
  }

  .delete-btn:hover {
    background: #991b1b;
    color: white;
  }

  .close-btn {
    background: #334155;
    color: #cbd5e1;
  }

  .close-btn:hover {
    background: #475569;
    color: white;
  }

  .icon-btn {
    padding: 6px 14px;
    font-size: 15px;
    line-height: 1;
  }

  .layer-content {
    flex: 1;
    overflow-y: auto;
    padding-top: 12px;
    padding-left: 16px;
    padding-right: 12px;
    padding-bottom: 12px;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #64748b;
    text-align: center;
  }

  .empty-icon {
    font-size: 40px;
    margin-bottom: 12px;
  }

  .layer-content::-webkit-scrollbar {
    width: 6px;
  }
  .layer-content::-webkit-scrollbar-track {
    background: transparent;
  }
  .layer-content::-webkit-scrollbar-thumb {
    background: #334155;
    border-radius: 3px;
  }
</style>
