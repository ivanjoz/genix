<script lang="ts">
  import type { ColorPalette, SectionTemplate } from '../../pkg-store/renderer/renderer-types';
  import type { EditableField, SelectedSection } from './EcommerceBuilder.svelte';
  import EditorTab from './components/EditorTab.svelte';
  import TemplatesTab from './components/TemplatesTab.svelte';
  import AIChatTab from './components/AIChatTab.svelte';
  import ConfigTab from './components/ConfigTab.svelte';

  interface Props {
    open: boolean;
    section: SelectedSection | null;
    palette?: ColorPalette;
    onFieldUpdate: (field: EditableField, newValue: string) => void;
    onClose: () => void;
  }

  let {
    open = $bindable(),
    section,
    palette,
    onFieldUpdate,
    onClose
  }: Props = $props();

  let activeTabId = $state('editor');

  const tabs = [
    { id: 'editor', name: 'Editor', icon: '✏️' },
    { id: 'templates', name: 'Templates', icon: '🧩' },
    { id: 'chat', name: 'AI Chat', icon: '🤖' },
    { id: 'config', name: 'Config', icon: '⚙️' }
  ];

  function handleTemplateSelect(template: SectionTemplate) {
    console.log('Selected template:', template);
    // Logic to add template to builder would go here
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
            title={tab.name}
          >
            <span class="tab-icon">{tab.icon}</span>
            <span class="tab-name">{tab.name}</span>
          </button>
        {/each}
      </div>
    </div>

    <div class="layer-content">
      {#if activeTabId === 'editor'}
        {#if section}
          <EditorTab {section} {onFieldUpdate} />
        {:else}
          <div class="empty-state">
            <span class="empty-icon">👆</span>
            <p>Select a section to edit</p>
          </div>
        {/if}
      {:else if activeTabId === 'templates'}
        <TemplatesTab onSelect={handleTemplateSelect} />
      {:else if activeTabId === 'chat'}
        <AIChatTab />
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
    transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    color: #e2e8f0;
  }

  .editor-layer:hover {
    width: 400px;
    box-shadow: -10px 0 30px rgba(0, 0, 0, 0.5);
  }

  /* Fixed width container to prevent internal resizing/squishing */
  .layer-inner {
    width: 400px;
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
    padding-top: 8px;
  }

  .tab-nav {
    display: flex;
    padding: 0 12px 12px;
    gap: 6px;
  }

  .tab-btn {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 10px 4px;
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
    font-size: 20px;
  }

  .tab-name {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }

  .layer-content {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
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

  .empty-state p {
    margin: 0;
    font-size: 14px;
  }

  /* Custom Scrollbar */
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
