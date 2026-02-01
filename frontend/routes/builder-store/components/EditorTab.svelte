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
  {#if section.fields.length === 0}
    <div class="empty-state">
      <span class="empty-icon">📭</span>
      <p>No editable content here</p>
    </div>
  {:else if section.fields.length === 1 && (section.fields[0].type === 'text' || section.fields[0].type === 'textLine')}
    {@const field = section.fields[0]}
    <div class="simple-editor">
      <label class="main-label">Content</label>
      <textarea
        class="main-textarea"
        value={getDisplayValue(field)}
        oninput={(e) => handleInput(field, e)}
        onblur={() => handleBlur(field)}
        onfocus={() => handleFocus(field)}
        onkeydown={(e) => handleKeyDown(e, field)}
        rows="8"
        placeholder="Enter text..."
      ></textarea>
      <div class="field-hint">
        Changes are saved automatically on blur
      </div>
    </div>
  {:else}
    <div class="fields-list">
      {#each section.fields as field}
        {@const fieldKey = getFieldKey(field)}
        {@const isActive = activeInput === fieldKey}
        <div class="field-item" class:active={isActive}>
          <div class="field-info">
            <span class="field-label">{field.label}</span>
            <span class="field-type">{field.type}</span>
          </div>

          {#if field.type === 'text' || field.type === 'textLine'}
            <textarea
              class="field-input textarea"
              value={getDisplayValue(field)}
              oninput={(e) => handleInput(field, e)}
              onblur={() => handleBlur(field)}
              onfocus={() => handleFocus(field)}
              onkeydown={(e) => handleKeyDown(e, field)}
              rows="2"
            ></textarea>
          {:else if field.type === 'variable'}
            <div class="variable-row">
              <input
                type="text"
                class="field-input"
                value={getDisplayValue(field)}
                oninput={(e) => handleInput(field, e)}
                onblur={() => handleBlur(field)}
                onfocus={() => handleFocus(field)}
                onkeydown={(e) => handleKeyDown(e, field)}
              />
              {#if field.variable?.min !== undefined}
                <input
                  type="range"
                  class="field-range"
                  value={parseInt(getDisplayValue(field)) || 0}
                  min={field.variable.min}
                  max={field.variable.max}
                  oninput={(e) => {
                    const val = (e.target as HTMLInputElement).value + 'px';
                    handleInput(field, { target: { value: val } });
                  }}
                  onchange={() => handleBlur(field)}
                />
              {/if}
            </div>
          {:else}
            <input
              type="text"
              class="field-input"
              value={getDisplayValue(field)}
              oninput={(e) => handleInput(field, e)}
              onblur={() => handleBlur(field)}
              onfocus={() => handleFocus(field)}
              onkeydown={(e) => handleKeyDown(e, field)}
            />
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .editor-tab {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .simple-editor {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .main-label {
    font-size: 12px;
    font-weight: 600;
    color: #94a3b8;
    text-transform: uppercase;
  }

  .main-textarea {
    width: 100%;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    color: #f1f5f9;
    padding: 16px;
    font-size: 15px;
    line-height: 1.6;
    resize: vertical;
    outline: none;
    transition: border-color 0.2s;
    box-sizing: border-box;
  }

  .main-textarea:focus {
    border-color: #3b82f6;
  }

  .fields-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .field-item {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-bottom: 16px;
    border-bottom: 1px solid #1e293b;
  }

  .field-item:last-child {
    border-bottom: none;
  }

  .field-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .field-label {
    font-size: 13px;
    font-weight: 500;
    color: #cbd5e1;
  }

  .field-type {
    font-size: 10px;
    color: #64748b;
    background: #1e293b;
    padding: 2px 6px;
    border-radius: 4px;
    text-transform: uppercase;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #f1f5f9;
    font-size: 14px;
    transition: all 0.2s;
    box-sizing: border-box;
  }

  .field-input:focus {
    outline: none;
    border-color: #3b82f6;
  }

  .field-input.textarea {
    resize: vertical;
    min-height: 80px;
  }

  .variable-row {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .field-range {
    width: 100%;
    height: 4px;
    background: #334155;
    border-radius: 2px;
    appearance: none;
    cursor: pointer;
  }

  .field-range::-webkit-slider-thumb {
    appearance: none;
    width: 14px;
    height: 14px;
    background: #3b82f6;
    border-radius: 50%;
  }

  .field-hint {
    font-size: 11px;
    color: #64748b;
    margin-top: 4px;
  }

  .empty-state {
    text-align: center;
    padding: 40px 0;
    color: #64748b;
  }

  .empty-icon {
    font-size: 32px;
    margin-bottom: 12px;
    display: block;
  }
</style>
