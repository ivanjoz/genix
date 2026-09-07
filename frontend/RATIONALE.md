## HMR is off in dev: a save must leave the open page untouched (supersedes the entry below)

**Context** — With Svelte HMR on, saving a `.svelte` file did *not* reload the page (verified on the
HMR socket: only self-accepted `js-update`s, never a `full-reload`; the browser kept the app shell,
the session and an open `Layer`). What it did do is what Svelte 5 HMR is defined to do — destroy and
re-create the edited component's subtree. So `const usuariosService = new UsuariosService()` in
`UsersTab.svelte` ran again on every save, firing a delta `GET /api/users` and rebuilding the table
under the cursor. Read from the dev terminal that is indistinguishable from a reload, and it is
disruptive in the middle of an edit.

**Decision** — `server.hmr: false` in `frontend/vite.config.ts`. Vite pushes nothing to the open
page — no component rebuild, no CSS swap, no refetch. Changes are picked up with a manual refresh.
`vite-plugin-svelte` reads `server.hmr` and forces `compilerOptions.hmr` off by itself
(`enforceOptionsForHmr`), so the svelte configs need no flag and the compiler stops emitting the HMR
wrapper. `frontend/webpage/` is a separate app and was left on HMR.

**Rationale** — The two knobs do opposite things and only one matches the goal: `compilerOptions.hmr:
false` alone (the pre-2026-09 state) leaves Vite unable to patch a component, so it escalates to a
*real* `full-reload` on every save — strictly worse. Turning the channel off at `server.hmr` is the
only setting where a save has zero effect on the running page. The cost is that every change now
needs an F5, and CSS edits no longer hot-swap either.

## An insecure origin has no WebCrypto, so an empty CipherKey asks for the UserInfo in clear

**Context** — `serve_tailscale` hands the dev app out at `http://100.x.y.z:3572`. That origin is not
a secure context (only https and loopback are), so the browser never defines `crypto.subtle`, and
`parseLogin` died on `Cannot read properties of undefined (reading 'importKey')` — the login POST
had already succeeded, so the token and the access blobs were in hand and only the AES-GCM decrypt
of `UserInfo` failed. Serving the dev app over real HTTPS would need `tailscale serve`, tailnet
certs, and the Go API moved behind the same origin to dodge mixed content: a lot of machinery for a
dev-only convenience.

**Decision** — The client decides. `makeCipherKey` (`frontend/services/login.ts`) returns `''` when
`crypto.subtle` is absent, and `MakeUsuarioResponse` reads an empty key as "this browser cannot
decrypt": on a dev launch it answers `UserInfoPlain` (the same JSON, unciphered) instead of
`UserInfo`, and otherwise errors exactly as before. The per-caller `CipherKey` checks in `PostLogin`,
`PostSignUpCompany` and `DevLogin` were removed or relaxed so the rule lives in the one function all
four login paths already funnel through.

**Rationale** — Keying the fallback on the *client's* capability rather than on the environment alone
keeps localhost dev exercising the real encrypt/decrypt path, so a break in it still surfaces before
production. `UserInfo` was never a trust boundary anyway: the client generates `CipherKey` and sends
it in the request body in clear, so anyone who can read the response can read the key — the real
session credential is `UserToken`, which is unaffected. The cost is a second response shape that only
a dev backend can emit. `makeCipherKey` also replaced the hardcoded `"12341234..."` key the login had
been sending, and merged the duplicate copy in `RegistrationModal.svelte`.
## Serving dev over Tailscale: the browser's own hostname is the API host

**Context** — Opening the dev app from another machine on the tailnet needed nothing on the
listening side: the frontend proxy (`scripts/proxy-server.js`) already binds `0.0.0.0`, the Go
backend binds `:<server.port>` and fareward binds `0.0.0.0:14013`. What broke was purely the URL the
browser builds. `core/env.ts` set `_isLocal` only for `localhost`/`127.0.0.1`, so a page served at
`100.64.0.2:3572` never got the "Local" entry in the login endpoint selector, and the entry itself
hardcoded `http://localhost:<port>/`, which on the remote machine points at *its own* loopback.

**Decision** — `serve_tailscale` in `config.toml` (root table). `start.js` resolves the tailnet
address with `tailscale ip -4` and passes it as `GENIX_TAILSCALE_HOST`; `scripts/setup-env.js`
writes it to `.env` as `PUBLIC_TAILSCALE_HOST`. `core/env.ts` treats that host as local, and builds
the local endpoint from `window.location.hostname` instead of the literal `localhost`.

**Rationale** — `tailscale serve` was the other option and would have given HTTPS on a MagicDNS
name, but this tailnet reports `CertDomains: None` (a self-hosted control server, MagicDNS suffix
`example.com`), so there is no cert to issue and `--https` cannot work. Plain HTTP on the tailnet
address needs no new listener, no `tailscale serve reset` on exit, and no second mount for the API —
an HTTPS page calling `http://…:14010` would have been blocked as mixed content anyway. Deriving the
API host from `window.location.hostname` rather than from `PUBLIC_TAILSCALE_HOST` means the endpoint
is right for *any* address the page is reached at, including a plain LAN IP, and is identical to the
old behaviour on localhost. `PUBLIC_TAILSCALE_HOST` is still needed for the `_isLocal` test, which
cannot be inferred from the hostname without guessing at CGNAT ranges. Cost: one more `PUBLIC_` var,
and the flag only takes effect through `start.js` — running `vite dev` by hand still serves
localhost-only endpoints.

## Svelte HMR is enabled in dev (no more full page reloads)

**Context** — `compilerOptions.hmr: false` was set in both `svelte.config.js` and
`webpage/svelte.config.js` since the initial Svelte frontend commit. With Svelte HMR
off, `@sveltejs/vite-plugin-svelte` has no way to swap a component in place, so every
edit to any `.svelte` file made the dev server issue a `full-reload`. That wiped the
in-memory delta caches and any open Layer/Modal state on each save.

**Decision** — Removed the `hmr: false` flag from both configs. The plugin's default
(HMR on in dev, ignored in build) now applies, so editing a component patches it in
place and the page keeps its state.

**Rationale** — The flag predates Svelte 5; the old `svelte-hmr` for Svelte 4 was
flaky enough that disabling it was a common workaround. Svelte 5 ships HMR in the
compiler itself and vite-plugin-svelte 7 relies on it. The cost is the usual HMR
caveat: module-level `$state` in a `*.svelte.ts` re-initializes when that module is
hot-replaced, so a stale-looking store after an edit is a reason to hard-refresh, not
a bug. Full reload is still the fallback for non-component modules Vite cannot patch.
