//! In-memory absolute usage rows with mutation versions for lossless dirty flushing.

use std::collections::HashMap;

use anyhow::{Result, anyhow};

use crate::credits_blob::{Credits, GroupedCredits};

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq)]
pub struct UsageKey {
    pub company_id: i32,
    pub user_id: i32,
    pub time_frame: i32,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct UsageSnapshot {
    pub key: UsageKey,
    pub groups: GroupedCredits,
    pub version: u64,
}

#[derive(Clone, Debug)]
pub struct UsageRecord {
    groups: GroupedCredits,
    version: u64,
    flushed_version: u64,
}

impl UsageRecord {
    pub fn loaded(groups: GroupedCredits) -> Self {
        // Loaded rows are already durable and must not be rewritten until they change.
        Self {
            groups,
            version: 0,
            flushed_version: 0,
        }
    }

    pub fn increment(&mut self, api_group: u8, credits: Credits) -> Result<()> {
        let current = self.groups.get(&api_group).copied().unwrap_or_default();
        let updated = current
            .checked_add(credits)
            .ok_or_else(|| anyhow!("in-memory usage credits overflowed uint64"))?;
        self.groups.insert(api_group, updated);
        self.version = self
            .version
            .checked_add(1)
            .ok_or_else(|| anyhow!("usage mutation version overflowed uint64"))?;
        Ok(())
    }

    pub fn snapshot(&self, key: UsageKey) -> Option<UsageSnapshot> {
        (self.version != self.flushed_version).then(|| UsageSnapshot {
            key,
            groups: self.groups.clone(),
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
    groups: GroupedCredits,
) {
    // Never replace an in-memory row that may already contain newer accepted usage.
    records
        .entry(key)
        .or_insert_with(|| UsageRecord::loaded(groups));
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
        let mut record = UsageRecord::loaded(GroupedCredits::new());
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
