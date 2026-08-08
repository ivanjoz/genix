# Set GitHub frontend variables

`set-github-frontend-vars.ts` reads the public frontend configuration from the
root `config.toml` and sets the matching GitHub Actions repository variables
for the repository resolved by `gh`:

- `github_account` selects the stored `gh` account used for the operation
- `aws.lambda_url` → `PUBLIC_LAMBDA_URL`
- `sse_bridge.url` → `PUBLIC_SSE_BRIDGE_URL` (opcional; sin bridge cae a `aws.lambda_url`)
- `frontend.cdn_url` → `PUBLIC_FRONTEND_CDN`
- `frontend.zone_name` → `PUBLIC_ZONE_NAME`
- `[[endpoints]]` → `PUBLIC_ENDPOINTS` as compact JSON

No passwords, tokens, database credentials, or other backend secrets are sent to
GitHub. The script requires Bun and an authenticated GitHub CLI account with
permission to edit Actions variables in the repository. It retrieves the token
for `github_account` from the local `gh` keyring and passes it only through the
child-process environment, so the globally active `gh` account is not changed.

Run it from the deployment menu with option **14**:

```bash
# Configure every public variable in one action.
./deploy.sh 14
```

To validate the config and GitHub repository without changing variables:

```bash
# Exercise parsing, authentication, and repository detection only.
bun run scripts/set-github-frontend-vars.ts --dry-run
```
