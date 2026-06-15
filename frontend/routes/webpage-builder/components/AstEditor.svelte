<script lang="ts">
  import type { ComponentAST, ColorPalette } from '$ecommerce/renderer/renderer-types';
  import {
    isNoEdit, isTextRun, isTextLeaf, isImageNode, isLinkNode,
    childrenAsUnits, unitChildren, unitLabel, unitNoun, humanizeLabel,
  } from '../html-ast/editable';
  import { slideSync } from '$ecommerce/stores/slide-sync.svelte';
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
  import TextBlockEditor from './TextBlockEditor.svelte';
  import ImageBlockEditor from './ImageBlockEditor.svelte';

  // Recursive editor mirroring AstRenderer: it walks the SAME section AST and emits
  // editor controls instead of DOM. Editability is derived from node shape (text / image /
  // link / slide-container) — no `data-role` required; `data-role` only labels a field.
  interface Props {
    nodes: ComponentAST | ComponentAST[];
    palette?: ColorPalette;
  }
  let { nodes, palette }: Props = $props();

  const list = $derived(Array.isArray(nodes) ? nodes : [nodes]);

  function setNodeHref(node: ComponentAST, value: string) {
    if (!node.attributes) node.attributes = {};
    node.attributes.href = value;
  }

  // Strip options for a navigable container (Slider slides / TabbedLayer tabs): one per
  // direct child. `id` is the index; the active unit lives in slideSync keyed by the
  // children array (the same reference the live preview component holds).
  function unitOptions(node: ComponentAST, children: ComponentAST[]) {
    return children.map((_, i) => ({ id: i, name: unitLabel(node, i) }));
  }
</script>

{#snippet field(label: string, node: ComponentAST)}
  <div class="field-item">
    {#if isImageNode(node)}
      <ImageBlockEditor {node} {palette} {label} />
    {:else}
      <TextBlockEditor {node} {palette} {label} rows={node.role === 'content' ? 3 : 2} />
      {#if isLinkNode(node)}
        <input
          type="text"
          class="field-input link-input"
          value={node.attributes?.href || ''}
          oninput={(e) => setNodeHref(node, e.currentTarget.value)}
          placeholder="Link URL (href)"
        />
      {/if}
    {/if}
  </div>
{/snippet}

{#snippet renderNode(node: ComponentAST)}
  {#if isNoEdit(node)}
    <!-- opted out via data-noedit -->
  {:else if isTextRun(node)}
    <!-- standalone text run inside mixed content: text only, no styling toolbar -->
    <div class="field-item">
      <span class="field-label">Text</span>
      <TextBlockEditor {node} {palette} textOnly rows={2} />
    </div>
  {:else if isImageNode(node)}
    {@render field(humanizeLabel(node), node)}
  {:else if childrenAsUnits(node)}
    {@const units = unitChildren(node)}
    {@const active = slideSync.get(node.children)}
    <div class="slides-group">
      <span class="field-label">{humanizeLabel(node)} — {units.length} {unitNoun(node)}s</span>
      {#if units.length > 1}
        <OptionsStrip
          options={unitOptions(node, units)}
          selected={active}
          keyId="id"
          keyName="name"
          onSelect={(o) => slideSync.set(node.children, o.id)}
          containerCss="slides-strip"
        />
      {/if}
      {#if units[active]}
        {@render renderNode(units[active])}
      {/if}
    </div>
  {:else if isLinkNode(node)}
    <!-- anchor: edit its text (if any) + href, then any nested children -->
    {@render field(humanizeLabel(node), node)}
    {#if node.children}
      {#each node.children as child, i (i)}{@render renderNode(child)}{/each}
    {/if}
  {:else if isTextLeaf(node)}
    {@render field(humanizeLabel(node), node)}
  {:else if node.children}
    <!-- plain container: no input of its own, just recurse (flat) -->
    {#each node.children as child, i (i)}{@render renderNode(child)}{/each}
  {/if}
{/snippet}

{#each list as node, i (i)}
  {@render renderNode(node)}
{/each}

<style>
  .field-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-input {
    width: 100%;
    padding: 10px 12px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #f1f5f9;
    font-size: 13px;
    box-sizing: border-box;
  }
  .field-input:focus {
    outline: none;
    border-color: #3b82f6;
    background: #0f172a;
  }
  .link-input {
    font-size: 12px;
    color: #94a3b8;
  }

  /* A slide container: its strip + the selected slide's fields, stacked flat (no box). */
  .slides-group {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  /* OptionsStrip lives on a light store theme; restyle its buttons for the dark editor. */
  .slides-group :global(.slides-strip) {
    gap: 6px;
  }
  .slides-group :global(.slides-strip button) {
    min-width: 0;
    flex: 1;
    color: #94a3b8;
    border-bottom-color: rgba(148, 163, 184, 0.2);
  }
  .slides-group :global(.slides-strip button[data-selected='true']) {
    color: #60a5fa;
    border-bottom-color: #3b82f6;
  }
</style>
