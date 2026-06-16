<script lang="ts">
  import type { ComponentAST, ColorPalette } from '$ecommerce/renderer/renderer-types';
  import {
    isTextRun, isImageNode, isLinkNode,
    childrenAsUnits, unitChildren, unitLabel, unitNoun, humanizeLabel,
    groupSiblings,
  } from '../html-ast/editable';
  import { slideSync } from '$ecommerce/stores/slide-sync.svelte';
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte';
  import TextBlockEditor from './TextBlockEditor.svelte';
  import ImageBlockEditor from './ImageBlockEditor.svelte';

  // AstEditor — the editing twin of AstRenderer.
  //
  // GOAL: walk the SAME reactive section AST that AstRenderer paints to the live preview,
  // but emit editor controls instead of DOM. Because both sides share the one AST, every
  // edit a control makes shows up in the preview instantly — there is no separate "model".
  //
  // HOW: per AST level, `groupSiblings(siblings)` partitions the children into items:
  //   - { kind: 'lines' } — a run of mergeable text lines (e.g. <h2>+<p>) OR a single
  //     standalone line (<li>); handed wholesale to ONE TextBlockEditor (which owns its
  //     own line/fragment editing). This is what collapses many old per-leaf editors into
  //     a few grouped ones.
  //   - { kind: 'node' } — anything else (image / link / slider-or-tab container / mixed
  //     text run / plain container), rendered by `specialNode`. Plain containers recurse
  //     via `renderSiblings`, so grouping happens independently at each nesting level.
  // `siblings` (the parent's children array) is passed down so TextBlockEditor can splice
  // new/removed line nodes into the real, reactive array — never a copy.
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

<!-- One special (non-grouped) node: image / link / navigable container / text run /
     plain container. Text leaves never reach here — they are grouped into a TextBlockEditor. -->
{#snippet specialNode(node: ComponentAST, siblings: ComponentAST[], container: ComponentAST | undefined)}
  {#if isImageNode(node)}
    <div class="field-item">
      <ImageBlockEditor {node} {palette} label={humanizeLabel(node)} />
    </div>
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
        {@render renderSiblings([units[active]], node)}
      {/if}
    </div>
  {:else if isLinkNode(node)}
    <!-- anchor: edit its text as a single-line group, plus its href -->
    <div class="field-item">
      <TextBlockEditor lines={[node]} {siblings} {container} {palette} />
      <input
        type="text"
        class="field-input link-input"
        value={node.attributes?.href || ''}
        oninput={(e) => setNodeHref(node, e.currentTarget.value)}
        placeholder="Link URL (href)"
      />
    </div>
  {:else if isTextRun(node)}
    <!-- standalone mixed-content text run: plain text, no styling -->
    <div class="field-item">
      <span class="field-label">Text</span>
      <textarea
        class="field-input"
        rows="2"
        value={node.text || ''}
        oninput={(e) => (node.text = e.currentTarget.value)}
      ></textarea>
    </div>
  {:else if node.children}
    <!-- plain container: no input of its own, recurse into its children (it IS their container) -->
    {@render renderSiblings(node.children, node)}
  {/if}
{/snippet}

<!-- Render a sibling list: group runs of text leaves / container lines into TextBlockEditors,
     and render everything else via specialNode. -->
{#snippet renderSiblings(siblings: ComponentAST[], container: ComponentAST | undefined)}
  {#each groupSiblings(siblings) as item, i (i)}
    {#if item.kind === 'lines'}
      <TextBlockEditor lines={item.lines} {siblings} {container} {palette} />
    {:else}
      {@render specialNode(item.node, siblings, container)}
    {/if}
  {/each}
{/snippet}

{@render renderSiblings(list, undefined)}

<style>
  .field-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
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
