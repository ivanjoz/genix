//! Loads rate-limiter and Scylla settings with environment-over-TOML precedence.

use std::{env, fs, net::SocketAddr, path::PathBuf, time::Duration};

use anyhow::{Context, Result, bail};
use toml::{Table, Value};

use crate::limiter::{CreditLimits, LimitPolicy, ScopeLimits};

const DEFAULT_LISTEN_ADDRESS: &str = "127.0.0.1:14013";
const DEFAULT_FLUSH_SECONDS: u64 = 15;
const DEFAULT_FRAME_TIMEOUT_SECONDS: u64 = 30;
const DEFAULT_MAX_CONNECTIONS: usize = 1_024;

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
    pub shard_count: usize,
    pub secret_phrase: Vec<u8>,
    pub database: DatabaseConfig,
    pub policy: LimitPolicy,
}

impl AppConfig {
    pub fn load() -> Result<Self> {
        let config = load_config()?;
        let listen_address = optional_string(&config, "RATE_LIMIT_ADDRESS", "rate_limit.address")
            .unwrap_or_else(|| DEFAULT_LISTEN_ADDRESS.to_owned())
            .parse()
            .context("rate_limit.address must be a socket address such as 127.0.0.1:14013")?;
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

        if flush_seconds == 0 || frame_timeout_seconds == 0 || max_connections == 0 {
            bail!("rate-limiter durations and connection limit must be positive");
        }

        let policy = LimitPolicy {
            company: load_scope_limits(&config, "COMPANY", "company")?,
            user: load_scope_limits(&config, "USER", "user")?,
        };
        validate_policy(policy)?;

        Ok(Self {
            listen_address,
            flush_interval: Duration::from_secs(flush_seconds),
            frame_timeout: Duration::from_secs(frame_timeout_seconds),
            max_connections,
            shard_count,
            secret_phrase: required_string(&config, "SECRET_PHRASE", "secret_phrase")?.into_bytes(),
            database: DatabaseConfig {
                host: required_string(&config, "DB_HOST", "db.host")?,
                port: required_u16(&config, "DB_PORT", "db.port")?,
                keyspace: required_string(&config, "DB_NAME", "db.name")?,
                user: required_string(&config, "DB_USER", "db.user")?,
                password: required_string(&config, "DB_PASSWORD", "db.password")?,
            },
            policy,
        })
    }
}

fn load_scope_limits(config: &Table, env_scope: &str, toml_scope: &str) -> Result<ScopeLimits> {
    // Environment names stay flat while TOML values live in the grouped rate_limit section.
    Ok(ScopeLimits {
        cpu: CreditLimits {
            ten_seconds: rate_limit_value(
                config, env_scope, toml_scope, "CPU", "cpu", "10S", "10s",
            )?,
            hour: rate_limit_value(config, env_scope, toml_scope, "CPU", "cpu", "1H", "1h")?,
            day: rate_limit_value(config, env_scope, toml_scope, "CPU", "cpu", "24H", "24h")?,
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
            day: rate_limit_value(
                config,
                env_scope,
                toml_scope,
                "INFERENCE",
                "inference",
                "24H",
                "24h",
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
            if limits.ten_seconds > limits.hour || limits.hour > limits.day {
                bail!(
                    "rate_limit.{scope_name}_{credit_name} limits must be nondecreasing from 10s to 1h to 24h"
                );
            }
            if limits.day > u64::from(u32::MAX) {
                bail!(
                    "rate_limit.{scope_name}_{credit_name}_24h exceeds the persisted uint32 format"
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
    fn zero_credit_limits_are_rejected() {
        let positive = CreditLimits {
            ten_seconds: 1,
            hour: 1,
            day: 1,
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
