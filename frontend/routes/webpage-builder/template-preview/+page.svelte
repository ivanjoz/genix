<script lang="ts">
  // Headless-review route: /webpage-builder/template-preview?id=<template-id>.
  // Renders a single section template both as its LIVE component (HTML -> AST ->
  // HtmlSection, the real thing a page would render) and as its authoring
  // THUMBNAIL, on a clean chrome-less page. A coding agent drives a headless
  // Chrome here to screenshot a template, eyeball the result, then iterate on the
  // template source. Stable selectors (#template-live / #template-thumbnail) and a
  // data-render-state hook let the agent wait for and target each artifact.
  import { page } from '$app/state';
  import { sectionTemplates } from '../templates';
  import { parseHTML } from '$ecommerce/html-ast/parse-html';
  import { generatePaletteStyles } from '$ecommerce/renderer/token-resolver';
  import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
  import HtmlSection from '$ecommerce/renderer/HtmlSection.svelte';

  // The template id from the query string (?id=html-hero-banner-v1). Defensive: strip
  // surrounding quotes so both ?id=foo and ?id="foo" resolve to the same template.
  const requestedId = $derived((page.url.searchParams.get('id') ?? '').replace(/^["']|["']$/g, ''));

  // Resolve the template; null when the id is missing or unknown (rendered as an error).
  const template = $derived(sectionTemplates.find(t => t.id === requestedId) ?? null);

  // The live model: parse the template HTML to the canonical AST exactly as the editor
  // does at add time (see stores/editor.svelte addSection), so this preview matches the
  // builder/storefront output rather than re-implementing rendering.
  const liveAst = $derived(template ? parseHTML(template.html ?? '') : []);

  // The builder's neutral default palette (mirrors [pageID]/+page.svelte). Templates use
  // color="N" / background-color="N" attributes that resolve against these CSS variables.
  const defaultPalette: ColorPalette = {
    id: 'default',
    name: 'Default Palette',
    colors: [
      '#f8fafc', '#f1f5f9', '#e2e8f0', '#cbd5e1', '#94a3b8',
      '#64748b', '#475569', '#334155', '#1e293b', '#0f172a',
    ],
  };
  const paletteStyles = $derived(generatePaletteStyles(defaultPalette));
</script>

<svelte:head>
  <title>Template preview · {template?.name ?? requestedId ?? '—'}</title>
</svelte:head>

<!-- data-render-state flips to "ready"/"missing" once resolved so a headless agent can
     poll for it before screenshotting instead of guessing at a timeout. -->
<main class="preview-root" data-template-id={requestedId} data-render-state={template ? 'ready' : 'missing'}>
  {#if !requestedId}
    <p class="preview-message">Pass a template id, e.g. <code>?id=html-hero-banner-v1</code></p>
  {:else if !template}
    <p class="preview-message">
      No template with id <code>{requestedId}</code>. Available:
      <code>{sectionTemplates.map(t => t.id).join(', ')}</code>
    </p>
  {:else}
    <header class="preview-header">
      <h1 class="preview-title">{template.name}</h1>
      <p class="preview-id"><code>{template.id}</code> · {template.category ?? 'uncategorized'}</p>
      {#if template.description}
        <p class="preview-desc">{template.description}</p>
      {/if}
    </header>

    <!-- LIVE: the real rendered component, full-width as it would appear on a page. -->
    <section class="preview-block">
      <h2 class="preview-block-title">Live component</h2>
      <div id="template-live" class="preview-live" style={paletteStyles}>
        <HtmlSection ast={liveAst} css={{}} />
      </div>
    </section>

    <!-- THUMBNAIL: the authoring preview markup shown in the builder's templates grid. -->
    <section class="preview-block">
      <h2 class="preview-block-title">Thumbnail</h2>
      <div id="template-thumbnail" class="preview-thumbnail">
        {@html template.thumbnail ?? '<em>no thumbnail</em>'}
      </div>
    </section>
  {/if}
</main>

<style>
  .preview-root {
    min-height: 100vh;
    background: #f8fafc;
    color: #0f172a;
    font-family: ui-sans-serif, system-ui, sans-serif;
  }
  .preview-header {
    padding: 24px;
    border-bottom: 1px solid #e2e8f0;
  }
  .preview-title { font-size: 22px; font-weight: 700; }
  .preview-id { margin-top: 4px; color: #64748b; font-size: 13px; }
  .preview-desc { margin-top: 8px; color: #475569; font-size: 14px; max-width: 720px; }
  .preview-block { padding: 24px; }
  .preview-block-title {
    margin-bottom: 12px;
    color: #94a3b8;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  /* The live block gets no inner styling — the template owns its full appearance. */
  .preview-live {
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    overflow: hidden;
    background: #fff;
  }
  /* Constrain the thumbnail to the builder card's footprint so it matches the grid. */
  .preview-thumbnail {
    width: 220px;
    height: 100px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    overflow: hidden;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: #fff;
  }
  .preview-message { padding: 24px; color: #475569; font-size: 14px; }
  code { background: #e2e8f0; border-radius: 4px; padding: 1px 5px; font-size: 0.92em; }
</style>
