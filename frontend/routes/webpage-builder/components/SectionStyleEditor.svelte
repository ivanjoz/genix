<script lang="ts">
  import type { ColorPalette } from '$ecommerce/renderer/renderer-types';
  import { editorStore } from '../stores/editor.svelte';
  import { TEXT_TAG } from '$ecommerce/html-ast/parse-html';
  import ImageBlockEditor from './ImageBlockEditor.svelte';

  // Active palette backs the swatch shortcuts (free hex is still allowed).
  interface Props { palette?: ColorPalette }
  let { palette }: Props = $props();

  const section = $derived(editorStore.selectedSection);

  // The template's root element node IS the section container — it already carries the
  // padding/background from the HTML. Editing it in place lets section styling override
  // the template instead of fighting a separate wrapper.
  const rootNode = $derived(section?.Ast?.find((n) => n.tagName !== TEXT_TAG) ?? null);

  // Discrete spacing steps (px). `--spacing` is 1px here, so the px value IS the
  // Tailwind number (py-8 = 8px). Mirrors TextBlockEditor's scale.
  const STEPS = [0, 2, 4, 6, 8, 10, 12, 16, 20, 24, 32, 40, 48, 64, 80];

  const SIDES = [
    { key: 't', label: 'Top' },
    { key: 'r', label: 'Right' },
    { key: 'b', label: 'Bottom' },
    { key: 'l', label: 'Left' },
  ] as const;

  // Section tools merged into ImageBlockEditor's toolbar (icons mirror TextBlockEditor).
  const SECTION_TOOLS = [
    {
      id: 'padding',
      label: 'Padding',
      icon: '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="1.5" y="1.5" width="13" height="13" rx="1"/><rect x="4.5" y="4.5" width="7" height="7" rx="1"/></svg>',
    },
    {
      id: 'bgColor',
      label: 'Background color',
      icon: '<svg viewBox="0 0 16 16" width="15" height="15"><rect x="2" y="2" width="12" height="12" rx="2" fill="currentColor"/></svg>',
    },
    {
      id: 'textColor',
      label: 'Text color',
      icon: '<svg viewBox="0 0 16 16" width="15" height="15"><text x="1.5" y="11" font-size="11" font-weight="700" fill="currentColor">A</text><rect x="1.5" y="13" width="11" height="2.5" fill="#3b82f6"/></svg>',
    },
  ];

  // --- Padding: Tailwind tokens on the root element node's css ---

  const tokens = () => (rootNode?.css || '').split(/\s+/).filter(Boolean);

  /** Effective px for one side, honoring p / px / py shorthand so sliders init right. */
  function sidePx(side: string): number {
    const axis = side === 'l' || side === 'r' ? 'x' : 'y';
    const list = tokens();
    const num = (p: string) => {
      const found = list.find((x) => x.startsWith(p + '-'));
      const n = found ? parseInt(found.slice(p.length + 1), 10) : NaN;
      return Number.isNaN(n) ? undefined : n;
    };
    return num('p' + side) ?? num('p' + axis) ?? num('p') ?? 0;
  }

  /** Rebuild all four padding sides, dropping any prior shorthand/axis tokens. */
  function setSide(side: string, px: number) {
    if (!rootNode) return;
    const vals: Record<string, number> = {
      t: sidePx('t'), r: sidePx('r'), b: sidePx('b'), l: sidePx('l'),
    };
    vals[side] = px;
    const families = ['p', 'px', 'py', 'pt', 'pr', 'pb', 'pl'];
    const kept = tokens().filter((x) => !families.some((f) => x.startsWith(f + '-')));
    for (const s of ['t', 'r', 'b', 'l']) if (vals[s] > 0) kept.push(`p${s}-${vals[s]}`);
    rootNode.css = kept.join(' ');
  }

  /** Nearest STEPS index for a px value (slider position). */
  function stepIndex(px: number): number {
    let best = 0;
    for (let i = 1; i < STEPS.length; i++) if (Math.abs(STEPS[i] - px) < Math.abs(STEPS[best] - px)) best = i;
    return best;
  }

  // --- Colors: free hex written to the root node's inline style ---
  // Setting the same declaration overrides the template's `var(--color-N)`.

  function styleDecls(): Record<string, string> {
    const out: Record<string, string> = {};
    for (const d of (rootNode?.style || '').split(';')) {
      const [k, v] = d.split(':');
      if (k && v) out[k.trim()] = v.trim();
    }
    return out;
  }
  /** Current value as a hex for the picker — resolves a palette var(--color-N) to its hex. */
  function colorValue(prop: string): string {
    const v = styleDecls()[prop];
    if (!v) return '';
    const m = v.match(/^var\(--color-(\d+)\)$/);
    if (m) return palette?.colors?.[parseInt(m[1], 10) - 1] ?? '';
    return v;
  }
  function setColor(prop: string, hex: string | null) {
    if (!rootNode) return;
    const decls = styleDecls();
    if (hex) decls[prop] = hex;
    else delete decls[prop];
    rootNode.style = Object.entries(decls).map(([k, v]) => `${k}: ${v}`).join('; ');
  }

  // --- Background image: ImageEffect props live on Attributes.background ---
  // Ensure the live (reactive) object exists outside render so ImageBlockEditor can
  // mutate node.props directly. Initializing in an $effect (not a derived/template)
  // avoids state_unsafe_mutation.
  $effect(() => {
    if (section) {
      (section.Attributes ??= {});
      section.Attributes.background ??= {};
    }
  });
  // tagName satisfies the ComponentAST type; ImageBlockEditor only touches node.props.
  const backgroundNode = $derived(
    section ? { tagName: 'ImageEffect', props: section.Attributes?.background ?? {} } : null
  );
</script>

{#if section && backgroundNode}
  <!-- Reuse ImageBlockEditor's toolbar/popup/URL chrome; inject section tools into it. -->
  <ImageBlockEditor node={backgroundNode} {palette} extraTools={SECTION_TOOLS}>
    {#snippet extraOptions(openTool)}
      {#if openTool === 'padding'}
        <div class="sliders">
          {#each SIDES as side}
            {@const idx = stepIndex(sidePx(side.key))}
            <label class="slider-row">
              <span class="slider-label">{side.label}</span>
              <input
                type="range" min="0" max={STEPS.length - 1} step="1" value={idx}
                oninput={(e) => setSide(side.key, STEPS[+e.currentTarget.value])}
              />
              <span class="slider-val">{STEPS[idx]}px</span>
            </label>
          {/each}
        </div>
      {:else if openTool === 'bgColor' || openTool === 'textColor'}
        {@const prop = openTool === 'bgColor' ? 'background-color' : 'color'}
        {@const current = colorValue(prop)}
        <div class="color-picker">
          <input
            type="color" value={current || '#000000'}
            oninput={(e) => setColor(prop, e.currentTarget.value)}
          />
          <input
            type="text" class="hex-input" placeholder="#rrggbb" value={current}
            oninput={(e) => setColor(prop, e.currentTarget.value || null)}
          />
          <button type="button" class="clear-btn" title="Clear" aria-label="Clear color" onclick={() => setColor(prop, null)}>✕</button>
        </div>
        <div class="swatches">
          {#each palette?.colors ?? [] as hex, i}
            <button
              type="button"
              class="swatch"
              class:sel={current?.toLowerCase() === hex.toLowerCase()}
              style={`background:${hex}`}
              title={`Color ${i + 1}`}
              aria-label={`Color ${i + 1}`}
              onclick={() => setColor(prop, hex)}
            ></button>
          {/each}
        </div>
      {/if}
    {/snippet}
  </ImageBlockEditor>
{/if}

<style>
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

  .color-picker {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .color-picker input[type='color'] {
    width: 36px;
    height: 28px;
    padding: 0;
    border: 1px solid #475569;
    border-radius: 5px;
    background: transparent;
    cursor: pointer;
  }
  .hex-input {
    flex: 1;
    min-width: 0;
    padding: 6px 10px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 5px;
    color: #f1f5f9;
    font-size: 12px;
    font-family: ui-monospace, Menlo, Monaco, Consolas, monospace;
  }
  .hex-input:focus {
    outline: none;
    border-color: #3b82f6;
  }
  .clear-btn {
    width: 28px;
    height: 28px;
    flex-shrink: 0;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 5px;
    color: #94a3b8;
    font-size: 12px;
    cursor: pointer;
  }
  .clear-btn:hover {
    background: #334155;
    color: #f1f5f9;
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
</style>
