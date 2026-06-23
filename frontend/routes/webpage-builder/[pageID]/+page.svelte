<script lang="ts">
  // Per-page builder route: /webpage-builder/<PageID>. The PageID comes from the
  // pages list (pencil button) and selects which page's content to load/save.
  // The bare /webpage-builder route redirects here (see ../+page.ts).
  import { page } from '$app/state';
  import EcommerceBuilder from '../builder/EcommerceBuilder.svelte';
  import type { SectionData } from '$ecommerce/renderer/section-types';
  import { getPageContent, setCurrentPageID } from '$services/ecommerce/page-content.svelte';
  import { editorStore } from '../stores/editor.svelte';
  import Header from '$ecommerce/components/Header.svelte';
  import Page from '$domain/Page.svelte';
  import T from '$components/misc/T.svelte';
  import { agentModes, type IAgentMode, type AgentSectionsPayload } from '$core/agent/agent.svelte';
  import { serializeAst } from '$ecommerce/html-ast/serialize-html';
  import { parseHTML } from '$ecommerce/html-ast/parse-html';
  import { absorbColors } from '../html-ast/absorb-colors';
  import { scopeCustomCss, nextGlobalId } from '../html-ast/scope-custom-css';

  // Mode IDs registered below; named so the context provider stays readable.
  const MODE_BUILD_PAGE = 2;
  const MODE_EDIT_SECTION = 3;

  // Pull the images/icons already present in the section HTML into an explicit
  // list, so the agent reuses them instead of re-deriving them by parsing the
  // markup (it was firing find_image / generate_svg speculatively because the
  // image lived inside an <ImageEffect src=…>, not a plain <img>). A src= can
  // sit on any tag (<img>, <ImageEffect>, …); icons are <Icon svg=…>.
  const extractAssets = (html: string): string => {
    const lines: string[] = [];
    const seenSrc = new Set<string>();
    const srcRe = /<([A-Za-z][\w-]*)\b[^>]*\bsrc="([^"]+)"/g;
    for (let m = srcRe.exec(html); m; m = srcRe.exec(html)) {
      const [, tag, src] = m;
      if (seenSrc.has(src)) continue;
      seenSrc.add(src);
      lines.push(`  - existing image → REUSE this EXACT src: src="${src}" (it is in <${tag}>)`);
    }
    const seenSvg = new Set<string>();
    const iconRe = /<Icon\b[^>]*\bsvg="([^"]+)"/g;
    for (let m = iconRe.exec(html); m; m = iconRe.exec(html)) {
      const svg = m[1];
      if (seenSvg.has(svg)) continue;
      seenSvg.add(svg);
      lines.push(`  - existing icon → REUSE this EXACT svg: svg="${svg}" (it is in <Icon>)`);
    }
    if (lines.length === 0) return '';
    return `ASSETS ALREADY IN THIS SECTION — you MUST reuse them as-is. Moving, resizing, or making one a circle is NOT a request for a new asset: keep the EXACT src/svg and only change the classes. Do NOT call find_image / generate_svg unless the user EXPLICITLY asks for a different image or icon.\n${lines.join('\n')}\n\n`;
  };

  // Build the per-turn context the agent receives: the current color palette (so it
  // can reuse a color by index), the assets already in the section, then the section
  // HTML — the whole page in "build page" mode, or just the selected section in "edit
  // section" mode. The backend appends this verbatim, so the framing/labels live here
  // (this side knows both the mode and the palette).
  const buildAgentContext = (modeID: number): string => {
    let html = '';
    let label = '';
    if (modeID === MODE_EDIT_SECTION) {
      html = serializeAst(editorStore.selectedSection?.Ast ?? []);
      label = 'the selected section';
    } else if (modeID === MODE_BUILD_PAGE) {
      // Prefix each section with a "=== SECTION N ===" marker so the backend's
      // intent classifier/verifier can target sections by number and require the
      // untargeted ones to come back unchanged. The agent echoes N as sourceId.
      html = editorStore.sections
        .map((s, i) => `=== SECTION ${i + 1} ===\n${serializeAst(s.Ast ?? [])}`)
        .join('\n');
      label = "the page's current sections";
    } else {
      return '';
    }
    const paletteLine = editorStore.palette.colors.map((c, i) => `${i + 1}=${c}`).join(' ');
    return `CURRENT COLOR PALETTE (reuse a color by its index, e.g. color="3"):\n${paletteLine}\n\n${extractAssets(html)}--- HTML of ${label} ---\n${html}`;
  };

  // Apply the page-builder agent's edited sections back into the editor. The
  // backend returns each section as HTML plus the SVG bodies it generated this
  // turn (keyed by sprite id); we parse the HTML into the canonical AST and
  // attach only the SVGs that section references (so each section's IconSprite
  // stays self-contained and sprite ids don't collide across sections).
  const pickReferencedSvgs = (html: string, svgs: Record<string, string>): Record<string, string> => {
    const out: Record<string, string> = {};
    for (const [id, body] of Object.entries(svgs ?? {})) {
      if (html.includes(id)) out[id] = body;
    }
    return out;
  };

  // Parse a returned section's HTML to AST, absorbing any new arbitrary-hex colors the
  // agent introduced (bg-[#hex]/text-[#hex]/…) into the page palette and rewriting them
  // to var(--color-N), so the palette stays the single source of truth and grows.
  const parseAgentHtml = (html: string) => {
    const ast = parseHTML(html ?? '');
    absorbColors(ast, editorStore.palette.colors);
    return ast;
  };

  const applyAgentSections = (payload: AgentSectionsPayload) => {
    const svgs = payload.Svgs ?? {};
    if (payload.ModeID === MODE_EDIT_SECTION) {
      // Edit section: replace the selected section's AST in place.
      const target = editorStore.selectedSection;
      const edited = payload.Sections[0];
      if (target && edited) {
        const ast = parseAgentHtml(edited.html ?? '');
        // Scope this turn's custom CSS against the rest of the page (this section's
        // own old CustomCss is replaced, so allocate ids past everyone else's).
        const others = editorStore.sections.filter((s) => s !== target);
        target.Ast = ast;
        target.CustomCss = scopeCustomCss(edited.css, ast, nextGlobalId(others));
        target.Svgs = { ...(target.Svgs ?? {}), ...pickReferencedSvgs(edited.html ?? '', svgs) };
      }
      return;
    }
    // Build page: the returned list is the complete page — replace all sections.
    // One allocator threads across sections so their custom-class ids don't collide.
    const allocId = nextGlobalId([]);
    editorStore.sections = payload.Sections.map((section) => {
      const ast = parseAgentHtml(section.html ?? '');
      return {
        id: crypto.randomUUID(),
        Type: 'HtmlSection',
        Ast: ast,
        CustomCss: scopeCustomCss(section.css, ast, allocId),
        Svgs: pickReferencedSvgs(section.html ?? '', svgs),
        Css: {},
      };
    });
    editorStore.select(null);
  };

  // Agent modes this page contributes to the shared chat widget. Mode 2 is the
  // default "build the page" intent; mode 3 targets the section the user has
  // selected in the editor.
  const BUILDER_MODES: IAgentMode[] = [
    { ID: MODE_BUILD_PAGE, Name: 'Build page|Construir página', Placeholder: 'I help you build your page...|Te ayudo a construir tu página...', Icon: 'icon-[fa--magic]' },
    { ID: MODE_EDIT_SECTION, Name: 'Edit section|Editar sección', Placeholder: 'Edit this section with a prompt...|Modifica esta sección con un promp...', Icon: 'icon-[fa--pencil]' },
  ];

  // Register the agent modes for this page. Mode 3 (edit section) only exists
  // while a section is selected, and becomes active then; otherwise just the
  // build mode (2) is offered, as the default. Re-runs on selection changes;
  // clears on unmount.
  $effect(() => {
    const hasSelection = !!editorStore.selectedId;
    agentModes.set(hasSelection ? BUILDER_MODES : [BUILDER_MODES[0]], hasSelection ? MODE_EDIT_SECTION : MODE_BUILD_PAGE);
    // Provide the sections-as-HTML context the chat widget sends with each message.
    agentModes.setContextProvider(buildAgentContext);
    // Apply the agent's edited sections back into the editor when they arrive.
    agentModes.setSectionsApplier(applyAgentSections);
    return () => agentModes.clear();
  });

  // params.pageID is a string; coerce to a number (0 falls back to the default page).
  const pageID = $derived(Number(page.params.pageID) || 0);

  let elements = $state<SectionData[]>([]);
  let values = $state<Record<string, string>>({});
  let loading = $state(false);

  // Load (re-load) the requested page whenever pageID changes — SvelteKit reuses
  // this component when navigating between /webpage-builder/<id> pages, so onMount
  // would fire only once. The editorStore is a singleton shared across builder
  // routes, so it must be reset before loading or it would keep showing the
  // previously opened page's sections (it only auto-loads when empty — see
  // EcommerceBuilder). A per-run token discards a stale response if pageID changes
  // again before the fetch resolves.
  let loadToken = 0;
  $effect(() => {
    const id = pageID;
    const token = ++loadToken;
    setCurrentPageID(id);
    editorStore.select(null);
    editorStore.sections = [];
    editorStore.setPalette(undefined); // reset to default until the page's palette loads
    elements = [];
    loading = true;

    getPageContent(id).then((stored) => {
      if (token !== loadToken) return; // a newer load superseded this one
      // Set the palette before `elements` triggers EcommerceBuilder's baseline snapshot.
      editorStore.setPalette(stored.palette);
      elements = stored.sections;
      loading = false;
    });
  });

  function handleUpdate(updated: SectionData[]) {
    console.log('Store updated:', updated);
  }
</script>

<Page title="Builder" containerCss="p-0! w-[calc(100%-280px)]!" useTopMinimalMenu fixedFullHeight>
  <!-- The storefront chrome is desktop-styled and lives outside the canvas. In mobile
       preview it would sit, full-width, atop a 390px body — so hide it; the iframe shows
       the page body on its own. -->
  {#if editorStore.viewMode !== 'mobile'}
    <Header />
  {/if}
  {#if loading}
    <div class="builder-loader">
      <div class="loader-spinner"></div>
      <span><T text="Loading page content...|Obteniendo contenido de la página..." /></span>
    </div>
  {:else}
    <EcommerceBuilder
      bind:elements
      bind:values
      palette={editorStore.palette}
      onUpdate={handleUpdate}
    />
  {/if}
</Page>

<style>
  .builder-loader {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    width: 100%;
    height: 100%;
    color: #475569;
    font-size: 15px;
  }
  .loader-spinner {
    width: 38px;
    height: 38px;
    border: 4px solid #e2e8f0;
    border-top-color: #64748b;
    border-radius: 50%;
    animation: builder-loader-spin 0.8s linear infinite;
  }
  @keyframes builder-loader-spin {
    to { transform: rotate(360deg); }
  }
</style>
