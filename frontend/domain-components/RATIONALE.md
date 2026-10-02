# RATIONALE

Design decisions for the app shell widgets. Newest first.

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
