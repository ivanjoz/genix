//! ScyllaDB adapter for loading and replacing absolute compact usage rows.

use anyhow::{Context, Result};
use async_trait::async_trait;
use scylla::{
    client::{session::Session, session_builder::SessionBuilder},
    statement::prepared::PreparedStatement,
};

use crate::{aggregation::UsageKey, config::DatabaseConfig};

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct StoredUsage {
    pub time_frame: i32,
    pub used_credits: Vec<u8>,
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
}

pub struct ScyllaUsageStore {
    session: Session,
    select_range: PreparedStatement,
    upsert_usage: PreparedStatement,
}

impl ScyllaUsageStore {
    pub async fn connect(config: &DatabaseConfig) -> Result<Self> {
        let endpoint = format!("{}:{}", config.host, config.port);
        let session = SessionBuilder::new()
            .known_node(endpoint)
            .user(&config.user, &config.password)
            .build()
            .await
            .context("failed to connect to ScyllaDB")?;
        session
            .use_keyspace(&config.keyspace, false)
            .await
            .with_context(|| format!("failed to use ScyllaDB keyspace {}", config.keyspace))?;

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

        Ok(Self {
            session,
            select_range,
            upsert_usage,
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
}
