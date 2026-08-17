//! ScyllaDB adapter for loading and replacing absolute compact usage rows.

use std::sync::Arc;

use anyhow::{Context, Result};
use async_trait::async_trait;
use scylla::{client::session::Session, statement::prepared::PreparedStatement};

use crate::limiter::credits_blob::Credits;
use crate::{config::DatabaseConfig, limiter::aggregation::UsageKey};

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct StoredUsage {
    pub time_frame: i32,
    pub used_credits: Vec<u8>,
}

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct StoredBudget {
    pub company_id: i32,
    pub daily: Credits,
    pub budget_month_start_day: i16,
    pub monthly_ceiling: Credits,
    pub updated: i32,
}

#[async_trait]
pub trait UsageStore: Send + Sync {
    async fn load_exact(&self, key: UsageKey) -> Result<Option<StoredUsage>>;
    async fn load_range(
        &self,
        company_id: i32,
        user_id: i32,
        start_time_frame: i32,
        end_time_frame: i32,
    ) -> Result<Vec<StoredUsage>>;
    async fn upsert(&self, key: UsageKey, used_credits: Vec<u8>) -> Result<()>;
    async fn load_budget(&self, company_id: i32) -> Result<Option<StoredBudget>>;
    async fn upsert_budget(&self, budget: StoredBudget) -> Result<()>;
}

pub struct ScyllaUsageStore {
    session: Arc<Session>,
    select_range: PreparedStatement,
    upsert_usage: PreparedStatement,
    select_budget: PreparedStatement,
    upsert_budget: PreparedStatement,
}

impl ScyllaUsageStore {
    /// Opens its own session. Kept for callers that have no session to hand over — the daemon uses
    /// `with_session`, so both services share one pool.
    pub async fn connect(config: &DatabaseConfig) -> Result<Self> {
        Self::with_session(crate::reqlog::writer::connect_session(config).await?).await
    }

    pub async fn with_session(session: Arc<Session>) -> Result<Self> {
        let mut select_range = session
            .prepare(
                "SELECT time_frame, used_credits FROM credit_usage \
                 WHERE company_id = ? AND user_id = ? AND time_frame >= ? AND time_frame <= ?",
            )
            .await
            .context("failed to prepare credit_usage range read")?;
        select_range.set_is_idempotent(true);

        let mut upsert_usage = session
            .prepare(
                "INSERT INTO credit_usage (company_id, user_id, time_frame, used_credits) \
                 VALUES (?, ?, ?, ?)",
            )
            .await
            .context("failed to prepare credit_usage upsert")?;
        // Absolute replacement is safe for driver retries; no database counters are involved.
        upsert_usage.set_is_idempotent(true);

        let mut select_budget = session
            .prepare(
                "SELECT daily_cpu, daily_inference, budget_month_start_day, \
                 monthly_cpu_ceiling, monthly_inference_ceiling, updated \
                 FROM company_credit_budget WHERE company_id = ?",
            )
            .await
            .context("failed to prepare company_credit_budget read")?;
        select_budget.set_is_idempotent(true);

        let mut upsert_budget = session
            .prepare(
                "INSERT INTO company_credit_budget \
                 (company_id, daily_cpu, daily_inference, budget_month_start_day, \
                  monthly_cpu_ceiling, monthly_inference_ceiling, updated) \
                 VALUES (?, ?, ?, ?, ?, ?, ?)",
            )
            .await
            .context("failed to prepare company_credit_budget upsert")?;
        upsert_budget.set_is_idempotent(true);

        Ok(Self {
            session,
            select_range,
            upsert_usage,
            select_budget,
            upsert_budget,
        })
    }
}

#[async_trait]
impl UsageStore for ScyllaUsageStore {
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
        let query_result = self
            .session
            .execute_unpaged(
                &self.select_range,
                (company_id, user_id, start_time_frame, end_time_frame),
            )
            .await
            .context("credit_usage range read failed")?;
        let rows_result = query_result
            .into_rows_result()
            .context("credit_usage read did not return rows")?;
        let mut usages = Vec::new();
        for row in rows_result
            .rows::<(i32, Vec<u8>)>()
            .context("credit_usage row shape is invalid")?
        {
            let (time_frame, used_credits) = row.context("credit_usage row decode failed")?;
            usages.push(StoredUsage {
                time_frame,
                used_credits,
            });
        }
        Ok(usages)
    }

    async fn upsert(&self, key: UsageKey, used_credits: Vec<u8>) -> Result<()> {
        self.session
            .execute_unpaged(
                &self.upsert_usage,
                (key.company_id, key.user_id, key.time_frame, used_credits),
            )
            .await
            .context("credit_usage absolute upsert failed")?;
        Ok(())
    }

    async fn load_budget(&self, company_id: i32) -> Result<Option<StoredBudget>> {
        let query_result = self
            .session
            .execute_unpaged(&self.select_budget, (company_id,))
            .await
            .context("company_credit_budget read failed")?;
        let rows_result = query_result
            .into_rows_result()
            .context("company_credit_budget read did not return rows")?;
        let mut rows = rows_result
            .rows::<(i64, i64, i16, i64, i64, i32)>()
            .context("company_credit_budget row shape is invalid")?;
        let Some(row) = rows.next() else {
            return Ok(None);
        };
        let (daily_cpu, daily_inference, month_start, monthly_cpu, monthly_inference, updated) =
            row.context("company_credit_budget row decode failed")?;
        if [daily_cpu, daily_inference, monthly_cpu, monthly_inference]
            .into_iter()
            .any(|value| value < 0)
        {
            anyhow::bail!("company_credit_budget contains negative credits");
        }
        Ok(Some(StoredBudget {
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
            updated,
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
                    budget.updated,
                ),
            )
            .await
            .context("company_credit_budget upsert failed")?;
        Ok(())
    }
}
