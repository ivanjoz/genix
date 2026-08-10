//! Domain-separated HMAC authentication bound to a connection nonce and frame sequence.
//!
//! Transport-level, not service-level: every opcode on the shared port authenticates the same
//! way, and the opcode byte is inside the signed bytes so a frame cannot be replayed as a
//! different operation.

use hmac::{Hmac, Mac};
use sha2::Sha256;
use subtle::ConstantTimeEq;
use thiserror::Error;

/// Names the whole shared-port framing, request *and* reply, not one service.
///
/// Bumped on every wire change so a mismatched peer fails loudly at the first frame instead of
/// misreading bytes. `genix-rate-limiter:v1` was the opcode-less 19-byte frame; `:v1` here added
/// the opcode byte; `:v2` widened the reply to 5 bytes; `:v3` gave `LOCK_RELEASE` a payload.
/// Replies are not themselves authenticated, so without the bump an old client would keep
/// authenticating fine, read 1 byte of a 5-byte reply, and silently misinterpret everything
/// after that.
const DOMAIN: &[u8] = b"genix-server-utils:v3";

type HmacSha256 = Hmac<Sha256>;

#[derive(Debug, Error)]
pub enum AuthError {
    #[error("operating system random source failed: {0}")]
    Random(#[from] getrandom::Error),
    #[error("authentication key is invalid")]
    InvalidKey,
}

pub fn new_nonce() -> Result<[u8; 8], AuthError> {
    let mut nonce = [0_u8; 8];
    getrandom::fill(&mut nonce)?;
    Ok(nonce)
}

pub fn compute_hash(
    secret: &[u8],
    nonce: &[u8; 8],
    sequence: u64,
    authenticated_payload: &[u8],
) -> Result<[u8; 8], AuthError> {
    let mut mac = HmacSha256::new_from_slice(secret).map_err(|_| AuthError::InvalidKey)?;
    mac.update(DOMAIN);
    mac.update(nonce);
    mac.update(&sequence.to_be_bytes());
    mac.update(authenticated_payload);
    let digest = mac.finalize().into_bytes();
    let mut truncated = [0_u8; 8];
    truncated.copy_from_slice(&digest[..8]);
    Ok(truncated)
}

pub fn verify_hash(
    secret: &[u8],
    nonce: &[u8; 8],
    sequence: u64,
    authenticated_payload: &[u8],
    received: &[u8; 8],
) -> Result<bool, AuthError> {
    // Constant-time comparison avoids revealing a valid truncated tag byte by byte.
    let expected = compute_hash(secret, nonce, sequence, authenticated_payload)?;
    Ok(bool::from(expected.ct_eq(received)))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn nonce_and_sequence_change_the_hash() {
        let secret = b"test secret";
        let payload = b"12345678901";
        let first = compute_hash(secret, &[1; 8], 0, payload).unwrap();
        assert_ne!(first, compute_hash(secret, &[2; 8], 0, payload).unwrap());
        assert_ne!(first, compute_hash(secret, &[1; 8], 1, payload).unwrap());
        assert!(verify_hash(secret, &[1; 8], 0, payload, &first).unwrap());
    }

    #[test]
    fn the_opcode_is_part_of_the_signature() {
        // Same charge bytes under two opcodes must not share a tag, or a frame could be replayed
        // into a different operation.
        let secret = b"test-secret";
        let charge = [0x12, 0x34, 0x56, 0x00, 0x00, 0x2A, 0x04, 0x00, 0x07, 0x00, 0x09];
        let mut as_charge = vec![0x01_u8];
        as_charge.extend_from_slice(&charge);
        let mut as_other = vec![0x02_u8];
        as_other.extend_from_slice(&charge);
        assert_ne!(
            compute_hash(secret, &[1; 8], 0, &as_charge).unwrap(),
            compute_hash(secret, &[1; 8], 0, &as_other).unwrap()
        );
    }

    #[test]
    fn matches_the_go_client_vectors() {
        let secret = b"test-secret";
        let nonce = [1, 2, 3, 4, 5, 6, 7, 8];
        // Opcode 0x01 followed by the 11-byte charge payload.
        let payload = [
            0x01, 0x12, 0x34, 0x56, 0x00, 0x00, 0x2A, 0x04, 0x00, 0x07, 0x00, 0x09,
        ];
        assert_eq!(
            compute_hash(secret, &nonce, 0, &payload).unwrap(),
            [0x62, 0x8C, 0xB7, 0x8A, 0x58, 0xDD, 0xE2, 0x6D]
        );
        assert_eq!(
            compute_hash(secret, &nonce, 1, &payload).unwrap(),
            [0x84, 0x25, 0x06, 0xDC, 0x3B, 0xB8, 0xDF, 0x6F]
        );

        // Opcode 0x02 with action 7, identifier -42, 3 waiters, 5000 ms wait, 15000 ms lease.
        let acquire = [
            0x02, 0x00, 0x07, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xD6, 0x03, 0x13, 0x88,
            0x3A, 0x98,
        ];
        assert_eq!(
            compute_hash(secret, &nonce, 0, &acquire).unwrap(),
            [0x89, 0x9E, 0x18, 0x73, 0xDC, 0xDF, 0x40, 0x29]
        );
    }
}
