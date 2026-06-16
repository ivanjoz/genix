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
    /**
     * The wrapping container node (the parent whose `children` is `siblings`), when one
     * exists. Its box (width / height / max-* / margin / padding) is what the "Container"
     * tool edits — shown only while no textarea is focused. Absent at the top level.
     */
    container?: ComponentAST;
    /** Active palette — its 10 colors back the color/background swatches. */
    palette?: ColorPalette;
  }

  let { lines, siblings, container, palette }: Props = $props();

  // Discrete spacing steps (px). `--spacing` is 1px in this project, so the px
  // value IS the Tailwind number (pt-8 = 8px). 2px steps up to 16, then 4px.
  const STEPS = [0, 2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32];

  // Discrete dimension steps for the container's width/height. The step grows as the
  // value does, so the slider stays usable across a big range. Each segment boundary is
  // always landed exactly (last gap may be short).
  function buildSteps(start: number, segments: { to: number; step: number }[]): number[] {
    const out = [start];
    let v = start;
    for (const seg of segments) {
      while (v < seg.to) {
        v = Math.min(v + seg.step, seg.to);
        out.push(v);
      }
    }
    return out;
  }
  // Height: 10→1000 — 10s up to 100, 20s to 300, 40s to 1000.
  const HEIGHT_STEPS = buildSteps(10, [
    { to: 100, step: 10 },
    { to: 300, step: 20 },
    { to: 1000, step: 40 },
  ]);
  // Width: 40→1400 — 10s up to 200, 20s to 600, 40s to 1400.
  const WIDTH_STEPS = buildSteps(40, [
    { to: 200, step: 10 },
    { to: 600, step: 20 },
    { to: 1400, step: 40 },
  ]);

  // The container's dimension controls — each a Tailwind arbitrary-value family. Grouped so
  // the Width tool edits width + max-width, and the Height tool edits height + max-height.
  const WIDTH_DIMS = [
    { label: 'Width', prefix: 'w', table: WIDTH_STEPS },
    { label: 'Max width', prefix: 'max-w', table: WIDTH_STEPS },
  ];
  const HEIGHT_DIMS = [
    { label: 'Height', prefix: 'h', table: HEIGHT_STEPS },
    { label: 'Max height', prefix: 'max-h', table: HEIGHT_STEPS },
  ];

  // Text-mode toolbar tools — act on the focused fragment. Each opens an options row.
  const TOOLS = [
    { id: 'padding', label: 'Padding' },
    { id: 'margin', label: 'Margin' },
    { id: 'fontSize', label: 'Font size' },
    { id: 'color', label: 'Text color' },
    { id: 'bgColor', label: 'Background color' },
  ];

  // Container-mode toolbar tools — act on the wrapping container box. Same separate-button
  // pattern as TOOLS; ids are prefixed `c-` so they never collide with the text tools.
  const CONTAINER_TOOLS = [
    { id: 'c-padding', label: 'Padding', icon: 'padding' },
    { id: 'c-margin', label: 'Margin', icon: 'margin' },
    { id: 'c-width', label: 'Width', icon: 'width' },
    { id: 'c-height', label: 'Height', icon: 'height' },
    { id: 'c-bgColor', label: 'Background color', icon: 'bgColor' },
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
    // Container width / height glyphs — double-headed arrows along each axis.
    width:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"><path d="M2 8h12M2 8l2.5-2.5M2 8l2.5 2.5M14 8l-2.5-2.5M14 8l-2.5 2.5"/></svg>',
    height:
      '<svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"><path d="M8 2v12M8 2L5.5 4.5M8 2l2.5 2.5M8 14l-2.5-2.5M8 14l2.5-2.5"/></svg>',
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

  // Which toolbar the bar shows: the per-fragment TEXT tools while a textarea (or a
  // toolbar control following one) holds focus, else the single CONTAINER tool. Set true
  // on textarea focus; cleared only when focus leaves the editor entirely (so clicking a
  // text tool, which blurs the textarea onto a button still inside `root`, keeps it true).
  let isFocused = $state(false);

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

  // Leaving the editor entirely (focus moves to an element outside `root`, or nowhere)
  // switches the toolbar back to container mode and closes any open popup. Focus moving
  // to a toolbar button inside `root` (a text-tool click) is NOT leaving — it stays.
  $effect(() => {
    if (!root) return;
    const onFocusOut = (e: FocusEvent) => {
      const next = e.relatedTarget as Node | null;
      if (!next || !root!.contains(next)) {
        isFocused = false;
        openTool = null;
      }
    };
    root.addEventListener('focusout', onFocusOut);
    return () => root!.removeEventListener('focusout', onFocusOut);
  });

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
  const ctokens = () => (container?.css || '').split(/\s+/).filter(Boolean);

  /** Effective px for one side of a spacing family in `list`, honoring p / px / py shorthand. */
  function sidePxFrom(list: string[], prefix: string, side: string): number {
    const axis = side === 'l' || side === 'r' ? 'x' : 'y';
    const num = (p: string) => {
      const found = list.find((x) => x.startsWith(p + '-'));
      const n = found ? parseInt(found.slice(p.length + 1), 10) : NaN;
      return Number.isNaN(n) ? undefined : n;
    };
    return num(prefix + side) ?? num(prefix + axis) ?? num(prefix) ?? 0;
  }

  /** Rebuild all four sides of a spacing family in `list`, dropping prior shorthand/axis tokens. */
  function setSideIn(list: string[], prefix: string, side: string, px: number): string {
    const vals: Record<string, number> = {
      t: sidePxFrom(list, prefix, 't'), r: sidePxFrom(list, prefix, 'r'),
      b: sidePxFrom(list, prefix, 'b'), l: sidePxFrom(list, prefix, 'l'),
    };
    vals[side] = px;
    const families = [prefix, prefix + 'x', prefix + 'y', prefix + 't', prefix + 'r', prefix + 'b', prefix + 'l'];
    const kept = list.filter((x) => !families.some((f) => x.startsWith(f + '-')));
    for (const s of ['t', 'r', 'b', 'l']) if (vals[s] > 0) kept.push(`${prefix}${s}-${vals[s]}`);
    return kept.join(' ');
  }

  // Per-fragment spacing (toolbar text tools) — operates on `target.css`.
  const sidePx = (prefix: string, side: string) => sidePxFrom(tokens(), prefix, side);
  function setSide(prefix: string, side: string, px: number) {
    if (target) target.css = setSideIn(tokens(), prefix, side, px);
  }

  // Container spacing (container tool) — operates on `container.css`.
  const cSidePx = (prefix: string, side: string) => sidePxFrom(ctokens(), prefix, side);
  function setCSide(prefix: string, side: string, px: number) {
    if (container) container.css = setSideIn(ctokens(), prefix, side, px);
  }

  // --- Container dimensions (width / height / max-*) live as `w-[Npx]` etc. on container.css ---

  // `--spacing` is 1px, so `w-40` = 40px (no need for the arbitrary `w-[40px]` form).
  // Read our own numeric token; matching ANY token in the family (incl. variant prefixes
  // and non-numeric utilities like w-full / w-3/4) so setting a value fully replaces them.
  const dimValRe = (prefix: string) => new RegExp(`^${prefix}-(\\d+)$`);
  const dimFamilyRe = (prefix: string) => new RegExp(`^(?:[\\w-]+:)*${prefix}-`);
  /** Current px for a dimension family (0 = unset). */
  function dimPx(prefix: string): number {
    const m = ctokens().map((x) => x.match(dimValRe(prefix))).find(Boolean);
    return m ? parseInt(m[1], 10) : 0;
  }
  /** Slider index for a dimension: 0 = Off, else nearest table entry + 1. */
  function dimIndex(prefix: string, table: number[]): number {
    const px = dimPx(prefix);
    if (!px) return 0;
    let best = 0;
    for (let i = 1; i < table.length; i++)
      if (Math.abs(table[i] - px) < Math.abs(table[best] - px)) best = i;
    return best + 1;
  }
  /** Set/clear a dimension family on the container (px = 0 removes the whole family). */
  function setDim(prefix: string, px: number) {
    const kept = ctokens().filter((x) => !dimFamilyRe(prefix).test(x));
    if (px > 0) kept.push(`${prefix}-${px}`);
    if (container) container.css = kept.join(' ');
  }

  /** Nearest STEPS index for a px value (slider position). */
  function stepIndex(px: number): number {
    let best = 0;
    for (let i = 1; i < STEPS.length; i++) if (Math.abs(STEPS[i] - px) < Math.abs(STEPS[best] - px)) best = i;
    return best;
  }

  // Every font-size utility — named scale (text-lg), numeric (text-40), or arbitrary size
  // (text-[18px]/rem/em) — with any variant prefix (md:text-xl). Matched so setting a size
  // strips ALL of them. Deliberately NOT matching colors (text-slate-500, text-[#fff]) or
  // keywords (text-center), which start differently / aren't sizes.
  const FONT_SIZE_RE = /^(?:[\w-]+:)*text-(?:xs|sm|base|lg|xl|[2-9]xl|\[[\d.]+(?:px|rem|em)\]|\d+)$/;
  const OUR_FONT_RE = /^text-\[(\d+)px\]$/;
  const fontPx = () => {
    const m = tokens().map((x) => x.match(OUR_FONT_RE)).find(Boolean);
    return m ? parseInt(m[1], 10) : 0; // text-[18px] -> 18; 0 = unset
  };
  function setFontPx(px: number) {
    const kept = tokens().filter((x) => !FONT_SIZE_RE.test(x));
    if (px > 0) kept.push(`text-[${px}px]`);
    if (target) target.css = kept.join(' ');
  }

  // Named Tailwind font sizes — quick presets alongside the px slider. Picking one is
  // mutually exclusive with a px value (both go through FONT_SIZE_RE on the way out).
  const FONT_SIZES = ['xs', 'sm', 'base', 'lg', 'xl', '2xl', '3xl', '4xl', '5xl'];
  const NAMED_FONT_RE = /^text-(xs|sm|base|lg|xl|[2-9]xl)$/;
  const fontNamed = () => {
    const m = tokens().map((x) => x.match(NAMED_FONT_RE)).find(Boolean);
    return m ? m[1] : null;
  };
  function setFontNamed(size: string) {
    const toggleOff = fontNamed() === size;
    const kept = tokens().filter((x) => !FONT_SIZE_RE.test(x));
    if (!toggleOff) kept.push(`text-${size}`);
    if (target) target.css = kept.join(' ');
  }

  // --- Color helpers (palette tokens live on a node's style as var(--color-N)) ---

  function parseStyle(style: string | undefined): Record<string, string> {
    const out: Record<string, string> = {};
    for (const d of (style || '').split(';')) {
      const [k, v] = d.split(':');
      if (k && v) out[k.trim()] = v.trim();
    }
    return out;
  }
  function colorIdxOf(style: string | undefined, prop: string): number | null {
    const m = parseStyle(style)[prop]?.match(/^var\(--color-(\d+)\)$/);
    return m ? parseInt(m[1], 10) : null;
  }
  function applyColor(style: string | undefined, prop: string, idx: number | null): string {
    const decls = parseStyle(style);
    if (idx) decls[prop] = `var(--color-${idx})`;
    else delete decls[prop];
    return Object.entries(decls).map(([k, v]) => `${k}: ${v};`).join(' ');
  }

  // Per-fragment colors (text tools) — on `target.style`.
  const colorIndex = (prop: string) => colorIdxOf(target?.style, prop);
  function setColor(prop: string, idx: number | null) {
    if (target) target.style = applyColor(target.style, prop, idx);
  }
  // Container color (container tool) — on `container.style`.
  const cColorIndex = (prop: string) => colorIdxOf(container?.style, prop);
  function setCColor(prop: string, idx: number | null) {
    if (container) container.style = applyColor(container.style, prop, idx);
  }
</script>

<div class="tbe" bind:this={root}>
  <!-- Toolbar (relative anchor for the absolute options panel below it). Acts on the
       focused fragment; the + button on the right adds a new line (block break). -->
  <div class="toolbar">
    {#if isFocused}
      <!-- Text mode: per-fragment tools acting on the focused textarea. -->
      {#each TOOLS as tool}
        <button
          type="button"
          class="tool-btn"
          class:active={openTool === tool.id}
          title={tool.label}
          aria-label={tool.label}
          onmousedown={(e) => e.preventDefault()}
          onclick={() => (openTool = openTool === tool.id ? null : tool.id)}
        >
          {@html ICONS[tool.id]}
        </button>
      {/each}
    {:else if container}
      <!-- Container mode (nothing focused): separate tools editing the wrapping container box. -->
      {#each CONTAINER_TOOLS as tool}
        <button
          type="button"
          class="tool-btn"
          class:active={openTool === tool.id}
          title={tool.label}
          aria-label={tool.label}
          onmousedown={(e) => e.preventDefault()}
          onclick={() => (openTool = openTool === tool.id ? null : tool.id)}
        >
          {@html ICONS[tool.icon]}
        </button>
      {/each}
    {/if}

    <button
      type="button"
      class="tool-btn add-line"
      title="Add line (block break)"
      aria-label="Add line"
      onmousedown={(e) => e.preventDefault()}
      onclick={addLine}
    >
      {@html ICONS.addLine}
    </button>

    {#if openTool}
      <div class="options">
        {#if openTool === 'c-padding' || openTool === 'c-margin'}
          {@const prefix = openTool === 'c-padding' ? 'p' : 'm'}
          <!-- Container spacing (4 sides) — same widget as the text tools, on container.css. -->
          <div class="sliders">
            {#each SIDES as side}
              {@const idx = stepIndex(cSidePx(prefix, side.key))}
              <label class="slider-row">
                <span class="slider-label">{side.label}</span>
                <input
                  type="range"
                  min="0"
                  max={STEPS.length - 1}
                  step="1"
                  value={idx}
                  oninput={(e) => setCSide(prefix, side.key, STEPS[+e.currentTarget.value])}
                />
                <span class="slider-val">{STEPS[idx]}px</span>
              </label>
            {/each}
          </div>
        {:else if openTool === 'c-width' || openTool === 'c-height'}
          {@const dims = openTool === 'c-width' ? WIDTH_DIMS : HEIGHT_DIMS}
          <!-- Container size: the dimension + its max, on container.css as w-[Npx] etc. -->
          <div class="sliders">
            {#each dims as dim}
              {@const di = dimIndex(dim.prefix, dim.table)}
              <label class="slider-row">
                <span class="slider-label">{dim.label}</span>
                <input
                  type="range"
                  min="0"
                  max={dim.table.length}
                  step="1"
                  value={di}
                  oninput={(e) => setDim(dim.prefix, +e.currentTarget.value === 0 ? 0 : dim.table[+e.currentTarget.value - 1])}
                />
                <span class="slider-val">{di === 0 ? 'Off' : dim.table[di - 1] + 'px'}</span>
              </label>
            {/each}
          </div>
        {:else if openTool === 'c-bgColor'}
          {@const current = cColorIndex('background-color')}
          <!-- Container background color — palette swatches, on container.style. -->
          <div class="swatches">
            <button
              type="button"
              class="swatch none"
              class:sel={current === null}
              title="None"
              aria-label="No color"
              onclick={() => setCColor('background-color', null)}>✕</button
            >
            {#each palette?.colors ?? [] as hex, i}
              <button
                type="button"
                class="swatch"
                class:sel={current === i + 1}
                style={`background:${hex}`}
                title={`Color ${i + 1}`}
                aria-label={`Color ${i + 1}`}
                onclick={() => setCColor('background-color', i + 1)}
              ></button>
            {/each}
          </div>
        {:else if openTool === 'padding' || openTool === 'margin'}
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
          {@const named = fontNamed()}
          <div class="sliders">
            <div class="size-presets">
              {#each FONT_SIZES as sz}
                <button
                  type="button"
                  class="size-btn"
                  class:sel={named === sz}
                  title={`text-${sz}`}
                  onmousedown={(e) => e.preventDefault()}
                  onclick={() => setFontNamed(sz)}>{sz}</button
                >
              {/each}
            </div>
            <label class="slider-row">
              <span class="slider-label">Size</span>
              <input type="range" min="0" max="72" step="2" value={fp} oninput={(e) => setFontPx(+e.currentTarget.value)} />
              <span class="slider-val">{fp ? fp + 'px' : 'Off'}</span>
            </label>
          </div>
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
              onfocus={() => { focusedFragment = frag; isFocused = true; if (openTool?.startsWith('c-')) openTool = null; }}
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
    max-height: 60vh;
    overflow-y: auto;
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

  /* Named font-size presets (xs … 5xl) above the px slider. */
  .size-presets {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .size-btn {
    min-width: 30px;
    padding: 3px 8px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 5px;
    color: #cbd5e1;
    font-size: 11px;
    cursor: pointer;
  }
  .size-btn:hover {
    background: #334155;
    color: #f1f5f9;
  }
  .size-btn.sel {
    background: #3b82f6;
    border-color: #3b82f6;
    color: #fff;
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
    /* Clip the flush textareas to the container's rounded bottom corners. */
    border-bottom-left-radius: 6px;
    border-bottom-right-radius: 6px;
    overflow: hidden;
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
    padding: 5px 8px 5px 6px;
    background: #1e293b;
    border: none;
    /* Reserve the focus accent up-front so focusing doesn't shift the text. */
    border-left: 4px solid transparent;
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
    border-left-color: #3b82f6;
  }
  /* Highlight the whole editor on focus. */
  .tbe:focus-within {
    border-color: #3b82f6;
  }
</style>
