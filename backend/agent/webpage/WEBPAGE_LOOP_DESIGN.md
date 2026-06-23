# Webpage Agent Loop — Design & Rationale (as-built)

A second agentic loop, parallel to the chat loop (`backend/agent/chat_loop.go`),
dedicated to the page builder. It authors and edits HTML "sections" for a website,
with two modes:

- **ModeID 2 — Build page** (`construir página`): the whole page's sections, each
  serialized to HTML, arrive in `Context`. The agent returns the complete page.
- **ModeID 3 — Edit section** (`editar sección`): only the selected section's HTML
  arrives. The agent returns that one section.

Mode 1 (`ask`) is untouched — it keeps running `chat_loop.go`.

The loop reuses the existing primitives verbatim: `llm.Client.Chat`, the `Message` /
`Tool` / `ToolCall` wire types, and the SSE push helpers on `*AgentSession`. What
differs from the chat loop: a pinned model, a custom toolset, a terminator tool
(`apply_sections` instead of `finish`), and — the substance of this loop — an
**intent classifier**, a **deterministic content-preservation gate**, and an
**aesthetic critic** wrapped around the model's output.

---

## 1. Routing — `agent` → `webpage`, one-directional

`onUserMessage` (`chat_ws.go:173`) branches on mode:

```go
switch msg.ModeID {
case webpage.ModeBuildPage, webpage.ModeEditSection: // 2, 3
    err = webpage.RunTurn(runCtx, s, msg.ModeID, text, msg.ModelHash, msg.Context)
default: // 1 (ask) and anything unknown
    err = s.RunTurn(runCtx, text, msg.ModelHash) // existing chat loop, unchanged
}
```

The mode constants live in `webpage` (`loop.go`) as the single source of truth,
shared with package `agent` (which imports `webpage` to route) and mirrored by the
frontend builder route, which sends these IDs on the wire.

**No import cycle:** `webpage` never imports `agent`. The session functionality the
loop needs (push status / reply / sections) is declared as the `Sink` interface in
`loop.go`, which `*AgentSession` satisfies structurally. Dependency is one-way:
`agent → webpage`.

---

## 2. Model & reasoning budgets (rationale)

The builder **pins its own model** (`builderModel = "tencent/hy3-preview"`) rather
than following the shared chat model picker — the `modelHash` from the wire is
ignored on purpose. hy3-preview honors disabled/low/high reasoning effort (unlike
DeepSeek V4 Flash, which ignored `effort:low` and reasoned to a huge default budget
— a single `generate_svg` was once seen at ~68s). Its one caveat — provider routing
rejects `tool_choice=required` — doesn't bite us; the loop uses `tool_choice:"auto"`
and the system prompt disciplines the model into ending via `apply_sections`.

Reasoning budgets (`loop.go`):
- `builderReasoning` — `effort:low, exclude:true`. The main loop plans (which asset,
  what to edit) but doesn't need a deep trace; low keeps each iteration snappy and
  `exclude` keeps the trace out of later prompts.
- `subagentNoReasoning` — disabled outright. `generate_svg`, `find_image`-select and
  the **intent classifier** are mechanical (emit markup / pick an index / emit a tiny
  JSON verdict) — no chain-of-thought.
- `criticReasoning` — `effort:low, exclude:true` for the aesthetic critic.

---

## 3. Turn lifecycle (`RunTurn` in `loop.go`)

```
RunTurn
 ├─ classifyTurn ........... intent classification → policy + system-prompt constraints
 │     └─ relevant=false? → reply "No se pudo interpretar la instrucción", stop
 ├─ loop (≤ maxBuilderIterations = 12):
 │     client.Chat(tools, auto)
 │     ├─ no tool call  → treat plain text as the final reply, stop
 │     ├─ generate_svg / find_image / get_component_docs → dispatch, feed result back
 │     └─ apply_sections (terminator):
 │           ├─ CONTENT GATE (hard, ≤ maxContentRevisions = 2)
 │           │     verifyContent → violations? feed them back, model reworks, continue
 │           └─ AESTHETIC CRITIC (soft, ≤ maxAestheticRevisions = 1)
 │                 reviewAesthetics → REVISE? feed critique back, continue
 │                 else → PushSections, stop
```

Per-turn state lives in `builderTurn`: the LLM client + pinned model (reused by
subagents), the SVG bodies generated this turn (keyed by sprite id, shipped with the
final payload), the turn log, and the content-preservation state set by the
classifier (§4).

The two gates run **content-first, aesthetics-second**: content is a hard
correctness requirement and bounces more cheaply, aesthetics is a quality nudge.
Each has its own revision budget so a stubborn model can't wedge the turn — once a
budget is spent, that gate is skipped and the next `apply_sections` is applied
verbatim. The aesthetic critic also **fails open**: any critic error approves the
result rather than blocking the turn.

There is no `pruneToolRounds`-style history clipping here (that's the chat loop):
builder turns are short — a few asset calls then `apply_sections` — so
`maxBuilderIterations` is the only bound needed.

---

## 4. Intent classification & content preservation

The core problem this loop solves: the model tended to change content the user did
**not** ask to change — rewording text, dropping an `<Icon>`, swapping an image. The
"make minimal edits" instruction alone didn't enforce it. Full design + decisions in
**`CONTENT_PRESERVATION_PLAN.md`**; the moving parts:

- **Classifier** (`classify.go`) — a cheap, reasoning-disabled subagent run *before*
  the loop. It reads the request + section HTML and returns a structured verdict:
  - Edit-section: per-dimension policy `{text, images, icons}` ∈
    `keep | add | modify | replace`, plus a `scope` hint (`rewrite` relaxes all to
    `replace`). Anything the request doesn't mention defaults to `keep`.
  - Build-page: a section plan `{operation, modifySectionIds, removeSectionIds,
    addSections}` over the `=== SECTION N ===`-numbered sections.
  - **Relevance gate:** a nonsense / off-topic request yields `relevant=false`; the
    turn aborts immediately with the reply **"No se pudo interpretar la instrucción"**.
  - Has its own retry budget (`classifyMaxAttempts`); on unrecoverable failure it
    falls back to the safest verdict — **lock everything to `keep`**.

- **Prompt conditioning** (`verify.go`) — the verdict renders a "PRESERVATION
  CONSTRAINTS THIS TURN" block appended to the system prompt, so the model
  self-limits before it's ever checked.

- **Deterministic gate** (`html_ast.go` + `verify.go`) — on `apply_sections`, old vs
  new HTML are parsed to ASTs (`ParseHTMLToAST`) and reduced to a content
  fingerprint `{Texts, Images, Icons}` (`ExtractSectionContent`). `VerifySectionContent`
  enforces the policy: `keep` → multiset identical; `add` → old ⊆ new;
  `modify`/`replace` → unverified. It compares *bags*, so the agent may freely
  restyle and move content between tags — only the content set is gated. Violations
  are fed back as a tool result naming the exact offending items; the model fixes
  only those and re-applies.

  Build-page mapping: the frontend prefixes each context section with
  `=== SECTION N ===`; the agent echoes `N` as `SectionEdit.SourceID` on each returned
  section. The verifier then requires untargeted sections back verbatim, removed
  sections absent, and new sections only when `addSections` is true. (`SourceID` is
  backend verification metadata only — the frontend applies sections positionally and
  ignores it.)

---

## 5. Tools (`prompts.go` schemas, `tools.go` dispatch)

The toolset registered every iteration: `generate_svg`, `find_image`,
`get_component_docs`, `apply_sections`.

### `generate_svg` — LLM subagent
- **Args:** `{ description, viewBox? }`.
- A fresh `client.Chat` with `svgSystemPrompt`, no tools, returns the bare inner SVG
  markup (`cleanSVGBody` strips fences / an accidental `<svg>` wrapper). Stored in the
  turn's `svgs` map under a fresh id (`genix-svg-N`); the tool returns just the id +
  viewBox. The main agent references it as `<Icon svg="genix-svg-N" vb="…"/>` — the
  same `<use href="#id">` dedup convention `Icon.svelte` / `IconSprite.svelte` use.
- The main agent **trusts** the returned markup — no round-trip validation.

### `find_image` — library search + LLM-select subagent
- **Args:** `{ keywords, intention?, ratio? }`.
- `business.FindImageCandidates(keywords, 10)` ranks matches (with a fallback so there
  is always at least one candidate). When >1, the select subagent (`imageSelectSystemPrompt`,
  no reasoning) picks the best index for the intention + ratio; on any failure it
  falls back to index 0.
- **Returns:** `{ ID, url, description, ratio, usage }`; the agent embeds
  `<img src="{url}"/>`. Ratio is `width/height` (1.0 ≈ 1:1, 1.78 ≈ 16:9, 0.75 ≈ 3:4);
  `0` ⇒ treated as 1:1.

### `get_component_docs` — pure lookup
- **Args:** `{ component }`. Returns the reference docs (attributes, defaults,
  example) for a custom builder component (`ProductGrid`, `ImageEffect`, `Slider`…).
  On a miss it returns the available names so the agent can self-correct. No LLM call.

### `apply_sections` — terminator
- **Args:** `{ message, summary, sections: [{ html, css?, sourceId? }] }`. Called
  **exactly once** to end the turn.
  - Edit-section (3): exactly one section.
  - Build-page (2): the **complete ordered page** — unchanged sections included
    verbatim, each tagged with its `sourceId`. Anything omitted is dropped.
- `message` is the short chat reply; `summary` is a brief change log. `css` is
  optional raw CSS for effects Tailwind can't express (gradients, clip-path,
  keyframes…) using the agent's own class names; the frontend scopes it to
  page-unique `.x{n}` classes and keeps only class selectors.

---

## 6. Apply-path: backend → builder

On a clean `apply_sections`, `applySections` calls `Sink.PushSections`, which emits
the `ChatTypeAgentSections = "agentSections"` event (`chat_ws.go`) with
`{ ModeID, Sections, Svgs, Message, Summary, Timestamp }`.

Frontend (`[pageID]/+page.svelte` `applyAgentSections`):
- **Edit section (3):** `parseHTML(html)` → AST, replace the selected section's `Ast`
  in place; scope its `css`; merge referenced SVGs.
- **Build page (2):** replace **all** sections from the returned ordered list (one
  custom-css id allocator threads across them).
- Each section gets only the sprite ids its HTML references
  (`pickReferencedSvgs`), so per-section `IconSprite`s stay self-contained and ids
  don't collide. Arbitrary-hex colors the agent introduced are absorbed into the
  palette (`absorbColors`) and rewritten to `var(--color-N)`.

---

## 7. File layout

```
backend/agent/webpage/
├── WEBPAGE_LOOP_DESIGN.md        this file — loop design & rationale
├── CONTENT_PRESERVATION_PLAN.md  intent classifier + deterministic verifier
├── loop.go         RunTurn, the iteration loop, both gates, apply_sections, subagent runner
├── classify.go     intent classifier (edit + page prompts, retries, JSON parse, fallbacks)
├── verify.go       classifyTurn dispatch, content gate, source/section parsing, constraint blocks
├── html_ast.go     ParseHTMLToAST + content fingerprint (Extract/VerifySectionContent)
├── tools.go        generate_svg, find_image, get_component_docs dispatch
├── prompts.go      system prompt (base + per-mode tail), tool schemas, subagent/critic prompts
├── components.go   custom-component registry + docs (get_component_docs source)
├── loop_log.go     per-turn structured log (turnLog)
└── *_test.go       html_ast / verify / components tests
backend/agent/chat_ws.go   mode routing, ChatTypeAgentSections event, PushSections, Mode consts use
frontend/routes/webpage-builder/[pageID]/+page.svelte
                           buildAgentContext (palette + assets + section markers), applyAgentSections
```

---

## Historical decisions (still in force)
- **Q1:** `image_assets.Ratio` is `float32` width/height (1.0=1:1, 1.78=16:9,
  0.75=3:4); `0` ⇒ treated as 1:1. Backfilling real ratios for old rows is a separate
  ingestion concern.
- **Q2:** Build-page MAY add/remove sections — the agent returns the complete page;
  new sections carry `sourceId:0`, omitted sections are dropped.
- **Q3:** Subagents (`generate_svg`, `find_image`-select, classifier, critic) reuse
  the turn's pinned model.
- **SectionEdit** carries `html`, optional `css`, and (build-page) `sourceId`. There
  is no runtime section uuid in the serialized context — apply semantics key off
  `ModeID`: edit replaces the selected section in place; build-page replaces the whole
  ordered list.
```
