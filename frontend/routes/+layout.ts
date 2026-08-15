// This file is required for SPA mode
export const ssr = false;

// Dev-only session bootstrap for the headless agent browser (scripts/agent_browser).
// Loading "/?devlogin=<companyID>:<userID>" mints that user's session before the layout renders,
// so the automated tab lands authenticated instead of bouncing to /welcome. It runs here — in
// load, not onMount — because +layout.svelte decides `redirectsToLogin` while rendering, and by
// then it is too late to seed the session.
//
// import.meta.env.DEV keeps both the branch and the dynamically imported module out of production
// builds.
export const load = async ({ url }: { url: URL }) => {
	if (!import.meta.env.DEV) { return }

	const devLoginTarget = url.searchParams.get('devlogin')
	if (!devLoginTarget) { return }

	const { applyDevLogin } = await import('$services/login')
	await applyDevLogin(devLoginTarget)
}
