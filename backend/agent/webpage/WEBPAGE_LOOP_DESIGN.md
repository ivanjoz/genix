# Webpage Agent Loop — Design / Implementation Plan

Status: **implemented** (steps 1–5). See "Implementation notes" at the bottom.

A second agentic loop, parallel to the existing chat loop (`backend/agent/chat_loop.go`),
that handles the page-builder's two modes:

- **ModeID 2 — Build page** (`construir página`): the whole page's sections (each
  serialized to HTML) arrive in `Context`. The agent (re)writes one or more sections.
- **ModeID 3 — Edit section** (`editar sección`): only the selected section's HTML
  arrives in `Context`. The agent rewrites that one section.

Mode 1 (`ask`) is untouched — it keeps running the existing `chat_loop.go`.

The loop reuses the existing primitives verbatim: `llm.Client.Chat`, the `Message`/
`Tool`/`ToolCall` wire types, token accounting, `finish`-as-terminator, and the
SSE push helpers on `AgentSession`. Only the toolset and system prompt differ.

---

## 1. Routing — get ModeID + Context to the loop

Today `ChatUserMessage.ModeID` / `Context` are decoded in `ws.go` but only logged;
`RunTurn(ctx, text, modelHash)` never receives them (`chat_ws.go:161`).

Change `onUserMessage` to branch on mode:

```go
// chat_ws.go onUserMessage goroutine
switch msg.ModeID {
case ModeBuildPage, ModeEditSection: // 2, 3
    err = webpage.RunTurn(runCtx, s, msg) // new loop, full msg (needs Context + ModeID)
default: // 1 (ask) and anything unknown
    err = s.RunTurn(runCtx, text, msg.ModelHash) // existing loop, unchanged
}
```

- New package `app/agent/webpage`. It takes `*AgentSession` (or a small interface
  exposing `sendJSON`/`pushStatus`/`CompanyID`/`UserID`/`TabID`) so it can push
  status + the final reply down the same SSE stream.
- Mode constants (`ModeAsk=1`, `ModeBuildPage=2`, `ModeEditSection=3`) move to a
  shared spot so both `chat_ws.go` and the frontend's `+page.svelte` agree. (Frontend
  already hardcodes 2/3.)

Open import note: `webpage` importing `agent` and `agent` importing `webpage` would
cycle. Resolve by having `webpage.RunTurn` accept a small **sink interface** defined
in `webpage`, which `*AgentSession` satisfies — no back-import.

---

## 2. The loop (`webpage/loop.go`)

Same shape as `chat_loop.go`: one `client.Chat` per iteration, dispatch tool calls,
feed results back as `tool` messages, terminate on `apply_sections`.

System prompt (`webpage/prompts.go`) tells the model:
- It is a section author for the Genix page builder. It receives the current
  section(s) as HTML and must return modified section HTML.
- The HTML vocabulary: native tags + the builder's custom components
  (`<Icon>`, product/grid components, etc.). Keep `data-role` attributes.
- It has the subagent tools below to obtain assets it should NOT hand-write.
- It MUST end by calling `apply_sections` exactly once with the final HTML.

History: build-page / edit-section turns are short-lived and asset-heavy; reuse the
existing `pruneToolRounds` + clip helpers so a turn can't blow the context window.

---

## 3. Tools

### 3.1 `generate_svg` — LLM subagent (decision 3 + 4)

- **Args:** `{ description: string, viewBox?: string }`.
- **Dispatch:** a *fresh* `client.Chat` call with a dedicated system prompt
  ("You output ONLY the inner markup of an SVG icon — `<path>`/`<g>`… — no `<svg>`
  wrapper, no prose."), **no tools**, returning the raw inner markup. Uses the same
  model as the turn.
- **Storage (decision 4):** the inner markup is stored in `SectionData.Svgs[id]`,
  keyed by a generated id (`genix-svg-1`, …). The tool returns that id + viewBox to
  the main agent, which references it as an AST node `<Icon svg="genix-svg-1" vb="…"/>`
  — exactly the dedup convention `Icon.svelte` / `IconSprite.svelte` already use
  (`<use href="#id">` against the section sprite).
- The main agent **trusts** the returned markup — no validation/round-trip.

### 3.2 `find_image` — DB search + LLM-select subagent (decision 3)

- **Args:** `{ keywords: string, intention?: string, ratio?: string }` (ratio e.g.
  `"16:9"`, `"1:1"`, `"3:4"`).
- **Dispatch:**
  1. `db.SearchText[ImageAsset](&slice, partition, keywords, 0, N)` — ranked matches.
     If empty, fall back to `db.Query(&slice).GroupID.Equals(partition).Limit(N).Exec()`
     so we **always** have candidates.
  2. A subagent `client.Chat` call is given the candidate list (id, description,
     ratio) + the desired `intention`/`ratio`, and picks the single best id. (This is
     the "subagent helps decide which image to pick" part.)
  3. Resolve the chosen image's URL backend-side: look up `ImageAssetCategory.Name`
     for `CategoryID`, build
     `https://ivanjoz.github.io/genix-assets/images/{Name}/{ID}.avif`.
- **Returns:** `{ ID, url, description, ratio }`. The main agent trusts it and embeds
  `<img src="{url}">`.
- **Ratio (decision 2):** new `Ratio` column on `image_assets` (see §4). When a
  candidate's ratio is 0/unset, the subagent treats it as `1:1`. With no good ratio
  match it still returns *some* image.
- **Partition:** images live under `imageAssetCategoryGroupID` (see
  `backend/business/image_assets.go`). The loop will reuse that constant / a small
  exported accessor.

### 3.3 `apply_sections` — terminator (replaces `finish` for these modes)

- **Args:** `{ message: string, summary: string, sections: [{ id?: string, html: string }] }`.
  - Edit-section (mode 3): exactly one section, `id` = the selected section's id.
  - Build-page (mode 2): one entry per section the agent created/changed.
- Ends the turn. Persists message/summary like `completeTurn`, and pushes a new
  reply event carrying the sections + their `Svgs` (see §5).

---

## 4. `image_assets.Ratio` column (decision 2)

- Add `Ratio float32` to `ImageAsset` + `ImageAssetTable` in
  `backend/business/types/image_assets.go` (width/height, e.g. `1.0`=1:1,
  `1.777`=16:9, `0.75`=3:4). Done via the `create-database-tables` skill /
  `scripts/CREATE_EDIT_TABLE.md`, then `./app.sh check_tables`.
- **Population gap (flagging):** existing rows will have `Ratio=0`. The ingestion
  pipeline that builds `image_assets` from the asset repo would need to fill it
  later. For now `0` ⇒ treat as `1:1`. Backfilling real ratios is **out of scope**
  of this task unless you say otherwise.

---

## 5. Apply-path: backend → builder (decision 1, in scope)

New reply event so the builder can write sections back into `editorStore`:

- **Backend** (`chat_ws.go`): add e.g. `ChatTypeAgentSections = "agentSections"` with
  payload `{ ModeID, Sections: [{ id, html }], Svgs: {id: body}, Message, Summary, Timestamp }`.
  `apply_sections` emits this instead of / in addition to `agentReply`.
- **Frontend** (`AgentChat.svelte` + `sse.ts`): handle `agentSections` →
  for each returned section, `parseHTML(html)` → `ComponentAST[]`, merge `Svgs` into
  the target `SectionData`, and write into `editorStore` (the selected section for
  mode 3; matched-by-id / appended for mode 2). The chat still shows `Message`.
  `parseHTML`/`serializeAst` already round-trip (`frontend/webpage/html-ast/`).

---

## 6. File layout

```
backend/agent/webpage/
├── WEBPAGE_LOOP_DESIGN.md   (this file)
├── loop.go                  RunTurn + dispatch + apply_sections
├── tools.go                 generate_svg, find_image dispatch
├── prompts.go               system prompt + tool schemas + subagent prompts
backend/business/types/image_assets.go   (+ Ratio column)
backend/agent/chat_ws.go     (mode routing + agentSections event + Mode consts)
frontend/core/agent/sse.ts          (agentSections handler/types)
frontend/core/agent/AgentChat.svelte (apply sections into editorStore)
```

---

## 7. Build order (stop + review after each)

1. **`Ratio` column** on `image_assets` (skill + check_tables). Smallest, isolated.
2. **Routing**: mode-branch in `onUserMessage`; `webpage.RunTurn` stub that just
   echoes (no tools) so modes 2/3 reach the new loop. Verifiable via logs.
3. **Subagent tools**: `generate_svg` + `find_image` (with DB search + select
   subagent + URL resolution). Verifiable via tool-result logs.
4. **`apply_sections`** terminator + the `agentSections` backend event.
5. **Frontend apply-path**: parse + write into `editorStore`; render preview.
6. Polish: status labels, iteration cap, clipping.

---

## Resolved decisions
- Q1: `Ratio` is `float32` width/height (1.0=1:1, 1.777=16:9, 0.75=3:4). ✓
- Q2: Build-page (mode 2) MAY add new sections — `apply_sections` entries without an
  `id` create a new section (appended). ✓
- Q3: Subagents (`generate_svg`, `find_image`-select) reuse the turn's model. ✓

## Implementation notes (divergence from §3.3 / §5)
- `SectionEdit` carries **only `html`** (no `id`). The builder's section ids are
  runtime uuids absent from the serialized HTML context, so the agent can't
  reference them. Apply semantics instead key off `ModeID`:
  - **Edit section (3):** the agent returns exactly one section; the frontend
    replaces the *selected* section's `Ast` in place.
  - **Build page (2):** the agent returns the **complete ordered page**; the
    frontend **replaces all sections**. This supports adding new sections (Q2)
    and is why the system prompt insists the model include unchanged sections
    verbatim — anything omitted is dropped.
- Generated SVG bodies ship in the `agentSections` event's `Svgs` map; the
  frontend attaches to each section only the sprite ids that section's HTML
  references, so per-section `IconSprite`s stay self-contained and ids don't
  collide across sections.
