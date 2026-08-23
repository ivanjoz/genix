## The last two cross-module calls move into `business/types`, and `agent/webpage` is renamed

**Context** — two violations remained. `sales` imported `app/business` for
`SaveClientProviders` (it resolves or creates the buyer while recording a sale), and
`agent/webpage` imported it for `FindImageCandidates` (the page-builder's image picker).

**Decision** — `SaveClientProviders` plus its private validators
(`companyRegistryNumberPattern`, `isValidEmailAddress`, used nowhere else) moved to
`business/types/client_provider_save.go`. The whole of `image_assets_agent.go` moved to
`business/types/image_assets_agent.go`. `imageAssetCategoryGroupID` moved with it into
`business/types/image_assets.go` and had to be **exported** as `ImageAssetCategoryGroupID`,
because six call sites in the `business` body still read it.

Separately, `agent/webpage` is now `agent/pagebuilder`.

**Rationale** — `imageAssetCategoryGroupID` is a schema constant: it names the group partition
every image asset lives in, so it belongs next to the table it partitions rather than in the sync
job that happened to declare it. Exporting it is the one avoidable-looking cost of the move, and
it is the honest one: the constant is genuinely shared between the module body and the types
package now.

The rename removes a real trap. `app/agent/webpage` (the page-builder loop) and `app/webpage`
(the storefront module) were two unrelated packages with the same name. Nothing imported both, so
it compiled — but any file that ever needed both would have required an alias, and `rg webpage`
spanned two unrelated domains. `pagebuilder` says what it is.

**All six violations from `MODULE_BOUNDARIES_PLAN.md` are now closed.** The only packages that
import a module body are the composition roots: root `main`, `exec` and `tests/sample_records`.

