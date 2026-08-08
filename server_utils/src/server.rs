//! Raw TCP listener, connection handshake, fixed-frame loop, and graceful stop coordination.

use std::{io::ErrorKind, net::SocketAddr, sync::Arc, time::Duration};

use anyhow::{Context, Result, anyhow};
use tokio::{
    io::{AsyncReadExt, AsyncWriteExt},
    net::{TcpListener, TcpStream},
    sync::{Semaphore, watch},
    task::JoinSet,
    time::timeout,
};
use tracing::{debug, info, warn};

use crate::{
    auth,
    limiter::RateLimiter,
    protocol::{AUTHENTICATED_PAYLOAD_SIZE, REQUEST_SIZE, parse_frame},
};

pub async fn run(
    listener: TcpListener,
    limiter: Arc<RateLimiter>,
    secret: Arc<Vec<u8>>,
    frame_timeout: Duration,
    max_connections: usize,
    mut shutdown: watch::Receiver<bool>,
) -> Result<()> {
    let connection_slots = Arc::new(Semaphore::new(max_connections));
    let mut connections = JoinSet::new();
    info!(address = %listener.local_addr()?, max_connections, "rate limiter listening");

    loop {
        tokio::select! {
            changed = shutdown.changed() => {
                if changed.is_err() || *shutdown.borrow() {
                    break;
                }
            }
            accepted = listener.accept() => {
                let (socket, peer) = accepted.context("TCP accept failed")?;
                let Ok(slot) = connection_slots.clone().try_acquire_owned() else {
                    warn!(%peer, "rejecting connection because the connection limit is full");
                    drop(socket);
                    continue;
                };
                socket.set_nodelay(true).context("failed to set TCP_NODELAY")?;
                let limiter = limiter.clone();
                let secret = secret.clone();
                let connection_shutdown = shutdown.clone();
                connections.spawn(async move {
                    let _slot = slot;
                    if let Err(connection_error) = handle_connection(
                        socket,
                        peer,
                        limiter,
                        &secret,
                        frame_timeout,
                        connection_shutdown,
                    ).await {
                        debug!(%peer, error = %connection_error, "connection closed with error");
                    }
                });
            }
            completed = connections.join_next(), if !connections.is_empty() => {
                if let Some(Err(join_error)) = completed {
                    warn!(error = %join_error, "connection task failed");
                }
            }
        }
    }

    // Connection tasks observe the same shutdown channel and drop their sockets promptly.
    while let Some(result) = connections.join_next().await {
        if let Err(join_error) = result {
            warn!(error = %join_error, "connection task failed during shutdown");
        }
    }
    Ok(())
}

async fn handle_connection(
    mut socket: TcpStream,
    peer: SocketAddr,
    limiter: Arc<RateLimiter>,
    secret: &[u8],
    frame_timeout: Duration,
    mut shutdown: watch::Receiver<bool>,
) -> Result<()> {
    let nonce = auth::new_nonce()?;
    socket
        .write_all(&nonce)
        .await
        .context("failed to write connection nonce")?;
    let mut sequence = 0_u64;
    debug!(%peer, "connection nonce sent");

    loop {
        let mut frame = [0_u8; REQUEST_SIZE];
        let read_result = tokio::select! {
            changed = shutdown.changed() => {
                if changed.is_err() || *shutdown.borrow() {
                    return Ok(());
                }
                continue;
            }
            result = timeout(frame_timeout, socket.read_exact(&mut frame)) => result,
        };

        match read_result {
            Err(_) => return Err(anyhow!("frame read timed out")),
            Ok(Err(error)) if error.kind() == ErrorKind::UnexpectedEof => return Ok(()),
            Ok(Err(error)) => return Err(error).context("frame read failed"),
            Ok(Ok(_)) => {}
        }

        let mut received_hash = [0_u8; 8];
        received_hash.copy_from_slice(&frame[AUTHENTICATED_PAYLOAD_SIZE..REQUEST_SIZE]);
        if !auth::verify_hash(
            secret,
            &nonce,
            sequence,
            &frame[..AUTHENTICATED_PAYLOAD_SIZE],
            &received_hash,
        )? {
            warn!(%peer, sequence, "frame authentication failed");
            return Ok(());
        }

        let parsed = parse_frame(&frame).context("authenticated frame is invalid")?;
        let decision = limiter.admit(parsed.request).await?;
        let response = decision.map_or(0, |violation| violation.response_byte());
        socket
            .write_all(&[response])
            .await
            .context("failed to write decision")?;
        debug!(
            %peer,
            sequence,
            company_id = parsed.request.company_id,
            user_id = parsed.request.user_id,
            api_group = parsed.request.api_group,
            cpu = parsed.request.credits.cpu,
            inference = parsed.request.credits.inference,
            response,
            "processed credit charge"
        );
        sequence = sequence
            .checked_add(1)
            .ok_or_else(|| anyhow!("frame sequence overflow"))?;
    }
}
