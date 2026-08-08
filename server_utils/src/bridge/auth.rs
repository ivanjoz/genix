//! The bridge's two authentication schemes, each keyed by a different config secret.
//!
//!   - The **browser** presents the session token the backend already issued
//!     (`Authorization: Bearer <token>`). It is self-contained (colbin payload + HMAC), so
//!     identity is verified without touching ScyllaDB. Keyed by `secret_phrase`, because
//!     that is what signed it.
//!   - The **backend** presents a timestamped HMAC header (`X-Bridge-Auth`). Keyed by
//!     `internal_apikey`, the project's service-to-service secret.
//!
//! The bridge only establishes *identity*. Permissions stay in the backend, which already
//! evaluated them when it accepted the turn.

use hmac::{Hmac, Mac};
use sha2::Sha256;
use subtle::ConstantTimeEq;
use thiserror::Error;

use crate::bridge::token::{TokenError, UserToken, decode_session_base64, decode_session_token};

pub const SERVICE_AUTH_HEADER: &str = "X-Bridge-Auth";

/// Domain separation: keeps a service signature from ever validating against another HMAC
/// the project computes with the same key.
const SERVICE_AUTH_PREFIX: &str = "sse-bridge:v1|";
/// Tolerates clock drift between the Lambda and this host while keeping a captured header
/// from being replayable forever.
const SERVICE_AUTH_MAX_SKEW_SECONDS: i64 = 300;
/// Domain separation for the session token, mirroring `core.ComputeUsuarioTokenHash`.
const SESSION_TOKEN_DOMAIN: &[u8] = b"usrToken:v1";

type HmacSha256 = Hmac<Sha256>;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum BridgeAuthError {
    #[error("session token is missing")]
    MissingSessionToken,
    #[error("session token is malformed: {0}")]
    MalformedSessionToken(#[from] TokenError),
    #[error("session token does not identify a user")]
    NoIdentity,
    #[error("session token signature is invalid")]
    InvalidSessionSignature,
    #[error("the {SERVICE_AUTH_HEADER} header is missing")]
    MissingServiceAuth,
    #[error("service authentication header is malformed")]
    MalformedServiceAuth,
    #[error("service authentication expired ({0}s of skew)")]
    ExpiredServiceAuth(i64),
    #[error("service authentication signature is invalid")]
    InvalidServiceSignature,
}

/// Recomputes the session token's own HMAC. Exact mirror of
/// `core.ComputeUsuarioTokenHash`: any change on the backend must be replicated here, since
/// a mismatch rejects every client.
fn compute_user_token_hash(user_token: &UserToken, secret_phrase: &[u8]) -> u64 {
    let mut mac = HmacSha256::new_from_slice(secret_phrase).expect("HMAC accepts any key length");
    let mut identity_bytes = [0_u8; 12];
    identity_bytes[0..4].copy_from_slice(&(user_token.company_id as u32).to_be_bytes());
    identity_bytes[4..8].copy_from_slice(&(user_token.id as u32).to_be_bytes());
    identity_bytes[8..12].copy_from_slice(&(user_token.created as u32).to_be_bytes());
    mac.update(SESSION_TOKEN_DOMAIN);
    mac.update(&identity_bytes);
    mac.update(user_token.user.as_bytes());

    let digest = mac.finalize().into_bytes();
    u64::from_be_bytes(digest[..8].try_into().expect("SHA-256 digests are 32 bytes"))
}

/// Verifies the `Authorization: Bearer <token>` header and returns the identity it proves.
pub fn authenticate_user(
    authorization_header: Option<&str>,
    secret_phrase: &[u8],
) -> Result<UserToken, BridgeAuthError> {
    let header_value = authorization_header.unwrap_or_default().trim();
    if header_value.len() < 8 {
        return Err(BridgeAuthError::MissingSessionToken);
    }

    let encoded_token = header_value.strip_prefix("Bearer ").unwrap_or(header_value).trim();
    let user_token = decode_session_token(&decode_session_base64(encoded_token)?)?;
    if user_token.company_id <= 0 || user_token.id <= 0 {
        return Err(BridgeAuthError::NoIdentity);
    }

    // Constant-time comparison: a byte-by-byte early exit would leak the expected hash to a
    // caller able to time many attempts.
    let expected_hash = compute_user_token_hash(&user_token, secret_phrase);
    if !bool::from(expected_hash.to_be_bytes().ct_eq(&user_token.hash.to_be_bytes())) {
        return Err(BridgeAuthError::InvalidSessionSignature);
    }
    Ok(user_token)
}

/// Builds the value the backend sends on `X-Bridge-Auth`. Mirrored in
/// `backend/agent/bridge.go`, which lives in another module and cannot import this one.
pub fn make_service_auth_header(internal_apikey: &[u8], unix_seconds: i64) -> String {
    let mut mac = HmacSha256::new_from_slice(internal_apikey).expect("HMAC accepts any key length");
    mac.update(format!("{SERVICE_AUTH_PREFIX}{unix_seconds}").as_bytes());
    let signature = mac.finalize().into_bytes();

    let mut header = format!("{unix_seconds}.");
    for byte in signature.iter() {
        header.push_str(&format!("{byte:02x}"));
    }
    header
}

/// Validates the backend's signature and its freshness.
pub fn verify_service_auth(
    header_value: Option<&str>,
    internal_apikey: &[u8],
    now_unix_seconds: i64,
) -> Result<(), BridgeAuthError> {
    let header_value = header_value.unwrap_or_default().trim();
    if header_value.is_empty() {
        return Err(BridgeAuthError::MissingServiceAuth);
    }

    let (timestamp_text, _) =
        header_value.split_once('.').ok_or(BridgeAuthError::MalformedServiceAuth)?;
    let signed_unix_seconds: i64 =
        timestamp_text.parse().map_err(|_| BridgeAuthError::MalformedServiceAuth)?;

    let elapsed_seconds = (now_unix_seconds - signed_unix_seconds).abs();
    if elapsed_seconds > SERVICE_AUTH_MAX_SKEW_SECONDS {
        return Err(BridgeAuthError::ExpiredServiceAuth(elapsed_seconds));
    }

    let expected_header = make_service_auth_header(internal_apikey, signed_unix_seconds);
    if !bool::from(header_value.as_bytes().ct_eq(expected_header.as_bytes())) {
        return Err(BridgeAuthError::InvalidServiceSignature);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::{Engine, engine::general_purpose};

    /// The secret the Go-generated vectors below were produced with.
    const TEST_SECRET: &[u8] = b"K1OzWIN0yarCc9ge";

    /// Vector 1 of the colbin set: company 7, user 42, created 1234, user "tester", whose
    /// Hash field was computed by Go's `core.ComputeUsuarioTokenHash` with TEST_SECRET. If
    /// this passes, the Rust and Go session-token HMACs agree byte for byte.
    const GO_SESSION_TOKEN_HEX: &str = "010105ca000600000001350029000000019f00d1040000011a68ed671d\
                                        93b93189b000000000000000006a02000500000001746573746572";

    fn go_session_token_base64() -> String {
        general_purpose::STANDARD.encode(decode_hex(GO_SESSION_TOKEN_HEX))
    }

    #[test]
    fn accepts_the_go_issued_session_token() {
        let authenticated =
            authenticate_user(Some(&format!("Bearer {}", go_session_token_base64())), TEST_SECRET)
                .unwrap();
        assert_eq!(authenticated.company_id, 7);
        assert_eq!(authenticated.id, 42);
        assert_eq!(authenticated.user, "tester");
    }

    #[test]
    fn rejects_a_session_token_signed_with_another_secret() {
        // This is what stops a client from minting its own identity.
        assert_eq!(
            authenticate_user(
                Some(&format!("Bearer {}", go_session_token_base64())),
                b"otro-secreto"
            ),
            Err(BridgeAuthError::InvalidSessionSignature)
        );
    }

    #[test]
    fn rejects_a_missing_session_token() {
        assert_eq!(authenticate_user(None, TEST_SECRET), Err(BridgeAuthError::MissingSessionToken));
        assert_eq!(
            authenticate_user(Some("Bearer "), TEST_SECRET),
            Err(BridgeAuthError::MissingSessionToken)
        );
    }

    /// Pins the service-auth header against the Go implementation. Produced by
    /// Go's `MakeServiceAuthHeader("K1OzWIN0yarCc9ge", 1700000000)`, which the backend mirrors
    /// in `backend/agent/bridge.go`.
    #[test]
    fn matches_the_go_service_auth_header() {
        assert_eq!(
            make_service_auth_header(TEST_SECRET, 1_700_000_000),
            "1700000000.d91e72e6afca0954d2f0b3c4f6b6603ffb146cb022dd7fbd9205d4d01099250b"
        );
    }

    #[test]
    fn accepts_a_fresh_service_signature_and_rejects_the_rest() {
        let now = 1_700_000_000_i64;
        let header = make_service_auth_header(TEST_SECRET, now);
        assert_eq!(verify_service_auth(Some(&header), TEST_SECRET, now), Ok(()));
        // Inside the skew window in both directions.
        assert_eq!(verify_service_auth(Some(&header), TEST_SECRET, now + 299), Ok(()));
        assert_eq!(verify_service_auth(Some(&header), TEST_SECRET, now - 299), Ok(()));

        assert_eq!(
            verify_service_auth(Some(&header), TEST_SECRET, now + 400),
            Err(BridgeAuthError::ExpiredServiceAuth(400))
        );
        let tampered = format!("{}ff", &header[..header.len() - 2]);
        assert_eq!(
            verify_service_auth(Some(&tampered), TEST_SECRET, now),
            Err(BridgeAuthError::InvalidServiceSignature)
        );
        assert_eq!(
            verify_service_auth(Some(&header), b"otra-clave", now),
            Err(BridgeAuthError::InvalidServiceSignature)
        );
        assert_eq!(
            verify_service_auth(Some("no-dot"), TEST_SECRET, now),
            Err(BridgeAuthError::MalformedServiceAuth)
        );
        assert_eq!(
            verify_service_auth(None, TEST_SECRET, now),
            Err(BridgeAuthError::MissingServiceAuth)
        );
    }

    fn decode_hex(text: &str) -> Vec<u8> {
        let compact: String = text.chars().filter(|character| !character.is_whitespace()).collect();
        (0..compact.len())
            .step_by(2)
            .map(|index| u8::from_str_radix(&compact[index..index + 2], 16).unwrap())
            .collect()
    }
}
