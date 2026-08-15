---
name: agent-browser
description: Drive and see the running Genix app from a headless, self-authenticating Chrome — navigate routes, read the agentic HTML a page produces, run component actions, and screenshot what the browser actually paints. Use when a task needs the real app rather than the source: verifying a UI change, reproducing a page bug, checking that a component is agent-navigable, or contrasting what the agent reads against what a user sees.
version: 0.1.0
---

# Agent browser (`agent_browser`)

A headless Chrome that logs itself in, holds the tab the agentic API needs, and gives you three
views of the same page:

| View | Command | What it is |
| --- | --- | --- |
| **Agentic HTML + registry** | `html` | the map the LLM agent navigates by |
| **Browser screenshot** | `shot` | ground truth — what a user would see |
| **In-page render** | (inside `compare`) | what `modern-screenshot` produces for the product's own agent |

Everything runs from the scripts module:

```bash
cd scripts
go run . agent_browser <command>
```

Development only. The session it mints is gated on `is_local` **and** loopback
(`backend/security/dev_login.go`), so none of this exists for an end user.

## Prerequisites

The dev stack must already be up (`start.js` — backend + `vite dev` on 3570). `agent_browser`
reads `server.port` from `config.toml` to find the backend, so no flag is needed on any machine.

## The loop

```bash
# 1. Launch. ALWAYS in the background — start is resident (it collects console output).
go run . agent_browser start                      # company 1, user 1, route /
go run . agent_browser start -company 3 -user 7 -route /sales/sale_order_create

# 2. Where am I, and is the tab live?
go run . agent_browser status

# 3. Read the page the way the agent reads it
go run . agent_browser html

# 4. Move
go run . agent_browser goto /security/users

# 5. Act — same contract as POST /agent
go run . agent_browser act '[{"ID":"10","Method":"click","Args":[]}]'
go run . agent_browser act '[{"ID":"15","Method":"setValue","Args":["agente.headless"]}]'

# 6. See
go run . agent_browser shot            # -full for the whole scrollable page
go run . agent_browser compare         # all three views at once

# 7. Why is the page broken?
go run . agent_browser console -level error

# 8. Done
go run . agent_browser stop
```

Outputs land in the temp dir: `genix-agent-shot.png` (browser), `genix-agent-screenshot.png`
(in-page), `genix-agent-page.html`, `genix-agent-console.jsonl`.

## Which view to trust

This is the reason the command exists. **The HTML is the map; the screenshot is the territory.**

- **Component in the screenshot but missing from `html`** → the UI component never called
  `agentRegister`. The agent is blind to it, and no amount of navigation will fix that. This is a
  bug in the component, not in the agent.
- **Component in `html` but invisible in the screenshot** → it is registered but not rendered, or
  it is behind a closed `Layer`/`Modal`. Open the parent first.
- **An action reports `ok` but the screenshot does not change** → the handle took the call and did
  nothing with it. Check `console -level error` before assuming the action was wrong.
- **The two screenshots disagree** → `frontend/core/agent/screenshot.ts` is dropping something
  (tainted `<canvas>`, a stripped `@font-face`). That is a real defect for the *product's* agent,
  which only ever gets the in-page render. Minor known divergence: the in-page version paints a
  scrollbar and clips a few pixels at the right edge.

## Component IDs

`html` prints `id  Type  Label  methods`. Only the methods listed for that handle are accepted.
Composite ids address children — `"11:100"` is row 100 of table 11; for `setValue`/`search`/
`getOptions` the backend rewrites the call to the table's `*Child` variant automatically, so you
always pass the bare verb.

Full contract: [`backend/agent/HTTP_API.md`](../../../backend/agent/HTTP_API.md) and
[`frontend/packages/genix-ui/docs/AGENTIC_COMPONENTS.md`](../../../frontend/packages/genix-ui/docs/AGENTIC_COMPONENTS.md).

## Gotchas

- **Never pipe `start` through `head`/`grep`.** Its output is block-buffered through a pipe, so you
  see nothing, and `head` exiting kills it with SIGPIPE. Run it backgrounded and unpiped, then read
  its output file.
- **`start` dying does not kill Chrome** — deliberately, so a lost shell does not strand the tab.
  Console collection *does* stop. `stop` reaps Chrome either way, by pid from the state file.
- **One tab only.** `agent.ResolveTab` auto-resolves solely when exactly one tab is connected, so a
  human browser sitting on the app with `?agent=1` will collide with the headless one. `start`
  refuses to launch a second browser.
- **The backend restarts on Go edits.** The tab reconnects on its own with backoff; a command that
  fails with `connection refused` usually just needs retrying a few seconds later.
- **Actions write real data.** The session is a real user against the real database — `Guardar`
  saves. Prefer read-only routes, or clean up what you create.
