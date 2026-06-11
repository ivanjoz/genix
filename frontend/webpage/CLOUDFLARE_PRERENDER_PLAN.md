# Cloudflare Multi-Tenant Storefront Serve Worker Plan

## Goal

Serve each company's already-prerendered storefront from its own hostname, for
example `tienda-x.un.pe`, using one minimal Cloudflare Worker.

The hostname is also the HTML artifact folder name. The Worker therefore needs no
subdomain-to-company map:

```text
webpages/
  tienda-x.un.pe/
    index.html
    sw.js
  tienda-y.un.pe/
    index.html
    sw.js

R2: websites/
  123/
    app.ABC.js
    app.XYZ.css
  456/
    app.DEF.js
    app.UVW.css
```

The HTML references company assets through
`FRONTEND_CDN/websites/<CompanyID>/`. `sw.js` stays on the storefront hostname
because browsers require service workers to be same-origin.

## Scope

This demo implements only:

1. The Serve Worker that selects an HTML folder from the request hostname.
2. A Go deployment action that deploys the Worker code and the complete
   `webpages/` Static Assets directory.
3. A Go company deployment action that receives `CompanyID`, reads its stored
   domain, prerenders the storefront, uploads JS/CSS to R2, deploys the HTML
   Static Assets, and provisions its Cloudflare Worker Custom Domain.
4. Integration with root `deploy.sh` options `[6]` and `[11]`.

## Explicitly out of scope

- Triggering a prerender from the domain form.
- KV, D1, or any hostname-to-company registry.
- Runtime company ID lookup or HTML injection.
- Public page-content API changes.
- Render-on-request, cache purges, or hydration changes.
- Customer-owned domains outside the managed Cloudflare zone.

## Architecture

```text
deploy.sh [11] + CompanyID
    |
    | backend Go CLI resolves parameters(Group=10, Key="domain")
    | and runs the prerender with the company CDN prefix
    v
temporary build
    |
    +-- JS/CSS --> R2 websites/<CompanyID>/
    |               |
    |               | loaded from FRONTEND_CDN
    |               v
    |             Browser module graph
    |
    +-- HTML + sw.js --> webpages/tienda-x.un.pe/
                            |
                            | Wrangler deploy
                            v
                      Serve Worker + Static Assets
                            |
                            | provision Custom Domain
                            v
                      tienda-x.un.pe
                            |
                            | hostname folder rewrite
                            v
                      webpages/tienda-x.un.pe/
```

Cloudflare Worker Custom Domains are preferred over manually creating DNS
records and Worker Routes. Cloudflare creates the DNS record and TLS certificate
and sends every path on that hostname to the Worker.

This is appropriate for the demo and a small number of managed hostnames.
Cloudflare currently recommends a wildcard route when a zone needs more than
100 Worker Custom Domains. That later scaling change would not affect the
hostname-folder Worker logic.

## Directory contract

Use the full normalized hostname as the folder name:

```text
frontend/webpage/cloudflare/
  src/serve-worker.ts
  webpages/
    tienda-x.un.pe/
      index.html
      sw.js
  wrangler.jsonc
  package.json

backend/exec/
  cloudflare_worker_deploy.go
  company_webpage_deploy.go
```

Rules:

- Folder names are lowercase full hostnames, without a scheme, port, path, or
  trailing dot.
- A folder must contain `index.html`.
- Requests may only resolve inside the folder selected from `url.hostname`.
- Unknown hostnames return `404`; they must never fall back to another tenant.
- Only `GET` and `HEAD` are served. Other methods return `405`.

Using the full hostname instead of only the first subdomain label avoids
collisions if another managed zone is added later.

## Serve Worker

`src/serve-worker.ts` should contain only routing and validation:

```ts
interface Env {
  ASSETS: Fetcher;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    // Static storefronts expose read-only GET and HEAD requests.
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      return new Response('Method Not Allowed', {
        status: 405,
        headers: { Allow: 'GET, HEAD' },
      });
    }

    const requestUrl = new URL(request.url);
    const hostname = requestUrl.hostname.toLowerCase();

    // Root and extensionless navigation requests use the tenant entry document.
    const requestedPath =
      requestUrl.pathname === '/' || !requestUrl.pathname.split('/').at(-1)?.includes('.')
        ? '/index.html'
        : requestUrl.pathname;

    // The hostname is the tenant key; no company lookup or mutable registry exists.
    const assetUrl = new URL(`/${hostname}${requestedPath}`, requestUrl.origin);
    const response = await env.ASSETS.fetch(new Request(assetUrl, request));

    if (response.status === 404) {
      console.warn('[serve-worker] asset not found', {
        hostname,
        requestedPath,
      });
    }

    return response;
  },
} satisfies ExportedHandler<Env>;
```

Implementation notes:

- Configure `assets.run_worker_first` so every request passes through the Worker
  before Cloudflare attempts a direct asset match.
- Configure the `ASSETS` binding to the `webpages/` directory.
- Set `assets.html_handling` to `none`; otherwise Cloudflare redirects the
  internal `/<hostname>/index.html` lookup and leaks that internal path.
- Keep `not_found_handling` disabled. A global SPA fallback could serve the wrong
  tenant's `index.html`.
- Preserve the query string only for browser/cache semantics; it does not affect
  the selected asset path.
- Do not log request headers, cookies, or query values.

## Wrangler configuration

`wrangler.jsonc`:

```jsonc
{
  // One Worker owns all provisioned storefront hostnames.
  "name": "genix-storefront",
  "main": "src/serve-worker.ts",
  "compatibility_date": "2026-06-10",
  "assets": {
    "directory": "./webpages",
    "binding": "ASSETS",
    "run_worker_first": true,
    "html_handling": "none",
    "not_found_handling": "none"
  },
  "observability": {
    "enabled": true
  }
}
```

Do not put tenant hostnames in `wrangler.jsonc`. They are created on demand by
the Go company deployment action.

## Go deployment actions

The deployment orchestration must be Go code in `backend/exec/`, not shell,
JavaScript, or an `.mjs` deployment script. Keeping it in the backend module
allows the company action to use the existing ORM and `core.Env` credentials.

Register two CLI handlers in `backend/exec/main.go`:

```text
fn-deploy-cloudflare-worker
fn-deploy-company-webpage <CompanyID>
```

Update the backend CLI argument handling so tokens after the `fn-*` command are
passed through `core.ExecArgs.Message`. The company handler parses exactly one
positive `int32` company ID and rejects missing or additional arguments.

### `fn-deploy-cloudflare-worker`

The shared `DeployCloudflareWorker()` Go function should:

1. Create `webpages/` when missing; zero tenant directories are valid so option
   `[6]` can bootstrap the Worker before the first company deploy.
2. Validate every directory name as a hostname and require its `index.html`.
3. Reject symlinks and duplicate normalized hostnames.
4. Validate `CLOUDFLARE_ACCOUNT` and `CLOUDFLARE_TOKEN` from
   `credentials.json`.
5. Run `bunx wrangler deploy` with `exec.Command`, setting
   `CLOUDFLARE_ACCOUNT_ID` and `CLOUDFLARE_API_TOKEN` only in the child process
   environment.
6. Print the Worker deployment result and tenant count.

The API token needs permission to deploy Workers and their Static Assets.

Command:

```bash
(cd backend && go run . fn-deploy-cloudflare-worker)
```

Cloudflare Static Assets are part of a Worker deployment, not an independently
writable filesystem. Therefore, adding or replacing a tenant folder requires
running this Go action again. Cloudflare deduplicates unchanged uploaded
assets.

### `fn-deploy-company-webpage <CompanyID>`

The company deployment must run these steps in order:

1. Parse and validate `CompanyID > 0`.
2. Query `parameters` using the ORM with:
   `CompanyID=<argument>`, `Group=10`, `Key="domain"`, `Status>0`.
3. Require exactly one active domain and normalize it with the same rules used
   by `POST.website-domain`.
4. Require that the hostname belongs to `ZONE_NAME` for this demo.
5. Configure R2 CORS for public `GET`/`HEAD` module requests.
6. Build into a temporary directory by invoking the prerender command with
   `exec.Command.Dir` set to `frontend/webpage`:
   `bun scripts/prerender.mjs --company <CompanyID> --asset-base
   <FRONTEND_CDN>/websites/<CompanyID> --out <temporary-directory>`.
7. Rewrite generated JS/CSS URLs to
   `FRONTEND_CDN/websites/<CompanyID>/<asset>`.
8. Upload non-HTML assets except `sw.js` to R2 under `websites/<CompanyID>/`,
   then remove those files from the Worker artifact.
9. Atomically replace `webpages/<hostname>/` with the HTML and `sw.js`.
10. Call the shared `DeployCloudflareWorker()` Go function. This uploads the
   Worker and the HTML-only `webpages/` asset collection.
11. Query existing Worker Custom Domains for the hostname.
12. Return success without mutation if it already points to
   `genix-storefront`.
13. Fail if the hostname is already attached to another Worker.
14. Create the Worker Custom Domain for `genix-storefront` through the
    Cloudflare API.
15. Poll until the Worker Custom Domain association is visible, with a bounded
    timeout. Cloudflare manages certificate issuance for the hostname.
16. Log the company ID, hostname, build path, deployment stage, and resulting
    status, but never log credentials or response bodies containing secrets.

Required credentials:

```text
CLOUDFLARE_ACCOUNT
CLOUDFLARE_TOKEN
FRONTEND_CDN=https://genix-dev-images.un.pe
ZONE_NAME=un.pe
```

The API token needs Worker Custom Domain write access for the account/zone.
Creating a Custom Domain also creates the required DNS record and certificate,
so the Go action must not separately create a DNS record.
`ZONE_NAME` identifies the Cloudflare zone and is required.

Command:

```bash
(cd backend && go run . fn-deploy-company-webpage 123)
```

The generated directory is selected from the stored domain, not from
`CompanyID`:

```text
CompanyID 123
  -> parameters domain = tienda-x.un.pe
  -> cloudflare/webpages/tienda-x.un.pe/
```

## `deploy.sh` integration

Change the menu entries to:

```text
[6] Desplegar: Tablas, Datos Iniciales, Cloudflare Worker
[10] Deploy Cloudflare Worker
[11] Deploy Company Webpage
```

### Action parsing

The current substring checks cannot support option `11`: the value `11` also
matches option `1`. Replace them with exact action tokens and a `has_action`
helper.

Accept space- or comma-separated actions:

```text
6
11
1 2 3
6,11
```

Do not keep the old compact `123` format because it is ambiguous once
multi-digit options exist.

### Option `[6]`

Option `[6]` must execute the complete pipeline and stop immediately if any
step fails:

1. Regenerate `controllers.generated.go`.
2. Run `(cd backend && go run . fn-homologate)`.
3. Run `(cd backend && go run . fn-init)`.
4. Run `(cd backend && go run . fn-deploy-cloudflare-worker)`.

The menu description must match this real behavior. Shared table deployment
logic should be a shell function reused by options `[5]` and `[6]`, so `[6]`
does not duplicate the commands.

### Option `[10]`

Option `[10]` deploys only the shared Cloudflare Worker:

```bash
(cd backend && "$GO_PATH" run . fn-deploy-cloudflare-worker)
```

### Option `[11]`

When `[11]` is selected:

1. Read `CompanyID` as a separate argument or prompt for it interactively.
2. Validate it as a positive integer before starting the backend.
3. Run:

```bash
(cd backend && "$GO_PATH" run . fn-deploy-company-webpage "$COMPANY_ID")
```

Support a non-interactive form for automation:

```bash
./deploy.sh 11 123
```

The shell only validates and forwards `CompanyID`; all domain lookup, build,
Cloudflare deployment, provisioning, and logging remain in Go.

## Domain change behavior

For this demo, changing `old.un.pe` to `new.un.pe` and then running `[11]`:

1. Resolves `new.un.pe` from the database.
2. Generates `webpages/new.un.pe/`.
3. Deploys the Worker assets.
4. Provisions `new.un.pe`.
5. Leaves the old Custom Domain and old artifact folder untouched.

Removing obsolete domains and folders is a separate cleanup feature. The new
domain is deployed first so a failed prerender or certificate issue does not
take the existing storefront offline.

## Validation

Local:

- `wrangler dev` serves a known tenant when the request `Host` matches its folder.
- `/` returns that tenant's `index.html`.
- JS and CSS requests resolve from the same tenant folder.
- An extensionless route returns that tenant's `index.html`.
- An unknown hostname returns `404`.
- A missing asset returns `404`, without cross-tenant fallback.
- `POST` returns `405`.

Deployed:

- `curl https://tienda-x.un.pe/` contains the expected prerendered SEO tags.
- JS and CSS return their correct content types.
- `tienda-x.un.pe` and `tienda-y.un.pe` cannot access each other's files.
- The Custom Domain reports an active certificate.
- Re-running `[6]` deploys the same Worker safely.
- Re-running `[11]` for the same company rebuilds its folder and treats the
  existing correct Custom Domain as success.
- `[11]` rejects missing, zero, negative, nonnumeric, and extra arguments.
- `[11]` fails before deployment when the company has no active stored domain.
- Selecting `11` does not execute option `1`.

## Implementation order

1. Create the `cloudflare/` package and Wrangler configuration.
2. Implement and test the minimal Serve Worker.
3. Add backend CLI argument forwarding into `core.ExecArgs.Message`.
4. Implement the shared Go `DeployCloudflareWorker()` function and
   `fn-deploy-cloudflare-worker`.
5. Implement `fn-deploy-company-webpage`: ORM domain lookup, prerender,
   Worker deployment, and idempotent Custom Domain provisioning.
6. Read the existing `ZONE_NAME` credential in backend and frontend deployment configuration.
7. Replace `deploy.sh` substring matching with exact action tokens.
8. Update option `[6]` and add option `[11]` with `CompanyID` forwarding.
9. Deploy one demo company and run the validation checklist.

## Cloudflare references

- [Workers Static Assets](https://developers.cloudflare.com/workers/static-assets/)
- [Static Assets binding](https://developers.cloudflare.com/workers/static-assets/binding/)
- [Worker Custom Domains](https://developers.cloudflare.com/workers/configuration/routing/custom-domains/)
- [Workers platform limits](https://developers.cloudflare.com/workers/platform/limits/)
- [Wrangler configuration](https://developers.cloudflare.com/workers/wrangler/configuration/)
