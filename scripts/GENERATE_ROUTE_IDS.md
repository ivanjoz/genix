# Generate Route IDs Script

Assigns one stable `int16` to every API route and writes them to `backend/core/api_routes.generated.go`.

## Why

A route is a string everywhere in the backend, which is right for a router and wrong for the two
places that now need it. A CloudWatch line pays 27 bytes for `"POST.almacen-producto-stock"` on
every entry, and `user_logs.frame_route_company_agg` packs the route into exactly two bytes
alongside the 15-minute frame and the company — a string cannot go in an integer key at all.

**An ID is handed out once and never reused.** A route that disappears keeps its number, annotated
`// retired`, because rows written last month still name it; recycling that number onto a different
route would silently rewrite what those rows mean.

## Usage

```bash
./app.sh generate_route_ids           # regenerate and write
./app.sh generate_route_ids --check   # write nothing, exit 1 if the file is stale
```

It runs automatically before **[2] Backend (AWS Cloud)** and **[3] Backend (VPS)** in the deployer,
so a route added today cannot reach production without a number. `--check` is for CI or a pre-commit
hook, where the point is to fail rather than to fix.

## What It Does

1. Walks `backend/` with `go/parser`, collecting the keys of every
   `var ModuleHandlers = core.AppRouterType{...}` literal. Parsing the source rather than running
   the binary keeps the generator usable while the backend does not compile — which is exactly when
   a new route is being added.
2. Parses the existing generated file to recover the IDs already assigned. The generated file is
   its own registry; there is no second source of truth to drift.
3. Keeps every existing assignment untouched, sorts the new routes alphabetically, and numbers them
   from the current maximum. Alphabetical ordering is what makes two branches adding routes in the
   same week produce the same file rather than a merge conflict.
4. Marks routes that are in the file but no longer in the source as `// retired`, keeping their ID.
5. Emits **one** literal map, `APIRouteIDs`, plus `MaxAPIRouteID` and the `APIRouteID(funcPath)`
   lookup. The reverse direction, `APIRouteNames`, is built by an `init()` that inverts it at
   start: a second generated literal would be a second thing to keep in step, and this one is
   exactly the first backwards. Retired routes stay in both — that is the only thing that keeps an
   old row readable.

`APIRouteID` returns **0** for an unknown route: a 404, or one added since the last generation.
Zero is never a valid assignment, so a `route_id` of 0 in `user_logs` always means "unrecognised".

## Failure Modes

The generator refuses to write, rather than guessing, when:

- no routes are found at all (an empty scan would retire the entire registry);
- a `ModuleHandlers` key is not a plain string literal;
- an existing entry has a non-literal or non-positive ID.

A missing output file is not an error — that is the first run.
