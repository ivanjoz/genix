//! Domain-separated HMAC authentication bound to a connection nonce and frame sequence.

use hmac::{Hmac, Mac};
use sha2::Sha256;
use subtle::ConstantTimeEq;
use thiserror::Error;

const DOMAIN: &[u8] = b"genix-rate-limiter:v1";

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
    fn matches_the_go_client_vectors() {
        let secret = b"test-secret";
        let nonce = [1, 2, 3, 4, 5, 6, 7, 8];
        let payload = [
            0x12, 0x34, 0x56, 0x00, 0x00, 0x2A, 0x04, 0x00, 0x07, 0x00, 0x09,
        ];
        assert_eq!(
            compute_hash(secret, &nonce, 0, &payload).unwrap(),
            [0x37, 0x79, 0x1B, 0xC2, 0x18, 0x3B, 0x8F, 0xE8]
        );
        assert_eq!(
            compute_hash(secret, &nonce, 1, &payload).unwrap(),
            [0x32, 0x8B, 0x6F, 0x7F, 0xCE, 0xD3, 0xC4, 0x07]
        );
    }
}
