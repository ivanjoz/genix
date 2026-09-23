# RATIONALE — exec

Design decisions for the one-off scripts, newest first.

## `identity_doc_type` was migrated in the database by hand, not by the ORM

**Context** — the column existed as `text` from before it became `int8` in Go, so the backfill
died on `can not marshal int8 into varchar`. genix-orm's deploy only ever issues `ALTER TABLE …
ADD`: it never changes an existing column's type, so `fn-homologate` is silent about the
mismatch. Cassandra 3.0.8 cannot alter `text` to `tinyint` either, and `client_provider` feeds a
materialized view that selects the column, which blocks a `DROP COLUMN` outright.

**Decision** — `DROP MATERIALIZED VIEW client_provider__pk_status_type_updated_version_rng_view`,
then `DROP`/`ADD` the column as `tinyint` on both `client_provider` and
`client_provider_snapshot`, then `fn-homologate` to recreate the view. Verified first that every
one of the 155 rows held `null` there and that the snapshot table was empty, so nothing was
converted and nothing was lost; the view rebuilt to 155 rows on its own.

**Rationale** — the type change is the user's, and the stored shape had to follow it rather than
the code bending back to `text` (CLAUDE.md §3: fix the data, never patch the flow). A drop/re-add
is only safe because the column was empty — with real values in it this would have had to be a
read-convert-write pass instead. Cost: the same mismatch will resurface on any other deployment
of this schema, and nothing in `fn-homologate` will report it. The tell is a marshal error at
write time, not at deploy time.

## The identity backfill rewrites `Status` and `Type` it does not change

**Context** — updating only `IdentityDocType` and `SnapshotID` panics inside the ORM: the delta
view is keyed on `(status, type, updated_version)`, `db.Update` always bumps the watermark, and a
partial update that moves one column of a composite key without the others is refused.

**Decision** — both columns are listed in the update with the values the read returned.

**Rationale** — the alternative is a full-row update, which would write a dozen columns this
script has no business touching on historical rows. Naming the two the view needs keeps the write
to four columns and says why they are there. Cost: if a row changes status between the read and
the write, this run writes the stale value back — acceptable for a one-off against data nobody
else is editing.

## The identity backfill reports before it writes, and pins to today's identity

**Context** — `fn-backfill-identity-snapshots` fills three columns that were added after the data
existed: `ClientProvider.IdentityDocType`, `ClientProvider.SnapshotID` and
`InvoiceDocument.ClientSnapshotID`. It writes to historical accounting rows, and the snapshot
table it feeds is append-only — nothing ever deletes a row from it.

**Decision** — the default invocation counts and writes nothing; `apply` is what writes. It
touches every company regardless of `Status`, and every snapshot it mints is stamped
`CreatedBy = -1`.

**Rationale** — a script whose whole job is to stamp filed accounting data should be readable
before it is trusted, and the counts are the only thing that says whether the run is going to do
what was expected. The status filter would have been wrong in both directions: every company in
this database sits at `Status = 0`, and a tenant that stopped operating still has to produce the
books for the periods it did operate in. `-1` is not a user id, which is the point — it is what
lets a later reader tell a reconstructed identity from a declared one. Cost: two invocations
instead of one, and one magic id that means "no human wrote this".

## A client that gains a document type is re-resolved, not just stamped

**Context** — the backfill derives `IdentityDocType` where it is 0. Some of those rows already
carry a `SnapshotID`, so the obvious move is to write the derived type and leave the pin alone.

**Decision** — any row whose document type changes is sent back through
`ResolveIdentitySnapshots`, even when it already had a snapshot.

**Rationale** — the snapshot table is content-addressed on a hash that covers `IdentityDocType`
(`ClientProviderSnapshot.SelfParse`). A pin resolved while the type was 0 names an identity built
from the empty value, which is a *different* identity from the one the row presents once the type
is filled. Leaving it would make the client and its snapshot disagree about who the client is —
silently, and only visible in a book months later. Cost: the backfill mints snapshot rows for
identities that look like they already had one.

## A comprobante with no identified buyer keeps its pin at zero

**Context** — the document pass reaches each comprobante's buyer through the sale it bills. Some
sales name no client: someone paid at the till and left.

**Decision** — those documents are counted and skipped, not filled.

**Rationale** — `ClientSnapshotID = 0` is not a missing value on those rows, it is the declared
one: the document printed `CLIENTES VARIOS` / `00000000`, and Anexo 3 makes fields 11-13 optional
for a sub-S/700 boleta. Inventing an identity now would make the Registro de Ventas print
something the comprobante never said, which is worse than the blank. The historical name for the
buyers that *were* identified is a separate loss and unavoidable: it was never stored, so the
backfill pins the identity the client presents today and the freeze only holds from there
forward.
