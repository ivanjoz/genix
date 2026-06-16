<!--
  TextBlockEditor — a small WYSIWYG editor for a run of text in a parsed HTML section.

  GOAL
  Edit rich text the way a non-technical user expects, while writing the result straight
  back into the section AST (the canonical, reactive model that AstRenderer paints to the
  live preview — there is NO HTML re-serialization step, so mutating nodes IS the edit).

  MODEL — block ▸ lines ▸ fragments
  - One editor owns a GROUP of sibling "line" nodes (`lines`), e.g. an `<h2>` + `<p>`, or a
    single `<li>`. AstEditor decides the grouping via `groupSiblings`; this component never
    looks beyond the nodes it is handed plus their parent array (`siblings`).
  - A LINE is one block-ish node (p / h2 / li / span). Lines are separated in the UI by a
    dashed rule and map to a real block break in the output.
  - A FRAGMENT is one inline run with its own css/style — a `<span>` (or icon span) child of
    the line. A bare leaf line (`<h2>Text</h2>`, text in `node.text`, no children) is treated
    as a single virtual fragment that IS the line node.
  - Fragments of one line stack with NO rule: they render contiguously (same visual line),
    so e.g. "The new" + "shoes" can be two spans, "shoes" bigger / a different colour.

  KEYS
  - Enter  → split the focused fragment at the caret into two contiguous spans on the SAME
             line (`splitFragment`). The first Enter on a leaf line MATERIALIZES it: the
             text moves into a clean `<span>`, the block keeps its own css/style and the
             spans inherit it via the CSS cascade — so nothing changes visually.
  - +  (toolbar, top-right) → add a new LINE after the focused one, copying its tag and
             styling (`addLine`); a new sibling block is spliced into `siblings`.
  - Backspace on an already-EMPTY fragment → delete it, then its line/node if nothing
             editable remains (`deleteFragment`); focus moves to the previous fragment.

  TOOLBAR
  Acts on the focused fragment (`focusedFragment`, kept across blur so toolbar clicks still
  target it). Padding/margin/font-size live as Tailwind tokens on `node.css`; text/background
  colour live on `node.style` as palette tokens `var(--color-N)`.

  ICON FRAGMENTS
  A fragment whose trimmed text has no word characters (e.g. "✅") is a read-only icon chip,
  not a textarea — it's decoration (a bullet), not editable copy.

  GOTCHA — $state proxy identity (do not "simplify" away)
  `lines`/`children` are deep `$state` proxies. A node literal you create and splice in is the
  RAW object; the proxy the array stores (and that flows through `{#each}` and `registerEl`)
  is a DIFFERENT identity, so `rawNode === proxyNode` is false. Always read a just-inserted
  node BACK OUT of the array (`line.children[i]`, `siblings[i]`) before using it as a focus
  key, or focus handoff silently breaks (Svelte logs `state_proxy_equality_mismatch`).
-->
<script lang="ts">
  import type { ComponentAST, ColorPalette } from '$ecommerce/renderer/renderer-types';

  interface Props {
    /**
     * The group of sibling "line" nodes this editor owns. Each line is a block-ish node
     * (p / h2 / li / span); its inline children are the editable fragments. A bare text
     * leaf (no children) is a line with a single fragment backed by `node.text`.
     */
    lines: ComponentAST[];
    /** The parent's children array — spliced when adding (+) or removing lines. */
    siblings: ComponentAST[];
    /** Active palette — its 10 colors back the color/background swatches. */
    palette?: ColorPalette;
  }

  let { lines, siblings, palette }: Props = $props();

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
    // The new "add line" glyph (plus) — splits the block at the focused line.
    addLine:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"><path d="M8 3.5v9M3.5 8h9"/></svg>',
  };

  const SIDES = [
    { key: 't', label: 'Top' },
    { key: 'r', label: 'Right' },
    { key: 'b', label: 'Bottom' },
    { key: 'l', label: 'Left' },
  ] as const;

  // Which tool's options row is open (null = collapsed). Toggled by the toolbar.
  let openTool = $state<string | null>(null);

  // Root element — used to detect clicks outside the popup.
  let root = $state<HTMLDivElement>();

  // The fragment whose styling the toolbar edits — set on textarea focus, kept across
  // blur so clicking a toolbar button still targets the last-focused fragment.
  let focusedFragment = $state<ComponentAST | null>(null);

  // --- Fragments model -----------------------------------------------------------

  // An icon fragment carries no word characters (e.g. "✅"); shown as a read-only chip.
  function isIconFragment(frag: ComponentAST): boolean {
    const t = (frag.text ?? '').trim();
    return t.length > 0 && !/[\p{L}\p{N}]/u.test(t);
  }
  const isEditableFragment = (frag: ComponentAST) => !isIconFragment(frag);

  // A line's fragments are its inline children, or — for a bare leaf — the line itself.
  function fragmentsOf(line: ComponentAST): ComponentAST[] {
    return line.children?.length ? line.children : [line];
  }

  // Flat, ordered list of every fragment across the group's lines (for focus moves).
  function allFragments(): { line: ComponentAST; frag: ComponentAST }[] {
    const out: { line: ComponentAST; frag: ComponentAST }[] = [];
    for (const line of lines) for (const frag of fragmentsOf(line)) out.push({ line, frag });
    return out;
  }

  // The toolbar target: the focused fragment if still present, else the first editable one.
  const target = $derived.by(() => {
    const frags = allFragments().map((x) => x.frag);
    if (focusedFragment && frags.includes(focusedFragment)) return focusedFragment;
    return frags.find(isEditableFragment) ?? lines[0];
  });

  // --- Focus handoff: focus a node's textarea after the DOM updates --------------

  const els = new Map<ComponentAST, HTMLTextAreaElement>();
  // A fragment awaiting focus. A freshly-split fragment isn't mounted yet, so we record
  // the request and let its textarea grab focus when its `registerEl` action runs.
  let pendingFocus: { frag: ComponentAST; caret: 'start' | 'end' } | null = null;

  function placeCaret(el: HTMLTextAreaElement, caret: 'start' | 'end') {
    console.log('[TBE] placeCaret', { caret, value: el.value });
    el.focus();
    const pos = caret === 'start' ? 0 : el.value.length;
    el.setSelectionRange(pos, pos);
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
    console.log('[TBE] placeCaret done, activeElement is this el?', document.activeElement === el);
  }

  function registerEl(el: HTMLTextAreaElement, frag: ComponentAST) {
    els.set(frag, el);
    console.log('[TBE] registerEl', { text: frag.text, isPending: pendingFocus?.frag === frag, elsSize: els.size });
    // The textarea this request was waiting for just mounted — focus it now.
    if (pendingFocus?.frag === frag) {
      const caret = pendingFocus.caret;
      pendingFocus = null;
      placeCaret(el, caret);
    }
    return { destroy: () => { els.delete(frag); console.log('[TBE] destroy el', { text: frag.text }); } };
  }

  // Auto-grow a textarea to its content so fragments read like inline text, not boxes.
  function autogrow(el: HTMLTextAreaElement) {
    const resize = () => {
      el.style.height = 'auto';
      el.style.height = `${el.scrollHeight}px`;
    };
    resize();
    el.addEventListener('input', resize);
    return { destroy: () => el.removeEventListener('input', resize) };
  }

  // Focus a fragment's textarea: immediately if already mounted (e.g. the previous
  // fragment after a delete), or on mount otherwise (a freshly-split fragment).
  function requestFocus(frag: ComponentAST, caret: 'start' | 'end') {
    const el = els.get(frag);
    console.log('[TBE] requestFocus', { text: frag.text, caret, alreadyMounted: !!el });
    if (el) placeCaret(el, caret);
    else pendingFocus = { frag, caret };
  }

  // Close the options popup when clicking anywhere outside this editor.
  $effect(() => {
    if (!openTool) return;
    const onPointerDown = (e: PointerEvent) => {
      if (root && !root.contains(e.target as Node)) openTool = null;
    };
    window.addEventListener('pointerdown', onPointerDown, true);
    return () => window.removeEventListener('pointerdown', onPointerDown, true);
  });

  // --- Editing actions -----------------------------------------------------------

  /** Enter: split the fragment at the caret into two contiguous spans on the SAME line. */
  function splitFragment(line: ComponentAST, frag: ComponentAST, el: HTMLTextAreaElement) {
    const caret = el.selectionStart ?? (frag.text ?? '').length;
    const text = frag.text ?? '';
    const before = text.slice(0, caret);
    const after = text.slice(caret);
    console.log('[TBE] splitFragment', { text, caret, before, after, lineHasChildren: !!line.children?.length });
    if (line.children?.length) {
      // Already materialized: insert a new span after `frag`, inheriting its styling.
      frag.text = before;
      const insertAt = line.children.indexOf(frag) + 1;
      line.children.splice(insertAt, 0, { tagName: 'span', text: after, css: frag.css, style: frag.style });
      // Read the node back out: the reactive array stores a PROXY, and that proxy (not the
      // raw literal) is what flows through {#each} / registerEl, so it must be the focus key.
      requestFocus(line.children[insertAt], 'start');
    } else {
      // Leaf line: materialize into two clean spans. The block keeps its own css/style,
      // which the spans inherit via the CSS cascade — so nothing changes visually.
      line.text = undefined;
      line.children = [{ tagName: 'span', text: before }, { tagName: 'span', text: after }];
      requestFocus(line.children[1], 'start');
    }
  }

  /** + button: add a new line (block) after the focused one, copying its tag and styling. */
  function addLine() {
    const current = focusedLine() ?? lines[lines.length - 1];
    if (!current) return;
    const insertAt = siblings.indexOf(current) + 1;
    siblings.splice(insertAt, 0, {
      tagName: current.tagName,
      text: '',
      css: current.css,
      style: current.style,
    });
    // Focus the proxied node the reactive array now holds (see splitFragment).
    requestFocus(siblings[insertAt], 'start');
  }

  /** Remove a line (block) node from the parent entirely. */
  function removeLine(line: ComponentAST) {
    const idx = siblings.indexOf(line);
    if (idx >= 0) siblings.splice(idx, 1);
  }

  /** Backspace on an empty fragment: drop it (and its line if it leaves no editable text). */
  function deleteFragment(line: ComponentAST, frag: ComponentAST) {
    const flat = allFragments();
    const here = flat.findIndex((x) => x.frag === frag);
    let prev: ComponentAST | null = null;
    for (let j = here - 1; j >= 0; j--)
      if (isEditableFragment(flat[j].frag)) {
        prev = flat[j].frag;
        break;
      }

    if (line.children?.length) {
      line.children.splice(line.children.indexOf(frag), 1);
      // If the line has no editable text left (e.g. only an icon), remove the whole line.
      if (!line.children.some(isEditableFragment)) removeLine(line);
    } else {
      removeLine(line); // leaf line: the fragment IS the line
    }
    if (prev) requestFocus(prev, 'end');
  }

  function focusedLine(): ComponentAST | undefined {
    if (!focusedFragment) return undefined;
    return lines.find((l) => fragmentsOf(l).includes(focusedFragment!));
  }

  function onKeydown(e: KeyboardEvent, line: ComponentAST, frag: ComponentAST) {
    const el = e.currentTarget as HTMLTextAreaElement;
    console.log('[TBE] onKeydown', { key: e.key, shift: e.shiftKey, value: el.value });
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      splitFragment(line, frag, el);
    } else if (e.key === 'Backspace' && el.value === '') {
      e.preventDefault();
      deleteFragment(line, frag);
    }
  }

  // --- Tailwind class helpers (padding / margin / font-size live on the fragment) ---

  const tokens = () => (target?.css || '').split(/\s+/).filter(Boolean);

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
    if (target) target.css = kept.join(' ');
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
    if (target) target.css = kept.join(' ');
  }

  // --- Color helpers (palette tokens live on the fragment style as var(--color-N)) ---

  function styleDecls(): Record<string, string> {
    const out: Record<string, string> = {};
    for (const d of (target?.style || '').split(';')) {
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
    if (target) target.style = Object.entries(decls).map(([k, v]) => `${k}: ${v};`).join(' ');
  }
</script>

<div class="tbe" bind:this={root}>
  <!-- Toolbar (relative anchor for the absolute options panel below it). Acts on the
       focused fragment; the + button on the right adds a new line (block break). -->
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

    <button
      type="button"
      class="tool-btn add-line"
      title="Add line (block break)"
      aria-label="Add line"
      onclick={addLine}
    >
      {@html ICONS.addLine}
    </button>

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

  <!-- Lines: fragments of one line stack with no rule (same output line, contiguous
       spans); a rule between lines marks a real block break. -->
  <div class="lines">
    {#each lines as line, li (line)}
      {#if li > 0}<hr class="line-sep" />{/if}
      <div class="line">
        {#each fragmentsOf(line) as frag (frag)}
          {#if isIconFragment(frag)}
            <span class="icon-chip" title="Icon">{frag.text}</span>
          {:else}
            <textarea
              class="tbe-text"
              rows="1"
              use:autogrow
              use:registerEl={frag}
              value={frag.text ?? ''}
              oninput={(e) => (frag.text = e.currentTarget.value)}
              onfocus={() => (focusedFragment = frag)}
              onkeydown={(e) => onKeydown(e, line, frag)}
            ></textarea>
          {/if}
        {/each}
      </div>
    {/each}
  </div>
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
    align-items: center;
    gap: 4px;
    padding: 4px 0;
    background: transparent;
    border-bottom: 1px solid #1e293b;
    transition: border-color 0.15s;
  }

  /* Push the + button to the far right of the toolbar. */
  .tool-btn.add-line {
    margin-left: auto;
    color: #60a5fa;
  }
  .tool-btn.add-line:hover {
    background: #1d4ed8;
    color: #fff;
  }

  /* While the popup is open, fade the host element's lines so they don't
     compete with the popup's accent border. */
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

  /* Second row: floats below the toolbar without pushing the textareas. */
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

  /* Lines container: padding so fragments don't touch the toolbar/border. */
  .lines {
    display: flex;
    flex-direction: column;
    padding: 6px;
    gap: 4px;
  }

  /* Fragments of one line: stacked tightly (no rule) — they are one visual line. */
  .line {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  /* A horizontal rule between lines marks a real block break. */
  .line-sep {
    border: none;
    border-top: 1px dashed #475569;
    margin: 6px 0;
  }

  /* A read-only icon fragment (e.g. an emoji bullet). */
  .icon-chip {
    display: inline-flex;
    align-items: center;
    width: fit-content;
    padding: 2px 8px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 5px;
    font-size: 14px;
    line-height: 1.4;
  }

  .tbe-text {
    width: 100%;
    padding: 6px 10px;
    background: #1e293b;
    border: 1px solid transparent;
    border-radius: 5px;
    color: #f1f5f9;
    font-size: 13px;
    line-height: 1.5;
    resize: none;
    overflow: hidden;
    min-height: 0;
    box-sizing: border-box;
    display: block;
  }
  .tbe-text:focus {
    outline: none;
    background: #0f172a;
    border-color: #3b82f6;
  }
  /* Highlight the whole editor on focus. */
  .tbe:focus-within {
    border-color: #3b82f6;
  }
</style>
