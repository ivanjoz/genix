# agent_browser — headless dev browser for the agentic API

```bash
cd scripts && go run . agent_browser <command>
```

Launches a headless Chrome that **logs itself in**, holds the browser tab the agentic HTTP API
needs, and lets an external agent read, drive and *see* the running app with no human involved.

Development only: the password-less session it mints is refused unless the backend is configured
`is_local` **and** the caller is on loopback.

The agent-facing walkthrough lives in the `agent-browser` skill
(`.agents/skills/agent-browser/SKILL.md`). This file documents the design.

Code lives in `scripts/agent_browser/` — `main.go` (lifecycle), `page.go` (read/act/capture),
`console.go` (event collection) and `browser/` (the CDP client), following the same
folder-plus-root-doc layout as `deployer/` + `DEPLOYER.md`.

## Why it exists

`backend/agent/` already exposes everything an agent needs: `POST /agent` runs component actions
and returns the page's agentic HTML plus its component registry, `GET /agent?get=menu` lists
routes, `GET /agent?get=screenshot` renders the DOM. All of it depended on two things only a human
could supply — a browser tab somebody opened, and a session somebody logged into.

`agent_browser` supplies both, and adds the one view the app cannot produce about itself: a real
pixel screenshot to contrast against the HTML the agent navigates by.

## Commands

| Command | Does |
| --- | --- |
| `start [-company 1] [-user 1] [-route /] [-app URL] [-backend URL] [-width 1440] [-height 900] [-port 9222] [-headed] [-chrome PATH]` | Launch Chrome, mint the session, wait for the tab to register, then stay resident collecting console output |
| `stop` | Kill Chrome and its renderers, drop the state file |
| `status` | Chrome alive, current URL, whether the backend can reach the tab |
| `goto <route>` | Navigate inside the SPA and print the new component registry |
| `shot [-out PATH] [-full]` | Chrome's own screenshot |
| `html [-out PATH]` | Agentic HTML + component registry |
| `act '<json actions>'` | Run an action batch — same contract as `POST /agent` |
| `compare [-full]` | Screenshot + in-page render + agentic HTML in one call |
| `console [-n 50] [-level error] [-follow]` | Console, exceptions and failed requests |

Artifacts land in the OS temp dir: `genix-agent-shot.png`, `genix-agent-screenshot.png`,
`genix-agent-page.html`, `genix-agent-console.jsonl`, plus the `genix-agent-browser.json` state
file and the `genix-agent-profile` Chrome profile.

## Design decisions

**Raw CDP, not Playwright.** Chrome is launched with `--headless=new --remote-debugging-port` and
driven over the DevTools Protocol (`scripts/agent_browser/browser/cdp.go`, ~200 lines on
`github.com/coder/websocket`). The browser is only a renderer and a session holder — every
*interaction* goes through the app's own action API, so a selector engine would be dead weight and
a second, divergent way to drive the app.

**Reading and acting go through the backend, only screenshots go to Chrome.** `goto`, `html` and
`act` are thin wrappers over `POST /agent`, the same endpoint the product's LLM agent uses. That
keeps this CLI from becoming a parallel implementation that drifts.

**Session minting reuses `parseLogin`.** `GET.p-dev-login` (`backend/security/dev_login.go`)
returns exactly what `POST.p-user-login` returns, built by the same `MakeUsuarioResponse`; the
frontend hydrates it with the unmodified `security.parseLogin`
(`frontend/routes/+layout.ts` → `frontend/services/login.ts:applyDevLogin`, both behind
`import.meta.env.DEV`). Seeding `localStorage` from Go was rejected: the session lives in six keys,
one wrapped by the bespoke rolling `checksum` in `packages/genix-ui/utilities/parsers.ts`, and a
second writer would eventually drift from the first.

**Two guards on the mint endpoint, not one.** `is_local` is a config value, and a config value that
is wrong turns a password-less session into a full auth bypass — so the client address is checked
independently. An `is_local=true` host that is publicly reachable still refuses everyone but itself.

**`start` is resident.** Collecting console events requires holding a CDP connection somewhere;
making `start` be that process avoids spawning a second daemon from a binary `go run` deletes on
exit. Chrome outlives it on purpose, so a lost shell does not strand the tab — `stop` reaps Chrome
by pid either way. Chrome accepts concurrent CDP clients, so the short-lived commands connect
alongside the collector.

**The backend URL comes from `config.toml`.** `server.port` is read from the repo root (3589 only
as the fallback), because hardcoding a port breaks on every machine whose config differs from the
author's — this one listens on 14010.

## Limits

- One tab at a time. `agent.ResolveTab` auto-resolves only when exactly one tab is connected, so a
  human browser on the app with `?agent=1` collides with the headless one; `start` refuses to
  launch a second browser.
- Console collection stops if the resident `start` dies, though Chrome and every other command
  keep working.
- The session is a real user against the real database. Actions that save, save.
