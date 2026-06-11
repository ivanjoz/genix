// Storefront-only stub for DOMPurify (~50 KB), aliased in only for the per-company
// prerender build (VITE_COMPANY_ID set) — see webpage/vite.config.ts.
//
// DOMPurify is reached exclusively from `getPageContent()` in
// frontend/ui-components/agent/registry.ts, which runs only when the UI automation
// agent is active (window.ENABLE_UI_AGENT === 1 or a local dev host). The public
// storefront never activates the agent, so this code path is unreachable there and
// the real library is dead weight on the critical path. This stub keeps the dynamic
// import resolvable while shipping nothing of substance; the identity `sanitize`
// below would only ever run if the agent were somehow enabled on the storefront.
const DOMPurify = {
  sanitize: (html) => html,
};

export default DOMPurify;
