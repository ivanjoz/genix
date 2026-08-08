use std::{sync::Arc, time::Instant};

use anyhow::{Context, Result};
use genix_server_utils::{
    config::AppConfig, limiter::RateLimiter, server, storage::ScyllaUsageStore,
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

    let config = AppConfig::load().context("rate-limiter configuration is invalid")?;
    let store = Arc::new(
        ScyllaUsageStore::connect(&config.database)
            .await
            .context("rate-limiter storage initialization failed")?,
    );
    let limiter = Arc::new(RateLimiter::new(config.shard_count, config.policy, store));
    let listener = TcpListener::bind(config.listen_address)
        .await
        .with_context(|| format!("failed to bind {}", config.listen_address))?;
    let secret = Arc::new(config.secret_phrase);
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

    let server_limiter = limiter.clone();
    let server_secret = secret.clone();
    let server_task = tokio::spawn(server::run(
        listener,
        server_limiter,
        server_secret,
        config.frame_timeout,
        config.max_connections,
        shutdown_receiver,
    ));

    tokio::signal::ctrl_c()
        .await
        .context("failed to listen for shutdown signal")?;
    info!("shutdown signal received");
    let _ = shutdown_sender.send(true);

    if let Err(join_error) = flush_task.await {
        error!(error = %join_error, "flush task failed");
    }
    server_task.await.context("server task failed")??;
    let written = limiter.flush_dirty().await;
    info!(written, "final usage flush finished");
    Ok(())
}
