# thirdparty — forked dependencies

One dependency is vendored here because the backend cannot use it unmodified, and it is not
optional: **without it the Go linker disables dead-method elimination for the entire binary, which
costs ~12 MB.**

Do not "tidy" it back to upstream. Read this file first.

| Fork | Upstream | Patched |
| --- | --- | --- |
| `protobuf/` | `google.golang.org/protobuf@v1.36.11` | 3 files, ~6 lines |

There is a second fork, `gocql`, in **`genix-orm/thirdparty/`**. It lives there because gocql is
genix-orm's dependency, not the app's — only `scylla/` imports it. The app's `go.mod` still has to
name that directory in a `replace`, because a replace in a non-main module is ignored; see that
directory's README.

Measurements and the full investigation: `docs/BINARY_SIZE_PLAN.md`.

---

## Why

The linker keeps one program-global flag, `reflectSeen`
(`cmd/link/internal/ld/deadcode.go`). It is set the moment any reachable function calls
`reflect.{Type,Value}.Method` / `.MethodByName` with a **non-constant** argument, because the linker
then cannot know which method will be looked up at runtime. While it is set, every exported method
of every reachable type is retained.

The ORM is the largest payer, having by far the most `(type × exported methods)`: `db.Col` alone has
258 instantiations and drops from 1,672,704 bytes to 313,152 once the flag is clear.

Two independent chains were setting it. The flag is boolean, so patching either one alone changes
nothing — both were measured at exactly zero on their own.

### Chain 1 — `internal/descfmt`

`descfmt/stringer.go` formatted a `ServiceDescriptor` with `rv.MethodByName("Methods")`. The name is
constant, so it emits `R_USENAMEDMETHOD` rather than setting the flag directly — but the linker's
`genericIfaceMethod` set is keyed **by name only**, so it retains every exported method called
`Methods` on every reachable type. That includes `reflect.Value.Methods`, the iterator added in
Go 1.27, which calls `v.Method(i)` with a non-constant index and is *absent from the exclusion list*
in `cmd/compile/internal/walk.usemethod`. Retaining it makes it reachable; reaching it sets the flag.

The patch takes the method value directly (`reflect.ValueOf(t.Methods)`). Same value, no name.

### Chain 2 — `internal/impl` and `xunsafe`

`impl` probed for proto1 legacy hooks with `t.MethodByName("XXX_OneofWrappers")`, where `t` is a
`reflect.Type` — an **interface**. That registers the methodsig
`{MethodByName, func(string) (Method, bool)}` in the linker's `ifaceMethod` set.
`github.com/viant/xunsafe.Field` embeds `reflect.Type`, so the compiler generates a promoted
`MethodByName` wrapper whose argument is not constant, that wrapper matches the registered
signature, and retaining it sets the flag.

The patch goes through `reflect.Value.MethodByName` instead — a concrete struct method, so no
interface registration, and a different signature that the xunsafe wrapper does not match. Same
method, same zero receiver, **proto1 support unchanged**.

The alternative was patching xunsafe to stop embedding `reflect.Type`. Patching protobuf was
chosen because it cuts both chains in one fork and leaves the ORM's dependencies untouched.

---

## Working on these forks

**Upgrading.** Bump the version in `../go.mod` *and* in `regenerate.sh`, then:

```sh
./regenerate.sh                 # rebuild the fork, re-apply patches
cd .. && go build ./... && go test ./thirdparty/
```

If a patch no longer applies, fix the `.patch` file in `patches/` against the new upstream — do not
hand-edit the vendored tree, or the next regeneration silently drops your change.

**Adding a protobuf package.** The fork is trimmed to the 33 packages the build reaches, out of 491.
If new code imports a protobuf package that was trimmed away, the build fails with
`no required module provides package …`. Refresh the list from a working tree:

```sh
./regenerate.sh refresh-packages && ./regenerate.sh protobuf
```

**`go mod tidy` does not run against this fork, by design.** Tidy walks the *test* imports of every
dependency, and `google.golang.org/grpc/status`'s own tests reach
`protobuf/testing/protocmp` and `protobuf/reflect/protodesc` — two of the 458 packages the trim
drops. Tidy therefore exits 1 with `module google.golang.org/protobuf@latest found ..., but does not
contain package ...`. Nothing is broken: `go build ./...`, `go vet ./...` and `go test ./...` all
resolve only non-test imports and are unaffected. `start.js` runs `go mod download` for this reason.

Carrying those two packages just to satisfy tidy is not worth it — `protodesc` drags in the
~250 KB generated `types/descriptorpb`, and `protocmp` adds a `github.com/google/go-cmp` require —
all of it dead weight no shipping code reaches. If you genuinely need to re-tidy after adding or
dropping a dependency, comment out the protobuf `replace` in `../go.mod` first:

```sh
cd .. && sed -i 's|^replace google.golang.org/protobuf|// &|' go.mod
go mod tidy                       # resolves against upstream v1.36.11
sed -i 's|^// replace google.golang.org/protobuf|replace google.golang.org/protobuf|' go.mod
go build ./... && go test ./thirdparty/
```

Tidy moves the `google.golang.org/protobuf` require out of the `// indirect` block while the replace
is off; that is cosmetic and can be left as tidy wrote it.

**Verifying the forks still do their job.** Binary size is a poor signal — removing a template
package shrinks the binary without clearing the flag. Count concrete `Col` instantiations instead:

```sh
# The tags are not optional here: without them net/rpc still sets the flag and the count
# stays high even though the forks are intact.
cd .. && go build -tags 'lambda.norpc,grpcnotrace' -o /tmp/uns . && go tool nm -size /tmp/uns > /tmp/syms.txt
rg 'genix-orm/db\..*Col\[' /tmp/syms.txt | rg -v 'go\.shape' | wc -l
# ~2,064 = elimination is on (good).  ~8,500 = a patch was lost, or the tags are missing.
```

`go test ./thirdparty/` checks the cheaper invariants on every run: that each patch is still present,
that the shipping build tags are declared in both deploy paths, and that patched protobuf still
round-trips the qdrant messages this repo sends. The gocql fork has its own equivalent test in
`genix-orm/thirdparty/`.

**The build tags are part of the same fix.** `lambda.norpc` and `grpcnotrace` each drop another
`reflectSeen` source, and the flag is boolean — the forks buy nothing without them. They are
declared in `cloud/main.go` (`BuildTags`) and `scripts/deploy_vps.go` (`backendBuildTags`), which the
tests keep in sync.

---

## Licensing

The fork keeps its upstream `LICENSE` and `PATENTS` and remains under those terms. Every modified
hunk is marked `PATCHED (genix)` with the reason inline.
