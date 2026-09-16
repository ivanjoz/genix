package db

import (
	"context"

	fareward "github.com/ivanjoz/fareward/go"
	"github.com/ivanjoz/genix-orm/scylla"
)

// ─────────────────────────────────────────────────────────────────────────────
// THIS IS THE PROJECT'S AUTOINCREMENT CHOICE — every id and every write sequence
// in Genix is reserved by the fareward daemon, never by the ORM itself.
//
// The ORM's own allocator reads a counter and then increments it. A Cassandra
// counter makes the increment atomic but will not report the result of *your*
// increment, so two concurrent writers read the same value and mint the same id —
// and when that id is a primary key, the second insert silently overwrites the
// first. Serializing the reservation in one process is the fix, and fareward is
// already this project's single active process.
//
// genix-orm stays a standalone library that knows nothing about fareward: it
// exposes ReserveCounterRange and Genix fills it in. This file is that decision,
// and it is the only place in the codebase that makes it.
// ─────────────────────────────────────────────────────────────────────────────

// Wired in init rather than beside a SetScyllaConnection call, because there are
// several of those — main.go and a handful of exec entry points — and a path that
// missed one would keep advancing the counters directly while the daemon held
// blocks it believed were its own. That is the one combination that produces
// duplicate ids, so it must not be possible to reach it by forgetting a line.
// Every module reaches the database through this package, so importing it is
// already unavoidable.
// Both hooks or neither. An allocator that reserves ranges but does not own the resets is the worst
// of the three states: ResetCounter would move a counter out from under a range the daemon is still
// serving, and the next range it claimed would repeat ids already written.
func init() {
	scylla.ReserveCounterRange = reserveAutoincrementRange
	scylla.SetCounterValue = setAutoincrementValue
}

// BypassFarewardSequences hands the counters back to the ORM's own GetCounter, which reads the
// `sequences` row and increments it in place. It undoes the init above and is the only way out of
// it, so every caller is one grep away.
//
// It exists for bootstrap entry points — fn-init — that must number a handful of seed rows on a
// database that has no daemon it can talk to: a wire protocol is versioned, and a checkout ahead of
// the deployed daemon cannot exchange a single frame with it. Refusing to seed for that reason
// makes the daemon a dependency of creating the database it stores its counters in.
//
// The safety argument is the caller's, not this function's, and it is narrow: GetCounter's
// read-then-increment races only against a CONCURRENT writer on the same counter. One process
// seeding a database nobody is serving yet has no such writer. Calling this anywhere a backend is
// live reintroduces the duplicate-id bug the daemon exists to remove — including a second process
// racing the daemon itself, which is why it can never move behind a flag or a config key.
func BypassFarewardSequences() {
	scylla.ReserveCounterRange = nil
	scylla.SetCounterValue = nil
}

// reserveAutoincrementRange asks the daemon for `increment` consecutive values on
// a counter and returns the first, which is the contract the ORM's own GetCounter
// had.
//
// The keyspace is dropped: the daemon writes the one it is configured against, and
// a backend pointed at a different keyspace than its daemon is a misconfiguration
// no per-call argument could repair.
//
// Errors propagate and the write fails. There is deliberately no fallback to the
// ORM's direct path — the daemon claims ranges in advance, so falling back would
// hand out ids from inside a range it already owns. That makes the fallback itself
// the duplicate-key bug, and a failed insert is the recoverable outcome.
func reserveAutoincrementRange(_ string, counterName string, increment int) (int64, error) {
	return fareward.ReserveSequence(context.Background(), counterName, increment)
}

// setAutoincrementValue moves a counter to an absolute value and returns what it held before. It
// backs ResetCounter, which realigns a counter with the rows that actually exist after a restore.
//
// It goes through the daemon for a reason that is not obvious: the daemon may be holding a range of
// ids it derived from the value being replaced. Writing the row directly would leave it serving
// from a range that no longer means anything, and the range it claimed next would hand out ids the
// abandoned one had already issued. Only the daemon can move the counter and drop its reservation
// as one step.
func setAutoincrementValue(_ string, counterName string, value int64) (int64, error) {
	return fareward.SetSequence(context.Background(), counterName, value)
}
