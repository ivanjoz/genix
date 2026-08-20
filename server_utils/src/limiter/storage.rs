//! ScyllaDB adapter for loading and replacing absolute compact usage rows.

use std::sync::Arc;

use anyhow::{Context, Result};
use async_trait::async_trait;
use scylla::{client::session::Session, statement::prepared::PreparedStatement};

use crate::limiter::credits_blob::Credits;
use crate::{
    config::DatabaseConfig,
    limiter::aggregation::{COMPANY_AGGREGATE_USER_ID, UsageKey},
};

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct StoredUsage {
    pub time_frame: i32,
    pub used_credits: Vec<u8>,
}

/// One user's authorization row, exactly as `users` holds it.
///
/// `grants_blob` is the raw `accesos_computed` column and is **little-endian** u16s — see
/// `decode_grants` in `access.rs` for why that is worth stating twice.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct StoredUserAccess {
    pub grants_blob: Vec<u8>,
    pub status: i8,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct StoredBudget {
    pub company_id: i32,
    pub daily: Credits,
    pub budget_month_start_day: i16,
    pub monthly_ceiling: Credits,
    /// The figure the last "set current" wrote, kept so a reader can tell how much of the granted
    /// credit is gone: the ceiling folds in the usage that existed when it was set, so on its own it
    /// cannot answer that.
    pub last_set: Credits,
    pub updated: i32,
}

/// The extra-pool counters as persisted, with the periods that say which window each belongs to.
///
/// Read back at cold start for two different reasons. The daily figure is the pool itself: without
/// it a restart would hand a company a fresh allowance mid-day. The monthly figure is not a quota at
/// all — there is no monthly extra ceiling — it is the correction term that keeps `month_used`
/// honest, because that counter is recovered by summing the usage rows and those rows include
/// whatever the pool paid for.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct StoredExtraUsage {
    pub day_period: i16,
    pub day_cpu: u64,
    pub month_start_day: i16,
    pub month_cpu: u64,
}

/// The two halves of one `company_credit_budget` row. Read together because it is one row and one
/// point query; written by two separate statements because the two writes race and each must leave
/// the other's columns alone.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct StoredBudgetRow {
    pub budget: StoredBudget,
    pub extra: StoredExtraUsage,
}

/// The daemon's own usage counters for one company, so a reader can subtract them from the ceilings
/// in `StoredBudget` instead of re-summing the usage rows the way the daemon does on a cold miss.
///
/// The two period fields say which window each counter belongs to: nothing rewrites a row when a day
/// or a month rolls over with no traffic, so a reader compares them against its own current window
/// and treats a mismatch as zero used.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct StoredBudgetUsage {
    pub company_id: i32,
    /// Local business UnixDay, the same day `time_frame::daily` buckets by.
    pub day_period: i16,
    pub day_used: Credits,
    pub month_start_day: i16,
    pub month_used: Credits,
    /// CPU paid out of the extra daily pool, in the two windows above. Not part of `day_used` or
    /// `month_used`: extra spending is counted apart from the quota it was granted outside of.
    pub day_extra_cpu: u64,
    pub month_extra_cpu: u64,
    pub updated: i32,
}

/// Every durable read and write the limiter needs. Named for the limiter rather than for usage
/// because it also answers the authorization question the charge frame now carries: one store
/// handle, one ScyllaDB session, one test double.
#[async_trait]
pub trait LimiterStore: Send + Sync {
    async fn load_exact(&self, key: UsageKey) -> Result<Option<StoredUsage>>;
    async fn load_range(
        &self,
        company_id: i32,
        user_id: i32,
        start_time_frame: i32,
        end_time_frame: i32,
    ) -> Result<Vec<StoredUsage>>;
    async fn upsert(&self, key: UsageKey, used_credits: Vec<u8>) -> Result<()>;
    async fn load_budget(&self, company_id: i32) -> Result<Option<StoredBudgetRow>>;
    async fn upsert_budget(&self, budget: StoredBudget) -> Result<()>;
    /// Writes only the usage columns, never the entitlement ones: the two writes race (a flush and a
    /// mutation can land in either order) and each must leave the other's columns untouched.
    async fn upsert_budget_usage(&self, usage: StoredBudgetUsage) -> Result<()>;
    /// `None` means no such user in this company, which is a denial and not an error.
    async fn load_user_access(
        &self,
        company_id: i32,
        user_id: i32,
    ) -> Result<Option<StoredUserAccess>>;
}

pub struct ScyllaLimiterStore {
    session: Arc<Session>,
    // Company totals and per-user totals live in separate tables: the company table carries the
    // per-route split and is read across every tenant by the SaaS panel, while the user table stays
    // narrow and is only ever read one user at a time. One table with a sentinel user id would make
    // the platform-wide read drag every user row through the same partition.
    select_company_range: PreparedStatement,
    upsert_company: PreparedStatement,
    select_user_range: PreparedStatement,
    upsert_user: PreparedStatement,
    select_budget: PreparedStatement,
    upsert_budget: PreparedStatement,
    upsert_budget_usage: PreparedStatement,
    // Authorization, not usage: a single-partition point read of the row the route gate needs.
    select_user_access: PreparedStatement,
}

impl ScyllaLimiterStore {
    /// Opens its own session. Kept for callers that have no session to hand over — the daemon uses
    /// `with_session`, so both services share one pool.
    pub async fn connect(config: &DatabaseConfig) -> Result<Self> {
        Self::with_session(crate::reqlog::writer::connect_session(config).await?).await
    }

    pub async fn with_session(session: Arc<Session>) -> Result<Self> {
        let mut select_company_range = session
            .prepare(
                "SELECT time_frame, used_credits FROM credit_usage_company \
                 WHERE company_id = ? AND time_frame >= ? AND time_frame <= ?",
            )
            .await
            .context("failed to prepare credit_usage_company range read")?;
        select_company_range.set_is_idempotent(true);

        let mut upsert_company = session
            .prepare(
                "INSERT INTO credit_usage_company (company_id, time_frame, used_credits) \
                 VALUES (?, ?, ?)",
            )
            .await
            .context("failed to prepare credit_usage_company upsert")?;
        // Absolute replacement is safe for driver retries; no database counters are involved.
        upsert_company.set_is_idempotent(true);

        let mut select_user_range = session
            .prepare(
                "SELECT time_frame, used_credits FROM credit_usage_user \
                 WHERE company_id = ? AND user_id = ? AND time_frame >= ? AND time_frame <= ?",
            )
            .await
            .context("failed to prepare credit_usage_user range read")?;
        select_user_range.set_is_idempotent(true);

        let mut upsert_user = session
            .prepare(
                "INSERT INTO credit_usage_user (company_id, user_id, time_frame, used_credits) \
                 VALUES (?, ?, ?, ?)",
            )
            .await
            .context("failed to prepare credit_usage_user upsert")?;
        upsert_user.set_is_idempotent(true);

        let mut select_budget = session
            .prepare(
                "SELECT daily_cpu, daily_inference, budget_month_start_day, \
                 monthly_cpu_ceiling, monthly_inference_ceiling, last_set_cpu, \
                 last_set_inference, updated, usage_day_period, day_extra_cpu_used, \
                 usage_month_start_day, month_extra_cpu_used \
                 FROM company_credit_budget WHERE company_id = ?",
            )
            .await
            .context("failed to prepare company_credit_budget read")?;
        select_budget.set_is_idempotent(true);

        let mut upsert_budget = session
            .prepare(
                "INSERT INTO company_credit_budget \
                 (company_id, daily_cpu, daily_inference, budget_month_start_day, \
                  monthly_cpu_ceiling, monthly_inference_ceiling, last_set_cpu, \
                  last_set_inference, updated) \
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
            )
            .await
            .context("failed to prepare company_credit_budget upsert")?;
        upsert_budget.set_is_idempotent(true);

        let mut upsert_budget_usage = session
            .prepare(
                "INSERT INTO company_credit_budget \
                 (company_id, usage_day_period, day_cpu_used, day_inference_used, \
                  usage_month_start_day, month_cpu_used, month_inference_used, \
                  day_extra_cpu_used, month_extra_cpu_used, usage_updated) \
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            )
            .await
            .context("failed to prepare company_credit_budget usage upsert")?;
        upsert_budget_usage.set_is_idempotent(true);

        // `users` is partitioned by company_id and clustered by id, so this is a point read of one
        // row in one partition — the cheapest question ScyllaDB can be asked.
        let mut select_user_access = session
            .prepare("SELECT accesos_computed, status FROM users WHERE company_id = ? AND id = ?")
            .await
            .context("failed to prepare users access read")?;
        select_user_access.set_is_idempotent(true);

        Ok(Self {
            session,
            select_company_range,
            upsert_company,
            select_user_range,
            upsert_user,
            select_budget,
            upsert_budget,
            upsert_budget_usage,
            select_user_access,
        })
    }
}

#[async_trait]
impl LimiterStore for ScyllaLimiterStore {
    async fn load_exact(&self, key: UsageKey) -> Result<Option<StoredUsage>> {
        let rows = self
            .load_range(key.company_id, key.user_id, key.time_frame, key.time_frame)
            .await?;
        Ok(rows.into_iter().next())
    }

    async fn load_range(
        &self,
        company_id: i32,
        user_id: i32,
        start_time_frame: i32,
        end_time_frame: i32,
    ) -> Result<Vec<StoredUsage>> {
        // The company table has no user_id column at all, so the aggregate read binds three values
        // and the per-user read binds four.
        let query_result = if user_id == COMPANY_AGGREGATE_USER_ID {
            self.session
                .execute_unpaged(
                    &self.select_company_range,
                    (company_id, start_time_frame, end_time_frame),
                )
                .await
        } else {
            self.session
                .execute_unpaged(
                    &self.select_user_range,
                    (company_id, user_id, start_time_frame, end_time_frame),
                )
                .await
        }
        .context("credit usage range read failed")?;
        let rows_result = query_result
            .into_rows_result()
            .context("credit usage read did not return rows")?;
        let mut usages = Vec::new();
        for row in rows_result
            .rows::<(i32, Vec<u8>)>()
            .context("credit usage row shape is invalid")?
        {
            let (time_frame, used_credits) = row.context("credit usage row decode failed")?;
            usages.push(StoredUsage {
                time_frame,
                used_credits,
            });
        }
        Ok(usages)
    }

    async fn upsert(&self, key: UsageKey, used_credits: Vec<u8>) -> Result<()> {
        if key.is_company_aggregate() {
            self.session
                .execute_unpaged(
                    &self.upsert_company,
                    (key.company_id, key.time_frame, used_credits),
                )
                .await
                .context("credit_usage_company absolute upsert failed")?;
        } else {
            self.session
                .execute_unpaged(
                    &self.upsert_user,
                    (key.company_id, key.user_id, key.time_frame, used_credits),
                )
                .await
                .context("credit_usage_user absolute upsert failed")?;
        }
        Ok(())
    }

    async fn load_budget(&self, company_id: i32) -> Result<Option<StoredBudgetRow>> {
        let query_result = self
            .session
            .execute_unpaged(&self.select_budget, (company_id,))
            .await
            .context("company_credit_budget read failed")?;
        let rows_result = query_result
            .into_rows_result()
            .context("company_credit_budget read did not return rows")?;
        let mut rows = rows_result
            // last_set_* and the two extra-pool columns arrived through an ALTER, so rows written
            // before them exist with those cells empty: decoding them as non-nullable would fail
            // every pre-existing company. usage_day_period is nullable for the same reason — a
            // company the daemon has never flushed for has no usage cells at all.
            .rows::<(
                i64,
                i64,
                i16,
                i64,
                i64,
                Option<i64>,
                Option<i64>,
                i32,
                Option<i16>,
                Option<i64>,
                Option<i16>,
                Option<i64>,
            )>()
            .context("company_credit_budget row shape is invalid")?;
        let Some(row) = rows.next() else {
            return Ok(None);
        };
        let (
            daily_cpu,
            daily_inference,
            month_start,
            monthly_cpu,
            monthly_inference,
            last_set_cpu,
            last_set_inference,
            updated,
            extra_day_period,
            extra_day_cpu,
            extra_month_start_day,
            extra_month_cpu,
        ) = row.context("company_credit_budget row decode failed")?;
        let (last_set_cpu, last_set_inference) =
            (last_set_cpu.unwrap_or(0), last_set_inference.unwrap_or(0));
        let (extra_day_cpu, extra_month_cpu) =
            (extra_day_cpu.unwrap_or(0), extra_month_cpu.unwrap_or(0));
        if [
            daily_cpu,
            daily_inference,
            monthly_cpu,
            monthly_inference,
            last_set_cpu,
            last_set_inference,
            extra_day_cpu,
            extra_month_cpu,
        ]
        .into_iter()
        .any(|value| value < 0)
        {
            anyhow::bail!("company_credit_budget contains negative credits");
        }
        let budget = StoredBudget {
            company_id,
            daily: Credits {
                cpu: daily_cpu as u64,
                inference: daily_inference as u64,
            },
            budget_month_start_day: month_start,
            monthly_ceiling: Credits {
                cpu: monthly_cpu as u64,
                inference: monthly_inference as u64,
            },
            last_set: Credits {
                cpu: last_set_cpu as u64,
                inference: last_set_inference as u64,
            },
            updated,
        };
        Ok(Some(StoredBudgetRow {
            budget,
            extra: StoredExtraUsage {
                day_period: extra_day_period.unwrap_or(0),
                day_cpu: extra_day_cpu as u64,
                month_start_day: extra_month_start_day.unwrap_or(0),
                month_cpu: extra_month_cpu as u64,
            },
        }))
    }

    async fn upsert_budget(&self, budget: StoredBudget) -> Result<()> {
        let daily_cpu =
            i64::try_from(budget.daily.cpu).context("daily CPU budget exceeds int64")?;
        let daily_inference = i64::try_from(budget.daily.inference)
            .context("daily inference budget exceeds int64")?;
        let monthly_cpu = i64::try_from(budget.monthly_ceiling.cpu)
            .context("monthly CPU ceiling exceeds int64")?;
        let monthly_inference = i64::try_from(budget.monthly_ceiling.inference)
            .context("monthly inference ceiling exceeds int64")?;
        let last_set_cpu =
            i64::try_from(budget.last_set.cpu).context("last set CPU credits exceed int64")?;
        let last_set_inference = i64::try_from(budget.last_set.inference)
            .context("last set inference credits exceed int64")?;
        self.session
            .execute_unpaged(
                &self.upsert_budget,
                (
                    budget.company_id,
                    daily_cpu,
                    daily_inference,
                    budget.budget_month_start_day,
                    monthly_cpu,
                    monthly_inference,
                    last_set_cpu,
                    last_set_inference,
                    budget.updated,
                ),
            )
            .await
            .context("company_credit_budget upsert failed")?;
        Ok(())
    }

    async fn upsert_budget_usage(&self, usage: StoredBudgetUsage) -> Result<()> {
        let day_cpu = i64::try_from(usage.day_used.cpu).context("daily CPU usage exceeds int64")?;
        let day_inference = i64::try_from(usage.day_used.inference)
            .context("daily inference usage exceeds int64")?;
        let month_cpu =
            i64::try_from(usage.month_used.cpu).context("monthly CPU usage exceeds int64")?;
        let month_inference = i64::try_from(usage.month_used.inference)
            .context("monthly inference usage exceeds int64")?;
        let day_extra_cpu =
            i64::try_from(usage.day_extra_cpu).context("daily extra CPU usage exceeds int64")?;
        let month_extra_cpu = i64::try_from(usage.month_extra_cpu)
            .context("monthly extra CPU usage exceeds int64")?;
        self.session
            .execute_unpaged(
                &self.upsert_budget_usage,
                (
                    usage.company_id,
                    usage.day_period,
                    day_cpu,
                    day_inference,
                    usage.month_start_day,
                    month_cpu,
                    month_inference,
                    day_extra_cpu,
                    month_extra_cpu,
                    usage.updated,
                ),
            )
            .await
            .context("company_credit_budget usage upsert failed")?;
        Ok(())
    }

    async fn load_user_access(
        &self,
        company_id: i32,
        user_id: i32,
    ) -> Result<Option<StoredUserAccess>> {
        let query_result = self
            .session
            .execute_unpaged(&self.select_user_access, (company_id, user_id))
            .await
            .context("users access read failed")?;
        let rows_result = query_result
            .into_rows_result()
            .context("users access read did not return rows")?;
        let mut rows = rows_result
            // A user saved before ever being given a profile has no cell in accesos_computed, and
            // status is nullable for the same reason every ORM column is: decoding either as
            // non-nullable would turn a legitimately empty user into a read failure, which fails
            // closed as a 503 rather than as the denial it actually is.
            .rows::<(Option<Vec<u8>>, Option<i8>)>()
            .context("users access row shape is invalid")?;
        let Some(row) = rows.next() else {
            return Ok(None);
        };
        let (grants_blob, status) = row.context("users access row decode failed")?;
        Ok(Some(StoredUserAccess {
            grants_blob: grants_blob.unwrap_or_default(),
            status: status.unwrap_or(0),
        }))
    }
}
