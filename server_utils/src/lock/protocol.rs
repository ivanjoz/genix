//! Payload codecs for opcodes `0x02` (acquire) and `0x03` (release).
//!
//! Transport concerns belong to `service`: this module decodes exactly the fifteen bytes that
//! describe one acquire, and release has no payload at all because the connection already
//! identifies the lock it holds.

use std::time::Duration;

use thiserror::Error;

pub const ACQUIRE_PAYLOAD_SIZE: usize = 15;
pub const RELEASE_PAYLOAD_SIZE: usize = 12;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct AcquireRequest {
    /// Namespace chosen by the Go caller. The daemon never interprets it; two features with
    /// different actions can never collide even on the same identifier.
    pub action: u16,
    /// Whatever the caller decided identifies the thing being serialized: an IP, a company, a
    /// client, a packed pair. Opaque here by design.
    pub identifier: i64,
    /// Queue ceiling. Zero means never queue, which turns the call into a try-lock.
    pub max_waiters: u8,
    pub wait: Duration,
    pub lease: Duration,
}

/// Which hold a release is ending.
///
/// The key alone is not enough once one connection can carry several locks and several callers:
/// a release sent by a caller that already gave up would otherwise end whichever hold replaced
/// it on the same key. The generation pins it to one specific grant.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ReleaseRequest {
    pub action: u16,
    pub identifier: i64,
    pub generation: u16,
}

/// The one-byte reply. Zero is success for every opcode on this port.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum LockReply {
    Ok = 0,
    /// `max_waiters` were already queued, so nothing was queued for this caller.
    Busy = 1,
    /// `wait` elapsed without reaching the front of the queue.
    WaitTimeout = 2,
    /// A process-wide ceiling was hit: too many live keys or too many waiters overall.
    Capacity = 3,
    /// Acquiring while already holding, or releasing while holding nothing.
    Misuse = 4,
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum LockProtocolError {
    #[error("lease_ms must be positive")]
    EmptyLease,
}

pub fn parse_acquire(
    payload: &[u8; ACQUIRE_PAYLOAD_SIZE],
) -> Result<AcquireRequest, LockProtocolError> {
    let action = u16::from_be_bytes([payload[0], payload[1]]);
    let identifier = i64::from_be_bytes([
        payload[2], payload[3], payload[4], payload[5], payload[6], payload[7], payload[8],
        payload[9],
    ]);
    let max_waiters = payload[10];
    let wait_ms = u16::from_be_bytes([payload[11], payload[12]]);
    let lease_ms = u16::from_be_bytes([payload[13], payload[14]]);

    // A zero lease would expire the hold the instant it was granted; a zero wait is legitimate
    // and means "try-lock".
    if lease_ms == 0 {
        return Err(LockProtocolError::EmptyLease);
    }
    Ok(AcquireRequest {
        action,
        identifier,
        max_waiters,
        wait: Duration::from_millis(u64::from(wait_ms)),
        lease: Duration::from_millis(u64::from(lease_ms)),
    })
}

pub fn parse_release(payload: &[u8; RELEASE_PAYLOAD_SIZE]) -> ReleaseRequest {
    ReleaseRequest {
        action: u16::from_be_bytes([payload[0], payload[1]]),
        identifier: i64::from_be_bytes([
            payload[2], payload[3], payload[4], payload[5], payload[6], payload[7], payload[8],
            payload[9],
        ]),
        generation: u16::from_be_bytes([payload[10], payload[11]]),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn release_parses_the_exact_wire_offsets() {
        let mut payload = [0_u8; RELEASE_PAYLOAD_SIZE];
        payload[0..2].copy_from_slice(&9_u16.to_be_bytes());
        payload[2..10].copy_from_slice(&(-7_i64).to_be_bytes());
        payload[10..12].copy_from_slice(&300_u16.to_be_bytes());

        let request = parse_release(&payload);
        assert_eq!(request.action, 9);
        assert_eq!(request.identifier, -7);
        assert_eq!(request.generation, 300);
    }

    #[test]
    fn parses_the_exact_wire_offsets() {
        let mut payload = [0_u8; ACQUIRE_PAYLOAD_SIZE];
        payload[0..2].copy_from_slice(&7_u16.to_be_bytes());
        payload[2..10].copy_from_slice(&(-42_i64).to_be_bytes());
        payload[10] = 3;
        payload[11..13].copy_from_slice(&5000_u16.to_be_bytes());
        payload[13..15].copy_from_slice(&15000_u16.to_be_bytes());

        let request = parse_acquire(&payload).unwrap();
        assert_eq!(request.action, 7);
        // Negative identifiers must survive the round trip: the field is opaque, so the Go side
        // is free to pack anything into it.
        assert_eq!(request.identifier, -42);
        assert_eq!(request.max_waiters, 3);
        assert_eq!(request.wait, Duration::from_millis(5000));
        assert_eq!(request.lease, Duration::from_millis(15000));
    }

    #[test]
    fn a_zero_lease_is_rejected() {
        let mut payload = [0_u8; ACQUIRE_PAYLOAD_SIZE];
        payload[13..15].copy_from_slice(&0_u16.to_be_bytes());
        assert_eq!(parse_acquire(&payload), Err(LockProtocolError::EmptyLease));
    }
}
