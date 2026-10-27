# RATIONALE — core

Design decisions, newest first.

## The DataFrame lock is a queuing fareward lock through a new AcquireLockWithOptions
**Context** — A read that finds a DataFrame partition due must wait up to 40 s for the compaction
under way, but `fareward.AcquireLock` fixes a 5 s wait and a 15 s lease.
**Decision** — fareward's Go client exports `AcquireLockWithOptions(ctx, action, identifier,
LockOptions)`. `acquireDataFrameLock` calls it with `ActionCompactDataFrame` (renamed from
`ActionMaterializeDataFrame`, same number 6), keyed on fnv64 of the frame partition's folder, the
wait and lease the ORM asks for (40 s each; the daemon clamps leases to 60 s) and at most 16
waiters per key. Past 16 a read is refused the lock and answers from the log, which stays correct.
**Rationale** — The daemon's own queue does the waiting: no polling of a lock timestamp. Costs one
exported function in the fareward repo, which must be pushed there.
