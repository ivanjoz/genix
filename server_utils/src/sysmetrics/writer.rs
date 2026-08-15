//! Turns sub-samples into `server_metrics` rows.
//!
//! The loop ticks once a second and writes on the tick that crosses a five-second boundary of the
//! wall clock, never on every fifth tick of a counter. That distinction is the whole reason a
//! restart lands back on the same grid instead of drifting, and it is why a skipped tick leaves an
//! honest hole in the series rather than shifting every later slot by one.
//!
//! Fails open throughout: a write that does not land is a warning and a counter.

use std::{
    sync::Arc,
    time::{Duration, SystemTime, UNIX_EPOCH},
};

use anyhow::{Context, Result};
use scylla::{client::session::Session, statement::prepared::PreparedStatement};
use tokio::{
    sync::watch,
    task::JoinHandle,
    time::{MissedTickBehavior, sleep},
};
use tracing::{info, warn};

use crate::{
    config::ServerMetricsConfig,
    sysmetrics::{
        MetricsSample, WindowPeaks,
        collector::{ServiceUnits, SystemMetricsCollector},
    },
};

const SECONDS_PER_DAY: i64 = 86_400;

pub struct ServerMetricsWriter {
    session: Arc<Session>,
    insert_row: PreparedStatement,
}

impl ServerMetricsWriter {
    pub async fn prepare(session: Arc<Session>, config: &ServerMetricsConfig) -> Result<Self> {
        let ttl_seconds = config.row_ttl.as_secs() as i32;

        // The TTL is interpolated because Scylla takes no bind marker in a USING TTL clause. It is
        // an i32 derived from config and validated at load, never from a request, so nothing
        // untrusted reaches this string.
        //
        // Every column is named: the Go side owns these names, and a rename there fails this
        // prepare at startup instead of writing values into the wrong columns.
        let mut insert_row = session
            .prepare(format!(
                "INSERT INTO server_metrics (date, slot, cpu_percent, mem_percent, disk_percent, \
                 net_rx_rate, net_tx_rate, backend_mem_mb, backend_cpu_percent, \
                 server_utils_mem_mb, server_utils_cpu_percent, search_mem_mb, search_cpu_percent, \
                 scylla_mem_mb, scylla_cpu_percent) \
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) USING TTL {ttl_seconds}"
            ))
            .await
            .context("failed to prepare the server_metrics insert")?;
        // A whole-row write with no counters: a driver retry rewrites the same row.
        insert_row.set_is_idempotent(true);

        Ok(Self {
            session,
            insert_row,
        })
    }

    /// Runs the sampling loop until shutdown.
    pub fn spawn(
        self,
        config: ServerMetricsConfig,
        mut shutdown: watch::Receiver<bool>,
    ) -> JoinHandle<()> {
        tokio::spawn(async move {
            let mut collector = SystemMetricsCollector::new(
                ServiceUnits {
                    backend: config.backend_unit.clone(),
                    server_utils: config.server_utils_unit.clone(),
                    search: config.search_unit.clone(),
                    scylla: config.scylla_unit.clone(),
                },
                config.disk_mount.clone(),
                config.network_interface.clone(),
            );
            let mut peaks = WindowPeaks::default();
            let mut pending_row_index: Option<i64> = None;
            let mut rows_written = 0_u64;
            let mut write_failures = 0_u64;

            // Start on a whole second so the sub-samples of a window are evenly spread inside it
            // rather than clustered around whenever the daemon happened to boot.
            sleep(nanoseconds_to_next_second()).await;
            let mut ticker = tokio::time::interval(config.sample_interval);
            ticker.set_missed_tick_behavior(MissedTickBehavior::Skip);

            info!(
                sample_ms = config.sample_interval.as_millis(),
                row_seconds = config.row_seconds,
                ttl_days = config.row_ttl.as_secs() / SECONDS_PER_DAY as u64,
                mount = %config.disk_mount,
                "server metrics collector started"
            );

            loop {
                tokio::select! {
                    changed = shutdown.changed() => {
                        if changed.is_err() || *shutdown.borrow() {
                            break;
                        }
                    }
                    _ = ticker.tick() => {
                        let row_index = unix_seconds() / config.row_seconds;

                        // The window this tick belongs to is not the pending one, so the pending
                        // window is complete and gets written before the new sub-sample is taken.
                        // Reading it here rather than after absorbing is what keeps a sub-sample
                        // from landing in two rows.
                        if pending_row_index.is_some_and(|pending| pending != row_index) {
                            self.write_window(
                                pending_row_index.unwrap_or_default(),
                                config.row_seconds,
                                &mut peaks,
                                &mut rows_written,
                                &mut write_failures,
                            )
                            .await;
                        }

                        if let Some(sample) = collector.sample() {
                            peaks.absorb(sample);
                        }
                        pending_row_index = Some(row_index);
                    }
                }
            }

            // The window in progress is written as it stands. A partial window is an honest peak
            // over fewer seconds; dropping it would leave a hole every time the daemon restarts.
            if let Some(pending) = pending_row_index {
                self.write_window(
                    pending,
                    config.row_seconds,
                    &mut peaks,
                    &mut rows_written,
                    &mut write_failures,
                )
                .await;
            }
            info!(
                rows_written,
                write_failures, "server metrics collector stopped"
            );
        })
    }

    /// Writes one window's peaks and clears the accumulator. A window that absorbed nothing writes
    /// nothing: a row of sentinels would claim the machine was observed and found absent.
    async fn write_window(
        &self,
        row_index: i64,
        row_seconds: i64,
        peaks: &mut WindowPeaks,
        rows_written: &mut u64,
        write_failures: &mut u64,
    ) {
        if peaks.is_empty() {
            return;
        }
        let sample = peaks.take();
        let window_start = row_index * row_seconds;
        let date = (window_start / SECONDS_PER_DAY) as i16;
        let slot = ((window_start % SECONDS_PER_DAY) / row_seconds) as i16;

        let values = row_values(date, slot, &sample);
        match self
            .session
            .execute_unpaged(&self.insert_row, &values[..])
            .await
        {
            Ok(_) => *rows_written += 1,
            Err(write_error) => {
                *write_failures += 1;
                warn!(error = %write_error, date, slot, "server_metrics write failed, dropping the row");
            }
        }
    }
}

/// The bound values, in the order the prepared statement names its columns. An array and not a
/// tuple so the row can be compared and printed whole in a test — Rust stops deriving `Debug` and
/// `PartialEq` for tuples at twelve elements, and this row has fifteen.
fn row_values(date: i16, slot: i16, sample: &MetricsSample) -> [i16; 15] {
    [
        date,
        slot,
        sample.cpu_percent,
        sample.memory_percent,
        sample.disk_percent,
        sample.network_rx_rate,
        sample.network_tx_rate,
        sample.backend.memory_mb,
        sample.backend.cpu_percent,
        sample.server_utils.memory_mb,
        sample.server_utils.cpu_percent,
        sample.search.memory_mb,
        sample.search.cpu_percent,
        sample.scylla.memory_mb,
        sample.scylla.cpu_percent,
    ]
}

/// Wall-clock seconds. The slot is a position in the day, so it can only come from an absolute
/// clock — an `Instant` would restart at zero with the process.
fn unix_seconds() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|elapsed| elapsed.as_secs() as i64)
        .unwrap_or(0)
}

fn nanoseconds_to_next_second() -> Duration {
    let subsecond = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|elapsed| elapsed.subsec_nanos())
        .unwrap_or(0);
    Duration::from_nanos(u64::from(1_000_000_000 - subsecond.min(999_999_999)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::sysmetrics::NOT_MEASURED;

    /// The identity of a row is derived from the window's start, not from a counter. These are the
    /// three cases that would corrupt a day: the first window, the last, and the rollover.
    fn window_identity(window_start: i64, row_seconds: i64) -> (i16, i16) {
        let row_index = window_start / row_seconds;
        let start = row_index * row_seconds;
        (
            (start / SECONDS_PER_DAY) as i16,
            ((start % SECONDS_PER_DAY) / row_seconds) as i16,
        )
    }

    #[test]
    fn slots_cover_a_day_and_roll_over_cleanly() {
        // 2026-01-01T00:00:00Z is unix day 20454.
        let midnight = 20_454 * SECONDS_PER_DAY;
        assert_eq!(window_identity(midnight, 5), (20_454, 0));
        assert_eq!(window_identity(midnight + 4, 5), (20_454, 0));
        assert_eq!(window_identity(midnight + 5, 5), (20_454, 1));
        // The last window of the day, and the first of the next.
        assert_eq!(
            window_identity(midnight + SECONDS_PER_DAY - 5, 5),
            (20_454, 17_279)
        );
        assert_eq!(window_identity(midnight + SECONDS_PER_DAY, 5), (20_455, 0));
    }

    /// The bound tuple is positional against a column list the Go side owns, so this pins the two
    /// orders together: a field read into the wrong position would silently mislabel every row.
    #[test]
    fn bound_values_follow_the_column_order() {
        let sample = MetricsSample {
            cpu_percent: 1,
            memory_percent: 2,
            disk_percent: 3,
            network_rx_rate: 4,
            network_tx_rate: 5,
            backend: crate::sysmetrics::ServiceSample {
                memory_mb: 6,
                cpu_percent: 7,
            },
            server_utils: crate::sysmetrics::ServiceSample {
                memory_mb: 8,
                cpu_percent: 9,
            },
            search: crate::sysmetrics::ServiceSample {
                memory_mb: 10,
                cpu_percent: 11,
            },
            scylla: crate::sysmetrics::ServiceSample {
                memory_mb: 12,
                cpu_percent: 13,
            },
        };
        assert_eq!(
            row_values(100, 200, &sample),
            [100, 200, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13]
        );
    }

    /// The Lambda row: the backend's two columns carry the sentinel while everything else is real.
    #[test]
    fn an_absent_backend_binds_the_sentinel() {
        let sample = MetricsSample {
            cpu_percent: 2_345,
            ..MetricsSample::default()
        };
        let values = row_values(20_454, 17_279, &sample);
        assert_eq!(values[2], 2_345);
        assert_eq!(values[7], NOT_MEASURED);
        assert_eq!(values[8], NOT_MEASURED);
    }

    #[test]
    fn the_first_tick_waits_for_a_whole_second() {
        let delay = nanoseconds_to_next_second();
        assert!(delay <= Duration::from_secs(1));
        assert!(delay > Duration::ZERO);
    }
}
