// SvelteKit 3 env entry (files.src is the frontend root): the PUBLIC_* values that
// scripts/setup-env.js writes to .env from config.toml. All are public and inlined at build
// time, the old $env/static/public behavior. The storefront re-exports this file.
import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	PUBLIC_ZONE_NAME: { public: true, static: true, description: 'DNS zone the company storefront subdomains live under.' },
	PUBLIC_LOCAL_API_PORT: { public: true, static: true, description: 'Port of the local backend, for the "Local" endpoint.' },
	PUBLIC_TAILSCALE_HOST: { public: true, static: true, description: 'Tailnet address a dev page may be served at. Empty = localhost only.' },
	PUBLIC_ENDPOINTS: { public: true, static: true, description: 'JSON array of the API endpoints from config.toml.' },
});
