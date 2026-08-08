//! Two independent token codecs the bridge needs at its HTTP boundary.
//!
//! 1. A fixed-shape colbin decoder for the browser's session token. colbin
//!    (`github.com/ivanjoz/colbin`) is a columnar format; this decodes exactly the
//!    five-field `UsuarioToken` struct the backend issues, not the general format.
//! 2. The channel token, a small custom varint format naming one browser tab.
//!
//! Both are mirrors of Go code in another repository, so every rule here is pinned by
//! vectors generated from that Go code (see the tests at the bottom).

use base64::{Engine, engine::general_purpose};
use thiserror::Error;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum TokenError {
    #[error("session token is not valid base64")]
    SessionBase64,
    #[error("session token is truncated")]
    Truncated,
    #[error("session token has an unsupported colbin version byte")]
    Version,
    #[error("session token must hold exactly one colbin record")]
    RecordCount,
    #[error("session token carries an unknown colbin field id {0}")]
    UnknownField(u8),
    #[error("session token carries an unexpected colbin column type for field {0}")]
    ColumnType(&'static str),
    #[error("session token user name is not valid UTF-8")]
    UserNotUtf8,
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

// --- colbin wire constants (mirror of colbin/format.go) ----------------------------

const COLBIN_VERSION: u8 = 0x01;
const FT_INT: u8 = 0;
const FT_STRING: u8 = 2;
/// Packed bit width per 3-bit precision code. 12/24/48 straddle byte boundaries.
const INT_WIDTHS: [u8; 7] = [8, 12, 16, 24, 32, 48, 64];

/// The session token's five encoded fields, in Go declaration order. colbin derives each
/// wire id by hashing the Go field name, so the order matters: it decides which field wins
/// a hash collision during linear probing.
const SESSION_FIELD_NAMES: [&str; 5] = ["CompanyID", "ID", "Created", "Hash", "User"];

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

/// LSB-first bit reader; mirror of colbin/bitstream.go's `bitReader`.
struct BitReader<'a> {
    buffer: &'a [u8],
    byte_position: usize,
    accumulator: u64,
    accumulated_bits: u8,
}

impl<'a> BitReader<'a> {
    fn new(buffer: &'a [u8]) -> Self {
        Self { buffer, byte_position: 0, accumulator: 0, accumulated_bits: 0 }
    }

    /// Reads `width` bits (0..=64) as the low bits of the result.
    fn read_bits(&mut self, width: u8) -> u64 {
        let mut output = 0_u64;
        let mut shift = 0_u8;
        let mut remaining = width;
        while remaining > 32 {
            output |= self.read_chunk(32) << shift;
            shift += 32;
            remaining -= 32;
        }
        output | (self.read_chunk(remaining) << shift)
    }

    /// Reads at most 32 bits, so every shift stays inside the u64 accumulator.
    fn read_chunk(&mut self, width: u8) -> u64 {
        if width == 0 {
            return 0;
        }
        while self.accumulated_bits < width && self.byte_position < self.buffer.len() {
            self.accumulator |= u64::from(self.buffer[self.byte_position]) << self.accumulated_bits;
            self.byte_position += 1;
            self.accumulated_bits += 8;
        }
        let output = self.accumulator & ((1_u64 << width) - 1);
        self.accumulator >>= width;
        self.accumulated_bits = self.accumulated_bits.saturating_sub(width);
        output
    }
}

/// Interprets the low `width` bits of `value` as two's complement and widens to i64.
fn sign_extend(value: u64, width: u8) -> i64 {
    if width < 64 && value & (1_u64 << (width - 1)) != 0 {
        return (value | !((1_u64 << width) - 1)) as i64;
    }
    value as i64
}

/// FNV-1a 32-bit over `name`, xor-folded to 8 bits. Mirror of colbin/typeinfo.go's `fnv8`.
fn fnv8(name: &str) -> u8 {
    let mut hash = 2_166_136_261_u32;
    for byte in name.as_bytes() {
        hash ^= u32::from(*byte);
        hash = hash.wrapping_mul(16_777_619);
    }
    (hash ^ (hash >> 8) ^ (hash >> 16) ^ (hash >> 24)) as u8
}

/// Assigns each field its colbin wire id: hash the name, then linear-probe past ids already
/// taken. Id 255 is reserved as colbin's terminator, so it is pre-marked as used.
fn session_field_ids() -> [u8; 5] {
    let mut used = [false; 256];
    used[255] = true;
    let mut assigned = [0_u8; 5];
    for (index, field_name) in SESSION_FIELD_NAMES.iter().enumerate() {
        let mut candidate = fnv8(field_name);
        while used[candidate as usize] {
            candidate = candidate.wrapping_add(1);
        }
        used[candidate as usize] = true;
        assigned[index] = candidate;
    }
    assigned
}

/// Cursor over the colbin message, advancing column by column.
struct ColbinCursor<'a> {
    data: &'a [u8],
    position: usize,
}

impl<'a> ColbinCursor<'a> {
    fn read_byte(&mut self) -> Result<u8, TokenError> {
        let byte = *self.data.get(self.position).ok_or(TokenError::Truncated)?;
        self.position += 1;
        Ok(byte)
    }

    fn take(&mut self, length: usize) -> Result<&'a [u8], TokenError> {
        let end = self.position.checked_add(length).ok_or(TokenError::Truncated)?;
        let slice = self.data.get(self.position..end).ok_or(TokenError::Truncated)?;
        self.position = end;
        Ok(slice)
    }

    /// Reads one integer column of `count` values. `native_width` comes from the Go field
    /// type (32 for the i32 fields, 64 for the u64 hash) and is known on both sides, which
    /// is why it is not on the wire.
    ///
    /// Frame-of-reference: every value is stored as a small delta from a per-column base.
    /// In unsigned mode 0 is a sentinel meaning "Go zero value"; in signed mode it is not.
    fn read_int_column(&mut self, count: usize, native_width: u8) -> Result<Vec<i64>, TokenError> {
        let flags = self.read_byte()?;
        if flags & 7 != FT_INT {
            return Err(TokenError::ColumnType("int"));
        }
        let is_signed = flags >> 3 & 1 == 1;
        let precision_code = flags >> 4 & 7;
        let is_empty = flags >> 7 & 1 == 1;
        if is_empty {
            return Ok(vec![0; count]);
        }

        let delta_width = INT_WIDTHS[precision_code as usize];
        let total_bits = usize::from(native_width) + count * usize::from(delta_width);
        let mut reader = BitReader::new(self.take(total_bits.div_ceil(8))?);

        let raw_base = reader.read_bits(native_width);
        let base = if is_signed { sign_extend(raw_base, native_width) } else { raw_base as i64 };
        Ok((0..count)
            .map(|_| {
                let delta = reader.read_bits(delta_width);
                if !is_signed && delta == 0 {
                    return 0;
                }
                base.wrapping_add(delta as i64)
            })
            .collect())
    }

    /// Reads one string column: a flags byte, an embedded 32-bit-native length column, then
    /// the concatenated UTF-8 bytes.
    fn read_string_column(&mut self, count: usize) -> Result<Vec<String>, TokenError> {
        let flags = self.read_byte()?;
        if flags & 7 != FT_STRING {
            return Err(TokenError::ColumnType("string"));
        }
        let lengths = self.read_int_column(count, 32)?;
        let mut values = Vec::with_capacity(count);
        for length in lengths {
            let length = usize::try_from(length).map_err(|_| TokenError::Truncated)?;
            let bytes = self.take(length)?;
            values.push(String::from_utf8(bytes.to_vec()).map_err(|_| TokenError::UserNotUtf8)?);
        }
        Ok(values)
    }
}

/// Decodes the colbin payload of a session token.
///
/// A single Go struct still travels through colbin's *records* path with `recordCount = 1`
/// (`topLevelIsRecords` is true for a struct), so the layout is
/// `[version][recordCount uvarint][colCount][column...]` and columns may arrive in any
/// order — each is self-identified by its field id.
pub fn decode_session_token(payload: &[u8]) -> Result<UserToken, TokenError> {
    let mut cursor = ColbinCursor { data: payload, position: 0 };
    if cursor.read_byte()? != COLBIN_VERSION {
        return Err(TokenError::Version);
    }

    // recordCount is a uvarint; only the single-record shape is valid for a session token.
    let mut record_count = 0_u64;
    let mut shift = 0_u32;
    loop {
        let byte = cursor.read_byte()?;
        record_count |= u64::from(byte & 0x7F) << shift;
        if byte & 0x80 == 0 {
            break;
        }
        shift += 7;
        if shift > 63 {
            return Err(TokenError::RecordCount);
        }
    }
    if record_count != 1 {
        return Err(TokenError::RecordCount);
    }

    let field_ids = session_field_ids();
    let column_count = cursor.read_byte()?;
    let mut token = UserToken::default();

    for _ in 0..column_count {
        let field_id = cursor.read_byte()?;
        match field_id {
            id if id == field_ids[0] => token.company_id = cursor.read_int_column(1, 32)?[0] as i32,
            id if id == field_ids[1] => token.id = cursor.read_int_column(1, 32)?[0] as i32,
            id if id == field_ids[2] => token.created = cursor.read_int_column(1, 32)?[0] as i32,
            // The hash is a u64 whose high bit is usually set, which colbin sees as a
            // negative i64 and stores in signed mode. Reinterpreting recovers the original.
            id if id == field_ids[3] => token.hash = cursor.read_int_column(1, 64)?[0] as u64,
            id if id == field_ids[4] => {
                token.user = cursor.read_string_column(1)?.remove(0);
            }
            unknown => return Err(TokenError::UnknownField(unknown)),
        }
    }
    Ok(token)
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

#[cfg(test)]
mod tests {
    use super::*;

    /// Field ids printed by the Go generator against the real colbin package. A change here
    /// means every session token silently decodes to zero values.
    #[test]
    fn field_ids_match_the_go_colbin_layout() {
        assert_eq!(session_field_ids(), [202, 53, 159, 26, 106]);
    }

    /// Vectors produced by `colbin.Marshal` on the real Go struct. Each covers a different
    /// column shape: the plain case, an all-zero (elided) column plus an empty string, the
    /// i32 maximum, multi-byte UTF-8, and a wider length column.
    #[test]
    fn decodes_the_go_colbin_vectors() {
        let vectors: [(&str, UserToken); 5] = [
            (
                "010105ca000600000001350029000000019f00d1040000011a68ed671d93b93189b00000000000\
                 0000006a02000500000001746573746572",
                UserToken {
                    company_id: 7,
                    id: 42,
                    created: 1234,
                    hash: 12_720_753_295_591_565_293,
                    user: "tester".to_owned(),
                },
            ),
            (
                "010105ca000000000001350000000000019f801a68cd5335e9a50952af00000000000000006a0280",
                UserToken {
                    company_id: 1,
                    id: 1,
                    created: 0,
                    hash: 12_633_170_512_914_502_605,
                    user: String::new(),
                },
            ),
            (
                "010105ca00feffff7f013500feffff7f019f00feffff7f011a6892cf323dc360a9d10000000000\
                 0000006a0200000000000178",
                UserToken {
                    company_id: 2_147_483_647,
                    id: 2_147_483_647,
                    created: 2_147_483_647,
                    hash: 15_107_712_816_652_668_818,
                    user: "x".to_owned(),
                },
            ),
            (
                "010105ca003e420f0001350038300000019f00fff05365011a00e1b6cc14f8bf0d20016a020012\
                 00000001c3b1616e64c3ba406578616d706c652e636f6d",
                UserToken {
                    company_id: 999_999,
                    id: 12_345,
                    created: 1_700_000_000,
                    hash: 2_309_713_256_132_687_586,
                    user: "ñandú@example.com".to_owned(),
                },
            ),
            (
                "010105ca007f0000000135007e000000019f00ffff0000011a6881d5a49e00b13ef90000000000\
                 0000006a02002600000001612d766572792d6c6f6e672d757365722d6e616d652d666f722d7769\
                 6474682d74657374696e67",
                UserToken {
                    company_id: 128,
                    id: 127,
                    created: 65_536,
                    hash: 17_959_986_980_219_835_777,
                    user: "a-very-long-user-name-for-width-testing".to_owned(),
                },
            ),
        ];

        for (hex_payload, expected) in vectors {
            let payload = decode_hex(hex_payload);
            assert_eq!(decode_session_token(&payload).unwrap(), expected);
        }
    }

    #[test]
    fn rejects_a_truncated_or_mistyped_session_token() {
        assert_eq!(decode_session_token(&[]), Err(TokenError::Truncated));
        assert_eq!(decode_session_token(&[0x02, 0x01, 0x00]), Err(TokenError::Version));
        // recordCount = 2: a session token names exactly one identity.
        assert_eq!(decode_session_token(&[0x01, 0x02, 0x00]), Err(TokenError::RecordCount));
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
        let overlong = general_purpose::URL_SAFE_NO_PAD.encode([0x81, 0x00, 0x01, 1, 2, 3, 4, 5, 6]);
        assert_eq!(decode_channel_token(&overlong), Err(TokenError::ChannelNotCanonical));

        // A zero id is not a valid identity.
        let zero_company = general_purpose::URL_SAFE_NO_PAD.encode([0x00, 0x01, 1, 2, 3, 4, 5, 6]);
        assert_eq!(decode_channel_token(&zero_company), Err(TokenError::ChannelRange));

        // Tab id must be exactly 6 bytes.
        let short_tab = general_purpose::URL_SAFE_NO_PAD.encode([0x01, 0x01, 1, 2, 3]);
        assert_eq!(decode_channel_token(&short_tab), Err(TokenError::ChannelTabID));

        assert_eq!(decode_channel_token("not base64!!"), Err(TokenError::ChannelBase64));
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

    fn decode_hex(text: &str) -> Vec<u8> {
        let compact: String = text.chars().filter(|character| !character.is_whitespace()).collect();
        (0..compact.len())
            .step_by(2)
            .map(|index| u8::from_str_radix(&compact[index..index + 2], 16).unwrap())
            .collect()
    }
}
