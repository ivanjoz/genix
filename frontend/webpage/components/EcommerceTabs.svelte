<script lang="ts">
  import { getContext, type Snippet } from 'svelte';
  import type { ComponentAST } from '$ecommerce/renderer/renderer-types';
  import { slideSync } from '$ecommerce/stores/slide-sync.svelte';
  import { EC_BUILDER_MODE } from '$ecommerce/renderer/builder-context';
  import OptionsStrip from '$components/navigation/OptionsStrip.svelte';

  interface Props {
    /**
     * The direct child AST nodes of the `<TabbedLayer>` tag — one tab panel each.
     * Injected by AstRenderer (not an HTML attribute); see `childNodes`/`renderChild`.
     */
    childNodes?: ComponentAST[];
    /** AstRenderer's node renderer, so a panel subtree renders without importing it. */
    renderChild?: Snippet<[ComponentAST]>;
    /** Pipe-separated tab labels, e.g. "My Style|Option 2|Option 3". */
    options?: string;
    css?: string;
    style?: string;
  }

  let {
    childNodes = [],
    renderChild,
    options = '',
    css = '',
    style = '',
  }: Props = $props();

  const panels = $derived(childNodes ?? []);
  const count = $derived(panels.length);

  // Authored tab labels; any panel without one falls back to "Tab N".
  const labels = $derived(options.split('|').map((s) => s.trim()).filter(Boolean));
  const tabOptions = $derived(
    panels.map((_, i) => ({ id: i, name: labels[i] ?? `Tab ${i + 1}` })),
  );

  // In the builder the active panel is shared with the editor (so picking a tab in the
  // editor's OptionsStrip selects it here, and vice-versa). Keyed by the panel array
  // identity — the same `childNodes` reference the editor holds as the node's children.
  // In production we keep a plain local index.
  const builderMode = getContext(EC_BUILDER_MODE) === true;

  let internalCurrent = $state(0);
  const current = $derived(builderMode ? slideSync.get(childNodes) : internalCurrent);

  function setCurrent(index: number) {
    if (builderMode) slideSync.set(childNodes, index);
    else internalCurrent = index;
  }

  // Keep the index valid if the panel set shrinks (e.g. while editing).
  $effect(() => {
    if (current > count - 1) setCurrent(Math.max(0, count - 1));
  });
</script>

<div class={["ec-tabs", css].join(" ")} {style}>
  {#if count > 1}
    <div class="ec-tabs-bar">
      <OptionsStrip
        options={tabOptions}
        selected={current}
        keyId="id"
        keyName="name"
        onSelect={(o) => setCurrent(o.id)}
      />
    </div>
  {/if}
  <div class="ec-tabs-panel">
    {#if panels[current] && renderChild}{@render renderChild(panels[current])}{/if}
  </div>
</div>

<style>
  .ec-tabs {
    width: 100%;
  }
  .ec-tabs-bar {
    display: flex;
    justify-content: center;
  }
  /* Only the selected panel is mounted, so it defines the layer's height; no track. */
  .ec-tabs-panel {
    position: relative;
    width: 100%;
  }
</style>
