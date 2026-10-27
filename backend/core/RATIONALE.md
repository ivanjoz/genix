# RATIONALE — core

Design decisions, newest first.

## The DataFrame lock is fareward's 15 s try-lock, installed with the file store
**Context** — A frame run can hold its lock for minutes (`dataframe` asks for 5), but
`fareward.AcquireLock` has a fixed 15 s lease and the daemon clamps any lease to its
`lock.max_lease_ms` (60 s by default). A longer lease needed a new export in the fareward repo.
**Decision** — `acquireDataFrameLock` calls `AcquireLock(ActionMaterializeDataFrame, fnv64(frame
folder), 0)` and ignores the requested lease. `ConfigureFileStore` installs it, so every entry
point that can run a frame has it.
**Rationale** — The lock only spares duplicate work. The frame's state file and files are written
conditioned on their ETags, so a second run that starts after the lease lapses fails its writes
instead of corrupting a file. Runs normally finish well within 15 s. No change in fareward.
