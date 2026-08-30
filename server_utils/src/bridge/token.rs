//! Two independent token codecs the bridge needs at its HTTP boundary.
//!
//! 1. The browser's session token: a colbin message holding the backend's
//!    `core.UsuarioToken`. The format itself lives in the `colbin` crate, which is the
//!    repository that defines it; what belongs here is the shape of *this* struct.
//! 2. The channel token, a small custom varint format naming one browser tab. It is not
//!    colbin and shares nothing with it beyond sitting at the same boundary.
//!
//! Both are mirrors of Go code in another repository, so every rule here is pinned by
//! vectors generated from that Go code — see `server_utils/vectors`, which prints them.

use std::sync::OnceLock;

use base64::{Engine, engine::general_purpose};
use colbin::{Kind, Schema};
use thiserror::Error;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum TokenError {
    #[error("session token is not valid base64")]
    SessionBase64,
    #[error("session token is not a valid colbin message: {0}")]
    Session(#[from] colbin::Error),
    #[error("channel token is not valid unpadded base64url")]
    ChannelBase64,
    #[error("channel token does not contain a company id")]
    ChannelCompanyID,
    #[error("channel token does not contain a user id")]
    ChannelUserID,
    #[error("channel token does not contain a 6-byte tab id")]
    ChannelTabID,
    #[error("channel token contains out-of-range identifiers")]
    ChannelRange,
    #[error("channel token is not canonically encoded")]
    ChannelNotCanonical,
}

/// Session identity proven by the token. Mirrors `core.UsuarioToken`; the transient `Error`
/// field carries `cb:"-"` in Go and is never on the wire.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct UserToken {
    pub company_id: i32,
    pub id: i32,
    pub created: i32,
    pub hash: u64,
    pub user: String,
}

/// The token's fields, in Go declaration order and with the Go widths.
///
/// Both matter. colbin derives each wire id by hashing the field name and linear-probing
/// past the ids already taken, so the order decides which field wins a collision; and the
/// width is what says where a value ends, since it never reaches the wire.
const SESSION_FIELDS: [(&str, Kind); 5] = [
    ("CompanyID", Kind::Int32),
    ("ID", Kind::Int32),
    ("Created", Kind::Int32),
    ("Hash", Kind::Uint64),
    ("User", Kind::String),
];

/// The schema and the ids it produced, resolved once. The ids are what the message names
/// its fields by, so they are read out of the schema rather than restated here.
struct SessionSchema {
    schema: Schema,
    ids: [u8; SESSION_FIELDS.len()],
}

fn session_schema() -> &'static SessionSchema {
    static SCHEMA: OnceLock<SessionSchema> = OnceLock::new();
    SCHEMA.get_or_init(|| {
        let schema =
            Schema::from_go(&SESSION_FIELDS).expect("the session layout is a valid schema");
        let ids = schema
            .ids()
            .try_into()
            .expect("the schema has one id per declared field");
        SessionSchema { schema, ids }
    })
}

/// Decodes the colbin payload of a session token.
///
/// A single struct is one record, which colbin routes through compact mode: the fields
/// arrive as a run of `[key][value]` pairs, and a field holding its zero value is not
/// there at all. That is why every field is read through an accessor that answers with the
/// zero value rather than an option — it is the same answer the Go decoder writes into the
/// destination struct.
pub fn decode_session_token(payload: &[u8]) -> Result<UserToken, TokenError> {
    let session = session_schema();
    let [company_id, id, created, hash, user] = session.ids;
    let record = colbin::decode_one(payload, &session.schema)?;
    Ok(UserToken {
        company_id: record.i64(company_id) as i32,
        id: record.i64(id) as i32,
        created: record.i64(created) as i32,
        hash: record.u64(hash),
        user: record.str(user).to_owned(),
    })
}

/// Undoes the backend's `MakeB64UrlEncode` alphabet substitution before standard base64
/// decoding (`core/helpers.go`).
pub fn decode_session_base64(encoded_token: &str) -> Result<Vec<u8>, TokenError> {
    let standard_alphabet: String = encoded_token
        .chars()
        .map(|character| match character {
            '_' => '/',
            '-' => '+',
            '~' => '=',
            other => other,
        })
        .collect();
    general_purpose::STANDARD
        .decode(standard_alphabet)
        .map_err(|_| TokenError::SessionBase64)
}

// --- Channel token (mirrored in backend/agent/channel.go, frontend/core/agent/channel.ts) ---

/// The tab's entropy: 6 bytes = 48 bits, exactly 8 base64url characters.
const TAB_RANDOM_BYTES: usize = 6;

/// Decodes a channel token into its company, user and tab parts.
///
/// Non-canonical encodings are rejected (an overlong varint names the same numbers with
/// different bytes). That rejection is what makes the token a bijection with the triple,
/// which is what lets it be used directly as the channel registry key: two distinct strings
/// can never name the same channel.
pub fn decode_channel_token(channel_token: &str) -> Result<(i32, i32, String), TokenError> {
    let token_bytes = general_purpose::URL_SAFE_NO_PAD
        .decode(channel_token)
        .map_err(|_| TokenError::ChannelBase64)?;

    let (company_value, company_byte_count) =
        read_uvarint(&token_bytes).ok_or(TokenError::ChannelCompanyID)?;
    let (user_value, user_byte_count) =
        read_uvarint(&token_bytes[company_byte_count..]).ok_or(TokenError::ChannelUserID)?;

    let tab_bytes = &token_bytes[company_byte_count + user_byte_count..];
    if tab_bytes.len() != TAB_RANDOM_BYTES {
        return Err(TokenError::ChannelTabID);
    }
    if company_value == 0
        || user_value == 0
        || company_value > i32::MAX as u64
        || user_value > i32::MAX as u64
    {
        return Err(TokenError::ChannelRange);
    }

    let company_id = company_value as i32;
    let user_id = user_value as i32;
    let tab_id = general_purpose::URL_SAFE_NO_PAD.encode(tab_bytes);

    // Canonicality by round-trip: cheaper than validating each varint by hand, and it
    // cannot miss a case.
    if encode_channel_token(company_id, user_id, &tab_id).as_deref() != Some(channel_token) {
        return Err(TokenError::ChannelNotCanonical);
    }
    Ok((company_id, user_id, tab_id))
}

/// Builds the token naming one tab. `tab_id` is the 8-character base64url form of the tab's
/// 6 random bytes; anything else yields `None`.
pub fn encode_channel_token(company_id: i32, user_id: i32, tab_id: &str) -> Option<String> {
    let tab_bytes = general_purpose::URL_SAFE_NO_PAD.decode(tab_id).ok()?;
    if tab_bytes.len() != TAB_RANDOM_BYTES || company_id <= 0 || user_id <= 0 {
        return None;
    }
    let mut token_bytes = Vec::with_capacity(8 + TAB_RANDOM_BYTES);
    append_uvarint(&mut token_bytes, company_id as u64);
    append_uvarint(&mut token_bytes, user_id as u64);
    token_bytes.extend_from_slice(&tab_bytes);
    Some(general_purpose::URL_SAFE_NO_PAD.encode(&token_bytes))
}

/// Reads one LEB128 varint, returning the value and the bytes it consumed.
fn read_uvarint(bytes: &[u8]) -> Option<(u64, usize)> {
    let mut value = 0_u64;
    let mut shift = 0_u32;
    for (index, byte) in bytes.iter().enumerate() {
        if shift > 63 {
            return None;
        }
        value |= u64::from(byte & 0x7F) << shift;
        if byte & 0x80 == 0 {
            return Some((value, index + 1));
        }
        shift += 7;
    }
    None
}

fn append_uvarint(output: &mut Vec<u8>, mut value: u64) {
    while value >= 0x80 {
        output.push(value as u8 | 0x80);
        value >>= 7;
    }
    output.push(value as u8);
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Field ids assigned by the Go colbin builder for this struct's field names. A change
    /// here means every session token silently decodes to zero values, since a message
    /// names its fields by id and an unknown one is the only thing that would be reported.
    #[test]
    fn field_ids_match_the_go_colbin_layout() {
        assert_eq!(session_schema().ids, [202, 53, 159, 26, 106]);
    }

    /// Tokens produced by `colbin.Marshal` on the real Go struct, printed by
    /// `go run ./server_utils/vectors`. Each covers a different shape: the plain case, an
    /// omitted field (`Created` is zero) with an empty string, the i32 maximum, multi-byte
    /// UTF-8, a long user name, and a negative value, which is what clears the message's
    /// ALL_POSITIVE flag and puts the integers on the zigzag path.
    #[test]
    fn decodes_the_go_colbin_vectors() {
        let vectors: [(&str, UserToken); 6] = [
            (
                "Q5mjBvVTyUTj9mc7Ts4bJyNY1FI+iZwkAv4B",
                UserToken {
                    company_id: 7,
                    id: 42,
                    created: 1234,
                    hash: 12_720_753_295_591_565_293,
                    user: "tester".to_owned(),
                },
            ),
            (
                "Q5mghkDj5lNrpi+bQtZV/gE=",
                UserToken {
                    company_id: 1,
                    id: 1,
                    created: 0,
                    hash: 12_633_170_512_914_502_605,
                    user: String::new(),
                },
            ),
            (
                "Q/n////9a/7//9//8/////01lvxsZq/h4LKGRg0B7x8=",
                UserToken {
                    company_id: 2_147_483_647,
                    id: 2_147_483_647,
                    created: 2_147_483_647,
                    hash: 15_107_712_816_652_668_818,
                    user: "x".to_owned(),
                },
            ),
            (
                "Q9k/gr7mHDA+Byh/SllD8XY5s6D/dwNQLa4dgzZ675BcwN4iXoVj4B8=",
                UserToken {
                    company_id: 999_999,
                    id: 12_345,
                    created: 1_700_000_000,
                    hash: 2_309_713_256_132_687_586,
                    user: "ñandú@example.com".to_owned(),
                },
            ),
            (
                "QzlAavrjcwAANQ5srbRnIHH9xEctWeBXEvFfrplPJYm/AUZ+cfFbNOb5k8iJmuEf",
                UserToken {
                    company_id: 128,
                    id: 127,
                    created: 65_536,
                    hash: 17_959_986_980_219_835_777,
                    user: "a-very-long-user-name-for-width-testing".to_owned(),
                },
            ),
            (
                "QRmjBuSTRMPrKiA5fv1uj00Nw63s7B8=",
                UserToken {
                    company_id: 3,
                    id: 4,
                    created: -5,
                    hash: 1_962_856_781_231_164_119,
                    user: "neg".to_owned(),
                },
            ),
        ];

        for (encoded, expected) in vectors {
            let payload = decode_session_base64(encoded).unwrap();
            assert_eq!(
                decode_session_token(&payload).unwrap(),
                expected,
                "{encoded}"
            );
            // The backend publishes the same bytes under its own alphabet, so both spellings
            // of one token must name one identity.
            let url_alphabet: String = encoded
                .chars()
                .map(|character| match character {
                    '/' => '_',
                    '+' => '-',
                    '=' => '~',
                    other => other,
                })
                .collect();
            assert_eq!(
                decode_session_base64(&url_alphabet).unwrap(),
                payload,
                "{url_alphabet}"
            );
        }
    }

    #[test]
    fn rejects_a_truncated_or_mistyped_session_token() {
        assert_eq!(
            decode_session_token(&[]),
            Err(TokenError::Session(colbin::Error::Truncated))
        );
        // An even first byte is a standard-mode version byte; 0x0a is not one colbin writes.
        assert_eq!(
            decode_session_token(&[0x0a, 0x01, 0x00]),
            Err(TokenError::Session(colbin::Error::BadVersion(0x0a)))
        );
        // A compact message whose first key names no field of this struct. It cannot be
        // stepped over: the wire carries no type tag, so there is no way to know how far.
        assert!(matches!(
            decode_session_token(&[0x43, 0x00, 0x00]),
            Err(TokenError::Session(colbin::Error::UnknownField(_)))
        ));
        assert_eq!(
            decode_session_base64("not base64!!"),
            Err(TokenError::SessionBase64)
        );
    }

    /// Cross-language vectors: the expected column was produced by the TypeScript codec in
    /// `frontend/core/agent/channel.ts` and already pinned Go. All three must agree.
    #[test]
    fn matches_the_cross_language_channel_vectors() {
        let vectors = [
            (1, 1, "N2xQaG8x", "AQE3bFBobzE"),
            (7, 42, "N2xQaG8x", "Byo3bFBobzE"),
            (127, 128, "N2xQaG8x", "f4ABN2xQaG8x"),
            (128, 127, "AAAAAAAA", "gAF_AAAAAAAA"),
            (999999, 1, "____buff", "v4Q9Af___27n3w"),
            (2147483647, 2147483647, "-_-_-_-_", "_____wf_____B_v_v_v_vw"),
            (16383, 16384, "N2xQaG8x", "_3-AgAE3bFBobzE"),
            (2097151, 2097152, "dGFyZGlv", "__9_gICAAXRhcmRpbw"),
        ];

        for (company_id, user_id, tab_id, expected_token) in vectors {
            assert_eq!(
                encode_channel_token(company_id, user_id, tab_id).as_deref(),
                Some(expected_token),
                "encode {company_id}/{user_id}/{tab_id}"
            );
            assert_eq!(
                decode_channel_token(expected_token).unwrap(),
                (company_id, user_id, tab_id.to_owned()),
                "decode {expected_token}"
            );
        }
    }

    #[test]
    fn rejects_non_canonical_and_malformed_channel_tokens() {
        // Overlong company varint (0x81 0x00 == 1): decodes to the same triple as "AQE...",
        // so accepting it would let one tab own two registry keys.
        let overlong =
            general_purpose::URL_SAFE_NO_PAD.encode([0x81, 0x00, 0x01, 1, 2, 3, 4, 5, 6]);
        assert_eq!(
            decode_channel_token(&overlong),
            Err(TokenError::ChannelNotCanonical)
        );

        // A zero id is not a valid identity.
        let zero_company = general_purpose::URL_SAFE_NO_PAD.encode([0x00, 0x01, 1, 2, 3, 4, 5, 6]);
        assert_eq!(
            decode_channel_token(&zero_company),
            Err(TokenError::ChannelRange)
        );

        // Tab id must be exactly 6 bytes.
        let short_tab = general_purpose::URL_SAFE_NO_PAD.encode([0x01, 0x01, 1, 2, 3]);
        assert_eq!(
            decode_channel_token(&short_tab),
            Err(TokenError::ChannelTabID)
        );

        assert_eq!(
            decode_channel_token("not base64!!"),
            Err(TokenError::ChannelBase64)
        );
        // Non-positive ids and a tab id that is not 6 decoded bytes have no valid encoding.
        assert_eq!(encode_channel_token(0, 1, "N2xQaG8x"), None);
        assert_eq!(encode_channel_token(1, -1, "N2xQaG8x"), None);
        assert_eq!(encode_channel_token(1, 1, "QUJD"), None);
    }

    #[test]
    fn session_base64_undoes_the_backend_alphabet() {
        // "_-~" stand in for "/+=" in the backend's URL-safe substitution.
        let payload = [0xFF_u8, 0xFE, 0xFD, 0x01];
        let standard = general_purpose::STANDARD.encode(payload);
        let substituted: String = standard
            .chars()
            .map(|character| match character {
                '/' => '_',
                '+' => '-',
                '=' => '~',
                other => other,
            })
            .collect();
        assert_eq!(decode_session_base64(&substituted).unwrap(), payload);
        // Standard base64 is accepted unchanged, which is what the Go tests emit.
        assert_eq!(decode_session_base64(&standard).unwrap(), payload);
    }
}
