//! Reads one sub-sample of the machine from `/proc` and cgroup v2.
//!
//! Per-service numbers come from the unit's cgroup rather than from walking its PIDs: one file read
//! covers a multi-process service like Scylla correctly, and a missing directory is exactly the
//! "this service is not on the box" signal the Lambda case needs. Host-wide numbers mirror the
//! algorithms already in `backend/system/metrics_collector.go`, so the stored series and the live
//! SSE panel cannot disagree about what the same second looked like.
//!
//! Every read is fallible and no failure is fatal: a metric that could not be read comes back as
//! [`NOT_MEASURED`] and the rest of the sub-sample is still produced.

use std::{
    ffi::CString,
    fs,
    path::{Path, PathBuf},
    time::{Duration, Instant},
};

use crate::sysmetrics::{
    MetricsSample, NOT_MEASURED, ServiceSample, megabytes, network_rate, percent_hundredths,
};

/// The cgroup v2 mount, searched for each unit rather than assumed to be its parent.
///
/// The first version of this hardcoded `system.slice`, which was wrong on a stock install: Scylla's
/// packaging puts its unit in its own slice, at
/// `/sys/fs/cgroup/scylla.slice/scylla-server.slice/scylla-server.service`, so the one service most
/// worth watching was the one that always reported absent. A unit's slice is a packaging decision
/// that no reader of this code can predict, so it gets looked up instead of guessed.
const CGROUP_ROOT: &str = "/sys/fs/cgroup";

/// How deep the search for a unit's directory goes. Scylla's is three levels down; the limit keeps
/// a pathological tree (thousands of user scopes) from turning discovery into a full walk.
const CGROUP_SEARCH_DEPTH: usize = 5;

/// How long to wait before searching again for a unit that was not found. Long enough that a
/// permanently absent unit — the backend on Lambda — costs nothing, short enough that a service
/// started after the daemon appears within half a minute.
const CGROUP_SEARCH_RETRY: Duration = Duration::from_secs(30);

/// The four units a row reports on, in the order the writer binds them.
#[derive(Clone, Debug)]
pub struct ServiceUnits {
    pub backend: String,
    pub server_utils: String,
    pub search: String,
    pub scylla: String,
}

/// Counters from the previous sub-sample. Rates are deltas, so the first sub-sample after startup
/// can only establish this and contributes no value of its own.
///
/// Every counter is optional, and that is not tidiness. A read that failed must not leave a zero
/// behind: the next sub-sample would subtract from it and report the whole counter since boot — a
/// since-boot average for CPU, and for the network a rate that saturates the column. Because the
/// row keeps the window's maximum, one such artefact would win its window outright.
struct PreviousCounters {
    taken_at: Instant,
    host_cpu_ticks: Option<(u64, u64)>,
    network_bytes: Option<(u64, u64)>,
    service_cpu_usec: [Option<u64>; 4],
}

pub struct SystemMetricsCollector {
    cgroup_root: PathBuf,
    units: ServiceUnits,
    disk_mount: String,
    /// Empty means "the first interface that is not `lo`", matching the live panel's `iface=auto`.
    network_interface: String,
    /// Divides the per-service CPU delta, which is what turns eight busy cores into 100.00% of the
    /// machine instead of a top-style 800% that would not fit an i16 at two decimals.
    cpu_count: f64,
    previous: Option<PreviousCounters>,
    /// Each unit's cgroup directory once found, in the order of `unit_names`. Searching the tree on
    /// every sub-sample would be a directory walk a second for a path that almost never moves.
    resolved_directories: [Option<PathBuf>; 4],
    /// When each unit was last searched for, so a unit that is not on this box does not re-walk the
    /// tree every second forever.
    last_search_attempt: [Option<Instant>; 4],
}

impl SystemMetricsCollector {
    pub fn new(units: ServiceUnits, disk_mount: String, network_interface: String) -> Self {
        Self::with_cgroup_root(
            PathBuf::from(CGROUP_ROOT),
            units,
            disk_mount,
            network_interface,
        )
    }

    /// Same collector with the cgroup tree pointed elsewhere, which is how the tests exercise an
    /// absent unit without depending on what is running on the machine.
    pub fn with_cgroup_root(
        cgroup_root: PathBuf,
        units: ServiceUnits,
        disk_mount: String,
        network_interface: String,
    ) -> Self {
        let cpu_count = read_cpu_count().unwrap_or(1) as f64;
        Self {
            cgroup_root,
            units,
            disk_mount,
            network_interface,
            cpu_count,
            previous: None,
            resolved_directories: [None, None, None, None],
            last_search_attempt: [None; 4],
        }
    }

    /// One sub-sample, or `None` on the very first call, which only establishes the baseline the
    /// rate metrics are measured against.
    pub fn sample(&mut self) -> Option<MetricsSample> {
        let taken_at = Instant::now();
        let host_cpu = read_proc_stat_ticks();
        let network = read_network_counters(&self.network_interface);
        let (service_memory, service_cpu_usec) = self.read_service_counters();

        let previous = self.previous.replace(PreviousCounters {
            taken_at,
            host_cpu_ticks: host_cpu,
            network_bytes: network,
            service_cpu_usec,
        })?;

        // Real elapsed time, never the configured interval: a tick that arrived late would
        // otherwise inflate every rate in the sub-sample.
        let elapsed_micros = taken_at.duration_since(previous.taken_at).as_micros() as f64;

        let make_service = |index: usize| ServiceSample {
            memory_mb: service_memory[index].map(megabytes).unwrap_or(NOT_MEASURED),
            cpu_percent: service_cpu_percent(
                service_cpu_usec[index],
                previous.service_cpu_usec[index],
                elapsed_micros,
                self.cpu_count,
            ),
        };

        let (network_rx_rate, network_tx_rate) = network_rates(
            network,
            previous.network_bytes,
            elapsed_micros / 1_000_000.0,
        );

        Some(MetricsSample {
            cpu_percent: host_cpu_percent(host_cpu, previous.host_cpu_ticks),
            memory_percent: read_memory_percent(),
            disk_percent: read_disk_percent(&self.disk_mount),
            network_rx_rate,
            network_tx_rate,
            backend: make_service(0),
            server_utils: make_service(1),
            search: make_service(2),
            scylla: make_service(3),
        })
    }

    fn unit_names(&self) -> [&str; 4] {
        [
            &self.units.backend,
            &self.units.server_utils,
            &self.units.search,
            &self.units.scylla,
        ]
    }

    /// Memory bytes and CPU microseconds for all four units, resolving each unit's directory once
    /// per sub-sample rather than once per file.
    fn read_service_counters(&mut self) -> ([Option<u64>; 4], [Option<u64>; 4]) {
        let mut memory_bytes = [None; 4];
        let mut cpu_microseconds = [None; 4];
        for index in 0..4 {
            let Some(directory) = self.unit_directory(index) else {
                continue;
            };
            memory_bytes[index] = fs::read_to_string(directory.join("memory.stat"))
                .ok()
                .map(|contents| resident_memory_bytes(&contents));
            cpu_microseconds[index] = fs::read_to_string(directory.join("cpu.stat"))
                .ok()
                .and_then(|contents| parse_keyed_field(&contents, "usage_usec"));
        }
        (memory_bytes, cpu_microseconds)
    }

    /// The unit's cgroup directory, discovered once and remembered. A cached path that has stopped
    /// existing is dropped and searched for again, so a service restarted into a different slice —
    /// or one that was simply not running when the daemon started — is picked up rather than
    /// reported absent forever.
    ///
    /// A search that finds nothing is what needs the rate limit, not the successful one. An absent
    /// unit is the *normal* state for the backend on Lambda, and re-walking the cgroup tree every
    /// second in that case was measured at 12 ms a pass on a quiet laptop — a cost that scales with
    /// somebody else's container count and buys nothing, since a unit that is not there now is
    /// unlikely to appear in the next second.
    fn unit_directory(&mut self, index: usize) -> Option<PathBuf> {
        if let Some(cached) = &self.resolved_directories[index] {
            if cached.is_dir() {
                return Some(cached.clone());
            }
            self.resolved_directories[index] = None;
        }
        if self.last_search_attempt[index]
            .is_some_and(|attempted| attempted.elapsed() < CGROUP_SEARCH_RETRY)
        {
            return None;
        }
        self.last_search_attempt[index] = Some(Instant::now());

        let unit = self.unit_names()[index].trim().to_owned();
        if unit.is_empty() {
            return None;
        }
        let found = find_cgroup_directory(&self.cgroup_root, &unit, CGROUP_SEARCH_DEPTH)?;
        self.resolved_directories[index] = Some(found.clone());
        Some(found)
    }
}

/// What the service actually holds in RAM: anonymous pages plus the file pages it has mapped.
///
/// `anon` alone was the first attempt and it under-reports badly — measured against `/proc`, it
/// matches `RssAnon` exactly but misses `RssFile`, which for a Go binary is 29 MB of the 41 MB the
/// process really occupies. `anon + file` would be worse in the other direction: it drags in the
/// cold page cache charged to the cgroup, 88 MB of it for a quiet Scylla. `file_mapped` is the part
/// of that cache the service has actually mapped, so the sum reconstructs `VmRSS` — which is what
/// the live panel shows, so both views of the same second now agree.
///
/// Falls back to `anon` where `file_mapped` is absent: under-reporting is better than reporting
/// nothing, and the field is only missing on kernels this daemon is unlikely to meet.
fn resident_memory_bytes(memory_stat: &str) -> u64 {
    let anonymous = parse_keyed_field(memory_stat, "anon").unwrap_or(0);
    anonymous + parse_keyed_field(memory_stat, "file_mapped").unwrap_or(0)
}

/// Depth-first search for the directory a unit's cgroup lives in, by exact directory name.
fn find_cgroup_directory(root: &Path, unit: &str, remaining_depth: usize) -> Option<PathBuf> {
    if remaining_depth == 0 {
        return None;
    }
    let entries = fs::read_dir(root).ok()?;
    let mut subdirectories = Vec::new();
    for entry in entries.flatten() {
        if !entry.file_type().is_ok_and(|kind| kind.is_dir()) {
            continue;
        }
        // A match at this level wins immediately: slices nest, but a unit name appears once.
        if entry.file_name() == unit {
            return Some(entry.path());
        }
        subdirectories.push(entry.path());
    }
    subdirectories
        .into_iter()
        .find_map(|directory| find_cgroup_directory(&directory, unit, remaining_depth - 1))
}

/// Busy share of the host over the window. Absent on either side of the subtraction means absent
/// in the row: without a predecessor the only honest answer is that this second was not observed,
/// and the tempting one — the counters as they stand — would be the average since boot.
fn host_cpu_percent(current: Option<(u64, u64)>, previous: Option<(u64, u64)>) -> i16 {
    let (Some((total, idle)), Some((previous_total, previous_idle))) = (current, previous) else {
        return NOT_MEASURED;
    };
    let total_delta = total.saturating_sub(previous_total) as f64;
    if total_delta <= 0.0 {
        return NOT_MEASURED;
    }
    let idle_delta = idle.saturating_sub(previous_idle) as f64;
    percent_hundredths((total_delta - idle_delta) / total_delta)
}

/// A service's share of the WHOLE machine: its cgroup microseconds over the wall-clock
/// microseconds every core could have supplied.
fn service_cpu_percent(
    current: Option<u64>,
    previous: Option<u64>,
    elapsed_micros: f64,
    cpu_count: f64,
) -> i16 {
    let (Some(current), Some(previous)) = (current, previous) else {
        return NOT_MEASURED;
    };
    if elapsed_micros <= 0.0 || cpu_count <= 0.0 {
        return NOT_MEASURED;
    }
    percent_hundredths(current.saturating_sub(previous) as f64 / (elapsed_micros * cpu_count))
}

/// Received and transmitted bytes per second, in the stored 5 KB/s units.
fn network_rates(
    current: Option<(u64, u64)>,
    previous: Option<(u64, u64)>,
    elapsed_seconds: f64,
) -> (i16, i16) {
    let (Some((received, transmitted)), Some((previous_received, previous_transmitted))) =
        (current, previous)
    else {
        return (NOT_MEASURED, NOT_MEASURED);
    };
    if elapsed_seconds <= 0.0 {
        return (NOT_MEASURED, NOT_MEASURED);
    }
    (
        // A counter that went backwards is an interface reset, not negative traffic.
        network_rate(received.saturating_sub(previous_received) as f64 / elapsed_seconds),
        network_rate(transmitted.saturating_sub(previous_transmitted) as f64 / elapsed_seconds),
    )
}

/// Finds `<field> <value>` in the flat `key value` format cgroup v2 uses for `cpu.stat` and
/// `memory.stat`.
fn parse_keyed_field(contents: &str, field: &str) -> Option<u64> {
    contents.lines().find_map(|line| {
        let (name, value) = line.split_once(' ')?;
        if name.trim() != field {
            return None;
        }
        value.trim().parse::<u64>().ok()
    })
}

/// Total and idle jiffies from the aggregate `cpu` line, the same two numbers the Go collector
/// takes: idle only, with iowait counted as busy.
fn read_proc_stat_ticks() -> Option<(u64, u64)> {
    let contents = fs::read_to_string("/proc/stat").ok()?;
    parse_proc_stat_ticks(&contents)
}

fn parse_proc_stat_ticks(contents: &str) -> Option<(u64, u64)> {
    let line = contents.lines().find(|line| line.starts_with("cpu "))?;
    let mut total = 0_u64;
    let mut idle = 0_u64;
    for (index, field) in line.split_whitespace().skip(1).enumerate() {
        let ticks = field.parse::<u64>().ok()?;
        total += ticks;
        // The fourth counter of the line is idle time.
        if index == 3 {
            idle = ticks;
        }
    }
    if total == 0 {
        None
    } else {
        Some((total, idle))
    }
}

/// How many cores the per-service CPU percentage is measured against, counted from the `cpuN`
/// lines rather than pulled in as a dependency.
fn read_cpu_count() -> Option<usize> {
    let contents = fs::read_to_string("/proc/stat").ok()?;
    Some(parse_cpu_count(&contents)).filter(|count| *count > 0)
}

fn parse_cpu_count(contents: &str) -> usize {
    contents
        .lines()
        .filter(|line| {
            line.strip_prefix("cpu")
                .is_some_and(|rest| rest.starts_with(|character: char| character.is_ascii_digit()))
        })
        .count()
}

fn read_memory_percent() -> i16 {
    match fs::read_to_string("/proc/meminfo")
        .ok()
        .and_then(|contents| parse_memory_used_ratio(&contents))
    {
        Some(ratio) => percent_hundredths(ratio),
        None => NOT_MEASURED,
    }
}

/// Used over total, with "used" defined as total minus MemAvailable — cache and reclaimable slab
/// count as free, which is the number a person means by "memory in use".
fn parse_memory_used_ratio(contents: &str) -> Option<f64> {
    let read_kilobytes = |field: &str| -> Option<u64> {
        contents.lines().find_map(|line| {
            let (name, value) = line.split_once(':')?;
            if name.trim() != field {
                return None;
            }
            value.split_whitespace().next()?.parse::<u64>().ok()
        })
    };
    let total = read_kilobytes("MemTotal")?;
    let available = read_kilobytes("MemAvailable")?;
    if total == 0 {
        return None;
    }
    Some((total.saturating_sub(available)) as f64 / total as f64)
}

fn read_network_counters(requested_interface: &str) -> Option<(u64, u64)> {
    let contents = fs::read_to_string("/proc/net/dev").ok()?;
    parse_network_counters(&contents, requested_interface)
}

/// Received and transmitted byte counters for the configured interface, or for the first interface
/// that is not `lo` when none was configured.
fn parse_network_counters(contents: &str, requested_interface: &str) -> Option<(u64, u64)> {
    let wanted = requested_interface.trim();
    for line in contents.lines() {
        let Some((name, counters)) = line.split_once(':') else {
            continue;
        };
        let name = name.trim();
        if name == "lo" || (!wanted.is_empty() && name != wanted) {
            continue;
        }
        let fields: Vec<&str> = counters.split_whitespace().collect();
        if fields.len() < 16 {
            continue;
        }
        // Columns of /proc/net/dev: received bytes first, transmitted bytes ninth.
        let received = fields[0].parse::<u64>().ok()?;
        let transmitted = fields[8].parse::<u64>().ok()?;
        return Some((received, transmitted));
    }
    None
}

/// Mount usage from `statvfs`, the one syscall in this module: no file under `/proc` reports free
/// space. Blocks minus *available* blocks, matching the Go collector and `df` — the root reserve
/// counts as used, because nothing that is not root can have it.
fn read_disk_percent(mount_path: &str) -> i16 {
    let path = mount_path.trim();
    let path = if path.is_empty() { "/" } else { path };
    let Ok(path) = CString::new(path) else {
        return NOT_MEASURED;
    };

    let mut stats: libc::statvfs = unsafe { std::mem::zeroed() };
    if unsafe { libc::statvfs(path.as_ptr(), &mut stats) } != 0 {
        return NOT_MEASURED;
    }
    let total_blocks = stats.f_blocks as u64;
    if total_blocks == 0 {
        return NOT_MEASURED;
    }
    let used_blocks = total_blocks.saturating_sub(stats.f_bavail as u64);
    percent_hundredths(used_blocks as f64 / total_blocks as f64)
}

#[cfg(test)]
mod tests {
    use super::*;

    const CPU_STAT: &str = "usage_usec 1234567\nuser_usec 900000\nsystem_usec 334567\n";
    const MEMORY_STAT: &str = "anon 268435456\nfile 1073741824\nkernel_stack 65536\n";

    /// The bug this replaced: Scylla's unit is not under `system.slice`, it is three levels down
    /// its own slice, so the hardcoded parent path reported the most important service on the box
    /// as absent on a stock install. The search has to find a nested unit and still refuse one that
    /// is not there.
    #[test]
    fn a_unit_is_found_wherever_its_slice_puts_it() {
        // The real layout from a Fedora host: /sys/fs/cgroup is the root, and Scylla nests.
        let root = std::env::temp_dir().join("genix-cgroup-search-test");
        let scylla = root
            .join("scylla.slice")
            .join("scylla-server.slice")
            .join("scylla-server.service");
        let backend = root.join("system.slice").join("genix.service");
        fs::create_dir_all(&scylla).unwrap();
        fs::create_dir_all(&backend).unwrap();

        assert_eq!(
            find_cgroup_directory(&root, "scylla-server.service", CGROUP_SEARCH_DEPTH),
            Some(scylla)
        );
        assert_eq!(
            find_cgroup_directory(&root, "genix.service", CGROUP_SEARCH_DEPTH),
            Some(backend)
        );
        assert_eq!(
            find_cgroup_directory(&root, "genixsearch.service", CGROUP_SEARCH_DEPTH),
            None
        );
        // The depth limit is a real bound, not decoration: Scylla sits below two slices.
        assert_eq!(
            find_cgroup_directory(&root, "scylla-server.service", 2),
            None
        );

        fs::remove_dir_all(&root).ok();
    }

    /// The absent unit is the *normal* case — the backend on Lambda has no cgroup at all — so it
    /// must not re-walk the tree on every sub-sample. Observable as the directory appearing without
    /// the collector noticing until the retry window elapses; the same rate limit is what keeps a
    /// permanently missing unit from costing a walk a second forever.
    #[test]
    fn an_absent_unit_is_not_searched_for_again_on_the_next_sample() {
        let root = std::env::temp_dir().join("genix-cgroup-retry-test");
        fs::create_dir_all(&root).unwrap();
        let mut collector = SystemMetricsCollector::with_cgroup_root(
            root.clone(),
            ServiceUnits {
                backend: "late.service".into(),
                server_utils: String::new(),
                search: String::new(),
                scylla: String::new(),
            },
            "/".into(),
            String::new(),
        );

        assert!(collector.unit_directory(0).is_none(), "nothing to find yet");

        fs::create_dir_all(root.join("system.slice").join("late.service")).unwrap();
        assert!(
            collector.unit_directory(0).is_none(),
            "the retry window has not elapsed, so no second walk happened"
        );

        // Clearing the attempt is what the elapsed window does, and then it is found.
        collector.last_search_attempt[0] = None;
        assert!(collector.unit_directory(0).is_some());

        fs::remove_dir_all(&root).ok();
    }

    /// Measured against `/proc` on a real host: `anon` alone matched `RssAnon` but missed `RssFile`,
    /// which for a Go backend was 29 MB of the 41 MB it actually held, while `anon + file` would
    /// have added 88 MB of cold cache to a quiet Scylla. These are that host's real numbers.
    #[test]
    fn resident_memory_is_anonymous_plus_mapped_file() {
        let backend =
            "anon 12308480\nfile 56344576\nfile_mapped 29622272\ninactive_file 55332864\n";
        // The kernel reported VmRSS 40948 kB for that same process. The sum reproduces it to the
        // byte, which is the whole claim of this formula.
        assert_eq!(resident_memory_bytes(backend), 40_948 * 1024);
        // Stored megabytes floor, so 39.99 MB is reported as 39 and never rounded up past reality.
        assert_eq!(megabytes(resident_memory_bytes(backend)), 39);

        // Cold page cache stays out: `file` is 56 MB but only the mapped part counts.
        assert!(resident_memory_bytes(backend) < 56_344_576);

        // A kernel without file_mapped falls back to anon rather than reporting nothing.
        assert_eq!(
            resident_memory_bytes("anon 12308480\nfile 56344576\n"),
            12_308_480
        );
        assert_eq!(resident_memory_bytes("nothing_useful 1\n"), 0);
    }

    #[test]
    fn cgroup_files_are_read_by_field_name() {
        assert_eq!(parse_keyed_field(CPU_STAT, "usage_usec"), Some(1_234_567));
        assert_eq!(parse_keyed_field(MEMORY_STAT, "anon"), Some(268_435_456));
        // `file` is the page cache and deliberately not what the memory column reports.
        assert_eq!(parse_keyed_field(MEMORY_STAT, "file"), Some(1_073_741_824));
        assert_eq!(parse_keyed_field(MEMORY_STAT, "missing"), None);
    }

    /// The failure that would be invisible: a `/proc` read that fails leaves no predecessor, and
    /// subtracting from a zero baseline would report the counters since boot — a plausible-looking
    /// CPU average and a network rate that pins the column at its ceiling. Since the row keeps the
    /// window's maximum, one such artefact would win its window outright.
    #[test]
    fn a_missing_predecessor_reports_absent_rather_than_the_counter_since_boot() {
        // A busy box that has been up for a while: 40% busy since boot, 300 MB transferred.
        let since_boot_cpu = Some((1_000_000, 600_000));
        let since_boot_network = Some((300_000_000, 300_000_000));

        assert_eq!(host_cpu_percent(since_boot_cpu, None), NOT_MEASURED);
        assert_eq!(
            network_rates(since_boot_network, None, 1.0),
            (NOT_MEASURED, NOT_MEASURED)
        );
        assert_eq!(
            service_cpu_percent(Some(500_000), None, 1_000_000.0, 4.0),
            NOT_MEASURED
        );

        // With a predecessor, the same counters describe the second that just passed.
        assert_eq!(
            host_cpu_percent(since_boot_cpu, Some((999_000, 599_800))),
            8_000
        );
        // 5.12 MB in the last second received, half that transmitted, in 5 KB/s units.
        assert_eq!(
            network_rates(since_boot_network, Some((294_880_000, 297_440_000)), 1.0),
            (1_000, 500)
        );
    }

    /// A service is measured against every core, so half of a four-core box is 50.00% and not the
    /// 200% a top-style number would give.
    #[test]
    fn a_service_is_a_share_of_the_whole_machine() {
        assert_eq!(
            service_cpu_percent(Some(2_000_000), Some(0), 1_000_000.0, 4.0),
            5_000
        );
        // Time that did not pass cannot be divided by.
        assert_eq!(
            service_cpu_percent(Some(2_000_000), Some(0), 0.0, 4.0),
            NOT_MEASURED
        );
    }

    #[test]
    fn proc_stat_yields_total_and_idle_ticks() {
        let contents = "cpu  100 20 30 400 5 0 0 0 0 0\ncpu0 50 10 15 200 2 0 0 0 0 0\ncpu1 50 10 15 200 3 0 0 0 0 0\nintr 999\n";
        assert_eq!(parse_proc_stat_ticks(contents), Some((555, 400)));
        // Two cores, and the aggregate `cpu ` line must not be counted as one of them.
        assert_eq!(parse_cpu_count(contents), 2);
    }

    #[test]
    fn memory_used_excludes_cache_via_mem_available() {
        let contents = "MemTotal:       16000000 kB\nMemFree:         1000000 kB\nMemAvailable:    4000000 kB\n";
        let ratio = parse_memory_used_ratio(contents).unwrap();
        assert_eq!(percent_hundredths(ratio), 7_500);
        assert_eq!(
            parse_memory_used_ratio("MemTotal: 0 kB\nMemAvailable: 0 kB\n"),
            None
        );
        assert_eq!(parse_memory_used_ratio("MemTotal: 100 kB\n"), None);
    }

    #[test]
    fn network_counters_skip_loopback_and_honour_a_named_interface() {
        let contents = concat!(
            "Inter-|   Receive                                                |  Transmit\n",
            " face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed\n",
            "    lo: 999999    100    0    0    0     0          0         0   999999     100    0    0    0     0       0          0\n",
            "  eth0: 500000    400    0    0    0     0          0         0   250000     300    0    0    0     0       0          0\n",
            "  eth1: 111111    111    0    0    0     0          0         0   222222     222    0    0    0     0       0          0\n",
        );
        // Auto picks the first interface that is not loopback.
        assert_eq!(
            parse_network_counters(contents, ""),
            Some((500_000, 250_000))
        );
        assert_eq!(
            parse_network_counters(contents, "eth1"),
            Some((111_111, 222_222))
        );
        assert_eq!(parse_network_counters(contents, "wlan0"), None);
    }

    /// The Lambda case, and the reason the sentinel exists: no cgroup directory means the service
    /// reports absent, not idle, and the rest of the sub-sample is still produced.
    #[test]
    fn an_absent_unit_reports_not_measured_rather_than_zero() {
        let mut collector = SystemMetricsCollector::with_cgroup_root(
            PathBuf::from("/nonexistent/cgroup/root"),
            ServiceUnits {
                backend: "genix.service".into(),
                server_utils: "genix-server-utils.service".into(),
                search: "genixsearch.service".into(),
                scylla: "scylla-server.service".into(),
            },
            "/".into(),
            String::new(),
        );

        // The first call only establishes the baseline the rates are measured against.
        assert!(collector.sample().is_none());

        let sample = collector
            .sample()
            .expect("the second sample must produce a row");
        for service in [
            sample.backend,
            sample.server_utils,
            sample.search,
            sample.scylla,
        ] {
            assert_eq!(service.memory_mb, NOT_MEASURED);
            assert_eq!(service.cpu_percent, NOT_MEASURED);
        }
        // Host-wide metrics come from /proc and are unaffected by the missing cgroup tree.
        assert!(sample.disk_percent > NOT_MEASURED);
        assert!(sample.memory_percent > NOT_MEASURED);
    }
}
