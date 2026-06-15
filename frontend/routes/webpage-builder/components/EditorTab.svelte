<script lang="ts">
import { editorStore } from '../stores/editor.svelte';
import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
import { collectEditableNodes, editorPropNames, editorPropLabel, getNodeProp, setNodeProp } from '../html-ast/editable';
import { getProductEcommerceData, type ProductCatalog } from '$ecommerce/services/productos.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import AstEditor from './AstEditor.svelte';
import SectionStyleEditor from './SectionStyleEditor.svelte';
    import T from '$components/misc/T.svelte';

  // Active palette drives the color swatches inside TextBlockEditor.
  interface Props { palette?: ColorPalette }
  let { palette }: Props = $props();

  const section = $derived(editorStore.selectedSection);
  const schema = $derived(editorStore.activeSchema);

  // Nodes whose schema declares a category-bound prop (ProductsByCategory / ProductGrid / CategoryDescription).
  const categoryNodes = $derived(
    section?.Type === 'HtmlSection' ? collectEditableNodes(section.Ast, 'category') : []
  );
  // Each node names its own category attribute (categoryID vs categoryIDs); the value is taken from the
  // first bound node and applied to all of them, so the description and grids share one category.
  const selectedCategoryID = $derived(
    categoryNodes.length ? getNodeProp(categoryNodes[0], editorPropNames(categoryNodes[0], 'category')[0]) : undefined
  );

  // Load the catalog once so the selector has the full category list to pick from.
  let catalog = $state<ProductCatalog | null>(null);
  $effect(() => {
    if (categoryNodes.length && !catalog) {
      getProductEcommerceData().then((loaded) => { catalog = loaded; });
    }
  });
  const categorias = $derived(catalog?.categorias ?? []);

  function handleCategoryChange(id: number) {
    for (const node of categoryNodes) setNodeProp(node, editorPropNames(node, 'category')[0], id);
  }

  // Nodes exposing numeric grid-layout knobs (maxWidth, rows...); the prop list comes from the schema.
  const gridNodes = $derived(
    section?.Type === 'HtmlSection' ? collectEditableNodes(section.Ast, 'grid') : []
  );
  const gridProps = $derived(gridNodes.length ? editorPropNames(gridNodes[0], 'grid') : []);
  // Layout is read from the first grid and applied to all of them, so multiple grids stay in sync.
  function handleGridPropInput(prop: string, raw: string) {
    const value = raw.trim() === '' ? undefined : Number(raw);
    for (const node of gridNodes) setNodeProp(node, prop, value);
  }

  function handleCssInput(slot: string, value: string) {
    if (section) {
      editorStore.updateCss(section.id, slot, value);
    }
  }

</script>

{#if section && schema}
  <div class="editor-tab" aria-label="Section content and styling editor">
    <div class="editor-groups">
      <!-- Section-level styling: padding, colors, and background image. -->
      <div class="editor-group">
        <h4 class="group-title"><T text="Section style|Estilo de sección" /></h4>
        <SectionStyleEditor {palette} />
      </div>
      <!-- Category selector for templates with category-bound components. -->
      {#if categoryNodes.length}
        <div class="editor-group">
          <h4 class="group-title"><T text="Category|Categoría" /> </h4>
          <div class="fields-list">
            <div class="field-item">
              <SearchSelect
                keyId="ID"
                keyName="Name"
                options={categorias}
                selected={selectedCategoryID}
                onChange={(e) => e && handleCategoryChange(e.ID)}
                noStyle
                css="w-full bg-[#0000003b] rounded-[8px] border border-[#99a4ce5c]"
                inputCss="text-sm px-10 text-[#f1f5f9]"
                optionsCss="w-full text-sm text-[#1e293b]"
              />
            </div>
          </div>
        </div>
      {/if}
      <!-- Layout controls for product grids (ProductsByCategory) present in the section. -->
      {#if gridNodes.length}
        <div class="editor-group">
          <h4 class="group-title"><T text="Grid layout|Diseño de grilla" /></h4>
          <div class="fields-list grid-fields">
            {#each gridProps as prop}
              <div class="field-item">
                <label class="field-label" for={`grid-${prop}`}>
                  <T text={editorPropLabel(gridNodes[0], prop)} />
                </label>
                <input
                  id={`grid-${prop}`}
                  type="number"
                  class="field-input"
                  value={getNodeProp(gridNodes[0], prop) ?? ''}
                  oninput={(e) => handleGridPropInput(prop, e.currentTarget.value)}
                  placeholder="auto"
                />
              </div>
            {/each}
          </div>
        </div>
      {/if}
      <!-- Every text, image, and link node in the AST is editable. -->
      <div class="editor-group">
        <h4 class="group-title">Content</h4>
        <div class="fields-list">
          {#if section.Ast && section.Ast.length}
            <AstEditor nodes={section.Ast} {palette} />
          {:else}
            <p class="empty-hint">No content. This section has no parsed HTML.</p>
          {/if}
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
                value={section.Css?.[slot] || ''}
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

  /* Compact numeric grid knobs into two columns to keep the panel short. */
  .grid-fields {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  /* Each knob lays its label on the left and the input on the right (55% of the row);
     the label takes the remaining space and wraps when its text is too long. */
  .grid-fields .field-item {
    flex-direction: row;
    align-items: center;
    gap: 8px;
  }
  .grid-fields .field-item .field-label {
    flex: 1;
    min-width: 0;
  }
  .grid-fields .field-item .field-input {
    flex: 0 0 55%;
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

  .empty-hint {
    font-size: 12px;
    color: #94a3b8;
    line-height: 1.5;
  }
</style>
