<script lang="ts">
  import type { ComponentAST, ColorPalette } from '$ecommerce/renderer/renderer-types';

  import type { Snippet } from 'svelte';
  import { Env } from '$core/env';
  import { openModal, tr } from '$core/store.svelte';
  import ImagePickerModal from './ImagePickerModal.svelte';

  interface Props {
    /** The ImageEffect AST node being edited; its visual treatment lives on node.props. */
    node: ComponentAST;
    /** Active palette — its 10 colors back the tint swatches. */
    palette?: ColorPalette;
    /** Optional label shown inside the toolbar to save vertical space. */
    label?: string;
    /** Extra toolbar tools (rendered before the image tools) whose option panels are
        supplied by `extraOptions`. Lets a host merge its own controls into this one bar. */
    extraTools?: { id: string; label: string; icon: string }[];
    /** Renders the option panel for an extra tool; receives the open tool id. */
    extraOptions?: Snippet<[string | null]>;
  }

  let { node, palette, label, extraTools, extraOptions }: Props = $props();

  // Composition presets: labels stay textual while the SVG previews show direction.
  const LAYOUTS = [
    { value: '', label: 'None' },
    { value: 'fade-left', label: 'Fade left' },
    { value: 'fade-right', label: 'Fade right' },
    { value: 'fade-up', label: 'Fade up' },
    { value: 'fade-down', label: 'Fade down' },
    { value: 'slash-left', label: 'Slash left' },
    { value: 'slash-right', label: 'Slash right' },
    { value: 'curve-left', label: 'Curve left' },
    { value: 'curve-right', label: 'Curve right' },
    { value: 'curve-convex-left', label: 'Convex left' },
    { value: 'curve-convex-right', label: 'Convex right' },
  ];

  // Visual filters applied on top of the image.
  const EFFECTS = [
    { value: '', label: 'None' },
    { value: 'duotone', label: 'Duotone' },
    { value: 'duotone-matrix', label: 'Duotone+' },
    { value: 'glass', label: 'Glass' },
    { value: 'vignette', label: 'Vignette' },
    { value: 'overlay', label: 'Overlay' },
  ];

  // How the photo sits in its box (and whether it fills the parent as a background).
  const FITS = [
    { value: 'cover', label: 'Cover' },
    { value: 'contain', label: 'Contain' },
    { value: 'contain-left', label: 'Contain ◀' },
    { value: 'contain-right', label: 'Contain ▶' },
  ];

  // Toolbar tools. Each opens an options row absolutely positioned below the bar.
  const TOOLS = [
    { id: 'layout', label: 'Layout' },
    { id: 'effect', label: 'Effect' },
    { id: 'colors', label: 'Tint colors' },
    { id: 'adjust', label: 'Intensity & blur' },
    { id: 'background', label: 'Background & fit' },
  ];

  // Compact inline SVG icons (currentColor) — one glyph per tool.
  const ICONS: Record<string, string> = {
    layout:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="1.5" y="1.5" width="13" height="13" rx="1"/><path d="M9 1.5 C9 6 6 10 1.5 13"/></svg>',
    effect:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><circle cx="6" cy="6" r="4"/><circle cx="10" cy="10" r="4"/></svg>',
    colors:
      '<svg viewBox="0 0 16 16" width="15" height="15"><rect x="1.5" y="1.5" width="7" height="13" rx="1" fill="currentColor"/><rect x="8.5" y="1.5" width="6" height="13" rx="1" fill="#64748b"/></svg>',
    adjust:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><line x1="2" y1="5" x2="14" y2="5"/><circle cx="6" cy="5" r="1.8" fill="currentColor"/><line x1="2" y1="11" x2="14" y2="11"/><circle cx="10" cy="11" r="1.8" fill="currentColor"/></svg>',
    background:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="1.5" y="1.5" width="13" height="13" rx="1"/><path d="M1.5 11 5 7.5 8 10.5 11 7 14.5 10.5" /></svg>',
  };

  // Use one real image so each thumbnail isolates the selected treatment.
  const PREVIEW_IMAGE =
    '<image href="/images/image_placeholder.webp" width="96" height="48" preserveAspectRatio="xMidYMid slice"/>';

  // Thumbnail artwork mirrors the gradients, clipping paths, and overlays in ImageEffect.
  const PREVIEW_SVGS: Record<string, string> = {
    none:
      `<svg viewBox="0 0 96 48" aria-hidden="true">${PREVIEW_IMAGE}</svg>`,
    'fade-left':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><linearGradient id="ibe-fade-left"><stop stop-color="#fff"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient></defs>${PREVIEW_IMAGE}<rect width="96" height="48" fill="url(#ibe-fade-left)"/></svg>`,
    'fade-right':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><linearGradient id="ibe-fade-right" x1="1" x2="0"><stop stop-color="#fff"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient></defs>${PREVIEW_IMAGE}<rect width="96" height="48" fill="url(#ibe-fade-right)"/></svg>`,
    'fade-up':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><linearGradient id="ibe-fade-up" x1="0" y1="1" x2="0" y2="0"><stop stop-color="#fff"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient></defs>${PREVIEW_IMAGE}<rect width="96" height="48" fill="url(#ibe-fade-up)"/></svg>`,
    'fade-down':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><linearGradient id="ibe-fade-down" x1="0" x2="0" y2="1"><stop stop-color="#fff"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient></defs>${PREVIEW_IMAGE}<rect width="96" height="48" fill="url(#ibe-fade-down)"/></svg>`,
    'slash-left':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><clipPath id="ibe-slash-left"><path d="M0 0H62L43 48H0Z"/></clipPath></defs><rect width="96" height="48" fill="#e2e8f0"/><g clip-path="url(#ibe-slash-left)">${PREVIEW_IMAGE}</g></svg>`,
    'slash-right':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><clipPath id="ibe-slash-right"><path d="M34 0H96V48H53Z"/></clipPath></defs><rect width="96" height="48" fill="#e2e8f0"/><g clip-path="url(#ibe-slash-right)">${PREVIEW_IMAGE}</g></svg>`,
    'curve-left':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><clipPath id="ibe-curve-left"><path d="M0 0H34C34 19 53 35 82 48H0Z"/></clipPath></defs><rect width="96" height="48" fill="#e2e8f0"/><g clip-path="url(#ibe-curve-left)">${PREVIEW_IMAGE}</g></svg>`,
    'curve-right':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><clipPath id="ibe-curve-right"><path d="M96 0H62C62 19 43 35 14 48H96Z"/></clipPath></defs><rect width="96" height="48" fill="#e2e8f0"/><g clip-path="url(#ibe-curve-right)">${PREVIEW_IMAGE}</g></svg>`,
    'curve-convex-left':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><clipPath id="ibe-convex-left"><path d="M0 0H34C53 14 62 34 58 48H0Z"/></clipPath></defs><rect width="96" height="48" fill="#e2e8f0"/><g clip-path="url(#ibe-convex-left)">${PREVIEW_IMAGE}</g></svg>`,
    'curve-convex-right':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><clipPath id="ibe-convex-right"><path d="M96 0H62C43 14 34 34 38 48H96Z"/></clipPath></defs><rect width="96" height="48" fill="#e2e8f0"/><g clip-path="url(#ibe-convex-right)">${PREVIEW_IMAGE}</g></svg>`,
    duotone:
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><filter id="ibe-duotone"><feColorMatrix type="saturate" values="0"/></filter></defs><g filter="url(#ibe-duotone)">${PREVIEW_IMAGE}</g><rect width="96" height="48" fill="#60a5fa" opacity=".65" style="mix-blend-mode:screen"/><rect width="96" height="48" fill="#172554" opacity=".8" style="mix-blend-mode:multiply"/></svg>`,
    'duotone-matrix':
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><filter id="ibe-duotone-matrix"><feColorMatrix type="matrix" values=".21 .72 .07 0 0 .21 .72 .07 0 0 .21 .72 .07 0 0 0 0 0 1 0"/><feComponentTransfer><feFuncR type="table" tableValues=".2 .95"/><feFuncG type="table" tableValues=".05 .35"/><feFuncB type="table" tableValues=".35 1"/></feComponentTransfer></filter></defs><g filter="url(#ibe-duotone-matrix)">${PREVIEW_IMAGE}</g></svg>`,
    glass:
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><filter id="ibe-glass"><feGaussianBlur stdDeviation="2"/></filter></defs>${PREVIEW_IMAGE}<g filter="url(#ibe-glass)" opacity=".8">${PREVIEW_IMAGE}</g><rect width="96" height="48" fill="#fff" opacity=".2"/></svg>`,
    vignette:
      `<svg viewBox="0 0 96 48" aria-hidden="true"><defs><radialGradient id="ibe-vignette"><stop offset=".35" stop-color="#000" stop-opacity="0"/><stop offset="1" stop-color="#000" stop-opacity=".9"/></radialGradient></defs>${PREVIEW_IMAGE}<rect width="96" height="48" fill="url(#ibe-vignette)"/></svg>`,
    overlay:
      `<svg viewBox="0 0 96 48" aria-hidden="true">${PREVIEW_IMAGE}<rect width="96" height="48" fill="#2563eb" opacity=".5"/></svg>`,
  };

  /** Empty values use the unmodified image thumbnail. */
  function previewSvg(value: string): string {
    return PREVIEW_SVGS[value || 'none'];
  }

  // Which tool's options row is open (null = collapsed). Toggled by the toolbar.
  let openTool = $state<string | null>(null);

  // Root element of this editor — used to detect clicks outside the popup.
  let root = $state<HTMLDivElement>();

  // Close the options popup when clicking anywhere outside this editor.
  $effect(() => {
    if (!openTool) return;
    const onPointerDown = (e: PointerEvent) => {
      if (root && !root.contains(e.target as Node)) openTool = null;
    };
    window.addEventListener('pointerdown', onPointerDown, true);
    return () => window.removeEventListener('pointerdown', onPointerDown, true);
  });

  // --- Prop helpers (visual treatment lives on node.props; mutating it re-renders) ---

  /** Read a prop, falling back to the value the ImageEffect component itself defaults to. */
  function prop<T>(key: string, fallback: T): T {
    const v = node.props?.[key];
    return (v === undefined || v === null ? fallback : v) as T;
  }

  function setProp(key: string, value: unknown) {
    const props = node.props ?? (node.props = {});
    props[key] = value;
  }

  // ImageEffect's own defaults (mirrors ImageEffect.svelte) so sliders/swatches init right.
  const currentLayout = $derived(prop('layout', ''));
  const currentEffect = $derived(prop('effect', ''));
  const tint = $derived(prop('color', '#ffffff'));
  const tint2 = $derived(prop('color2', '#000000'));
  const intensity = $derived(prop('intensity', 1));
  const blur = $derived(prop('blur', 0));
  const fill = $derived(prop('fill', false));
  const fit = $derived(prop('fit', 'cover'));

  // Current image source; drives the preview and toggles the remove/add overlay.
  const src = $derived(prop('src', ''));
  // Unique numeric id so each editor instance opens its own picker modal.
  const pickerModalId = Env.getComponentID();

  // Host-provided tools come first, then this editor's image tools. Each carries its
  // own icon html so the toolbar can render one unified row.
  const allTools = $derived([
    ...(extraTools ?? []),
    ...TOOLS.map((t) => ({ ...t, icon: ICONS[t.id] })),
  ]);
  // True when the open tool belongs to the host (its panel is rendered via extraOptions).
  const isExtraTool = $derived(!!openTool && (extraTools ?? []).some((t) => t.id === openTool));
</script>

<div class="ibe" bind:this={root}>
  <div class="toolbar">
    {#if label}<span class="toolbar-label">{label}</span>{/if}
    {#each allTools as tool}
      <button
        type="button"
        class="tool-btn"
        class:active={openTool === tool.id}
        title={tool.label}
        aria-label={tool.label}
        onclick={() => (openTool = openTool === tool.id ? null : tool.id)}
      >
        {@html tool.icon}
      </button>
    {/each}

    {#if openTool}
      <div class="options">
        {#if openTool === 'layout' || openTool === 'effect'}
          {@const opts = openTool === 'layout' ? LAYOUTS : EFFECTS}
          {@const current = openTool === 'layout' ? currentLayout : currentEffect}
          {@const key = openTool === 'layout' ? 'layout' : 'effect'}
          <div class="visual-options">
            {#each opts as opt}
              <button
                type="button"
                class="visual-option"
                class:sel={current === opt.value}
                onclick={() => setProp(key, opt.value)}
              >
                <span>{opt.label}</span>
                <span class="effect-preview">{@html previewSvg(opt.value)}</span>
              </button>
            {/each}
          </div>
        {:else if openTool === 'colors'}
          {#each [{ key: 'color', current: tint, label: 'Primary tint' }, { key: 'color2', current: tint2, label: 'Secondary tint' }] as row}
            <div class="color-row">
              <span class="row-label">{row.label}</span>
              <div class="swatches">
                {#each palette?.colors ?? [] as hex, i}
                  <button
                    type="button"
                    class="swatch"
                    class:sel={row.current?.toLowerCase() === hex.toLowerCase()}
                    style={`background:${hex}`}
                    title={`Color ${i + 1}`}
                    aria-label={`Color ${i + 1}`}
                    onclick={() => setProp(row.key, hex)}
                  ></button>
                {/each}
              </div>
            </div>
          {/each}
        {:else if openTool === 'background'}
          <!-- Fill the parent as a background, plus object-fit control. -->
          <label class="toggle-row">
            <input type="checkbox" checked={!!fill} onchange={(e) => setProp('fill', e.currentTarget.checked)} />
            <span>Use as section background (fill parent)</span>
          </label>
          <div class="chips">
            {#each FITS as opt}
              <button
                type="button"
                class="chip"
                class:sel={fit === opt.value}
                onclick={() => setProp('fit', opt.value)}
              >{opt.label}</button>
            {/each}
          </div>
        {:else if isExtraTool}
          <!-- Option panel for a host-provided tool. -->
          {@render extraOptions?.(openTool)}
        {:else if openTool === 'adjust'}
          <!-- adjust: intensity (0..1) and blur (px) -->
          <label class="slider-row">
            <span class="slider-label">Intensity</span>
            <input
              type="range" min="0" max="1" step="0.05" value={intensity}
              oninput={(e) => setProp('intensity', +e.currentTarget.value)}
            />
            <span class="slider-val">{intensity}</span>
          </label>
          <label class="slider-row">
            <span class="slider-label">Blur</span>
            <input
              type="range" min="0" max="20" step="1" value={blur}
              oninput={(e) => setProp('blur', +e.currentTarget.value)}
            />
            <span class="slider-val">{blur}px</span>
          </label>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Preview strip: the source image rendered raw so edits read against the real photo. -->
  <div class="preview" style={`background-image:url('${src}')`}>
    {#if src}
      <!-- Hover reveals a circular remove button centered at the bottom. -->
      <button
        type="button"
        class="preview-remove"
        title={tr('Remove image|Quitar imagen')}
        aria-label={tr('Remove image|Quitar imagen')}
        onclick={() => setProp('src', '')}
      >
        <i class="icon-[fa--close]"></i>
      </button>
    {:else}
      <!-- Empty: hover reveals an "Agregar" button that opens the image picker. -->
      <button type="button" class="preview-add" onclick={() => openModal(pickerModalId)}>
        <i class="icon-[fa--plus]"></i><span>{tr('Add|Agregar')}</span>
      </button>
    {/if}
  </div>
</div>

<!-- Picker modal: selecting an image writes its full-resolution url back to node.props.src. -->
<ImagePickerModal modalId={pickerModalId} onSelect={(url) => setProp('src', url)} />

<style>
  .ibe {
    display: flex;
    flex-direction: column;
    border: 1px solid #334155;
    border-radius: 6px;
    transition: border-color 0.15s;
  }

  .preview {
    position: relative;
    height: 64px;
    background-size: cover;
    background-position: center;
    background-color: #0b1120;
    border-radius: 0 0 6px 6px;
  }

  /* Circular remove button, centered at the bottom edge, revealed on hover. */
  .preview-remove {
    position: absolute;
    bottom: 6px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    border: none;
    border-radius: 50%;
    background: #e75c5c;
    color: #fff;
    font-size: 13px;
    cursor: pointer;
    opacity: 0;
    box-shadow: 0 4px 10px rgba(0, 0, 0, 0.55);
    transition: opacity 0.15s, background 0.15s;
  }
  .preview:hover .preview-remove {
    opacity: 1;
  }
  .preview-remove:hover {
    background: #f77d7d;
  }

  /* "Agregar" button shown over the empty placeholder on hover. */
  .preview-add {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border: none;
    background: rgba(15, 23, 42, 0.55);
    color: #e2e8f0;
    font-size: 12px;
    font-weight: 700;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s;
  }
  .preview:hover .preview-add {
    opacity: 1;
  }
  .preview-add:hover {
    color: #fff;
    background: rgba(15, 23, 42, 0.7);
  }

  .toolbar {
    position: relative;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 0;
    background: transparent;
    border-bottom: 1px solid #1e293b;
  }

  .toolbar-label {
    width: 68px;
    flex-shrink: 0;
    font-size: 10px;
    font-weight: 700;
    color: #e2e8f0;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    padding-left: 8px;
  }

  /* While the popup is open, fade the host's lines so they don't compete
     with the popup's accent border (matches TextBlockEditor). */
  .ibe:has(.options),
  .ibe:has(.options):focus-within {
    border-color: #1e293b;
  }
  .ibe:has(.options) .toolbar {
    border-bottom-color: #1e293b;
  }

  .tool-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 28px;
    background: transparent;
    border: none;
    border-radius: 5px;
    color: #cbd5e1;
    cursor: pointer;
    transition: all 0.15s;
  }
  .tool-btn:hover {
    background: #334155;
    color: #f1f5f9;
  }
  .tool-btn.active {
    background: #3b82f6;
    color: #fff;
  }

  /* Second row: floats below the toolbar without pushing the input. */
  .options {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    right: 0;
    z-index: 10;
    padding: 12px;
    background: #0b1120;
    border: 1px solid #3b82f6;
    border-radius: 8px;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.6);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .visual-options {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
    max-height: 300px;
    overflow-y: auto;
  }
  .visual-option {
    min-width: 0;
    padding: 0;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #cbd5e1;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s;
  }
  .visual-option > span:first-child {
    display: block;
    padding: 5px 2px;
  }
  .visual-option:hover {
    background: #334155;
    color: #f1f5f9;
  }
  .visual-option.sel {
    background: #1d4ed8;
    border-color: #60a5fa;
    color: #fff;
  }
  .effect-preview {
    display: block;
    overflow: hidden;
    border-radius: 0 0 5px 5px;
    line-height: 0;
  }
  .effect-preview :global(svg) {
    display: block;
    width: 100%;
    height: 42px;
  }
  .chip {
    padding: 4px 9px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #cbd5e1;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s;
  }
  .chip:hover {
    background: #334155;
    color: #f1f5f9;
  }
  .chip.sel {
    background: #3b82f6;
    border-color: #3b82f6;
    color: #fff;
  }

  .color-row {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }
  .row-label {
    font-size: 11px;
    color: #94a3b8;
  }
  .swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .swatch {
    width: 24px;
    height: 24px;
    border: 1px solid #475569;
    border-radius: 5px;
    cursor: pointer;
    padding: 0;
  }
  .swatch.sel {
    outline: 2px solid #3b82f6;
    outline-offset: 1px;
    border-color: #3b82f6;
  }

  .toggle-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #cbd5e1;
    cursor: pointer;
  }
  .toggle-row input {
    accent-color: #3b82f6;
  }

  .slider-row {
    display: grid;
    grid-template-columns: 60px 1fr 40px;
    align-items: center;
    gap: 8px;
  }
  .slider-label {
    font-size: 11px;
    color: #94a3b8;
  }
  .slider-val {
    font-size: 11px;
    color: #cbd5e1;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .slider-row input[type='range'] {
    width: 100%;
    accent-color: #3b82f6;
  }

  .ibe:focus-within {
    border-color: #3b82f6;
  }
</style>
