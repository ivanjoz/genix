//! Loads rate-limiter and Scylla settings with environment-over-TOML precedence.

use std::{env, fs, net::SocketAddr, path::PathBuf, time::Duration};

use anyhow::{Context, Result, bail};
use toml::{Table, Value};

use crate::{
    limiter::quota::{CreditLimits, LimitPolicy, ScopeLimits},
    lock::registry::LockLimits,
};

const DEFAULT_LISTEN_PORT: u16 = 14013;
const DEFAULT_FLUSH_SECONDS: u64 = 15;
/// How long a user's authorization grants stay cached. The backend invalidates actively after
/// rewriting them, so this bounds only the case where that frame was lost or the backend restarted.
const DEFAULT_ACCESS_CACHE_SECONDS: u64 = 600;
const DEFAULT_FRAME_TIMEOUT_SECONDS: u64 = 30;
const DEFAULT_MAX_CONNECTIONS: usize = 1_024;
/// Requests one connection may have in flight at once. With one request per socket the socket
/// itself was the ceiling; multiplexing removes that, so it has to be stated.
const DEFAULT_MAX_INFLIGHT_PER_CONNECTION: usize = 64;
const DEFAULT_LOCK_MAX_KEYS: usize = 100_000;
const DEFAULT_LOCK_MAX_TOTAL_WAITERS: u64 = 4_096;
/// The wire carries the lease as a `u16` of milliseconds, so no ceiling above 65_535 is
/// reachable in the first place.
const DEFAULT_LOCK_MAX_LEASE_MS: u64 = 60_000;
/// Keeps the bridge next to the other Genix services on the same hosts (14008 ScyllaDB,
/// 14010 backend, 14013 this process's rate limiter, 14446 GenixSearch).
const DEFAULT_BRIDGE_PORT: u16 = 14012;

/// A month of request history is what a "what broke last week" question needs; the partition is
/// the date, so a whole day expires together and Scylla drops it wholesale.
const DEFAULT_REQUEST_LOG_TTL_DAYS: u64 = 30;
/// Long enough to batch a busy second, short enough that a row is queryable while the person who
/// caused it is still looking at the screen.
const DEFAULT_REQUEST_LOG_FLUSH_MS: u64 = 1_000;
const DEFAULT_REQUEST_LOG_MAX_BATCH: usize = 128;
/// How long a written `request_errors` row is considered fresh enough to leave alone.
const DEFAULT_REQUEST_LOG_ERROR_CACHE_SECONDS: u64 = 600;
/// Bounds the suppression map. Distinct failing code lines are bounded by the codebase, so this is
/// a backstop against pathology rather than a working limit.
const DEFAULT_REQUEST_LOG_ERROR_CACHE_ENTRIES: usize = 20_000;
/// Records waiting to be written. Past this the writer drops them: a log row must never be the
/// reason a request, or this daemon, slows down.
const DEFAULT_REQUEST_LOG_QUEUE_CAPACITY: usize = 8_192;

/// One sub-sample a second, and the peak of five of them per row. Both mirror the Go side:
/// `ServerMetricSlotSeconds` in backend/core/types/server_metrics.go is 5, and the clustering key
/// of every stored row already means that. Changing `row_seconds` alone would leave the two sides
/// disagreeing about what slot 4 covers.
const DEFAULT_SERVER_METRICS_SAMPLE_SECONDS: u64 = 1;
const DEFAULT_SERVER_METRICS_ROW_SECONDS: u64 = 5;
/// A month of history, matching the request log. The partition is the day, so it expires whole.
const DEFAULT_SERVER_METRICS_TTL_DAYS: u64 = 30;
/// A row is 17280 slots a day of int16s, so the useful floor is set by the int16 key rather than by
/// storage: fewer than three seconds a slot overflows it.
const MINIMUM_SERVER_METRICS_ROW_SECONDS: u64 = 3;
const DEFAULT_SERVER_METRICS_BACKEND_UNIT: &str = "genix.service";
const DEFAULT_SERVER_METRICS_SERVER_UTILS_UNIT: &str = "genix-server-utils.service";
const DEFAULT_SERVER_METRICS_SEARCH_UNIT: &str = "genixsearch.service";
const DEFAULT_SERVER_METRICS_SCYLLA_UNIT: &str = "scylla-server.service";

#[derive(Clone, Debug)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub keyspace: String,
    pub user: String,
    pub password: String,
}

#[derive(Clone, Debug)]
pub struct AppConfig {
    pub listen_address: SocketAddr,
    pub flush_interval: Duration,
    pub frame_timeout: Duration,
    pub max_connections: usize,
    pub max_inflight_per_connection: usize,
    pub shard_count: usize,
    /// TTL of the cached `users.accesos_computed` grants the charge frame is authorized against.
    pub access_cache_seconds: i64,
    /// Signs the browser session tokens the SSE bridge verifies. Nothing else reads it.
    pub secret_phrase: Vec<u8>,
    /// Service-to-service secret: the rate limiter's TCP frame HMAC and the bridge's
    /// `X-Bridge-Auth` header. Distinct from `secret_phrase` so token signing and
    /// inter-service authentication can be rotated independently.
    pub internal_apikey: Vec<u8>,
    pub database: DatabaseConfig,
    pub policy: LimitPolicy,
    /// Process-wide ceilings for the lock service. Per-action policy is deliberately absent:
    /// that belongs to the Go call sites, which is what keeps this service generic.
    pub locks: LockLimits,
    pub bridge: BridgeConfig,
    pub request_log: RequestLogConfig,
    pub server_metrics: ServerMetricsConfig,
}

#[derive(Clone, Debug)]
pub struct ServerMetricsConfig {
    /// Off means no sampling loop at all. Unlike the request log there is no client to keep from
    /// breaking here — nothing calls into this, it only ticks.
    pub enabled: bool,
    pub sample_interval: Duration,
    /// Seconds per stored row, which is also what the clustering key counts. Kept as a number
    /// rather than a Duration because it is arithmetic on the key, not a delay.
    pub row_seconds: i64,
    pub row_ttl: Duration,
    pub disk_mount: String,
    /// Empty means the first interface that is not `lo`.
    pub network_interface: String,
    /// systemd units under `system.slice`. An absent one is not an error: the backend has no unit
    /// at all when it runs on Lambda, which is the case the `-1` sentinel exists for.
    pub backend_unit: String,
    pub server_utils_unit: String,
    pub search_unit: String,
    pub scylla_unit: String,
}

#[derive(Clone, Debug)]
pub struct RequestLogConfig {
    /// Off means opcode 0x04 is accepted and discarded rather than refused. Refusing would close
    /// the connection of a backend that is only trying to log, taking its charges and locks with
    /// it — the switch is for not writing rows, not for breaking clients.
    pub enabled: bool,
    pub row_ttl: Duration,
    pub flush_interval: Duration,
    pub max_batch: usize,
    pub error_freshness: Duration,
    pub error_cache_entries: usize,
    pub queue_capacity: usize,
}

#[derive(Clone, Debug)]
pub struct BridgeConfig {
    /// The SSE bridge's HTTP listener. Bound on all interfaces because nginx terminates TLS
    /// in front of it, matching the Go bridge this replaced.
    pub listen_address: SocketAddr,
    pub verbose_logs: bool,
}

impl AppConfig {
    pub fn load() -> Result<Self> {
        let config = load_config()?;
        // [server_utils] is its own section, not a key under [rate_limit]: this one port serves
        // every raw-TCP operation and the frame's opcode picks the service, so it belongs to the
        // process, not to one of the services running inside it.
        //
        // The bind address is DERIVED from `public` and never read from `host`, which is the
        // client's half of this section. Binding a literal host cannot work behind NAT — a cloud
        // VM's public IP is never on its NIC, so bind returns EADDRNOTAVAIL and the daemon dies at
        // startup. A boolean has no such unrepresentable state.
        let listen_port = optional_u64(&config, "SERVER_UTILS_PORT", "server_utils.port")?
            .unwrap_or(u64::from(DEFAULT_LISTEN_PORT));
        let listen_port = u16::try_from(listen_port)
            .ok()
            .filter(|port| *port > 0)
            .context("server_utils.port must be a port number between 1 and 65535")?;
        let listen_octets = if optional_bool(&config, "SERVER_UTILS_PUBLIC", "server_utils.public")
        {
            [0, 0, 0, 0]
        } else {
            [127, 0, 0, 1]
        };
        let listen_address = SocketAddr::from((listen_octets, listen_port));
        let flush_seconds = optional_u64(
            &config,
            "RATE_LIMIT_FLUSH_SECONDS",
            "rate_limit.flush_seconds",
        )?
        .unwrap_or(DEFAULT_FLUSH_SECONDS);
        let frame_timeout_seconds = optional_u64(
            &config,
            "RATE_LIMIT_FRAME_TIMEOUT_SECONDS",
            "rate_limit.frame_timeout_seconds",
        )?
        .unwrap_or(DEFAULT_FRAME_TIMEOUT_SECONDS);
        let max_connections = optional_usize(
            &config,
            "RATE_LIMIT_MAX_CONNECTIONS",
            "rate_limit.max_connections",
        )?
        .unwrap_or(DEFAULT_MAX_CONNECTIONS);
        let default_shards = std::thread::available_parallelism()
            .map(usize::from)
            .unwrap_or(1);
        let configured_shards =
            optional_usize(&config, "RATE_LIMIT_SHARDS", "rate_limit.shards")?.unwrap_or(0);
        let shard_count = if configured_shards == 0 {
            default_shards
        } else {
            configured_shards
        };

        let max_inflight_per_connection = optional_usize(
            &config,
            "RATE_LIMIT_MAX_INFLIGHT_PER_CONNECTION",
            "rate_limit.max_inflight_per_connection",
        )?
        .unwrap_or(DEFAULT_MAX_INFLIGHT_PER_CONNECTION);
        let access_cache_seconds = optional_u64(
            &config,
            "RATE_LIMIT_ACCESS_CACHE_SECONDS",
            "rate_limit.access_cache_seconds",
        )?
        .unwrap_or(DEFAULT_ACCESS_CACHE_SECONDS);

        if flush_seconds == 0
            || frame_timeout_seconds == 0
            || max_connections == 0
            || max_inflight_per_connection == 0
            || access_cache_seconds == 0
        {
            bail!("rate-limiter durations and connection limits must be positive");
        }

        let policy = LimitPolicy {
            company: load_scope_limits(&config, "COMPANY", "company")?,
            user: load_scope_limits(&config, "USER", "user")?,
        };
        validate_policy(policy)?;

        let lock_max_keys = optional_usize(&config, "LOCK_MAX_KEYS", "lock.max_keys")?
            .unwrap_or(DEFAULT_LOCK_MAX_KEYS);
        let lock_max_total_waiters =
            optional_u64(&config, "LOCK_MAX_TOTAL_WAITERS", "lock.max_total_waiters")?
                .unwrap_or(DEFAULT_LOCK_MAX_TOTAL_WAITERS);
        let lock_max_lease_ms = optional_u64(&config, "LOCK_MAX_LEASE_MS", "lock.max_lease_ms")?
            .unwrap_or(DEFAULT_LOCK_MAX_LEASE_MS);
        if lock_max_keys == 0 || lock_max_total_waiters == 0 || lock_max_lease_ms == 0 {
            bail!("lock ceilings must be positive");
        }
        let locks = LockLimits {
            max_keys: lock_max_keys,
            max_total_waiters: u32::try_from(lock_max_total_waiters)
                .context("lock.max_total_waiters must fit in uint32")?,
            max_lease: Duration::from_millis(lock_max_lease_ms),
        };

        // Every value here has a default, unlike the eight credit ceilings: a guessed quota is
        // worse than none, but a guessed flush interval for a log table is simply a flush interval.
        // An absent [request_log] section therefore means "on, with these", not a refusal to start.
        let request_log_ttl_days =
            optional_u64(&config, "REQUEST_LOG_TTL_DAYS", "request_log.ttl_days")?
                .unwrap_or(DEFAULT_REQUEST_LOG_TTL_DAYS);
        let request_log_flush_ms =
            optional_u64(&config, "REQUEST_LOG_FLUSH_MS", "request_log.flush_ms")?
                .unwrap_or(DEFAULT_REQUEST_LOG_FLUSH_MS);
        let request_log_max_batch =
            optional_usize(&config, "REQUEST_LOG_MAX_BATCH", "request_log.max_batch")?
                .unwrap_or(DEFAULT_REQUEST_LOG_MAX_BATCH);
        let request_log_error_cache_seconds = optional_u64(
            &config,
            "REQUEST_LOG_ERROR_CACHE_SECONDS",
            "request_log.error_cache_seconds",
        )?
        .unwrap_or(DEFAULT_REQUEST_LOG_ERROR_CACHE_SECONDS);
        let request_log_error_cache_entries = optional_usize(
            &config,
            "REQUEST_LOG_ERROR_CACHE_ENTRIES",
            "request_log.error_cache_entries",
        )?
        .unwrap_or(DEFAULT_REQUEST_LOG_ERROR_CACHE_ENTRIES);
        let request_log_queue_capacity = optional_usize(
            &config,
            "REQUEST_LOG_QUEUE_CAPACITY",
            "request_log.queue_capacity",
        )?
        .unwrap_or(DEFAULT_REQUEST_LOG_QUEUE_CAPACITY);
        if request_log_ttl_days == 0
            || request_log_flush_ms == 0
            || request_log_max_batch == 0
            || request_log_error_cache_entries == 0
            || request_log_queue_capacity == 0
        {
            bail!("request_log intervals, batch sizes and capacities must be positive");
        }
        // Scylla's TTL is seconds in an i32, so a ttl_days that overflows it would be silently
        // truncated into a retention nobody asked for.
        let request_log_ttl_seconds = request_log_ttl_days
            .checked_mul(86_400)
            .filter(|seconds| *seconds <= i32::MAX as u64)
            .context("request_log.ttl_days is too large for a ScyllaDB TTL")?;
        let request_log = RequestLogConfig {
            enabled: optional_string(&config, "REQUEST_LOG_ENABLED", "request_log.enabled")
                .is_none_or(|value| value == "1" || value.eq_ignore_ascii_case("true")),
            row_ttl: Duration::from_secs(request_log_ttl_seconds),
            flush_interval: Duration::from_millis(request_log_flush_ms),
            max_batch: request_log_max_batch,
            error_freshness: Duration::from_secs(request_log_error_cache_seconds),
            error_cache_entries: request_log_error_cache_entries,
            queue_capacity: request_log_queue_capacity,
        };

        let server_metrics = load_server_metrics(&config)?;

        let bridge_port = optional_u64(&config, "SSE_BRIDGE_PORT", "sse_bridge.port")?
            .map(|port| u16::try_from(port).context("sse_bridge.port must be a valid TCP port"))
            .transpose()?
            .unwrap_or(DEFAULT_BRIDGE_PORT);
        if bridge_port == 0 {
            bail!("sse_bridge.port must be greater than zero");
        }

        Ok(Self {
            listen_address,
            flush_interval: Duration::from_secs(flush_seconds),
            frame_timeout: Duration::from_secs(frame_timeout_seconds),
            max_connections,
            max_inflight_per_connection,
            shard_count,
            access_cache_seconds: access_cache_seconds as i64,
            secret_phrase: required_string(&config, "SECRET_PHRASE", "secret_phrase")?.into_bytes(),
            internal_apikey: required_string(&config, "INTERNAL_APIKEY", "internal_apikey")?
                .into_bytes(),
            bridge: BridgeConfig {
                listen_address: SocketAddr::from(([0, 0, 0, 0], bridge_port)),
                verbose_logs: optional_string(&config, "SSE_BRIDGE_VERBOSE", "sse_bridge.verbose")
                    .is_some_and(|value| value == "1" || value.eq_ignore_ascii_case("true")),
            },
            database: DatabaseConfig {
                host: required_string(&config, "DB_HOST", "db.host")?,
                port: required_u16(&config, "DB_PORT", "db.port")?,
                keyspace: required_string(&config, "DB_NAME", "db.name")?,
                user: required_string(&config, "DB_USER", "db.user")?,
                password: required_string(&config, "DB_PASSWORD", "db.password")?,
            },
            policy,
            locks,
            request_log,
            server_metrics,
        })
    }
}

/// Every value defaults, like the request log's: a guessed sampling interval is a sampling
/// interval, not a guessed quota. An absent `[server_metrics]` section means "on, with these".
fn load_server_metrics(config: &Table) -> Result<ServerMetricsConfig> {
    let sample_seconds = optional_u64(
        config,
        "SERVER_METRICS_SAMPLE_SECONDS",
        "server_metrics.sample_seconds",
    )?
    .unwrap_or(DEFAULT_SERVER_METRICS_SAMPLE_SECONDS);
    let row_seconds = optional_u64(
        config,
        "SERVER_METRICS_ROW_SECONDS",
        "server_metrics.row_seconds",
    )?
    .unwrap_or(DEFAULT_SERVER_METRICS_ROW_SECONDS);
    let ttl_days = optional_u64(config, "SERVER_METRICS_TTL_DAYS", "server_metrics.ttl_days")?
        .unwrap_or(DEFAULT_SERVER_METRICS_TTL_DAYS);

    // row_seconds decides what the clustering key MEANS, so it is validated rather than trusted.
    // A value that does not divide the day leaves a short last slot whose peak covers a different
    // span than every other row; below three seconds the slot count overflows the int16 key.
    if sample_seconds == 0 || ttl_days == 0 {
        bail!("server_metrics.sample_seconds and ttl_days must be positive");
    }
    if row_seconds < MINIMUM_SERVER_METRICS_ROW_SECONDS || 86_400 % row_seconds != 0 {
        bail!(
            "server_metrics.row_seconds must divide 86400 evenly and be at least {MINIMUM_SERVER_METRICS_ROW_SECONDS}"
        );
    }
    if row_seconds % sample_seconds != 0 || sample_seconds > row_seconds {
        bail!("server_metrics.sample_seconds must divide server_metrics.row_seconds evenly");
    }
    // Scylla's TTL is seconds in an i32, so a ttl_days that overflows it would be silently
    // truncated into a retention nobody asked for.
    let ttl_seconds = ttl_days
        .checked_mul(86_400)
        .filter(|seconds| *seconds <= i32::MAX as u64)
        .context("server_metrics.ttl_days is too large for a ScyllaDB TTL")?;

    let unit_name = |env_key: &str, toml_path: &str, default: &str| {
        optional_string(config, env_key, toml_path).unwrap_or_else(|| default.to_owned())
    };

    Ok(ServerMetricsConfig {
        enabled: optional_string(config, "SERVER_METRICS_ENABLED", "server_metrics.enabled")
            .is_none_or(|value| value == "1" || value.eq_ignore_ascii_case("true")),
        sample_interval: Duration::from_secs(sample_seconds),
        row_seconds: row_seconds as i64,
        row_ttl: Duration::from_secs(ttl_seconds),
        disk_mount: optional_string(
            config,
            "SERVER_METRICS_DISK_MOUNT",
            "server_metrics.disk_mount",
        )
        .unwrap_or_else(|| "/".to_owned()),
        network_interface: optional_string(
            config,
            "SERVER_METRICS_NETWORK_INTERFACE",
            "server_metrics.network_interface",
        )
        .unwrap_or_default(),
        backend_unit: unit_name(
            "SERVER_METRICS_BACKEND_UNIT",
            "server_metrics.backend_unit",
            DEFAULT_SERVER_METRICS_BACKEND_UNIT,
        ),
        server_utils_unit: unit_name(
            "SERVER_METRICS_SERVER_UTILS_UNIT",
            "server_metrics.server_utils_unit",
            DEFAULT_SERVER_METRICS_SERVER_UTILS_UNIT,
        ),
        search_unit: unit_name(
            "SERVER_METRICS_SEARCH_UNIT",
            "server_metrics.search_unit",
            DEFAULT_SERVER_METRICS_SEARCH_UNIT,
        ),
        scylla_unit: unit_name(
            "SERVER_METRICS_SCYLLA_UNIT",
            "server_metrics.scylla_unit",
            DEFAULT_SERVER_METRICS_SCYLLA_UNIT,
        ),
    })
}

fn load_scope_limits(config: &Table, env_scope: &str, toml_scope: &str) -> Result<ScopeLimits> {
    // Environment names stay flat while TOML values live in the grouped rate_limit section.
    Ok(ScopeLimits {
        cpu: CreditLimits {
            ten_seconds: rate_limit_value(
                config, env_scope, toml_scope, "CPU", "cpu", "10S", "10s",
            )?,
            hour: rate_limit_value(config, env_scope, toml_scope, "CPU", "cpu", "1H", "1h")?,
        },
        inference: CreditLimits {
            ten_seconds: rate_limit_value(
                config,
                env_scope,
                toml_scope,
                "INFERENCE",
                "inference",
                "10S",
                "10s",
            )?,
            hour: rate_limit_value(
                config,
                env_scope,
                toml_scope,
                "INFERENCE",
                "inference",
                "1H",
                "1h",
            )?,
        },
    })
}

fn rate_limit_value(
    config: &Table,
    env_scope: &str,
    toml_scope: &str,
    env_credit: &str,
    toml_credit: &str,
    env_window: &str,
    toml_window: &str,
) -> Result<u64> {
    required_u64(
        config,
        &format!("RATE_LIMIT_{env_scope}_{env_credit}_{env_window}"),
        &format!("rate_limit.{toml_scope}_{toml_credit}_{toml_window}"),
    )
}

fn validate_policy(policy: LimitPolicy) -> Result<()> {
    for (scope_name, scope) in [("company", policy.company), ("user", policy.user)] {
        for (credit_name, limits) in [("cpu", scope.cpu), ("inference", scope.inference)] {
            if limits.ten_seconds == 0 {
                bail!("rate_limit.{scope_name}_{credit_name} limits must be positive");
            }
            if limits.ten_seconds > limits.hour {
                bail!(
                    "rate_limit.{scope_name}_{credit_name} limits must be nondecreasing from 10s to 1h"
                );
            }
        }
    }
    Ok(())
}

fn load_config() -> Result<Table> {
    let explicit_path = env::var("GENIX_CONFIG_FILE")
        .ok()
        .filter(|value| !value.trim().is_empty())
        .map(PathBuf::from);
    let mut candidates = Vec::new();
    if let Some(path) = explicit_path {
        candidates.push(path);
    }
    candidates.push(PathBuf::from("../config.toml"));
    candidates.push(PathBuf::from("config.toml"));

    for path in candidates {
        if !path.is_file() {
            continue;
        }
        let text = fs::read_to_string(&path)
            .with_context(|| format!("failed to read config file {}", path.display()))?;
        let value: Value =
            toml::from_str(&text).with_context(|| format!("invalid TOML in {}", path.display()))?;
        return value
            .as_table()
            .cloned()
            .with_context(|| format!("{} must contain a TOML table", path.display()));
    }

    // An empty table still permits an environment-only deployment.
    Ok(Table::new())
}

fn value_text(config: &Table, env_key: &str, toml_path: &str) -> Option<String> {
    if let Ok(value) = env::var(env_key)
        && !value.trim().is_empty()
    {
        return Some(value.trim().to_owned());
    }
    match value_at_path(config, toml_path) {
        Some(Value::String(value)) if !value.trim().is_empty() => Some(value.trim().to_owned()),
        Some(Value::Integer(value)) => Some(value.to_string()),
        Some(Value::Float(value)) => Some(value.to_string()),
        Some(Value::Boolean(value)) => Some(value.to_string()),
        _ => None,
    }
}

fn value_at_path<'a>(config: &'a Table, path: &str) -> Option<&'a Value> {
    let mut parts = path.split('.');
    let mut value = config.get(parts.next()?)?;
    for part in parts {
        value = value.as_table()?.get(part)?;
    }
    Some(value)
}

fn optional_string(config: &Table, env_key: &str, toml_path: &str) -> Option<String> {
    value_text(config, env_key, toml_path)
}

/// Absent reads as false, and "1" is accepted next to "true" so an environment override can be
/// written either way.
fn optional_bool(config: &Table, env_key: &str, toml_path: &str) -> bool {
    value_text(config, env_key, toml_path)
        .is_some_and(|value| value == "1" || value.eq_ignore_ascii_case("true"))
}

fn required_string(config: &Table, env_key: &str, toml_path: &str) -> Result<String> {
    value_text(config, env_key, toml_path)
        .with_context(|| format!("missing required setting {toml_path} (environment: {env_key})"))
}

fn optional_u64(config: &Table, env_key: &str, toml_path: &str) -> Result<Option<u64>> {
    value_text(config, env_key, toml_path)
        .map(|value| {
            value
                .parse::<u64>()
                .with_context(|| format!("{toml_path}/{env_key} must be an unsigned integer"))
        })
        .transpose()
}

fn required_u64(config: &Table, env_key: &str, toml_path: &str) -> Result<u64> {
    let value = optional_u64(config, env_key, toml_path)?.with_context(|| {
        format!("missing required setting {toml_path} (environment: {env_key})")
    })?;
    if value == 0 {
        bail!("{toml_path}/{env_key} must be greater than zero");
    }
    Ok(value)
}

fn required_u16(config: &Table, env_key: &str, toml_path: &str) -> Result<u16> {
    let value = required_u64(config, env_key, toml_path)?;
    u16::try_from(value).with_context(|| format!("{toml_path}/{env_key} must fit in uint16"))
}

fn optional_usize(config: &Table, env_key: &str, toml_path: &str) -> Result<Option<usize>> {
    optional_u64(config, env_key, toml_path)?
        .map(|value| {
            usize::try_from(value).with_context(|| format!("{toml_path}/{env_key} is too large"))
        })
        .transpose()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nested_numeric_and_string_values_are_accepted() {
        // TOML integers and quoted values both support deployment templating.
        let config: Table = toml::from_str("[test]\na = 42\nb = '43'").unwrap();
        assert_eq!(
            optional_u64(&config, "IGNORED_A", "test.a").unwrap(),
            Some(42)
        );
        assert_eq!(
            optional_u64(&config, "IGNORED_B", "test.b").unwrap(),
            Some(43)
        );
    }

    #[test]
    fn a_toml_boolean_is_read_as_a_flag() {
        // `public = true` is a real TOML boolean, not a quoted one: value_text has to decode that
        // arm or the daemon silently keeps binding loopback and the off-box backend never connects.
        let config: Table = toml::from_str("[server_utils]\npublic = true").unwrap();
        assert!(optional_bool(&config, "IGNORED", "server_utils.public"));

        let private: Table = toml::from_str("[server_utils]\nport = 14013").unwrap();
        assert!(!optional_bool(&private, "IGNORED", "server_utils.public"));
    }

    /// The section is optional, and its defaults are what the nested deployment scripts under
    /// `scripts/configure/` actually produce.
    #[test]
    fn server_metrics_defaults_to_the_installed_unit_names() {
        let metrics = load_server_metrics(&Table::new()).unwrap();
        assert!(metrics.enabled);
        assert_eq!(metrics.row_seconds, 5);
        assert_eq!(metrics.sample_interval, Duration::from_secs(1));
        assert_eq!(metrics.backend_unit, "genix.service");
        assert_eq!(metrics.scylla_unit, "scylla-server.service");
        assert!(metrics.network_interface.is_empty());
    }

    /// row_seconds is the meaning of the clustering key, so the values that would corrupt it are
    /// refused at startup rather than discovered as a day with a strange last row.
    #[test]
    fn a_row_width_that_does_not_divide_the_day_is_rejected() {
        let uneven: Table = toml::from_str("[server_metrics]\nrow_seconds = 7").unwrap();
        assert!(load_server_metrics(&uneven).is_err());

        // Under three seconds the 17280-slot key stops fitting an int16.
        let too_fine: Table = toml::from_str("[server_metrics]\nrow_seconds = 2").unwrap();
        assert!(load_server_metrics(&too_fine).is_err());

        // A sub-sample longer than the window it feeds would leave rows with no samples at all.
        let mismatched: Table =
            toml::from_str("[server_metrics]\nrow_seconds = 5\nsample_seconds = 2").unwrap();
        assert!(load_server_metrics(&mismatched).is_err());

        let valid: Table =
            toml::from_str("[server_metrics]\nrow_seconds = 10\nsample_seconds = 5").unwrap();
        assert_eq!(load_server_metrics(&valid).unwrap().row_seconds, 10);
    }

    #[test]
    fn zero_credit_limits_are_rejected() {
        let positive = CreditLimits {
            ten_seconds: 1,
            hour: 1,
        };
        let policy = LimitPolicy {
            company: ScopeLimits {
                cpu: CreditLimits {
                    ten_seconds: 0,
                    ..positive
                },
                inference: positive,
            },
            user: ScopeLimits {
                cpu: positive,
                inference: positive,
            },
        };

        assert!(validate_policy(policy).is_err());
    }
}
