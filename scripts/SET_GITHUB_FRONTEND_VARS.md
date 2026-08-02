# Set GitHub frontend variables

`set-github-frontend-vars.ts` reads the public frontend configuration from the
root `credentials.json` and sets the matching GitHub Actions repository variables
for the repository resolved by `gh`:

- `GITHUB_ACCOUNT` selects the stored `gh` account used for the operation
- `LAMBDA_URL` → `PUBLIC_LAMBDA_URL`
- `FRONTEND_CDN` → `PUBLIC_FRONTEND_CDN`
- `ZONE_NAME` → `PUBLIC_ZONE_NAME`
- `ENPOINTS` → `PUBLIC_ENDPOINTS` as compact JSON

No passwords, tokens, database credentials, or other backend secrets are sent to
GitHub. The script requires Bun and an authenticated GitHub CLI account with
permission to edit Actions variables in the repository. It retrieves the token
for `GITHUB_ACCOUNT` from the local `gh` keyring and passes it only through the
child-process environment, so the globally active `gh` account is not changed.

Run it from the deployment menu with option **14**:

```bash
# Configure all four variables in one action.
./deploy.sh 14
```

To validate the credentials and GitHub repository without changing variables:

```bash
# Exercise parsing, authentication, and repository detection only.
bun run scripts/set-github-frontend-vars.ts --dry-run
```
