//! One row every five seconds in `server_metrics` with what this machine was doing.
//!
//! This daemon owns the writes and the Go ORM owns the schema, the same split `reqlog` uses for
//! `user_logs`. Here it is not a preference: the backend may be a Lambda, so `server_utils` is the
//! only Genix process guaranteed to be on the box and therefore the only one that can promise a
//! continuous series.
//!
//! **Every stored value is a peak, not an average.** The collector samples once a second and the
//! row carries the highest of the five sub-samples in its window, so a one-second spike survives
//! into a five-second row. The cost is stated in the table's own doc comment and worth repeating:
//! these rows cannot be summed into totals, because each value is a peak standing in for five
//! seconds.
//!
//! Like `reqlog` and unlike the limiter, everything here fails open. A dropped row costs a gap in a
//! chart; taking the process down would stop the rate limiter, the lock service and the SSE bridge
//! along with it.

pub mod collector;
pub mod writer;

/// Nothing could be read for this column: the service has no cgroup on this box — the backend
/// running on Lambda is the case this exists for — or every sub-sample of the window failed.
///
/// A sentinel and not a null because every column is an `i16` in which `0` is a legitimate
/// reading, so an absent backend and an idle backend would otherwise be the same row.
pub const NOT_MEASURED: i16 = -1;

/// Percentages are stored as hundredths, so 23.45% is 2345 and the column tops out at 100.00%.
const PERCENT_CEILING: i16 = 10_000;

/// Network rates are stored in 5 KB/s units: an `i16` then reaches 163 MB/s while still resolving
/// the single-digit KB/s an idle box shows, which whole-10 KB units would have flattened to zero.
const NETWORK_RATE_UNIT_BYTES: f64 = 5_120.0;

/// One service's half of a sub-sample. Both fields default to [`NOT_MEASURED`] so a reader that
/// the collector never filled reports as absent rather than as an idle service.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ServiceSample {
    pub memory_mb: i16,
    pub cpu_percent: i16,
}

impl Default for ServiceSample {
    fn default() -> Self {
        Self {
            memory_mb: NOT_MEASURED,
            cpu_percent: NOT_MEASURED,
        }
    }
}

/// Everything read in one second. The same shape is reused for the window's peaks, since taking a
/// maximum does not change what a field means.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct MetricsSample {
    pub cpu_percent: i16,
    pub memory_percent: i16,
    pub disk_percent: i16,
    pub network_rx_rate: i16,
    pub network_tx_rate: i16,
    pub backend: ServiceSample,
    pub server_utils: ServiceSample,
    pub search: ServiceSample,
    pub scylla: ServiceSample,
}

impl Default for MetricsSample {
    fn default() -> Self {
        Self {
            cpu_percent: NOT_MEASURED,
            memory_percent: NOT_MEASURED,
            disk_percent: NOT_MEASURED,
            network_rx_rate: NOT_MEASURED,
            network_tx_rate: NOT_MEASURED,
            backend: ServiceSample::default(),
            server_utils: ServiceSample::default(),
            search: ServiceSample::default(),
            scylla: ServiceSample::default(),
        }
    }
}

/// The running maximum of one row's window.
///
/// The whole subtlety is in [`peak`]: `NOT_MEASURED` is `-1`, so a naive `max` would let a single
/// valid reading of `0` win against it — which is right — but would also let `-1` win against
/// nothing and turn "absent" into "idle" the moment any column started at `-1`. Absence has to lose
/// to every real value and survive only when no real value ever arrived.
#[derive(Debug, Default)]
pub struct WindowPeaks {
    peaks: MetricsSample,
    absorbed: u32,
}

/// Keeps the higher of two readings, with [`NOT_MEASURED`] always losing to a real one.
fn peak(current: i16, candidate: i16) -> i16 {
    if candidate == NOT_MEASURED {
        return current;
    }
    if current == NOT_MEASURED {
        return candidate;
    }
    current.max(candidate)
}

fn peak_service(current: ServiceSample, candidate: ServiceSample) -> ServiceSample {
    ServiceSample {
        memory_mb: peak(current.memory_mb, candidate.memory_mb),
        cpu_percent: peak(current.cpu_percent, candidate.cpu_percent),
    }
}

impl WindowPeaks {
    pub fn absorb(&mut self, sample: MetricsSample) {
        let current = self.peaks;
        self.peaks = MetricsSample {
            cpu_percent: peak(current.cpu_percent, sample.cpu_percent),
            memory_percent: peak(current.memory_percent, sample.memory_percent),
            disk_percent: peak(current.disk_percent, sample.disk_percent),
            network_rx_rate: peak(current.network_rx_rate, sample.network_rx_rate),
            network_tx_rate: peak(current.network_tx_rate, sample.network_tx_rate),
            backend: peak_service(current.backend, sample.backend),
            server_utils: peak_service(current.server_utils, sample.server_utils),
            search: peak_service(current.search, sample.search),
            scylla: peak_service(current.scylla, sample.scylla),
        };
        self.absorbed += 1;
    }

    /// Nothing was absorbed, so there is no row to write. Distinct from a row of `NOT_MEASURED`,
    /// which would claim the window was observed and found empty.
    pub fn is_empty(&self) -> bool {
        self.absorbed == 0
    }

    /// Returns the window's peaks and clears the accumulator in one move, so no sub-sample can be
    /// counted into two rows or dropped between them.
    pub fn take(&mut self) -> MetricsSample {
        self.absorbed = 0;
        std::mem::take(&mut self.peaks)
    }
}

/// A ratio in 0.0..=1.0 as hundredths of a percent, clamped. NaN reads as absent rather than as
/// zero: a division that produced no answer is not an observation of an idle machine.
pub fn percent_hundredths(ratio: f64) -> i16 {
    if ratio.is_nan() {
        return NOT_MEASURED;
    }
    let hundredths = (ratio * 10_000.0).round();
    if hundredths <= 0.0 {
        return 0;
    }
    if hundredths >= f64::from(PERCENT_CEILING) {
        return PERCENT_CEILING;
    }
    hundredths as i16
}

/// Bytes as megabytes, saturating at the `i16` ceiling. A host with more than 32 GB in one service
/// reports its ceiling instead of wrapping into a negative that would read as [`NOT_MEASURED`].
pub fn megabytes(bytes: u64) -> i16 {
    let megabytes = bytes / (1024 * 1024);
    i16::try_from(megabytes).unwrap_or(i16::MAX)
}

/// Bytes per second in 5 KB/s units, saturating. Rounds rather than truncates so a steady 3 KB/s
/// trickle registers as 1 instead of disappearing.
pub fn network_rate(bytes_per_second: f64) -> i16 {
    if bytes_per_second.is_nan() || bytes_per_second <= 0.0 {
        return 0;
    }
    let units = (bytes_per_second / NETWORK_RATE_UNIT_BYTES).round();
    if units >= f64::from(i16::MAX) {
        return i16::MAX;
    }
    units as i16
}

#[cfg(test)]
mod tests {
    use super::*;

    /// The window keeps the worst second, which is the entire reason this table samples at 1 s and
    /// writes at 5 s. The last sample winning, or the mean, would defeat the point.
    #[test]
    fn a_window_keeps_the_highest_sub_sample() {
        let mut window = WindowPeaks::default();
        for cpu in [1_000, 9_500, 200, 4_000, 300] {
            window.absorb(MetricsSample {
                cpu_percent: cpu,
                ..MetricsSample::default()
            });
        }
        assert_eq!(window.take().cpu_percent, 9_500);
    }

    /// Absence must lose to any real reading, and `0` is a real reading. Getting this backwards
    /// turns "the backend is on Lambda" into "the backend is idle", which is the one distinction
    /// the sentinel exists to preserve.
    #[test]
    fn not_measured_loses_to_every_real_reading_including_zero() {
        assert_eq!(peak(NOT_MEASURED, 0), 0);
        assert_eq!(peak(0, NOT_MEASURED), 0);
        assert_eq!(peak(NOT_MEASURED, NOT_MEASURED), NOT_MEASURED);
        assert_eq!(peak(50, NOT_MEASURED), 50);
        assert_eq!(peak(NOT_MEASURED, 50), 50);
    }

    /// A service absent for the whole window stays absent in the row, while one that answered once
    /// out of five is reported from that one answer.
    #[test]
    fn a_service_absent_all_window_stays_not_measured() {
        let mut window = WindowPeaks::default();
        window.absorb(MetricsSample::default());
        window.absorb(MetricsSample {
            search: ServiceSample {
                memory_mb: 128,
                cpu_percent: 0,
            },
            ..MetricsSample::default()
        });
        window.absorb(MetricsSample::default());

        let row = window.take();
        assert_eq!(row.backend.memory_mb, NOT_MEASURED);
        assert_eq!(row.backend.cpu_percent, NOT_MEASURED);
        assert_eq!(row.search.memory_mb, 128);
        assert_eq!(row.search.cpu_percent, 0);
    }

    /// Taking the peaks must also reset them, or the next window inherits this one's spike and the
    /// series never comes back down.
    #[test]
    fn taking_the_peaks_resets_the_accumulator() {
        let mut window = WindowPeaks::default();
        window.absorb(MetricsSample {
            cpu_percent: 8_000,
            ..MetricsSample::default()
        });
        assert!(!window.is_empty());
        assert_eq!(window.take().cpu_percent, 8_000);

        assert!(window.is_empty());
        window.absorb(MetricsSample {
            cpu_percent: 100,
            ..MetricsSample::default()
        });
        assert_eq!(window.take().cpu_percent, 100);
    }

    #[test]
    fn percentages_are_hundredths_and_clamped() {
        assert_eq!(percent_hundredths(0.2345), 2_345);
        assert_eq!(percent_hundredths(0.0), 0);
        assert_eq!(percent_hundredths(1.0), 10_000);
        // A service pinning more cores than the divisor accounted for cannot exceed the ceiling.
        assert_eq!(percent_hundredths(3.5), 10_000);
        // Negative would encode as the absence sentinel, so it clamps to zero instead.
        assert_eq!(percent_hundredths(-0.5), 0);
        assert_eq!(percent_hundredths(f64::NAN), NOT_MEASURED);
    }

    #[test]
    fn memory_saturates_instead_of_wrapping() {
        assert_eq!(megabytes(0), 0);
        assert_eq!(megabytes(32 * 1024 * 1024), 32);
        assert_eq!(megabytes(u64::MAX), i16::MAX);
    }

    #[test]
    fn network_rates_resolve_kilobytes_and_saturate_at_the_ceiling() {
        assert_eq!(network_rate(0.0), 0);
        // The idle traffic the live panel shows: visible as 1, not rounded away to 0.
        assert_eq!(network_rate(3_000.0), 1);
        assert_eq!(network_rate(5_120.0), 1);
        assert_eq!(network_rate(51_200.0), 10);
        // A saturated gigabit link clamps rather than wrapping negative.
        assert_eq!(network_rate(1_000_000_000.0), i16::MAX);
    }
}
