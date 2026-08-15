//! Suppression of repeated `request_errors` writes.
//!
//! A code line that is failing is usually failing a lot: the same route, the same query, thousands
//! of times an hour. The row it produces is identical every time, so writing it every time buys
//! nothing and costs a round trip per failing request.
//!
//! What the row is *for* is knowing that a place in the code is broken and roughly what it said.
//! Ten minutes of staleness on the text costs nothing against that, because the current message is
//! already in CloudWatch under the request id that referenced it.

use std::{
    collections::HashMap,
    time::{Duration, Instant},
};

/// Keyed on the pair, not on the id alone: two distinct code lines that collide on the same int32
/// are two real rows, and keying on the hash would let one of them permanently suppress the other.
type ErrorKey = (i32, String);

pub struct ErrorWriteGate {
    last_written: HashMap<ErrorKey, Instant>,
    freshness: Duration,
    capacity: usize,
}

impl ErrorWriteGate {
    pub fn new(freshness: Duration, capacity: usize) -> Self {
        Self {
            last_written: HashMap::new(),
            freshness,
            // A zero capacity would mean evicting on every insert and suppressing nothing; one
            // entry is the smallest thing that still behaves like a gate.
            capacity: capacity.max(1),
        }
    }

    /// Whether this error should be written now, recording the decision when it is.
    ///
    /// `now` is passed in rather than read here so the tests can age entries without sleeping.
    pub fn should_write(&mut self, id: i32, code_line: &str, now: Instant) -> bool {
        let key = (id, code_line.to_string());
        if let Some(written_at) = self.last_written.get(&key)
            && now.duration_since(*written_at) < self.freshness
        {
            return false;
        }

        if self.last_written.len() >= self.capacity && !self.last_written.contains_key(&key) {
            self.evict_oldest();
        }
        self.last_written.insert(key, now);
        true
    }

    /// Drops the least recently written entry. Linear, and deliberately so: the map is bounded by
    /// the number of distinct failing code lines in the codebase, which is small, and eviction
    /// only runs once the ceiling is actually reached. A heap here would be more machinery than
    /// the problem has.
    fn evict_oldest(&mut self) {
        if let Some(oldest) = self
            .last_written
            .iter()
            .min_by_key(|(_, written_at)| **written_at)
            .map(|(key, _)| key.clone())
        {
            self.last_written.remove(&oldest);
        }
    }

    #[cfg(test)]
    pub fn tracked(&self) -> usize {
        self.last_written.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn gate() -> ErrorWriteGate {
        ErrorWriteGate::new(Duration::from_secs(600), 100)
    }

    #[test]
    fn the_first_write_goes_through_and_the_repeat_does_not() {
        let mut gate = gate();
        let now = Instant::now();
        assert!(gate.should_write(1, "responses.go:539", now));
        assert!(!gate.should_write(1, "responses.go:539", now + Duration::from_secs(1)));
        assert!(!gate.should_write(1, "responses.go:539", now + Duration::from_secs(599)));
    }

    #[test]
    fn a_stale_entry_is_written_again() {
        let mut gate = gate();
        let now = Instant::now();
        assert!(gate.should_write(1, "responses.go:539", now));
        assert!(gate.should_write(1, "responses.go:539", now + Duration::from_secs(600)));
        // …and the clock restarts from the new write, not from the first one.
        assert!(!gate.should_write(1, "responses.go:539", now + Duration::from_secs(700)));
    }

    /// Two code lines that hash to the same id are two different errors. Keying on the id alone
    /// would let whichever arrived first hide the other for ten minutes at a time, forever.
    #[test]
    fn a_hash_collision_does_not_suppress_the_other_line() {
        let mut gate = gate();
        let now = Instant::now();
        assert!(gate.should_write(1, "responses.go:539", now));
        assert!(gate.should_write(1, "stock.go:120", now));
    }

    #[test]
    fn distinct_errors_are_independent() {
        let mut gate = gate();
        let now = Instant::now();
        assert!(gate.should_write(1, "a.go:1", now));
        assert!(gate.should_write(2, "b.go:2", now));
        assert!(!gate.should_write(1, "a.go:1", now));
        assert!(!gate.should_write(2, "b.go:2", now));
    }

    #[test]
    fn the_map_stays_bounded_and_evicts_the_oldest() {
        let mut gate = ErrorWriteGate::new(Duration::from_secs(600), 3);
        let now = Instant::now();
        gate.should_write(1, "a.go:1", now);
        gate.should_write(2, "b.go:2", now + Duration::from_secs(1));
        gate.should_write(3, "c.go:3", now + Duration::from_secs(2));
        gate.should_write(4, "d.go:4", now + Duration::from_secs(3));

        assert_eq!(gate.tracked(), 3);
        // a.go:1 was the oldest, so it was dropped and is writable again despite being fresh.
        assert!(gate.should_write(1, "a.go:1", now + Duration::from_secs(4)));
        // c.go:3 survived and is still suppressed.
        assert!(!gate.should_write(3, "c.go:3", now + Duration::from_secs(4)));
    }

    /// Re-suppressing an entry already in the map must not evict anything: it is an update, not a
    /// new key, and evicting on it would churn the map for a request that writes nothing.
    #[test]
    fn refreshing_an_existing_entry_does_not_evict() {
        let mut gate = ErrorWriteGate::new(Duration::from_secs(60), 2);
        let now = Instant::now();
        gate.should_write(1, "a.go:1", now);
        gate.should_write(2, "b.go:2", now + Duration::from_secs(50));

        // a.go:1 has gone stale and is written again. The map is at capacity, but this is an
        // update to a key already in it, so nothing may be evicted to make room.
        assert!(gate.should_write(1, "a.go:1", now + Duration::from_secs(61)));
        assert_eq!(gate.tracked(), 2);
        // b.go:2 is still fresh and still tracked, which it would not be had the refresh evicted it.
        assert!(!gate.should_write(2, "b.go:2", now + Duration::from_secs(62)));
    }
}
