<script lang="ts">
  import type { ColorPalette, SectionTemplate } from '../../pkg-store/renderer/renderer-types';
  import type { EditableField, SelectedSection } from './EcommerceBuilder.svelte';
  import OptionsStrip from '../../pkg-components/OptionsStrip.svelte';
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
  <div class="editor-header-nav">
    <OptionsStrip
      options={tabs}
      selected={activeTabId}
      keyId="id"
      keyName="name"
      onSelect={(tab) => activeTabId = tab.id}
      containerCss="tab-nav"
      itemCss="tab-item"
      activeClass="tab-active"
    />
  </div>

  <div class="editor-content">
    {#if activeTabId === 'editor'}
      {#if section}
        <EditorTab {section} {onFieldUpdate} />
      {:else}
        <div class="empty-state centered">
          <span class="empty-icon">👆</span>
          <p>Click on a section to edit</p>
          <p class="empty-hint">Hover over sections to see their boundaries</p>
        </div>
      {/if}
    {:else}
      <div class="tab-body">
        {#if activeTabId === 'templates'}
          <TemplatesTab onSelect={handleTemplateSelect} />
        {:else if activeTabId === 'chat'}
          <AIChatTab />
        {:else if activeTabId === 'config'}
          <ConfigTab {palette} />
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .editor-layer {
    position: fixed;
    top: var(--header-height);
    right: 0;
    width: 260px;
    height: calc(100vh - var(--header-height));
    background: #0f172a;
    border-left: 1px solid #1e293b;
    transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    z-index: 205;
    display: flex;
    flex-direction: column;
    color: #e2e8f0;
    overflow: hidden;
  }

  .editor-layer:hover {
    width: 420px;
    box-shadow: -10px 0 20px -4px rgb(0 0 0 / 42%);
  }

  .editor-header-nav {
    background: #1e293b;
    border-bottom: 1px solid #334155;
    padding: 0 10px;
    flex-shrink: 0;
  }

  :global(.tab-nav) {
    border: none !important;
    gap: 4px;
    padding: 8px 0 !important;
  }

  :global(.tab-item) {
    min-width: 80px !important;
    height: 36px !important;
    border-radius: 6px !important;
    border: none !important;
    font-size: 13px !important;
    color: #94a3b8 !important;
    background: transparent !important;
    transition: all 0.2s ease !important;
    padding: 0 12px !important;
    align-items: center !important;
  }

  :global(.tab-item:hover) {
    background: rgba(255, 255, 255, 0.05) !important;
    color: #f1f5f9 !important;
  }

  :global(.tab-active) {
    background: #3b82f6 !important;
    color: white !important;
    font-weight: 600 !important;
  }

  .editor-content {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
    min-width: 420px;
  }

  .tab-body {
    height: 100%;
  }

  .empty-state {
    text-align: center;
    padding: 40px 20px;
    color: #64748b;
  }

  .empty-state.centered {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
  }

  .empty-icon {
    font-size: 48px;
    margin-bottom: 16px;
    display: block;
  }

  .empty-state p {
    margin: 0;
    font-size: 14px;
  }

  .empty-hint {
    color: #475569;
    font-size: 12px !important;
    margin-top: 8px !important;
  }

  /* Scrollbar styling */
  .editor-content::-webkit-scrollbar {
    width: 8px;
  }

  .editor-content::-webkit-scrollbar-track {
    background: #1e293b;
  }

  .editor-content::-webkit-scrollbar-thumb {
    background: #475569;
    border-radius: 4px;
  }

  .editor-content::-webkit-scrollbar-thumb:hover {
    background: #64748b;
  }
</style>
