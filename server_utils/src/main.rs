//! Genix server utilities: one process hosting two independent services.
//!
//!   - The raw-TCP credit rate limiter (`server`, `limiter`, `storage`).
//!   - The HTTP SSE bridge that relays agent events to browser tabs (`bridge`).
//!
//! They share only this process: the config load, the shutdown signal, and the tokio runtime.
//! Neither calls into the other.

use std::{sync::Arc, time::Instant};

use anyhow::{Context, Result};
use genix_server_utils::{
    bridge::{self, channel::ChannelRegistry, http::BridgeState},
    config::AppConfig,
    limiter::{quota::RateLimiter, server, storage::ScyllaUsageStore},
};
use tokio::{net::TcpListener, sync::watch, time::MissedTickBehavior};
use tracing::{error, info};
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .init();

    let config = AppConfig::load().context("server-utils configuration is invalid")?;
    let store = Arc::new(
        ScyllaUsageStore::connect(&config.database)
            .await
            .context("rate-limiter storage initialization failed")?,
    );
    let limiter = Arc::new(RateLimiter::new(config.shard_count, config.policy, store));
    let listener = TcpListener::bind(config.listen_address)
        .await
        .with_context(|| format!("failed to bind {}", config.listen_address))?;
    let bridge_listener = TcpListener::bind(config.bridge.listen_address)
        .await
        .with_context(|| format!("failed to bind {}", config.bridge.listen_address))?;
    // The rate limiter authenticates its frames with the service secret, not with the
    // token-signing one.
    let internal_apikey = Arc::new(config.internal_apikey.clone());
    let (shutdown_sender, shutdown_receiver) = watch::channel(false);

    let flush_limiter = limiter.clone();
    let flush_interval = config.flush_interval;
    let mut flush_shutdown = shutdown_receiver.clone();
    let flush_task = tokio::spawn(async move {
        let mut interval =
            tokio::time::interval_at(tokio::time::Instant::now() + flush_interval, flush_interval);
        interval.set_missed_tick_behavior(MissedTickBehavior::Skip);
        loop {
            tokio::select! {
                changed = flush_shutdown.changed() => {
                    if changed.is_err() || *flush_shutdown.borrow() {
                        break;
                    }
                }
                _ = interval.tick() => {
                    let started = Instant::now();
                    let written = flush_limiter.flush_dirty().await;
                    info!(written, elapsed_ms = started.elapsed().as_millis(), "periodic flush finished");
                }
            }
        }
    });

    let server_task = tokio::spawn(server::run(
        listener,
        limiter.clone(),
        internal_apikey.clone(),
        config.frame_timeout,
        config.max_connections,
        shutdown_receiver.clone(),
    ));

    let bridge_state = Arc::new(BridgeState {
        registry: ChannelRegistry::new(),
        secret_phrase: config.secret_phrase,
        internal_apikey: config.internal_apikey,
        verbose_logs: config.bridge.verbose_logs,
        started_at: Instant::now(),
    });
    let mut bridge_shutdown = shutdown_receiver;
    let bridge_task = tokio::spawn(async move {
        info!(
            address = %config.bridge.listen_address,
            verbose = config.bridge.verbose_logs,
            "sse bridge listening"
        );
        // No request timeouts on this listener: every deadline here would kill a healthy
        // long-lived SSE stream. Dead connections surface as the keepalive write failing.
        axum::serve(bridge_listener, bridge::http::routes(bridge_state))
            .with_graceful_shutdown(async move {
                while !*bridge_shutdown.borrow_and_update() {
                    if bridge_shutdown.changed().await.is_err() {
                        break;
                    }
                }
            })
            .await
    });

    tokio::signal::ctrl_c().await.context("failed to listen for shutdown signal")?;
    info!("shutdown signal received");
    let _ = shutdown_sender.send(true);

    if let Err(join_error) = flush_task.await {
        error!(error = %join_error, "flush task failed");
    }
    if let Err(bridge_error) = bridge_task.await.context("bridge task failed")? {
        error!(error = %bridge_error, "sse bridge stopped with an error");
    }
    server_task.await.context("server task failed")??;
    let written = limiter.flush_dirty().await;
    info!(written, "final usage flush finished");
    Ok(())
}
