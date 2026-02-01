<script lang="ts">
  import type { ColorPalette, ComponentVariable, ITextLine } from '../../pkg-store/renderer/renderer-types';
  import type { EditableField, SelectedSection } from './EcommerceBuilder.svelte';

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

  let pendingUpdates = $state<Map<string, string>>(new Map());
  let activeInput = $state<string | null>(null);

  function getFieldKey(field: EditableField): string {
    return field.path.join('.');
  }

  function getDisplayValue(field: EditableField): string {
    const key = getFieldKey(field);
    if (pendingUpdates.has(key)) {
      return pendingUpdates.get(key)!;
    }
    return field.value;
  }

  function handleInput(field: EditableField, event: Event) {
    const target = event.target as HTMLInputElement | HTMLTextAreaElement;
    const key = getFieldKey(field);
    pendingUpdates.set(key, target.value);
    pendingUpdates = new Map(pendingUpdates);
  }

  function handleBlur(field: EditableField) {
    const key = getFieldKey(field);
    activeInput = null;
    
    if (pendingUpdates.has(key)) {
      const newValue = pendingUpdates.get(key)!;
      if (newValue !== field.value) {
        onFieldUpdate(field, newValue);
      }
      pendingUpdates.delete(key);
      pendingUpdates = new Map(pendingUpdates);
    }
  }

  function handleFocus(field: EditableField) {
    activeInput = getFieldKey(field);
  }

  function handleKeyDown(event: KeyboardEvent, field: EditableField) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      (event.target as HTMLInputElement).blur();
    }
    if (event.key === 'Escape') {
      const key = getFieldKey(field);
      pendingUpdates.delete(key);
      pendingUpdates = new Map(pendingUpdates);
      (event.target as HTMLInputElement).blur();
    }
  }

  function formatVariableType(variable: ComponentVariable): string {
    const parts: string[] = [];
    if (variable.type) parts.push(variable.type);
    if (variable.min !== undefined && variable.max !== undefined) {
      parts.push(`${variable.min} - ${variable.max}`);
    }
    if (variable.units?.length) {
      parts.push(variable.units.join(', '));
    }
    return parts.join(' · ');
  }

  // Get all text fields (text + textLine) in order they appear
  const textFields = $derived(() => {
    if (!section?.fields) return [];
    return section.fields.filter(f => f.type === 'text' || f.type === 'textLine');
  });

  // Get other field types
  const variableFields = $derived(() => {
    if (!section?.fields) return [];
    return section.fields.filter(f => f.type === 'variable');
  });

  const imageFields = $derived(() => {
    if (!section?.fields) return [];
    return section.fields.filter(f => f.type === 'image');
  });

  const attributeFields = $derived(() => {
    if (!section?.fields) return [];
    return section.fields.filter(f => f.type === 'attribute');
  });

  // Reset pending updates when section changes
  $effect(() => {
    if (section) {
      pendingUpdates = new Map();
    }
  });
</script>

<div class="editor-layer" class:open>
  <div class="editor-header">
    <div class="editor-title">
      <span class="editor-icon">🎨</span>
      <span>Section Editor</span>
    </div>
    <button class="close-btn" onclick={onClose} aria-label="Close editor">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 6L6 18M6 6l12 12"/>
      </svg>
    </button>
  </div>

  {#if section}
    <div class="editor-content">
      <div class="section-info">
        <span class="section-tag">{section.element.tagName || 'div'}</span>
        {#if section.element.id}
          <span class="section-id">#{section.element.id}</span>
        {/if}
      </div>

      {#if section.fields.length === 0}
        <div class="empty-state">
          <span class="empty-icon">📭</span>
          <p>No editable content in this section</p>
          <p class="empty-hint">This section might only contain layout elements</p>
        </div>
      {:else}
        <div class="fields-container">
          <!-- Text Content Fields -->
          {#if textFields().length > 0}
            <div class="field-group">
              <div class="group-header">
                <span class="group-icon">✏️</span>
                <span class="group-label">Text Content</span>
                <span class="group-count">{textFields().length}</span>
              </div>
              
              <div class="group-fields">
                {#each textFields() as field, idx}
                  {@const fieldKey = getFieldKey(field)}
                  {@const isActive = activeInput === fieldKey}
                  {@const tag = field.elementTag || field.textLine?.tag || 'text'}
                  
                  <div class="field-item" class:active={isActive}>
                    <div class="field-header">
                      <span class="field-number">{idx + 1}</span>
                      <span class="tag-badge">&lt;{tag}&gt;</span>
                      <span class="field-preview">{field.label}</span>
                    </div>
                    
                    <textarea
                      id={fieldKey}
                      class="field-input textarea"
                      value={getDisplayValue(field)}
                      oninput={(e) => handleInput(field, e)}
                      onblur={() => handleBlur(field)}
                      onfocus={() => handleFocus(field)}
                      onkeydown={(e) => handleKeyDown(e, field)}
                      rows="2"
                      placeholder="Enter text..."
                    ></textarea>
                    
                    {#if isActive}
                      <div class="field-hint">
                        Press <kbd>Enter</kbd> to apply · <kbd>Esc</kbd> to cancel
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          <!-- Variable Fields -->
          {#if variableFields().length > 0}
            <div class="field-group">
              <div class="group-header">
                <span class="group-icon">⚙️</span>
                <span class="group-label">Variables</span>
                <span class="group-count">{variableFields().length}</span>
              </div>
              
              <div class="group-fields">
                {#each variableFields() as field}
                  {@const fieldKey = getFieldKey(field)}
                  {@const isActive = activeInput === fieldKey}
                  
                  <div class="field-item" class:active={isActive}>
                    <label class="field-label" for={fieldKey}>
                      {field.label}
                      {#if field.variable}
                        <span class="variable-info">{formatVariableType(field.variable)}</span>
                      {/if}
                    </label>
                    
                    <div class="variable-input-wrapper">
                      {#if field.variable?.min !== undefined && field.variable?.max !== undefined}
                        <input
                          type="range"
                          class="field-range"
                          value={parseInt(getDisplayValue(field)) || field.variable.min}
                          min={field.variable.min}
                          max={field.variable.max}
                          step={field.variable.step || 1}
                          oninput={(e) => {
                            const val = (e.target as HTMLInputElement).value + 'px';
                            handleInput(field, { target: { value: val } } as any);
                          }}
                          onchange={() => handleBlur(field)}
                        />
                      {/if}
                      <input
                        id={fieldKey}
                        type="text"
                        class="field-input"
                        value={getDisplayValue(field)}
                        oninput={(e) => handleInput(field, e)}
                        onblur={() => handleBlur(field)}
                        onfocus={() => handleFocus(field)}
                        onkeydown={(e) => handleKeyDown(e, field)}
                        placeholder={field.variable?.defaultValue || 'Value'}
                      />
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          <!-- Image Fields -->
          {#if imageFields().length > 0}
            <div class="field-group">
              <div class="group-header">
                <span class="group-icon">🖼️</span>
                <span class="group-label">Images</span>
                <span class="group-count">{imageFields().length}</span>
              </div>
              
              <div class="group-fields">
                {#each imageFields() as field}
                  {@const fieldKey = getFieldKey(field)}
                  {@const isActive = activeInput === fieldKey}
                  
                  <div class="field-item" class:active={isActive}>
                    <label class="field-label" for={fieldKey}>{field.label}</label>
                    
                    <div class="image-input-wrapper">
                      <input
                        id={fieldKey}
                        type="text"
                        class="field-input"
                        value={getDisplayValue(field)}
                        oninput={(e) => handleInput(field, e)}
                        onblur={() => handleBlur(field)}
                        onfocus={() => handleFocus(field)}
                        onkeydown={(e) => handleKeyDown(e, field)}
                        placeholder="Image URL..."
                      />
                      {#if getDisplayValue(field)}
                        <div class="image-preview">
                          <img src={getDisplayValue(field)} alt="Preview" />
                        </div>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          <!-- Attribute Fields -->
          {#if attributeFields().length > 0}
            <div class="field-group">
              <div class="group-header">
                <span class="group-icon">🔗</span>
                <span class="group-label">Links & Attributes</span>
                <span class="group-count">{attributeFields().length}</span>
              </div>
              
              <div class="group-fields">
                {#each attributeFields() as field}
                  {@const fieldKey = getFieldKey(field)}
                  {@const isActive = activeInput === fieldKey}
                  
                  <div class="field-item" class:active={isActive}>
                    <label class="field-label" for={fieldKey}>
                      {field.label}
                      {#if field.elementTag}
                        <span class="tag-badge">&lt;{field.elementTag}&gt;</span>
                      {/if}
                    </label>
                    
                    <input
                      id={fieldKey}
                      type="text"
                      class="field-input"
                      value={getDisplayValue(field)}
                      oninput={(e) => handleInput(field, e)}
                      onblur={() => handleBlur(field)}
                      onfocus={() => handleFocus(field)}
                      onkeydown={(e) => handleKeyDown(e, field)}
                      placeholder="Enter value..."
                    />
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      {/if}

      {#if palette}
        <div class="palette-preview">
          <div class="group-header">
            <span class="group-icon">🎨</span>
            <span class="group-label">Color Palette</span>
          </div>
          <div class="palette-colors">
            {#each palette.colors as color, idx}
              <div 
                class="color-swatch" 
                style="background-color: {color}"
                title="COLOR:{idx + 1} - {color}"
              >
                <span class="color-index">{idx + 1}</span>
              </div>
            {/each}
          </div>
          <div class="palette-name">{palette.name}</div>
        </div>
      {/if}
    </div>
  {:else}
    <div class="empty-state centered">
      <span class="empty-icon">👆</span>
      <p>Click on a section to edit</p>
      <p class="empty-hint">Hover over sections to see their boundaries</p>
    </div>
  {/if}
</div>

<style>
  .editor-layer {
    position: fixed;
    top: 0;
    right: 0;
    width: 400px;
    height: 100vh;
    background: #0f172a;
    border-left: 1px solid #1e293b;
    transform: translateX(100%);
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    z-index: 1000;
    display: flex;
    flex-direction: column;
    font-family: 'Inter', system-ui, -apple-system, sans-serif;
    color: #e2e8f0;
  }

  .editor-layer.open {
    transform: translateX(0);
  }

  .editor-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    background: #1e293b;
    border-bottom: 1px solid #334155;
  }

  .editor-title {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 16px;
    font-weight: 600;
    color: #f1f5f9;
  }

  .editor-icon {
    font-size: 20px;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    color: #94a3b8;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .close-btn:hover {
    background: #334155;
    color: #f1f5f9;
  }

  .editor-content {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  .section-info {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 20px;
  }

  .section-tag {
    background: #3b82f6;
    color: white;
    font-size: 12px;
    font-weight: 600;
    padding: 4px 10px;
    border-radius: 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .section-id {
    color: #64748b;
    font-size: 13px;
    font-family: 'JetBrains Mono', monospace;
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

  .fields-container {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .field-group {
    background: #1e293b;
    border-radius: 12px;
    overflow: hidden;
  }

  .group-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background: #334155;
    font-size: 13px;
    font-weight: 600;
    color: #cbd5e1;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .group-icon {
    font-size: 14px;
  }

  .group-label {
    flex: 1;
  }

  .group-count {
    background: #475569;
    color: #94a3b8;
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 10px;
  }

  .group-fields {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .field-item {
    background: #0f172a;
    border-radius: 8px;
    padding: 12px;
    border: 1px solid transparent;
    transition: all 0.15s ease;
  }

  .field-item.active {
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.15);
  }

  .field-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
  }

  .field-number {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    background: #3b82f6;
    color: white;
    font-size: 11px;
    font-weight: 600;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .field-preview {
    flex: 1;
    font-size: 12px;
    color: #94a3b8;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .field-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    font-weight: 500;
    color: #94a3b8;
    margin-bottom: 8px;
  }

  .variable-info {
    color: #64748b;
    font-size: 10px;
    font-weight: 400;
  }

  .tag-badge {
    background: #475569;
    color: #cbd5e1;
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: 'JetBrains Mono', monospace;
    flex-shrink: 0;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #f1f5f9;
    font-size: 14px;
    font-family: inherit;
    transition: all 0.15s ease;
    box-sizing: border-box;
  }

  .field-input:focus {
    outline: none;
    border-color: #3b82f6;
    background: #1e293b;
  }

  .field-input::placeholder {
    color: #475569;
  }

  .field-input.textarea {
    resize: vertical;
    min-height: 60px;
    line-height: 1.5;
  }

  .variable-input-wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .field-range {
    width: 100%;
    height: 6px;
    background: #334155;
    border-radius: 3px;
    appearance: none;
    cursor: pointer;
  }

  .field-range::-webkit-slider-thumb {
    appearance: none;
    width: 16px;
    height: 16px;
    background: #3b82f6;
    border-radius: 50%;
    cursor: pointer;
    transition: transform 0.15s ease;
  }

  .field-range::-webkit-slider-thumb:hover {
    transform: scale(1.2);
  }

  .image-input-wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .image-preview {
    width: 100%;
    height: 80px;
    background: #1e293b;
    border-radius: 6px;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .image-preview img {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }

  .field-hint {
    margin-top: 8px;
    font-size: 11px;
    color: #64748b;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .field-hint kbd {
    background: #334155;
    color: #94a3b8;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 10px;
  }

  .palette-preview {
    margin-top: 24px;
    background: #1e293b;
    border-radius: 12px;
    overflow: hidden;
  }

  .palette-colors {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(36px, 1fr));
    gap: 4px;
    padding: 12px;
  }

  .color-swatch {
    aspect-ratio: 1;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: transform 0.15s ease;
    position: relative;
  }

  .color-swatch:hover {
    transform: scale(1.1);
    z-index: 1;
  }

  .color-index {
    font-size: 10px;
    font-weight: 600;
    color: white;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
  }

  .palette-name {
    padding: 8px 12px;
    background: #334155;
    font-size: 12px;
    color: #94a3b8;
    text-align: center;
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
