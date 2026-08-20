//! In-memory absolute usage rows with mutation versions for lossless dirty flushing.

use std::collections::HashMap;

use anyhow::{Result, anyhow};

use crate::limiter::credits_blob::{Credits, RoutedCredits};

/// Charging a request updates the user's row and the company's total, so the in-memory map needs a
/// key for "the company itself". It is a discriminator, not a user, and it is not stored: the two
/// usage tables are separate, and `UsageKey::is_company_aggregate` is what routes a flush to the
/// right one.
pub const COMPANY_AGGREGATE_USER_ID: i32 = -1;

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq)]
pub struct UsageKey {
    pub company_id: i32,
    pub user_id: i32,
    pub time_frame: i32,
}

impl UsageKey {
    pub fn is_company_aggregate(&self) -> bool {
        self.user_id == COMPANY_AGGREGATE_USER_ID
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct UsageSnapshot {
    pub key: UsageKey,
    pub routes: RoutedCredits,
    pub version: u64,
}

#[derive(Clone, Debug)]
pub struct UsageRecord {
    routes: RoutedCredits,
    version: u64,
    flushed_version: u64,
}

impl UsageRecord {
    pub fn loaded(routes: RoutedCredits) -> Self {
        // Loaded rows are already durable and must not be rewritten until they change.
        Self {
            routes,
            version: 0,
            flushed_version: 0,
        }
    }

    pub fn increment(&mut self, route_id: u16, credits: Credits) -> Result<()> {
        let current = self.routes.get(&route_id).copied().unwrap_or_default();
        let updated = current
            .checked_add(credits)
            .ok_or_else(|| anyhow!("in-memory usage credits overflowed uint64"))?;
        self.routes.insert(route_id, updated);
        self.version = self
            .version
            .checked_add(1)
            .ok_or_else(|| anyhow!("usage mutation version overflowed uint64"))?;
        Ok(())
    }

    pub fn snapshot(&self, key: UsageKey) -> Option<UsageSnapshot> {
        (self.version != self.flushed_version).then(|| UsageSnapshot {
            key,
            routes: self.routes.clone(),
            version: self.version,
        })
    }

    pub fn mark_flushed(&mut self, version: u64) {
        // A concurrent mutation stays dirty instead of being hidden by an older completed write.
        if self.version == version {
            self.flushed_version = version;
        }
    }

    pub fn is_clean(&self) -> bool {
        self.version == self.flushed_version
    }
}

pub fn merge_loaded(
    records: &mut HashMap<UsageKey, UsageRecord>,
    key: UsageKey,
    routes: RoutedCredits,
) {
    // Never replace an in-memory row that may already contain newer accepted usage.
    records
        .entry(key)
        .or_insert_with(|| UsageRecord::loaded(routes));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mutation_during_flush_remains_dirty() {
        let key = UsageKey {
            company_id: 7,
            user_id: 42,
            time_frame: 105_954_061,
        };
        let mut record = UsageRecord::loaded(RoutedCredits::new());
        record
            .increment(
                0,
                Credits {
                    cpu: 2,
                    inference: 3,
                },
            )
            .unwrap();
        let snapshot = record.snapshot(key).unwrap();
        record
            .increment(
                0,
                Credits {
                    cpu: 1,
                    inference: 1,
                },
            )
            .unwrap();
        record.mark_flushed(snapshot.version);
        assert!(record.snapshot(key).is_some());
    }
}
