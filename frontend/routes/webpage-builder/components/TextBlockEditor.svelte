<script lang="ts">
  import type { ComponentAST, ColorPalette } from '$ecommerce/renderer/renderer-types';

  interface Props {
    /** The AST text node being edited; styling is applied to the whole node. */
    node: ComponentAST;
    /** Active palette — its 10 colors back the color/background swatches. */
    palette?: ColorPalette;
    rows?: number;
    /** Hide the styling toolbar — for raw text runs (#text nodes) that carry no css/style. */
    textOnly?: boolean;
  }

  let { node, palette, rows = 3, textOnly = false }: Props = $props();

  // Discrete spacing steps (px). `--spacing` is 1px in this project, so the px
  // value IS the Tailwind number (pt-8 = 8px). 2px steps up to 16, then 4px.
  const STEPS = [0, 2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32];

  // Toolbar tools. Each opens an options row absolutely positioned below the bar.
  const TOOLS = [
    { id: 'padding', label: 'Padding' },
    { id: 'margin', label: 'Margin' },
    { id: 'fontSize', label: 'Font size' },
    { id: 'color', label: 'Text color' },
    { id: 'bgColor', label: 'Background color' },
  ];

  // Compact inline SVG icons (currentColor) — one glyph per tool.
  const ICONS: Record<string, string> = {
    padding:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="1.5" y="1.5" width="13" height="13" rx="1"/><rect x="4.5" y="4.5" width="7" height="7" rx="1"/></svg>',
    margin:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="1" y="1" width="14" height="14" rx="1" stroke-dasharray="2 2"/><rect x="4.5" y="4.5" width="7" height="7" rx="1"/></svg>',
    fontSize:
      '<svg viewBox="0 0 16 16" width="15" height="15"><text x="2" y="13" font-size="14" font-weight="700" fill="currentColor">A</text></svg>',
    color:
      '<svg viewBox="0 0 16 16" width="15" height="15"><text x="1.5" y="11" font-size="11" font-weight="700" fill="currentColor">A</text><rect x="1.5" y="13" width="11" height="2.5" fill="#3b82f6"/></svg>',
    bgColor:
      '<svg viewBox="0 0 16 16" width="15" height="15"><rect x="2" y="2" width="12" height="12" rx="2" fill="currentColor"/></svg>',
  };

  const SIDES = [
    { key: 't', label: 'Top' },
    { key: 'r', label: 'Right' },
    { key: 'b', label: 'Bottom' },
    { key: 'l', label: 'Left' },
  ] as const;

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

  // --- Tailwind class helpers (padding / margin / font-size live on node.css) ---

  const tokens = () => (node.css || '').split(/\s+/).filter(Boolean);

  /** Effective px for one side, honoring shorthand (p / px / py) so sliders init right. */
  function sidePx(prefix: string, side: string): number {
    const axis = side === 'l' || side === 'r' ? 'x' : 'y';
    const list = tokens();
    const num = (p: string) => {
      const found = list.find((x) => x.startsWith(p + '-'));
      const n = found ? parseInt(found.slice(p.length + 1), 10) : NaN;
      return Number.isNaN(n) ? undefined : n;
    };
    return num(prefix + side) ?? num(prefix + axis) ?? num(prefix) ?? 0;
  }

  /** Rebuild all four sides of a spacing family, dropping any prior shorthand/axis tokens. */
  function setSide(prefix: string, side: string, px: number) {
    const vals: Record<string, number> = {
      t: sidePx(prefix, 't'), r: sidePx(prefix, 'r'),
      b: sidePx(prefix, 'b'), l: sidePx(prefix, 'l'),
    };
    vals[side] = px;
    const families = [prefix, prefix + 'x', prefix + 'y', prefix + 't', prefix + 'r', prefix + 'b', prefix + 'l'];
    const kept = tokens().filter((x) => !families.some((f) => x.startsWith(f + '-')));
    for (const s of ['t', 'r', 'b', 'l']) if (vals[s] > 0) kept.push(`${prefix}${s}-${vals[s]}`);
    node.css = kept.join(' ');
  }

  /** Nearest STEPS index for a px value (slider position). */
  function stepIndex(px: number): number {
    let best = 0;
    for (let i = 1; i < STEPS.length; i++) if (Math.abs(STEPS[i] - px) < Math.abs(STEPS[best] - px)) best = i;
    return best;
  }

  const FONT_RE = /^text-\[\d+px\]$/;
  const fontPx = () => {
    const t = tokens().find((x) => FONT_RE.test(x));
    return t ? parseInt(t.slice(6, -3), 10) : 0; // text-[18px] -> 18; 0 = unset
  };
  function setFontPx(px: number) {
    const kept = tokens().filter((x) => !FONT_RE.test(x));
    if (px > 0) kept.push(`text-[${px}px]`);
    node.css = kept.join(' ');
  }

  // --- Color helpers (palette tokens live on node.style as var(--color-N)) ---

  function styleDecls(): Record<string, string> {
    const out: Record<string, string> = {};
    for (const d of (node.style || '').split(';')) {
      const [k, v] = d.split(':');
      if (k && v) out[k.trim()] = v.trim();
    }
    return out;
  }
  function colorIndex(prop: string): number | null {
    const m = styleDecls()[prop]?.match(/^var\(--color-(\d+)\)$/);
    return m ? parseInt(m[1], 10) : null;
  }
  function setColor(prop: string, idx: number | null) {
    const decls = styleDecls();
    if (idx) decls[prop] = `var(--color-${idx})`;
    else delete decls[prop];
    node.style = Object.entries(decls).map(([k, v]) => `${k}: ${v};`).join(' ');
  }
</script>

<div class="tbe" class:text-only={textOnly} bind:this={root}>
  <!-- Toolbar (relative anchor for the absolute options panel below it). -->
  {#if !textOnly}
  <div class="toolbar">
    {#each TOOLS as tool}
      <button
        type="button"
        class="tool-btn"
        class:active={openTool === tool.id}
        title={tool.label}
        aria-label={tool.label}
        onclick={() => (openTool = openTool === tool.id ? null : tool.id)}
      >
        {@html ICONS[tool.id]}
      </button>
    {/each}

    {#if openTool}
      <div class="options">
        {#if openTool === 'padding' || openTool === 'margin'}
          {@const prefix = openTool === 'padding' ? 'p' : 'm'}
          <div class="sliders">
            {#each SIDES as side}
              {@const idx = stepIndex(sidePx(prefix, side.key))}
              <label class="slider-row">
                <span class="slider-label">{side.label}</span>
                <input
                  type="range"
                  min="0"
                  max={STEPS.length - 1}
                  step="1"
                  value={idx}
                  oninput={(e) => setSide(prefix, side.key, STEPS[+e.currentTarget.value])}
                />
                <span class="slider-val">{STEPS[idx]}px</span>
              </label>
            {/each}
          </div>
        {:else if openTool === 'fontSize'}
          {@const fp = fontPx()}
          <label class="slider-row">
            <span class="slider-label">Size</span>
            <input type="range" min="0" max="72" step="2" value={fp} oninput={(e) => setFontPx(+e.currentTarget.value)} />
            <span class="slider-val">{fp ? fp + 'px' : 'Off'}</span>
          </label>
        {:else}
          {@const prop = openTool === 'color' ? 'color' : 'background-color'}
          {@const current = colorIndex(prop)}
          <div class="swatches">
            <button
              type="button"
              class="swatch none"
              class:sel={current === null}
              title="None"
              aria-label="No color"
              onclick={() => setColor(prop, null)}>✕</button
            >
            {#each palette?.colors ?? [] as hex, i}
              <button
                type="button"
                class="swatch"
                class:sel={current === i + 1}
                style={`background:${hex}`}
                title={`Color ${i + 1}`}
                aria-label={`Color ${i + 1}`}
                onclick={() => setColor(prop, i + 1)}
              ></button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
  {/if}

  <textarea
    class="tbe-text"
    {rows}
    value={node.text || ''}
    oninput={(e) => (node.text = e.currentTarget.value)}
  ></textarea>
</div>

<style>
  .tbe {
    display: flex;
    flex-direction: column;
    border: 1px solid #334155;
    border-radius: 6px;
    transition: border-color 0.15s;
  }

  .toolbar {
    position: relative;
    display: flex;
    gap: 4px;
    padding: 5px;
    background: #0b1120;
    border-bottom: 1px solid #334155;
    border-radius: 6px 6px 0 0;
    transition: border-color 0.15s;
  }

  /* While the popup is open, fade the host element's lines so they don't
     compete with the popup's accent border. The :focus-within variant is
     needed because clicking a toolbar button focuses it and would otherwise
     re-trigger the blue focus border below. */
  .tbe:has(.options),
  .tbe:has(.options):focus-within {
    border-color: #1e293b;
  }
  .tbe:has(.options) .toolbar {
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

  /* Second row: floats below the toolbar without pushing the textarea. */
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
  }

  .sliders {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .slider-row {
    display: grid;
    grid-template-columns: 52px 1fr 40px;
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
  .swatch.none {
    display: flex;
    align-items: center;
    justify-content: center;
    background: #0f172a;
    color: #94a3b8;
    font-size: 12px;
  }
  .swatch.sel {
    outline: 2px solid #3b82f6;
    outline-offset: 1px;
    border-color: #3b82f6;
  }

  .tbe-text {
    width: 100%;
    padding: 10px 12px;
    background: #1e293b;
    border: none;
    border-radius: 0 0 6px 6px;
    color: #f1f5f9;
    font-size: 13px;
    line-height: 1.5;
    resize: vertical;
    min-height: 60px;
    box-sizing: border-box;
    display: block;
  }
  .tbe-text:focus {
    outline: none;
    background: #0f172a;
  }
  /* No toolbar (raw text run): round all corners of the textarea. */
  .tbe.text-only .tbe-text {
    border-radius: 6px;
  }
  /* Highlight the whole fused element (header + textarea) on focus. */
  .tbe:focus-within {
    border-color: #3b82f6;
  }
</style>
