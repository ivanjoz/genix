# RATIONALE — thirdparty

## start.js runs `go mod download`, not `go mod tidy`
**Context** — `bun start.js` failed at "Instalando los paquetes de Go" the first time it ran after
the trimmed protobuf fork landed. `go mod tidy` resolves the test imports of dependencies, and
`grpc/status`'s tests reach `protobuf/testing/protocmp` and `protobuf/reflect/protodesc`, which the
trim removes — so tidy exits 1 against a tree that builds, vets and tests fine.
**Decision** — `start.js` now runs `go mod download`. The stamp file in `tmp/` that gated the tidy
call was deleted along with it. The `go mod tidy` limitation and its workaround are documented in
`README.md`.
**Rationale** — Tidy was the wrong command for what that step does ("install the Go packages if they
are not already there"): it rewrites `go.mod`/`go.sum` as a side effect, which is what invalidated
the build cache and forced the stamp file in the first place. `go mod download` fetches exactly the
requires already in `go.mod`, touches nothing, and measured 23 ms warm — so the stamp became dead
code. The alternative, carrying `protocmp` and `protodesc` in the fork, adds the ~250 KB generated
`descriptorpb` and a `go-cmp` require to satisfy an import path no shipping code reaches. Cost:
`go mod tidy` needs the protobuf `replace` commented out to run at all, which is only felt when a
dependency is added or dropped.

## Named thirdparty/, not vendor/
**Context** — Two patched dependency forks need a home in the repo. `vendor/` is the obvious name.
**Decision** — `backend/thirdparty/`, with a `replace` per fork in `../go.mod`. `vendor/` is
unusable.
**Rationale** — `vendor/` is reserved: the mere existence of that directory at the module root
switches the toolchain into vendor mode, which then requires a `vendor/modules.txt` covering *every*
dependency ("inconsistent vendoring ... is replaced in go.mod, but not marked as replaced in
vendor/modules.txt"). Using it would mean vendoring the whole graph -- the AWS SDK, grpc, qdrant,
gocql -- for the sake of a six-line diff. Worse, `go mod vendor` *regenerates* that tree, so a
routine command would silently drop the patches and cost 12 MB with nothing failing, which is the
exact failure mode the tests in this directory exist to catch. `thirdparty/` is touched only by
`regenerate.sh`, which re-applies the patches and then verifies they landed.

## Vendor and patch protobuf rather than patch xunsafe or drop the qdrant client
**Context** — The linker's global `reflectSeen` flag was set, retaining every exported method of
every reachable type and costing 12,058,624 bytes (27.3% of the binary). Two independent chains set
it, both routed through `google.golang.org/protobuf`, which enters the graph through exactly one
importer: `github.com/qdrant/go-client`. Three fixes were possible — patch protobuf, patch
`github.com/viant/xunsafe`, or remove the qdrant Go client and talk to Qdrant over REST.
**Decision** — Vendored a trimmed `google.golang.org/protobuf` under `thirdparty/protobuf` and
patched three files. Nothing else in the repo changed: xunsafe, genix-orm and all application code
are untouched.
**Rationale** — Patching protobuf cuts *both* chains, measured; patching xunsafe cuts only one and
leaves the binary unchanged on its own. Dropping the qdrant client is ~3 MB better still (it removes
grpc and protobuf outright, 1,527,744 bytes of text) but costs a hand-written REST client for the
hybrid-search path and owns a protocol surface forever. The fork is 39,385 inert lines against a
6-line diff, and reverts with one `go.mod` line. If the qdrant client is ever replaced, this fork
should be deleted rather than kept.

## Route the proto1 probes through reflect.Value instead of stubbing them
**Context** — Chain 2 was `internal/impl` calling `MethodByName` through the `reflect.Type`
*interface*, which registers a methodsig that `xunsafe.Field`'s promoted wrapper matches. The first
working patch replaced the three probes with a stub returning "not found", which cut the chain but
silently removed proto1 legacy oneof and extension-range discovery.
**Decision** — Call `reflect.Value.MethodByName` on a zero value of the same type instead. Behaviour
is identical; proto1 support is retained.
**Rationale** — The stub was measured at the same binary size, so the only difference was risk. A
pre-2016 `github.com/golang/protobuf` message would have lost its oneof metadata *silently* rather
than failing — and silent is the problem, not proto1. `reflect.Value` is a concrete struct, so the
call registers no interface methodsig, which is the whole mechanism; the constant method name still
emits a harmless `R_USENAMEDMETHOD`. Costs one extra `reflect.Zero` per message type, once, at
init.

## Trim the protobuf fork to reachable packages, from a checked-in list
**Context** — Vendoring protobuf whole is 15 MB and 491 files. Trimming to what the build reaches is
1.6 MB and 145 files, but the reachable set has to come from somewhere at regeneration time.
**Decision** — `patches/protobuf-packages.txt` holds the 33 reachable packages, checked in.
`regenerate.sh refresh-packages` rewrites it from the live dependency graph; `regenerate.sh` reads it.
**Rationale** — Deriving the list during regeneration needs `go list` to resolve the `replace`
target, which is the very tree being rebuilt — the first attempt did exactly that and produced a
truncated fork that failed to compile. The list is the union of the tagged and untagged builds, so a
development `go build` cannot reach a package that was trimmed away. Cost: importing a new protobuf
package needs one refresh command, and the failure when you forget is a clear
`no required module provides package`.

## Verify the patches from the script and from a test, not from binary size
**Context** — The first `regenerate.sh` used `git apply` from inside `thirdparty/protobuf`. git
resolves patch paths against the repository root and silently ignores paths outside the current
subdirectory, so it applied nothing and exited 0. The binary grew back by 11 MB with no error
anywhere.
**Decision** — `git apply` now runs from the repository root with `--directory`, the script asserts
that exactly three files carry the `PATCHED (genix)` marker afterwards, and
`thirdparty/patches_test.go` asserts each patch is present, that `gocql/recreate.go` has not
returned, and that both deploy paths declare the build tags.
**Rationale** — This class of change fails silently by construction: a dropped patch still compiles
and still passes every functional test. Binary size is also a poor detector, because removing a
dependency shrinks the binary without clearing `reflectSeen` — the two are easy to confuse and the
earlier investigation did confuse them. The tests encode the invariant directly instead.

## The gocql fork was moved out to genix-orm
**Context** — `recreate.go` (which links `text/template` for an unused helper) was first deleted in
a gocql fork placed here, alongside protobuf, because both were done in one pass.
**Decision** — Moved to `genix-orm/thirdparty/gocql`, with its own README, regeneration script and
test. This directory now holds protobuf only.
**Rationale** — gocql is imported only by `genix-orm/scylla`; the app never names it, and
genix-orm's `go.mod` already declared the scylladb swap itself. A fork belongs with the module that
owns the dependency, or upgrading the ORM means editing a directory in the app's repo. protobuf is
the mirror image and correctly stays here: it arrives through `qdrant/go-client`, which genix-orm
never touches. The one cost is that the app's `go.mod` must repeat the replace against
`./genix-orm/thirdparty/gocql`, because Go ignores replaces in non-main modules — a duplicated line
pointing at one tree, guarded by a test on the genix-orm side.
