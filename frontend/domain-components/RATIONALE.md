# RATIONALE

Design decisions for the app shell widgets. Newest first.

## Settings panel remembers its last tab in localStorage
**Context** — `HeaderConfig` remounts every time the gear opens it, so the selected tab reset on each open.
**Decision** — The last tab is saved under `header-config-selected-tab`; with nothing stored the panel
opens on Config. (now the first tab). The key is not in `BROWSER_PREFERENCE_STORAGE_KEYS`, so logout resets it.
**Rationale** — localStorage also survives a page reload, unlike module-level state. Resetting on logout
costs nothing for a tab choice and keeps the preserved-keys list to session-critical preferences.

## Mobile header: only the info button; settings and reload in the drawer footer (from berryapps)
**Context** — Ported from berryapps: below 749px the header keeps only the info button, and settings and
reload move to genix-ui's new `SideMenu` `footer` snippet. berryapps puts its logo in that footer
because a module selector replaced its drawer header; genix's drawer header still shows the logo.
**Decision**
- The footer holds only the gear `ButtonLayer` and the reload button, right-aligned, no logo.
- The drawer's gear keeps its own open state, not `ui.state.headerSettingsOpen`: with both gears bound
  to that flag, the hidden one's click-outside would close the visible one.
- `ReloadPageButton.svelte` is shared by the header and the footer.
- "Reqs. Logs" in `HeaderConfig` also sets `ui.state.mobileMenuOpen = false`: the drawer (z 301) sits
  above modals (z 260), so its modal would open hidden behind it.
**Rationale** — Smallest port that keeps one source for reload and settings. Costs: two settings
`ButtonLayer`s register with the agent under the same label; closing the drawer from "Reqs. Logs"
leaves its settings layer open for the next time the drawer opens.
