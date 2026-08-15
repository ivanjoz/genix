//! Persists finished-request records into `user_logs`, and the code lines that failed into
//! `request_errors`.
//!
//! **This half fails open, and that is the difference from the limiter.** The limiter exits when
//! ScyllaDB is unreachable because admitting traffic it cannot account for is worse than not
//! serving it. A log row has no such stake: dropping one costs a line in a table, while taking the
//! process down would stop the limiter and the SSE bridge with it. So every failure here is a
//! warning and a counter, never a propagated error.
//!
//! The channel is bounded for the same reason. A backend that logs faster than Scylla accepts must
//! see its records dropped, not slow down: the whole point of the fire-and-forget opcode is that
//! logging never appears on a request's critical path.

use std::{
    sync::{
        Arc,
        atomic::{AtomicU64, Ordering},
    },
    time::Instant,
};

use anyhow::{Context, Result};
use scylla::{client::session::Session, statement::batch::Batch, statement::prepared::PreparedStatement};
use tokio::sync::{mpsc, watch};
use tracing::{debug, info, warn};

use crate::{
    config::RequestLogConfig,
    reqlog::{errors::ErrorWriteGate, protocol::RequestLogRecord},
};

/// What the writer could not do, kept as counters rather than logged per event: a Scylla outage
/// would otherwise produce one warning per request.
#[derive(Debug, Default)]
pub struct RequestLogMetrics {
    pub queued: AtomicU64,
    pub dropped_queue_full: AtomicU64,
    pub rows_written: AtomicU64,
    pub errors_written: AtomicU64,
    pub write_failures: AtomicU64,
}

/// The handle the connection reader holds. Cloning is cheap and every clone feeds the same queue.
#[derive(Clone)]
pub struct RequestLogSink {
    sender: Option<mpsc::Sender<RequestLogRecord>>,
    metrics: Arc<RequestLogMetrics>,
}

impl RequestLogSink {
    /// A sink that accepts records and discards them, for `request_log.enabled = false`. The
    /// opcode still parses and still authenticates — only the row is not written.
    pub fn disabled() -> Self {
        Self {
            sender: None,
            metrics: Arc::new(RequestLogMetrics::default()),
        }
    }

    /// Hands a record to the writer without waiting for it. Never blocks and never fails: a full
    /// queue drops the record and counts it.
    pub fn submit(&self, record: RequestLogRecord) {
        let Some(sender) = &self.sender else {
            return;
        };
        match sender.try_send(record) {
            Ok(()) => {
                self.metrics.queued.fetch_add(1, Ordering::Relaxed);
            }
            Err(_) => {
                self.metrics
                    .dropped_queue_full
                    .fetch_add(1, Ordering::Relaxed);
            }
        }
    }

    pub fn metrics(&self) -> Arc<RequestLogMetrics> {
        self.metrics.clone()
    }
}

pub struct RequestLogWriter {
    session: Arc<Session>,
    insert_user_log: PreparedStatement,
    insert_request_error: PreparedStatement,
    config: RequestLogConfig,
}

impl RequestLogWriter {
    pub async fn prepare(session: Arc<Session>, config: RequestLogConfig) -> Result<Self> {
        let ttl_seconds = config.row_ttl.as_secs() as i32;

        // TTL is interpolated rather than bound because Scylla will not take a bind marker in a
        // USING TTL clause. It is an i32 derived from config and validated at load, never from a
        // request, so nothing untrusted reaches this string.
        let mut insert_user_log = session
            .prepare(format!(
                "INSERT INTO user_logs (date, request_id, company_id, user_id, route_id, \
                 frame_route_company_agg, elapsed_ms, error_count, error_ids) \
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) USING TTL {ttl_seconds}"
            ))
            .await
            .context("failed to prepare the user_logs insert")?;
        // Whole-row writes with no counters: a driver retry rewrites the same row.
        insert_user_log.set_is_idempotent(true);

        // No TTL: a code line that failed once is worth keeping until it is rewritten. The table is
        // bounded by the codebase, not by traffic, so it does not grow the way user_logs does.
        let mut insert_request_error = session
            .prepare(
                "INSERT INTO request_errors (id, code_line, text, updated) VALUES (?, ?, ?, ?)",
            )
            .await
            .context("failed to prepare the request_errors insert")?;
        insert_request_error.set_is_idempotent(true);

        Ok(Self {
            session,
            insert_user_log,
            insert_request_error,
            config,
        })
    }

    /// Spawns the flush loop and returns the sink the connection readers submit through.
    pub fn spawn(self, mut shutdown: watch::Receiver<bool>) -> RequestLogSink {
        let (sender, mut receiver) = mpsc::channel(self.config.queue_capacity);
        let metrics = Arc::new(RequestLogMetrics::default());
        let sink = RequestLogSink {
            sender: Some(sender),
            metrics: metrics.clone(),
        };

        let flush_interval = self.config.flush_interval;
        let max_batch = self.config.max_batch;
        let mut gate = ErrorWriteGate::new(
            self.config.error_freshness,
            self.config.error_cache_entries,
        );

        tokio::spawn(async move {
            let mut pending: Vec<RequestLogRecord> = Vec::with_capacity(max_batch);
            let mut ticker = tokio::time::interval(flush_interval);
            ticker.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);

            loop {
                tokio::select! {
                    changed = shutdown.changed() => {
                        if changed.is_err() || *shutdown.borrow() {
                            break;
                        }
                    }
                    received = receiver.recv() => {
                        match received {
                            Some(record) => {
                                pending.push(record);
                                if pending.len() >= max_batch {
                                    self.flush(&mut pending, &mut gate, &metrics).await;
                                }
                            }
                            // Every sink was dropped: nothing more can arrive.
                            None => break,
                        }
                    }
                    _ = ticker.tick() => {
                        self.flush(&mut pending, &mut gate, &metrics).await;
                    }
                }
            }

            // Drain what is already queued before the process goes away. These rows cost one round
            // trip and are the only record of the requests that were in flight during a restart.
            while let Ok(record) = receiver.try_recv() {
                pending.push(record);
            }
            self.flush(&mut pending, &mut gate, &metrics).await;
            info!(
                queued = metrics.queued.load(Ordering::Relaxed),
                rows_written = metrics.rows_written.load(Ordering::Relaxed),
                errors_written = metrics.errors_written.load(Ordering::Relaxed),
                dropped = metrics.dropped_queue_full.load(Ordering::Relaxed),
                failures = metrics.write_failures.load(Ordering::Relaxed),
                "request log writer stopped"
            );
        });

        sink
    }

    async fn flush(
        &self,
        pending: &mut Vec<RequestLogRecord>,
        gate: &mut ErrorWriteGate,
        metrics: &RequestLogMetrics,
    ) {
        if pending.is_empty() {
            return;
        }

        let now = Instant::now();
        let updated = unix_seconds();
        // Unlogged, not logged: these rows share no partition and need no atomicity between them.
        // A logged batch would buy a guarantee nobody wants and pay for it on every write.
        let mut batch = Batch::new(scylla::statement::batch::BatchType::Unlogged);
        let mut row_values = Vec::with_capacity(pending.len());
        let mut error_values = Vec::new();
        for record in pending.iter() {
            batch.append_statement(self.insert_user_log.clone());
            row_values.push((
                record.date,
                record.request_id,
                record.company_id,
                record.user_id,
                record.route_id,
                record.frame_route_company_agg(),
                record.elapsed_ms,
                record.error_count(),
                record.error_ids(),
            ));

            for entry in &record.errors {
                if gate.should_write(entry.id, &entry.code_line, now) {
                    error_values.push((
                        entry.id,
                        entry.code_line.clone(),
                        entry.text.clone(),
                        updated,
                    ));
                }
            }
        }

        let row_count = row_values.len();
        match self.session.batch(&batch, row_values).await {
            Ok(_) => {
                metrics
                    .rows_written
                    .fetch_add(row_count as u64, Ordering::Relaxed);
            }
            Err(batch_error) => {
                metrics.write_failures.fetch_add(1, Ordering::Relaxed);
                warn!(error = %batch_error, rows = row_count, "user_logs batch write failed, dropping it");
            }
        }

        for values in error_values {
            let code_line = values.1.clone();
            match self
                .session
                .execute_unpaged(&self.insert_request_error, values)
                .await
            {
                Ok(_) => {
                    metrics.errors_written.fetch_add(1, Ordering::Relaxed);
                }
                Err(write_error) => {
                    metrics.write_failures.fetch_add(1, Ordering::Relaxed);
                    warn!(error = %write_error, %code_line, "request_errors write failed, dropping it");
                }
            }
        }

        debug!(rows = row_count, "request log flush finished");
        pending.clear();
    }
}

/// Wall-clock seconds for the `updated` column. Not `Instant`: this one is read back by people and
/// compared against other timestamps, so it has to be an absolute time.
fn unix_seconds() -> i32 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs() as i32)
        .unwrap_or(0)
}

/// Opens the session both the limiter and this writer use. One pool to the same cluster is enough;
/// two would double the connections for no gain.
pub async fn connect_session(config: &crate::config::DatabaseConfig) -> Result<Arc<Session>> {
    use scylla::client::session_builder::SessionBuilder;

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
    Ok(Arc::new(session))
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A sink with no writer behind it must swallow records rather than panic or block: that is
    /// exactly the shape the disabled config takes, and it is on the request path.
    #[test]
    fn a_disabled_sink_accepts_and_discards() {
        let sink = RequestLogSink::disabled();
        sink.submit(RequestLogRecord {
            date: 1,
            request_id: 2,
            route_id: 3,
            frame: 4,
            company_id: 5,
            user_id: 6,
            elapsed_ms: 7,
            errors: vec![],
        });
        assert_eq!(sink.metrics().queued.load(Ordering::Relaxed), 0);
        assert_eq!(
            sink.metrics().dropped_queue_full.load(Ordering::Relaxed),
            0
        );
    }

    /// The guarantee that matters on the request path: a queue nobody is draining drops records
    /// and counts them instead of blocking the caller.
    #[tokio::test]
    async fn a_full_queue_drops_instead_of_blocking() {
        let (sender, _receiver) = mpsc::channel(2);
        let sink = RequestLogSink {
            sender: Some(sender),
            metrics: Arc::new(RequestLogMetrics::default()),
        };

        let record = || RequestLogRecord {
            date: 1,
            request_id: 2,
            route_id: 3,
            frame: 4,
            company_id: 5,
            user_id: 6,
            elapsed_ms: 7,
            errors: vec![],
        };
        for _ in 0..10 {
            sink.submit(record());
        }

        let metrics = sink.metrics();
        assert_eq!(metrics.queued.load(Ordering::Relaxed), 2);
        assert_eq!(metrics.dropped_queue_full.load(Ordering::Relaxed), 8);
    }
}
