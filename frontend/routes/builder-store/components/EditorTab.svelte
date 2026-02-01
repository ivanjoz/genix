<script lang="ts">
  import type { ComponentVariable } from '../../../pkg-store/renderer/renderer-types';
  import type { EditableField, SelectedSection } from '../EcommerceBuilder.svelte';

  interface Props {
    section: SelectedSection;
    onFieldUpdate: (field: EditableField, newValue: string) => void;
  }

  let {
    section,
    onFieldUpdate
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

  function handleInput(field: EditableField, event: any) {
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

  const textFields = $derived(() => section.fields.filter(f => f.type === 'text' || f.type === 'textLine'));
  const variableFields = $derived(() => section.fields.filter(f => f.type === 'variable'));
  const imageFields = $derived(() => section.fields.filter(f => f.type === 'image'));
  const attributeFields = $derived(() => section.fields.filter(f => f.type === 'attribute'));

  $effect(() => {
    if (section) pendingUpdates = new Map();
  });
</script>

<div class="editor-tab">
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
                        handleInput(field, { target: { value: val } });
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
</div>

<style>
  .editor-tab {
    display: flex;
    flex-direction: column;
    gap: 20px;
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
</style>
