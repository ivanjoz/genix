//! Sharded key registry: one FIFO mutex per `(action, identifier)`, created on demand and
//! removed as soon as nobody holds or wants it.

use std::{
    collections::HashMap,
    sync::{
        Arc, Mutex,
        atomic::{AtomicU16, AtomicU32, AtomicUsize, Ordering},
    },
    time::Duration,
};

use tokio::{
    sync::{OwnedSemaphorePermit, Semaphore},
    time::timeout,
};

use crate::lock::protocol::AcquireRequest;

type LockKey = (u16, i64);

/// Process-wide ceilings. Per-action policy deliberately lives in the Go call sites; these exist
/// only so a misbehaving client cannot exhaust the daemon's memory.
#[derive(Clone, Copy, Debug)]
pub struct LockLimits {
    pub max_keys: usize,
    pub max_total_waiters: u32,
    pub max_lease: Duration,
}

#[derive(Debug)]
struct LockEntry {
    /// One permit: every lock on this port is mutual exclusion. Tokio's semaphore hands the
    /// permit to waiters in arrival order, which is what makes a queued caller's turn arrive.
    semaphore: Arc<Semaphore>,
    queued: AtomicU32,
}

#[derive(Debug)]
pub struct LockRegistry {
    shards: Vec<Mutex<HashMap<LockKey, Arc<LockEntry>>>>,
    total_queued: AtomicU32,
    key_count: AtomicUsize,
    /// Stamped on every grant so a release can prove which hold it is ending.
    ///
    /// Registry-wide rather than per-key on purpose: an idle key's entry is pruned, so a counter
    /// living on the entry would restart at zero the next time that key was taken, and a stale
    /// release from the previous hold would match the new one exactly. Only successive holds of
    /// the *same* key need distinct values — release matches the key first — so wrapping is
    /// harmless unless 65_536 grants land across the whole process while one stale release is
    /// still in flight.
    next_generation: AtomicU16,
    limits: LockLimits,
}

/// Held by the connection task for as long as the lock is held. Dropping it — on release, on
/// disconnect, on lease expiry, or on a panic — is the only way a lock is ever freed.
#[derive(Debug)]
pub struct LockGuard {
    registry: Arc<LockRegistry>,
    key: LockKey,
    entry: Arc<LockEntry>,
    permit: Option<OwnedSemaphorePermit>,
    generation: u16,
}

impl LockGuard {
    /// Identifies this hold specifically, so a release arriving from a caller that already gave
    /// up cannot end the hold that replaced it.
    pub fn generation(&self) -> u16 {
        self.generation
    }
}

#[derive(Debug)]
pub enum LockOutcome {
    Acquired(LockGuard),
    Busy,
    WaitTimeout,
    Capacity,
}

impl LockRegistry {
    pub fn new(shard_count: usize, limits: LockLimits) -> Self {
        let shard_count = shard_count.max(1);
        Self {
            shards: (0..shard_count).map(|_| Mutex::new(HashMap::new())).collect(),
            total_queued: AtomicU32::new(0),
            key_count: AtomicUsize::new(0),
            next_generation: AtomicU16::new(0),
            limits,
        }
    }

    /// Clamps a client-supplied lease to the configured ceiling, so one caller cannot pin a key
    /// for longer than the deployment allows.
    pub fn clamp_lease(&self, lease: Duration) -> Duration {
        lease.min(self.limits.max_lease)
    }

    pub async fn acquire(self: &Arc<Self>, request: AcquireRequest) -> LockOutcome {
        let key = (request.action, request.identifier);
        let Some(entry) = self.entry_for(key) else {
            return LockOutcome::Capacity;
        };

        // Uncontended path: the common case never touches a queue counter.
        if let Ok(permit) = entry.semaphore.clone().try_acquire_owned() {
            return LockOutcome::Acquired(self.make_guard(key, entry, permit));
        }

        // Admission before queueing, so a flood is refused instead of parking thousands of
        // connections inside the daemon — the wait itself would otherwise be the denial of
        // service.
        let queued_before = entry.queued.fetch_add(1, Ordering::AcqRel);
        let total_before = self.total_queued.fetch_add(1, Ordering::AcqRel);
        let refusal = if queued_before >= u32::from(request.max_waiters) {
            Some(LockOutcome::Busy)
        } else if total_before >= self.limits.max_total_waiters {
            Some(LockOutcome::Capacity)
        } else {
            None
        };
        if let Some(outcome) = refusal {
            self.leave_queue(&entry);
            self.prune(key, &entry);
            return outcome;
        }

        let waited = timeout(request.wait, entry.semaphore.clone().acquire_owned()).await;
        self.leave_queue(&entry);
        match waited {
            Ok(Ok(permit)) => LockOutcome::Acquired(self.make_guard(key, entry, permit)),
            // The semaphore is never closed, so this arm is unreachable in practice.
            Ok(Err(_)) => LockOutcome::Capacity,
            Err(_) => {
                self.prune(key, &entry);
                LockOutcome::WaitTimeout
            }
        }
    }

    fn make_guard(
        self: &Arc<Self>,
        key: LockKey,
        entry: Arc<LockEntry>,
        permit: OwnedSemaphorePermit,
    ) -> LockGuard {
        // Incremented per grant, so no two live holds of one key can ever carry the same value.
        // Zero is skipped so it can keep meaning "no generation".
        let mut generation = self.next_generation.fetch_add(1, Ordering::Relaxed).wrapping_add(1);
        if generation == 0 {
            generation = self.next_generation.fetch_add(1, Ordering::Relaxed).wrapping_add(1);
        }
        LockGuard {
            registry: self.clone(),
            key,
            entry,
            permit: Some(permit),
            generation,
        }
    }

    fn leave_queue(&self, entry: &Arc<LockEntry>) {
        entry.queued.fetch_sub(1, Ordering::AcqRel);
        self.total_queued.fetch_sub(1, Ordering::AcqRel);
    }

    fn entry_for(&self, key: LockKey) -> Option<Arc<LockEntry>> {
        let mut shard = self.shard(key);
        if let Some(entry) = shard.get(&key) {
            return Some(entry.clone());
        }
        if self.key_count.load(Ordering::Relaxed) >= self.limits.max_keys {
            return None;
        }
        let entry = Arc::new(LockEntry {
            semaphore: Arc::new(Semaphore::new(1)),
            queued: AtomicU32::new(0),
        });
        shard.insert(key, entry.clone());
        self.key_count.fetch_add(1, Ordering::Relaxed);
        Some(entry)
    }

    fn prune(&self, key: LockKey, entry: &Arc<LockEntry>) {
        let mut shard = self.shard(key);
        // Two handles means the map's and the caller's, and nobody else's. Every clone is made
        // under this same shard lock, so a key that someone still holds or is waiting on can
        // never be removed here — which is what stops a fresh entry, with a fresh permit, from
        // being handed to a second holder.
        if Arc::strong_count(entry) == 2 && shard.remove(&key).is_some() {
            self.key_count.fetch_sub(1, Ordering::Relaxed);
        }
    }

    fn shard(&self, key: LockKey) -> std::sync::MutexGuard<'_, HashMap<LockKey, Arc<LockEntry>>> {
        // A std mutex, not a tokio one: it is never held across an await, and the critical
        // section is a single hash lookup.
        self.shards[self.shard_index(key)]
            .lock()
            .unwrap_or_else(|poisoned| poisoned.into_inner())
    }

    fn shard_index(&self, (action, identifier): LockKey) -> usize {
        // Fibonacci hashing spreads sequential identifiers — company ids, IPv4 addresses — across
        // shards instead of piling them onto one.
        let mixed = (identifier as u64 ^ (u64::from(action) << 48)).wrapping_mul(0x9E37_79B9_7F4A_7C15);
        (mixed >> 32) as usize % self.shards.len()
    }

    #[cfg(test)]
    fn live_keys(&self) -> usize {
        self.key_count.load(Ordering::Relaxed)
    }
}

impl Drop for LockGuard {
    fn drop(&mut self) {
        // Release first so a queued waiter wakes immediately, then drop the key if this was the
        // last interest in it.
        self.permit = None;
        self.registry.prune(self.key, &self.entry);
    }
}

#[cfg(test)]
mod tests {
    use std::time::Duration;

    use super::*;

    fn registry() -> Arc<LockRegistry> {
        Arc::new(LockRegistry::new(
            4,
            LockLimits {
                max_keys: 8,
                max_total_waiters: 8,
                max_lease: Duration::from_secs(60),
            },
        ))
    }

    fn request(max_waiters: u8, wait_ms: u64) -> AcquireRequest {
        AcquireRequest {
            action: 1,
            identifier: 99,
            max_waiters,
            wait: Duration::from_millis(wait_ms),
            lease: Duration::from_secs(15),
        }
    }

    #[tokio::test]
    async fn a_second_caller_waits_for_the_first_to_release() {
        let registry = registry();
        let first = registry.acquire(request(4, 1000)).await;
        let LockOutcome::Acquired(guard) = first else {
            panic!("the uncontended acquire must succeed");
        };

        let waiting = tokio::spawn({
            let registry = registry.clone();
            async move { matches!(registry.acquire(request(4, 1000)).await, LockOutcome::Acquired(_)) }
        });
        tokio::time::sleep(Duration::from_millis(50)).await;
        assert!(!waiting.is_finished(), "the second caller must still be queued");

        drop(guard);
        assert!(waiting.await.unwrap(), "releasing must hand the lock over");
    }

    #[tokio::test]
    async fn the_queue_ceiling_refuses_instead_of_parking() {
        let registry = registry();
        let LockOutcome::Acquired(_holder) = registry.acquire(request(1, 5000)).await else {
            panic!("the uncontended acquire must succeed");
        };
        let queued = tokio::spawn({
            let registry = registry.clone();
            async move { registry.acquire(request(1, 5000)).await }
        });
        tokio::time::sleep(Duration::from_millis(50)).await;

        // One waiter is allowed, so the next caller is turned away rather than queued.
        assert!(matches!(
            registry.acquire(request(1, 5000)).await,
            LockOutcome::Busy
        ));
        queued.abort();
    }

    #[tokio::test]
    async fn waiting_longer_than_the_patience_gives_up() {
        let registry = registry();
        let LockOutcome::Acquired(_holder) = registry.acquire(request(4, 1000)).await else {
            panic!("the uncontended acquire must succeed");
        };
        assert!(matches!(
            registry.acquire(request(4, 30)).await,
            LockOutcome::WaitTimeout
        ));
    }

    #[tokio::test]
    async fn a_try_lock_never_queues() {
        let registry = registry();
        let LockOutcome::Acquired(_holder) = registry.acquire(request(0, 5000)).await else {
            panic!("the uncontended acquire must succeed");
        };
        assert!(matches!(
            registry.acquire(request(0, 5000)).await,
            LockOutcome::Busy
        ));
    }

    #[tokio::test]
    async fn keys_do_not_accumulate() {
        let registry = registry();
        for identifier in 0..100_i64 {
            let outcome = registry
                .acquire(AcquireRequest {
                    identifier,
                    ..request(4, 1000)
                })
                .await;
            assert!(matches!(outcome, LockOutcome::Acquired(_)));
        }
        // Each guard dropped at the end of its iteration, so nothing may be left behind — and
        // max_keys is 8, so a leak would have surfaced as Capacity long before 100.
        assert_eq!(registry.live_keys(), 0);
    }

    #[tokio::test]
    async fn different_actions_do_not_share_a_lock() {
        let registry = registry();
        let LockOutcome::Acquired(_first) = registry.acquire(request(0, 0)).await else {
            panic!("the uncontended acquire must succeed");
        };
        let other_action = AcquireRequest {
            action: 2,
            ..request(0, 0)
        };
        assert!(matches!(
            registry.acquire(other_action).await,
            LockOutcome::Acquired(_)
        ));
    }

    #[tokio::test]
    async fn the_key_ceiling_is_enforced() {
        let registry = registry();
        let mut guards = Vec::new();
        for identifier in 0..8_i64 {
            let LockOutcome::Acquired(guard) = registry
                .acquire(AcquireRequest {
                    identifier,
                    ..request(4, 1000)
                })
                .await
            else {
                panic!("the first max_keys acquires must succeed");
            };
            guards.push(guard);
        }
        assert!(matches!(
            registry
                .acquire(AcquireRequest {
                    identifier: 999,
                    ..request(4, 1000)
                })
                .await,
            LockOutcome::Capacity
        ));
    }
}
