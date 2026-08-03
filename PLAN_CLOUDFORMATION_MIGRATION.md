# Plan: replace CDK with a plain CloudFormation template

## Decisions (agreed)

| Topic | Decision |
|---|---|
| IaC tool | Drop CDK entirely. One committed CloudFormation template + a `cloudformation` API call from the existing Go deploy tool. |
| Logical IDs | Clean, hand-written names. No CDK hashes. |
| Stack identity | **New stack.** `APP_NAME` in `credentials.json` gets bumped (`genix-2` → e.g. `genix-3`) so every physical name is new and nothing collides with the RETAINed leftovers of the old stack. |
| Lambda runtime | `provided.al2023` (was `provided.al2`). |
| Function URL invoke mode | `RESPONSE_STREAM` on both URLs, `LAMBDA_RESPONSE_STREAMING=1` in both function environments. Backend streaming code stays as-is. |

## Why this makes sense

The CDK usage here is thin — 17 static resources, no assets, no constructs, no cross-stack refs. It costs:

- `npx --yes aws-cdk@latest` downloaded on every `deploy` (requires Node on the deploy machine)
- the jsii runtime plus ~25 CDK Go modules in `cloud/go.mod`
- a CDK bootstrap stack, which the synthesized template *hard-asserts* on via an SSM parameter rule

None of that buys anything for a static resource list. `aws-sdk-go-v2/service/cloudformation` is already a dependency, so the deploy becomes one API call with no Node and no bootstrap.

## Out of scope

- `p2p/` has its own separate CDK project (`p2p/deploy.sh`). Untouched.
- `cloud/cloudflare_infra.go` (the `CLOUD_PROVIDER=cloudflare` path). Untouched.
- Backend response code. Streaming stays.

---

## 1. New file: `cloud/template.yml`

The full stack, parameterized. Values come from `credentials.json` and are passed as CFN parameters rather than string-interpolated, so the template stays static and committed.

**Parameters:** `NamePrefix`, `FrontendBucketName`, `DeploymentBucket`, `CompiledS3Key`, `LambdaIamRole`, `AppCode`.

**Resources** (clean IDs, replacing the hashed CDK ones):

| New logical ID | Type | Was |
|---|---|---|
| `FrontendBucket` | `AWS::S3::Bucket` (Retain) | `FrontendBucketEFE2E19C` |
| `FrontendOriginAccessIdentity` | `AWS::CloudFront::CloudFrontOriginAccessIdentity` | `OAIE1EFC67F` |
| `FrontendBucketPolicy` | `AWS::S3::BucketPolicy` | `FrontendBucketPolicy1DFF75D9` |
| `FrontendDistribution` | `AWS::CloudFront::Distribution` | `CloudfrontFrontendA1B4DC03` |
| `BackendFunction` | `AWS::Lambda::Function` (192 MB) | `LambdaGO79EDB5AF` |
| `BackendFunctionUrl` | `AWS::Lambda::Url` | `LambdaGOFunctionUrlE15EE033` |
| `BackendFunctionUrlPermission` + `BackendFunctionInvokePermission` | `AWS::Lambda::Permission` | the two `LambdaGOinvoke*` |
| `BackendLogGroup` | `AWS::Logs::LogGroup` | `LambdaGOLogGroup89A64633` |
| `BackendHeavyFunction` + URL + 2 permissions + log group | same set, 2048 MB | the `LambdaGOn2*` set |
| `CronRule` | `AWS::Events::Rule` (every 10 min, body `exec:cron`) | `EventBridgeRule15224D08` |
| `CronRuleInvokePermission` | `AWS::Lambda::Permission` | the long `EventBridgeRuleAllow…` |
| `MainTable` | `AWS::DynamoDB::Table` + 5 GSIs (Retain) | `MainTable74195DAB` |

Everything else — CloudFront cache-policy IDs, custom error responses, CORS, GSI shapes, TTL attribute, retention — is transcribed verbatim from the current stack so behaviour is unchanged.

Two deliberate changes, per the decisions above:
- `Runtime: provided.al2023`
- `AWS::Lambda::Url` → `InvokeMode: RESPONSE_STREAM`, and `LAMBDA_RESPONSE_STREAMING: "1"` in both function environments. These two must always agree — a mismatch fails every request — so the template is the single place that sets both, with a comment saying so.

`CDKMetadata`, the `BootstrapVersion` parameter and the `CheckBootstrapVersion` rule are all dropped.

**Outputs:** `BackendUrl`, `BackendHeavyUrl`, `FrontendDistributionDomain`, `FrontendBucket`. More useful than CDK's single `WebsiteURL` — these are the values that go back into `credentials.json`.

No `Capabilities` are needed: the stack creates no IAM resources, it consumes an existing role by ARN.

## 2. New file: `cloud/cloudformation.go`

Replaces `cdk_infra.go`. Roughly:

- `//go:embed template.yml` — template ships inside the binary.
- `DeployCloudFormation(params)`:
  1. `DescribeStacks` to decide create vs. update.
  2. `CreateStack` / `UpdateStack` with the parameters built from `DeployParams`.
  3. Swallow the `No updates are to be performed` validation error as success.
  4. Poll `DescribeStackEvents` every 5s and print each **new** event (time, logical ID, status, reason) until a terminal stack status. This replaces the progress output CDK used to give and satisfies the extensive-logging rule; failures name the exact resource that broke.
  5. On success print the stack outputs.

## 3. Edits to `cloud/main.go`

- Delete the `"cdk"` argument branch and the `validArgs` entry — no synth step exists anymore.
- `DeployIfraestructure`: replace the `npx aws-cdk deploy` shell-out with `DeployCloudFormation(params)`. The `CompileBackendToS3(params, true)` call before it stays.
- **Bug fix in `UpdateEnviromentVariables`** (action `2`): it currently rewrites the whole environment to `{APP_CODE, CONFIG}`, which would silently **wipe `LAMBDA_RESPONSE_STREAMING`** from a deployed function and break every request against a `RESPONSE_STREAM` URL. Add the flag to that map. This bug is live today; it becomes fatal once streaming is on.

## 4. Deletions

- `cloud/cdk_infra.go`
- `cloud/cdk.json`
- `cloud/cdk.out/` (also stale — it predates commit `bdbb0bad`)
- CDK deps from `cloud/go.mod`: `aws-cdk-go/awscdk/v2`, `constructs-go`, `jsii-runtime-go`, the three `cdklabs/*`, and transitives. Then `go mod tidy` and `go build ./...` to verify.

## 5. Docs

Add a short **Lambda Deployment** section to `DEPLOYMENT.md` (the heading exists but is empty): the three actions, what the template creates, and the post-deploy step below.

---

## Manual steps for you, in order

1. Bump `APP_NAME` in `credentials.json` (`genix-2` → `genix-3`).
2. Run action `3` to create the new stack.
3. Copy the new `BackendUrl` into `LAMBDA_URL`, and the new CloudFront domain into `FRONTEND_CDN`, in `credentials.json`. Both currently point at `genix-2` resources.
4. Re-upload the frontend to the new `genix-3-frontend` bucket.
5. Once the new stack is verified, delete `genix-2-stack`. Its bucket, DynamoDB table and log groups are `Retain` and will survive as orphans — delete them by hand when you're sure you don't need them.

## Added during execution, beyond the plan above

1. **`LAMBDA_URL` write-back** (requested). After a successful deploy the tool writes the
   `BackendUrl` output into `credentials.json`, printing the old and new values when they
   differ. Done by regex on the raw text, not by re-serializing the JSON, so key order and
   hand-maintained formatting survive. If the previous value was not a `*.lambda-url.*` host
   the tool warns that a custom domain was just overwritten.
2. **`APP_CODE` inconsistency fixed.** The template inherited `gerp-prd` (hyphen) from CDK
   while action `2` sent `gerp_prd` (underscore) — so the deployed value flipped depending on
   which action ran last. `backend/core/security.go:206` derives `IS_PROD` from
   `strings.Contains(APP_CODE, "_prd")`, meaning the hyphen form silently ran production as
   non-prod. Both now read one constant, `appCodeEnvValue = "gerp_prd"` in `cloud/main.go`.
3. Log groups are `DeletionPolicy: Delete` rather than CDK's `Retain`. Retained log groups are
   part of what forced this stack rename; logs are disposable and orphans block recreation.

## Verification I will run

- `go build ./...` in `cloud/`
- `aws cloudformation validate-template` against the new template — **only if you want me to**; it is a live AWS API call under your profile. Otherwise I'll validate the YAML parses and self-review the resource properties.

I will not deploy anything.
