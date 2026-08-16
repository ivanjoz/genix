//! Atomic company/user quota admission and accepted-usage aggregation.

use std::{
    collections::HashMap,
    sync::Arc,
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

use anyhow::{Context, Result, anyhow};
use tokio::sync::Mutex;
use tracing::{debug, error, info};

use crate::limiter::{
    aggregation::{UsageKey, UsageRecord, UsageSnapshot, merge_loaded},
    credits_blob::{Credits, RoutedCredits, decode, encode, sum},
    protocol::{LimitViolation, Request, Scope, Window},
    storage::UsageStore,
    time_frame,
};

const COMPANY_AGGREGATE_USER_ID: i32 = -1;
const PLATFORM_AGGREGATE_COMPANY_ID: i32 = 0;
const TOKEN_PERIOD: Duration = Duration::from_secs(10);

#[derive(Clone, Copy, Debug)]
pub struct CreditLimits {
    pub ten_seconds: u64,
    pub hour: u64,
    pub day: u64,
}

#[derive(Clone, Copy, Debug)]
pub struct ScopeLimits {
    pub cpu: CreditLimits,
    pub inference: CreditLimits,
}

#[derive(Clone, Copy, Debug)]
pub struct LimitPolicy {
    pub company: ScopeLimits,
    pub user: ScopeLimits,
}

#[derive(Debug)]
struct TokenBucket {
    scaled_tokens: u128,
    last_refill: Instant,
}

impl TokenBucket {
    fn full(limit: u64, now: Instant) -> Self {
        Self {
            scaled_tokens: capacity(limit),
            last_refill: now,
        }
    }

    fn refill(&mut self, limit: u64, now: Instant) {
        let elapsed_nanos = now.duration_since(self.last_refill).as_nanos();
        let refill = elapsed_nanos.saturating_mul(u128::from(limit));
        self.scaled_tokens = self
            .scaled_tokens
            .saturating_add(refill)
            .min(capacity(limit));
        self.last_refill = now;
    }

    fn can_charge(&self, credits: u64) -> bool {
        self.scaled_tokens >= u128::from(credits) * TOKEN_PERIOD.as_nanos()
    }

    fn charge(&mut self, credits: u64) {
        self.scaled_tokens -= u128::from(credits) * TOKEN_PERIOD.as_nanos();
    }
}

fn capacity(limit: u64) -> u128 {
    u128::from(limit) * TOKEN_PERIOD.as_nanos()
}

#[derive(Debug)]
struct SubjectState {
    cpu_bucket: TokenBucket,
    inference_bucket: TokenBucket,
    hour_period: i64,
    day_period: i64,
    hour_used: Credits,
    day_used: Credits,
}

impl SubjectState {
    fn recovered(
        limits: ScopeLimits,
        unix_seconds: i64,
        now: Instant,
        hour_used: Credits,
        day_used: Credits,
    ) -> Self {
        Self {
            cpu_bucket: TokenBucket::full(limits.cpu.ten_seconds, now),
            inference_bucket: TokenBucket::full(limits.inference.ten_seconds, now),
            hour_period: unix_seconds / 3_600,
            day_period: unix_seconds / 86_400,
            hour_used,
            day_used,
        }
    }

    fn refresh_periods(&mut self, limits: ScopeLimits, unix_seconds: i64, now: Instant) {
        self.cpu_bucket.refill(limits.cpu.ten_seconds, now);
        self.inference_bucket
            .refill(limits.inference.ten_seconds, now);
        let hour_period = unix_seconds / 3_600;
        if self.hour_period != hour_period {
            self.hour_period = hour_period;
            self.hour_used = Credits::default();
        }
        let day_period = unix_seconds / 86_400;
        if self.day_period != day_period {
            self.day_period = day_period;
            self.day_used = Credits::default();
        }
    }

    fn violation(
        &self,
        scope: Scope,
        window: Window,
        limits: ScopeLimits,
        requested: Credits,
    ) -> Option<LimitViolation> {
        let (cpu, inference) = match window {
            Window::TenSeconds => (
                !self.cpu_bucket.can_charge(requested.cpu),
                !self.inference_bucket.can_charge(requested.inference),
            ),
            Window::Hour => (
                exceeds(self.hour_used.cpu, requested.cpu, limits.cpu.hour),
                exceeds(
                    self.hour_used.inference,
                    requested.inference,
                    limits.inference.hour,
                ),
            ),
            Window::Day => (
                exceeds(self.day_used.cpu, requested.cpu, limits.cpu.day),
                exceeds(
                    self.day_used.inference,
                    requested.inference,
                    limits.inference.day,
                ),
            ),
        };
        (cpu || inference).then_some(LimitViolation {
            scope,
            window,
            cpu,
            inference,
        })
    }

    fn charge(&mut self, requested: Credits) -> Result<()> {
        self.cpu_bucket.charge(requested.cpu);
        self.inference_bucket.charge(requested.inference);
        self.hour_used = self
            .hour_used
            .checked_add(requested)
            .ok_or_else(|| anyhow!("hour usage overflowed uint64"))?;
        self.day_used = self
            .day_used
            .checked_add(requested)
            .ok_or_else(|| anyhow!("daily usage overflowed uint64"))?;
        Ok(())
    }
}

fn exceeds(current: u64, requested: u64, limit: u64) -> bool {
    current
        .checked_add(requested)
        .is_none_or(|total| total > limit)
}

#[derive(Default)]
struct ShardState {
    subjects: HashMap<(i32, i32), SubjectState>,
    usage: HashMap<UsageKey, UsageRecord>,
}

pub struct RateLimiter {
    shards: Vec<Mutex<ShardState>>,
    // Platform usage has one reserved key shared by every company. Keeping it outside the
    // company-sharded maps prevents competing absolute snapshots from overwriting one another.
    platform_usage: Mutex<HashMap<UsageKey, UsageRecord>>,
    policy: LimitPolicy,
    store: Arc<dyn UsageStore>,
}

impl RateLimiter {
    pub fn new(shard_count: usize, policy: LimitPolicy, store: Arc<dyn UsageStore>) -> Self {
        let shards = (0..shard_count.max(1))
            .map(|_| Mutex::new(ShardState::default()))
            .collect();
        Self {
            shards,
            platform_usage: Mutex::new(HashMap::new()),
            policy,
            store,
        }
    }

    pub async fn admit(&self, request: Request) -> Result<Option<LimitViolation>> {
        let unix_seconds = current_unix_seconds()?;
        self.admit_at(request, unix_seconds, Instant::now()).await
    }

    async fn admit_at(
        &self,
        request: Request,
        unix_seconds: i64,
        now: Instant,
    ) -> Result<Option<LimitViolation>> {
        let shard_index = request.company_id as usize % self.shards.len();
        let mut shard = self.shards[shard_index].lock().await;

        self.ensure_subject(
            &mut shard,
            request.company_id,
            COMPANY_AGGREGATE_USER_ID,
            self.policy.company,
            unix_seconds,
            now,
        )
        .await?;
        self.ensure_subject(
            &mut shard,
            request.company_id,
            request.user_id,
            self.policy.user,
            unix_seconds,
            now,
        )
        .await?;

        let company_key = (request.company_id, COMPANY_AGGREGATE_USER_ID);
        let user_key = (request.company_id, request.user_id);
        shard
            .subjects
            .get_mut(&company_key)
            .expect("company state initialized")
            .refresh_periods(self.policy.company, unix_seconds, now);
        shard
            .subjects
            .get_mut(&user_key)
            .expect("user state initialized")
            .refresh_periods(self.policy.user, unix_seconds, now);

        // Shortest window and company scope win, matching the documented response priority.
        for window in [Window::TenSeconds, Window::Hour, Window::Day] {
            let company = shard
                .subjects
                .get(&company_key)
                .expect("company state initialized");
            if let Some(violation) =
                company.violation(Scope::Company, window, self.policy.company, request.credits)
            {
                return Ok(Some(violation));
            }
            let user = shard
                .subjects
                .get(&user_key)
                .expect("user state initialized");
            if let Some(violation) =
                user.violation(Scope::User, window, self.policy.user, request.credits)
            {
                return Ok(Some(violation));
            }
        }

        // Load the reserved absolute row before mutating accepted usage. A storage failure then
        // fails admission cleanly instead of charging company/user state without the platform row.
        let mut platform_usage = self.platform_usage.lock().await;
        self.ensure_platform_usage(&mut platform_usage, unix_seconds)
            .await?;

        shard
            .subjects
            .get_mut(&company_key)
            .expect("company state initialized")
            .charge(request.credits)?;
        shard
            .subjects
            .get_mut(&user_key)
            .expect("user state initialized")
            .charge(request.credits)?;
        increment_usage(&mut shard.usage, request, unix_seconds)?;
        increment_platform_usage(&mut platform_usage, request, unix_seconds)?;
        Ok(None)
    }

    async fn ensure_platform_usage(
        &self,
        platform_usage: &mut HashMap<UsageKey, UsageRecord>,
        unix_seconds: i64,
    ) -> Result<()> {
        let key = platform_usage_key(unix_seconds)?;
        if platform_usage.contains_key(&key) {
            return Ok(());
        }

        let stored_row = self
            .store
            .load_exact(key)
            .await
            .context("failed to initialize platform usage")?;
        let routes = decode_optional(
            stored_row
                .as_ref()
                .map(|stored_usage| stored_usage.used_credits.as_slice()),
        )?;
        merge_loaded(platform_usage, key, routes);
        debug!(time_frame = key.time_frame, "initialized platform usage");
        Ok(())
    }

    async fn ensure_subject(
        &self,
        shard: &mut ShardState,
        company_id: i32,
        user_id: i32,
        limits: ScopeLimits,
        unix_seconds: i64,
        now: Instant,
    ) -> Result<()> {
        if shard.subjects.contains_key(&(company_id, user_id)) {
            return Ok(());
        }

        let daily_time_frame = time_frame::daily(unix_seconds)?;
        let daily_key = UsageKey {
            company_id,
            user_id,
            time_frame: daily_time_frame,
        };
        let daily_row = self
            .store
            .load_exact(daily_key)
            .await
            .context("failed to initialize daily usage")?;
        let daily_routes =
            decode_optional(daily_row.as_ref().map(|row| row.used_credits.as_slice()))?;
        merge_loaded(&mut shard.usage, daily_key, daily_routes.clone());

        let (hour_start, hour_end) = time_frame::hour_five_minute_range(unix_seconds)?;
        let hour_rows = self
            .store
            .load_range(company_id, user_id, hour_start, hour_end)
            .await
            .context("failed to initialize hourly usage")?;
        let mut hour_used = Credits::default();
        for row in hour_rows {
            let routes = decode(&row.used_credits)
                .with_context(|| format!("corrupt usage blob in time frame {}", row.time_frame))?;
            hour_used = hour_used
                .checked_add(sum(&routes)?)
                .ok_or_else(|| anyhow!("recovered hour usage overflowed uint64"))?;
            merge_loaded(
                &mut shard.usage,
                UsageKey {
                    company_id,
                    user_id,
                    time_frame: row.time_frame,
                },
                routes,
            );
        }

        let day_used = sum(&daily_routes)?;
        shard.subjects.insert(
            (company_id, user_id),
            SubjectState::recovered(limits, unix_seconds, now, hour_used, day_used),
        );
        debug!(
            company_id,
            user_id,
            hour_cpu = hour_used.cpu,
            hour_inference = hour_used.inference,
            day_cpu = day_used.cpu,
            day_inference = day_used.inference,
            "initialized quota state"
        );
        Ok(())
    }

    pub async fn flush_dirty(&self) -> usize {
        let mut snapshots = Vec::new();
        for shard in &self.shards {
            let shard = shard.lock().await;
            snapshots.extend(
                shard
                    .usage
                    .iter()
                    .filter_map(|(&key, record)| record.snapshot(key)),
            );
        }
        {
            let platform_usage = self.platform_usage.lock().await;
            snapshots.extend(
                platform_usage
                    .iter()
                    .filter_map(|(&key, record)| record.snapshot(key)),
            );
        }
        if snapshots.is_empty() {
            debug!("usage flush skipped because no records are dirty");
            return 0;
        }

        info!(dirty_records = snapshots.len(), "starting usage flush");
        let mut written = 0;
        for snapshot in snapshots {
            match self.flush_snapshot(&snapshot).await {
                Ok(()) => {
                    self.mark_flushed(&snapshot).await;
                    written += 1;
                }
                Err(flush_error) => {
                    error!(
                        company_id = snapshot.key.company_id,
                        user_id = snapshot.key.user_id,
                        time_frame = snapshot.key.time_frame,
                        error = %flush_error,
                        "usage snapshot remains dirty after failed flush"
                    );
                }
            }
        }
        if let Err(cleanup_error) = self.prune_clean_usage().await {
            error!(error = %cleanup_error, "failed to prune clean historical usage from memory");
        }
        info!(written, "usage flush completed");
        written
    }

    async fn flush_snapshot(&self, snapshot: &UsageSnapshot) -> Result<()> {
        let encoded = encode(&snapshot.routes).context("failed to encode usage snapshot")?;
        self.store.upsert(snapshot.key, encoded).await
    }

    async fn mark_flushed(&self, snapshot: &UsageSnapshot) {
        if snapshot.key.company_id == PLATFORM_AGGREGATE_COMPANY_ID {
            let mut platform_usage = self.platform_usage.lock().await;
            if let Some(record) = platform_usage.get_mut(&snapshot.key) {
                record.mark_flushed(snapshot.version);
            }
            return;
        }

        let shard_index = snapshot.key.company_id as usize % self.shards.len();
        let mut shard = self.shards[shard_index].lock().await;
        if let Some(record) = shard.usage.get_mut(&snapshot.key) {
            record.mark_flushed(snapshot.version);
        }
    }

    async fn prune_clean_usage(&self) -> Result<()> {
        let unix_seconds = current_unix_seconds()?;
        let current_five_minute = time_frame::five_minute(unix_seconds)?;
        let current_day = time_frame::daily(unix_seconds)?;
        for shard in &self.shards {
            let mut shard = shard.lock().await;
            shard.usage.retain(|key, record| {
                !record.is_clean()
                    || key.time_frame == current_five_minute
                    || key.time_frame == current_day
            });
        }
        let mut platform_usage = self.platform_usage.lock().await;
        platform_usage
            .retain(|key, record| !record.is_clean() || key.time_frame == current_five_minute);
        Ok(())
    }
}

fn decode_optional(encoded: Option<&[u8]>) -> Result<RoutedCredits> {
    encoded
        .map(decode)
        .transpose()
        .map(|value| value.unwrap_or_default())
        .map_err(Into::into)
}

fn increment_usage(
    records: &mut HashMap<UsageKey, UsageRecord>,
    request: Request,
    unix_seconds: i64,
) -> Result<()> {
    let five_minute = time_frame::five_minute(unix_seconds)?;
    let daily = time_frame::daily(unix_seconds)?;
    for user_id in [request.user_id, COMPANY_AGGREGATE_USER_ID] {
        for time_frame in [five_minute, daily] {
            let key = UsageKey {
                company_id: request.company_id,
                user_id,
                time_frame,
            };
            records
                .entry(key)
                .or_insert_with(|| UsageRecord::loaded(RoutedCredits::new()))
                .increment(request.route_id, request.credits)?;
        }
    }
    Ok(())
}

fn platform_usage_key(unix_seconds: i64) -> Result<UsageKey> {
    Ok(UsageKey {
        company_id: PLATFORM_AGGREGATE_COMPANY_ID,
        user_id: COMPANY_AGGREGATE_USER_ID,
        time_frame: time_frame::five_minute(unix_seconds)?,
    })
}

fn increment_platform_usage(
    records: &mut HashMap<UsageKey, UsageRecord>,
    request: Request,
    unix_seconds: i64,
) -> Result<()> {
    let key = platform_usage_key(unix_seconds)?;
    records
        .get_mut(&key)
        .expect("platform usage initialized")
        .increment(request.route_id, request.credits)
}

fn current_unix_seconds() -> Result<i64> {
    let seconds = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .context("system clock is before the Unix epoch")?
        .as_secs();
    i64::try_from(seconds).context("Unix time does not fit in int64")
}

#[cfg(test)]
mod tests {
    use std::sync::Mutex as StdMutex;

    use async_trait::async_trait;

    use super::*;
    use crate::limiter::storage::StoredUsage;

    #[derive(Default)]
    struct MemoryStore {
        rows: StdMutex<HashMap<UsageKey, Vec<u8>>>,
    }

    #[async_trait]
    impl UsageStore for MemoryStore {
        async fn load_exact(&self, key: UsageKey) -> Result<Option<StoredUsage>> {
            Ok(self
                .rows
                .lock()
                .unwrap()
                .get(&key)
                .cloned()
                .map(|used_credits| StoredUsage {
                    time_frame: key.time_frame,
                    used_credits,
                }))
        }

        async fn load_range(
            &self,
            company_id: i32,
            user_id: i32,
            start_time_frame: i32,
            end_time_frame: i32,
        ) -> Result<Vec<StoredUsage>> {
            Ok(self
                .rows
                .lock()
                .unwrap()
                .iter()
                .filter(|(key, _)| {
                    key.company_id == company_id
                        && key.user_id == user_id
                        && (start_time_frame..=end_time_frame).contains(&key.time_frame)
                })
                .map(|(key, blob)| StoredUsage {
                    time_frame: key.time_frame,
                    used_credits: blob.clone(),
                })
                .collect())
        }

        async fn upsert(&self, key: UsageKey, used_credits: Vec<u8>) -> Result<()> {
            self.rows.lock().unwrap().insert(key, used_credits);
            Ok(())
        }
    }

    fn test_policy(limit: u64) -> LimitPolicy {
        let limits = ScopeLimits {
            cpu: CreditLimits {
                ten_seconds: limit,
                hour: limit,
                day: limit,
            },
            inference: CreditLimits {
                ten_seconds: limit,
                hour: limit,
                day: limit,
            },
        };
        LimitPolicy {
            company: limits,
            user: limits,
        }
    }

    #[tokio::test]
    async fn accepted_request_dirties_company_user_and_platform_rows() {
        let store = Arc::new(MemoryStore::default());
        let limiter = RateLimiter::new(1, test_policy(100), store.clone());
        let request = Request {
            company_id: 7,
            user_id: 42,
            route_id: 34,
            credits: Credits {
                cpu: 4,
                inference: 5,
            },
        };
        assert_eq!(
            limiter
                .admit_at(request, 1_800_000_000, Instant::now())
                .await
                .unwrap(),
            None
        );
        assert_eq!(limiter.flush_dirty().await, 5);
        assert_eq!(store.rows.lock().unwrap().len(), 5);
        let platform_key = platform_usage_key(1_800_000_000).unwrap();
        let platform_routes = decode(
            store
                .rows
                .lock()
                .unwrap()
                .get(&platform_key)
                .expect("platform row persisted"),
        )
        .unwrap();
        assert_eq!(platform_routes.get(&34), Some(&request.credits));
        assert_eq!(limiter.flush_dirty().await, 0);
    }

    #[tokio::test]
    async fn platform_aggregate_is_shared_across_company_shards() {
        let store = Arc::new(MemoryStore::default());
        let limiter = RateLimiter::new(4, test_policy(100), store.clone());
        let unix_seconds = 1_800_000_000;

        for company_id in [1, 2] {
            let result = limiter
                .admit_at(
                    Request {
                        company_id,
                        user_id: company_id * 10,
                        route_id: 34,
                        credits: Credits {
                            cpu: 4,
                            inference: 5,
                        },
                    },
                    unix_seconds,
                    Instant::now(),
                )
                .await
                .unwrap();
            assert_eq!(result, None);
        }

        limiter.flush_dirty().await;
        let platform_key = platform_usage_key(unix_seconds).unwrap();
        let platform_routes = decode(
            store
                .rows
                .lock()
                .unwrap()
                .get(&platform_key)
                .expect("one shared platform row persisted"),
        )
        .unwrap();
        assert_eq!(
            platform_routes.get(&34),
            Some(&Credits {
                cpu: 8,
                inference: 10
            })
        );
    }

    #[tokio::test]
    async fn platform_aggregate_extends_the_absolute_row_after_restart() {
        let store = Arc::new(MemoryStore::default());
        let unix_seconds = 1_800_000_000;
        let platform_key = platform_usage_key(unix_seconds).unwrap();
        store.rows.lock().unwrap().insert(
            platform_key,
            encode(&RoutedCredits::from([(
                34,
                Credits {
                    cpu: 7,
                    inference: 8,
                },
            )]))
            .unwrap(),
        );
        let limiter = RateLimiter::new(1, test_policy(100), store.clone());

        limiter
            .admit_at(
                Request {
                    company_id: 7,
                    user_id: 42,
                    route_id: 34,
                    credits: Credits {
                        cpu: 4,
                        inference: 5,
                    },
                },
                unix_seconds,
                Instant::now(),
            )
            .await
            .unwrap();
        limiter.flush_dirty().await;

        let platform_routes = decode(
            store
                .rows
                .lock()
                .unwrap()
                .get(&platform_key)
                .expect("platform row persisted"),
        )
        .unwrap();
        assert_eq!(
            platform_routes.get(&34),
            Some(&Credits {
                cpu: 11,
                inference: 13
            })
        );
    }

    #[tokio::test]
    async fn exact_limit_is_allowed_and_next_credit_is_rejected() {
        let store = Arc::new(MemoryStore::default());
        let limiter = RateLimiter::new(1, test_policy(10), store);
        let now = Instant::now();
        let request = Request {
            company_id: 7,
            user_id: 42,
            route_id: 0,
            credits: Credits {
                cpu: 10,
                inference: 0,
            },
        };
        assert_eq!(
            limiter.admit_at(request, 1_800_000_000, now).await.unwrap(),
            None
        );
        let violation = limiter
            .admit_at(
                Request {
                    credits: Credits {
                        cpu: 1,
                        inference: 0,
                    },
                    ..request
                },
                1_800_000_000,
                now,
            )
            .await
            .unwrap()
            .unwrap();
        assert_eq!(violation.scope, Scope::Company);
        assert_eq!(violation.window, Window::TenSeconds);
        assert!(violation.cpu);
        // The rejected second request must not create another dirty platform mutation.
        assert_eq!(limiter.flush_dirty().await, 5);
        assert_eq!(limiter.flush_dirty().await, 0);
    }
}
