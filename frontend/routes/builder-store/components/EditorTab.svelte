<script lang="ts">
import { editorStore } from '$stores/editor.svelte';
import type { StandardContent } from '$ecommerce/renderer/section-types';

  const section = $derived(editorStore.selectedSection);
  const schema = $derived(editorStore.activeSchema);

  function handleContentInput(key: keyof StandardContent, value: any) {
    if (section) {
      editorStore.updateContent(section.id, key as string, value);
    }
  }

  function handleCssInput(slot: string, value: string) {
    if (section) {
      editorStore.updateCss(section.id, slot, value);
    }
  }

  function handleTextLineChange(index: number, value: string) {
    if (!section) return;
    const lines = [...(section.content.textLines || [])];
    if (!lines[index]) {
      lines[index] = { text: value, css: '' };
    } else {
      lines[index] = { ...lines[index], text: value };
    }
    editorStore.updateContent(section.id, 'textLines', lines);
  }

  function addTextLine() {
    if (!section) return;
    const lines = [...(section.content.textLines || []), { text: 'New Line', css: '' }];
    editorStore.updateContent(section.id, 'textLines', lines);
  }

  function removeTextLine(index: number) {
    if (!section) return;
    const lines = (section.content.textLines || []).filter((_, i) => i !== index);
    editorStore.updateContent(section.id, 'textLines', lines);
  }
</script>

{#if section && schema}
  <div class="editor-tab">
    <div class="section-meta">
      <h3>{schema.name}</h3>
      <p>{schema.description}</p>
    </div>

    <div class="editor-groups">
      <!-- CONTENT GROUP -->
      <div class="editor-group">
        <h4 class="group-title">Content</h4>
        <div class="fields-list">
          {#each schema.content as fieldKey}
            <div class="field-item">
              <label class="field-label" for={`content-${fieldKey}`}>{fieldKey}</label>
              
              {#if fieldKey === 'textLines'}
                <div class="text-lines-editor">
                  {#each section.content.textLines || [] as line, i}
                    <div class="text-line-item">
                      <input 
                        type="text" 
                        class="field-input" 
                        value={line.text}
                        oninput={(e) => handleTextLineChange(i, e.currentTarget.value)}
                      />
                      <button class="remove-btn" onclick={() => removeTextLine(i)}>✕</button>
                    </div>
                  {/each}
                  <button class="add-btn" onclick={addTextLine}>+ Add Line</button>
                </div>
              {:else if fieldKey === 'description' || fieldKey.includes('text')}
                <textarea
                  id={`content-${fieldKey}`}
                  class="field-input textarea"
                  value={section.content[fieldKey] || ''}
                  oninput={(e) => handleContentInput(fieldKey, e.currentTarget.value)}
                  rows="3"
                ></textarea>
              {:else if fieldKey.includes('IDs')}
                 <input
                  id={`content-${fieldKey}`}
                  type="text"
                  class="field-input"
                  value={(section.content[fieldKey] || []).join(', ')}
                  oninput={(e) => {
                    const ids = e.currentTarget.value.split(',').map(s => parseInt(s.trim())).filter(n => !isNaN(n));
                    handleContentInput(fieldKey, ids);
                  }}
                  placeholder="e.g. 1, 2, 3"
                />
              {:else}
                <input
                  id={`content-${fieldKey}`}
                  type="text"
                  class="field-input"
                  value={section.content[fieldKey] || ''}
                  oninput={(e) => handleContentInput(fieldKey, e.currentTarget.value)}
                />
              {/if}
            </div>
          {/each}
        </div>
      </div>

      <!-- STYLING GROUP -->
      <div class="editor-group">
        <h4 class="group-title">Styling (Tailwind)</h4>
        <div class="fields-list">
          {#each schema.css as slot}
            <div class="field-item">
              <label class="field-label" for={`css-${slot}`}>{slot} Classes</label>
              <textarea
                id={`css-${slot}`}
                class="field-input textarea css-textarea"
                value={section.css[slot] || ''}
                oninput={(e) => handleCssInput(slot, e.currentTarget.value)}
                rows="2"
                placeholder="Enter tailwind classes..."
              ></textarea>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .editor-tab {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .section-meta h3 {
    font-size: 18px;
    font-weight: 700;
    margin-bottom: 4px;
    color: white;
  }

  .section-meta p {
    font-size: 12px;
    color: #94a3b8;
    line-height: 1.4;
  }

  .editor-groups {
    display: flex;
    flex-direction: column;
    gap: 32px;
  }

  .group-title {
    font-size: 11px;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: #3b82f6;
    margin-bottom: 16px;
    padding-bottom: 8px;
    border-bottom: 1px solid #1e293b;
  }

  .fields-list {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .field-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .field-label {
    font-size: 12px;
    font-weight: 600;
    color: #cbd5e1;
    text-transform: capitalize;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #f1f5f9;
    font-size: 13px;
    transition: all 0.2s;
    box-sizing: border-box;
  }

  .field-input:focus {
    outline: none;
    border-color: #3b82f6;
    background: #0f172a;
  }

  .field-input.textarea {
    resize: vertical;
    min-height: 60px;
    line-height: 1.5;
  }

  .css-textarea {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 12px;
    color: #94a3b8;
  }

  .text-lines-editor {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .text-line-item {
    display: flex;
    gap: 8px;
  }

  .remove-btn {
    background: transparent;
    border: none;
    color: #ef4444;
    cursor: pointer;
    padding: 0 8px;
  }

  .add-btn {
    background: #1e293b;
    border: 1px dashed #334155;
    color: #94a3b8;
    padding: 8px;
    border-radius: 6px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .add-btn:hover {
    background: #334155;
    color: white;
  }
</style>