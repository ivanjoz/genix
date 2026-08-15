//! Payload codec for opcode `0x04`, the end-of-request record.
//!
//! Transport concerns — the opcode byte, the length header, the authentication tag — belong to
//! `service`, so this module never sees them: it decodes exactly the bytes that describe one
//! finished request and the handful of code lines that failed inside it.
//!
//! This is the first variable-length payload on the port. Everything before it described a fixed
//! record; a request log carries strings, so every length it states is checked against what is
//! actually left in the buffer before it is trusted.

use thiserror::Error;

/// Header before the first error block: date, request id, route, frame, company, user, elapsed,
/// error count.
pub const REQUEST_LOG_HEADER_SIZE: usize = 2 + 8 + 2 + 1 + 3 + 4 + 2 + 1;

/// Four errors is the cap the Go side enforces; past that a request is one failure cascading.
pub const MAX_ERRORS_PER_REQUEST: usize = 4;
/// Enough for "product-stock-movement.go:1204" several times over.
pub const MAX_CODE_LINE_BYTES: usize = 64;
/// The preview only. CloudWatch has the message in full.
pub const MAX_ERROR_TEXT_BYTES: usize = 200;

/// Ceiling on one request-log payload, and therefore on what a client can make the daemon buffer
/// before its tag has been verified. Every field is at its widest here.
///
/// One error block is: the 4-byte id, a 1-byte code-line length and its bytes, then a **2-byte**
/// text length and its bytes. The text prefix is two bytes because 200 does not have to fit in one
/// forever — and a constant that undercounts it would size MAX_FRAME_SIZE too small, so the widest
/// legitimate frame would be refused as oversized.
const REQUEST_LOG_MAX_ERROR_BLOCK_SIZE: usize =
    4 + 1 + MAX_CODE_LINE_BYTES + 2 + MAX_ERROR_TEXT_BYTES;
pub const REQUEST_LOG_MAX_PAYLOAD_SIZE: usize =
    REQUEST_LOG_HEADER_SIZE + MAX_ERRORS_PER_REQUEST * REQUEST_LOG_MAX_ERROR_BLOCK_SIZE;

/// Fifteen-minute slots in a day, four per hour.
pub const FRAMES_PER_DAY: u8 = 96;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ErrorEntry {
    /// Hashed from the code line by the Go side, which is the sole authority on the value. The
    /// daemon stores it and never recomputes it.
    pub id: i32,
    pub code_line: String,
    pub text: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct RequestLogRecord {
    pub date: i16,
    pub request_id: i64,
    pub route_id: i16,
    pub frame: u8,
    pub company_id: i32,
    pub user_id: i32,
    pub elapsed_ms: i16,
    pub errors: Vec<ErrorEntry>,
}

impl RequestLogRecord {
    /// Packs the three dimensions the dashboard groups by into one sortable integer.
    ///
    /// The frame leads deliberately: it makes one fifteen-minute slice of the day a single
    /// contiguous clustering range, which is what lets the dashboard poll forward instead of
    /// rereading the day.
    ///
    ///   bits 47..40  frame     (0..95)
    ///   bits 39..24  route_id
    ///   bits 23..0   company_id
    ///
    /// Mirrored in backend/core/types/user_logs.go — that is the side that *reads* the column, so
    /// the two must agree byte for byte or the dashboard silently ranges over the wrong rows. The
    /// vectors in both test files pin them together.
    pub fn frame_route_company_agg(&self) -> i64 {
        (self.frame as i64) << 40
            | ((self.route_id as u16) as i64) << 24
            | (self.company_id as i64) & 0xFF_FFFF
    }

    /// What lands in the error_count column. Capped by the parser, so this is always small.
    pub fn error_count(&self) -> i8 {
        self.errors.len() as i8
    }

    pub fn error_ids(&self) -> Vec<i32> {
        self.errors.iter().map(|entry| entry.id).collect()
    }
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum RequestLogError {
    #[error("payload is {0} bytes, shorter than the {REQUEST_LOG_HEADER_SIZE}-byte header")]
    TooShort(usize),
    #[error("frame {0} is outside the {FRAMES_PER_DAY} slots of a day")]
    InvalidFrame(u8),
    #[error("error count {0} exceeds the cap of {MAX_ERRORS_PER_REQUEST}")]
    TooManyErrors(u8),
    #[error("error block {0} runs past the end of the payload")]
    Truncated(usize),
    #[error("error block {0} declares a {1}-byte {2}, over its ceiling")]
    FieldTooLong(usize, usize, &'static str),
    #[error("error block {0} has a {1} that is not valid UTF-8")]
    InvalidUtf8(usize, &'static str),
    #[error("payload has {0} bytes left after the last error block")]
    TrailingBytes(usize),
}

pub fn parse_request_log(payload: &[u8]) -> Result<RequestLogRecord, RequestLogError> {
    if payload.len() < REQUEST_LOG_HEADER_SIZE {
        return Err(RequestLogError::TooShort(payload.len()));
    }

    let date = i16::from_be_bytes([payload[0], payload[1]]);
    let request_id = i64::from_be_bytes(payload[2..10].try_into().expect("eight bytes"));
    let route_id = i16::from_be_bytes([payload[10], payload[11]]);
    let frame = payload[12];
    let company_id = read_u24(&payload[13..16]) as i32;
    let user_id = i32::from_be_bytes(payload[16..20].try_into().expect("four bytes"));
    let elapsed_ms = i16::from_be_bytes([payload[20], payload[21]]);
    let error_count = payload[22];

    if frame >= FRAMES_PER_DAY {
        return Err(RequestLogError::InvalidFrame(frame));
    }
    if error_count as usize > MAX_ERRORS_PER_REQUEST {
        return Err(RequestLogError::TooManyErrors(error_count));
    }

    let mut errors = Vec::with_capacity(error_count as usize);
    let mut offset = REQUEST_LOG_HEADER_SIZE;
    for block in 0..error_count as usize {
        // The id and the code line's one length byte. Every read below is preceded by the check
        // that the bytes it wants are actually there — a length header is an instruction from the
        // other side of a socket, and acting on one before verifying it is how a parser reads
        // memory it was never given.
        if payload.len() < offset + 5 {
            return Err(RequestLogError::Truncated(block));
        }
        let id = i32::from_be_bytes(payload[offset..offset + 4].try_into().expect("four bytes"));
        offset += 4;

        let code_line_length = payload[offset] as usize;
        let code_line = read_length_prefixed(
            payload,
            &mut offset,
            code_line_length,
            1,
            MAX_CODE_LINE_BYTES,
            block,
            "code line",
        )?;

        if payload.len() < offset + 2 {
            return Err(RequestLogError::Truncated(block));
        }
        let text_length = u16::from_be_bytes([payload[offset], payload[offset + 1]]) as usize;
        let text = read_length_prefixed(
            payload,
            &mut offset,
            text_length,
            2,
            MAX_ERROR_TEXT_BYTES,
            block,
            "text",
        )?;

        errors.push(ErrorEntry {
            id,
            code_line,
            text,
        });
    }

    // A payload that describes fewer bytes than it carries means the two sides disagree about the
    // layout. Accepting it would let a mismatch go unnoticed until it produced wrong rows.
    if offset != payload.len() {
        return Err(RequestLogError::TrailingBytes(payload.len() - offset));
    }

    Ok(RequestLogRecord {
        date,
        request_id,
        route_id,
        frame,
        company_id,
        user_id,
        elapsed_ms,
        errors,
    })
}

/// Reads one length-prefixed string, advancing past both the prefix and the bytes it counts.
/// `prefix_size` is how wide the already-read length was, so the offset lands correctly.
fn read_length_prefixed(
    payload: &[u8],
    offset: &mut usize,
    length: usize,
    prefix_size: usize,
    maximum: usize,
    block: usize,
    field: &'static str,
) -> Result<String, RequestLogError> {
    if length > maximum {
        return Err(RequestLogError::FieldTooLong(block, length, field));
    }
    let start = *offset + prefix_size;
    let end = start + length;
    if payload.len() < end {
        return Err(RequestLogError::Truncated(block));
    }
    let value = std::str::from_utf8(&payload[start..end])
        .map_err(|_| RequestLogError::InvalidUtf8(block, field))?
        .to_string();
    *offset = end;
    Ok(value)
}

fn read_u24(bytes: &[u8]) -> u32 {
    (u32::from(bytes[0]) << 16) | (u32::from(bytes[1]) << 8) | u32::from(bytes[2])
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Builds the payload the Go client writes, so the offsets below are the wire contract.
    fn encode(record: &RequestLogRecord) -> Vec<u8> {
        let mut payload = Vec::new();
        payload.extend_from_slice(&record.date.to_be_bytes());
        payload.extend_from_slice(&record.request_id.to_be_bytes());
        payload.extend_from_slice(&record.route_id.to_be_bytes());
        payload.push(record.frame);
        payload.extend_from_slice(&record.company_id.to_be_bytes()[1..4]);
        payload.extend_from_slice(&record.user_id.to_be_bytes());
        payload.extend_from_slice(&record.elapsed_ms.to_be_bytes());
        payload.push(record.errors.len() as u8);
        for entry in &record.errors {
            payload.extend_from_slice(&entry.id.to_be_bytes());
            payload.push(entry.code_line.len() as u8);
            payload.extend_from_slice(entry.code_line.as_bytes());
            payload.extend_from_slice(&(entry.text.len() as u16).to_be_bytes());
            payload.extend_from_slice(entry.text.as_bytes());
        }
        payload
    }

    fn sample() -> RequestLogRecord {
        RequestLogRecord {
            date: 20_500,
            request_id: 1_767_225_600_123,
            route_id: 102,
            frame: 41,
            company_id: 7,
            user_id: 42,
            elapsed_ms: 318,
            errors: vec![
                ErrorEntry {
                    id: 1_234_567,
                    code_line: "responses.go:539".to_string(),
                    text: "no se pudo obtener el registro".to_string(),
                },
                ErrorEntry {
                    id: 7_654_321,
                    code_line: "product-stock.go:1204".to_string(),
                    text: "error al consultar el stock".to_string(),
                },
            ],
        }
    }

    #[test]
    fn round_trips_a_record_with_errors() {
        let record = sample();
        assert_eq!(parse_request_log(&encode(&record)).unwrap(), record);
    }

    #[test]
    fn round_trips_a_record_with_no_errors() {
        let mut record = sample();
        record.errors.clear();
        let payload = encode(&record);
        assert_eq!(payload.len(), REQUEST_LOG_HEADER_SIZE);
        assert_eq!(parse_request_log(&payload).unwrap(), record);
    }

    #[test]
    fn parses_the_exact_wire_offsets() {
        let record = parse_request_log(&encode(&sample())).unwrap();
        assert_eq!(record.date, 20_500);
        assert_eq!(record.request_id, 1_767_225_600_123);
        assert_eq!(record.route_id, 102);
        assert_eq!(record.frame, 41);
        assert_eq!(record.company_id, 7);
        assert_eq!(record.user_id, 42);
        assert_eq!(record.elapsed_ms, 318);
        assert_eq!(record.error_count(), 2);
        assert_eq!(record.error_ids(), vec![1_234_567, 7_654_321]);
    }

    /// These are the same vectors as TestMakeFrameRouteCompanyAgg in
    /// backend/core/types/user_logs_test.go. If either side moves, the dashboard ranges over rows
    /// that were packed under a different layout and silently reports the wrong numbers.
    #[test]
    fn the_packed_key_matches_the_go_vectors() {
        let pack = |frame: u8, route_id: i16, company_id: i32| {
            RequestLogRecord {
                date: 0,
                request_id: 0,
                route_id,
                frame,
                company_id,
                user_id: 0,
                elapsed_ms: 0,
                errors: vec![],
            }
            .frame_route_company_agg()
        };

        assert_eq!(pack(0, 0, 0), 0);
        assert_eq!(pack(1, 0, 0), 1 << 40);
        assert_eq!(pack(0, 1, 0), 1 << 24);
        assert_eq!(pack(0, 0, 1), 1);
        assert_eq!(pack(95, 0, 0), 95 << 40);
        assert_eq!(pack(41, 102, 7), 41 << 40 | 102 << 24 | 7);
        assert_eq!(
            pack(95, 32767, 16_777_215),
            95_i64 << 40 | 32767_i64 << 24 | 16_777_215
        );
        // A frame's rows must never leak into the next frame's clustering range.
        assert!(pack(41, 32767, 16_777_215) < pack(42, 0, 0));
    }

    /// Bytes produced by the Go encoder, pasted in verbatim.
    ///
    /// Every other test here round-trips through this module's own `encode`, which would agree
    /// with itself even if both halves drifted from Go together. This one cannot: it is the actual
    /// output of `encodeRequestLog(sampleRecord())` in backend/core/server_utils/request_log.go,
    /// and the values below are that function's sample record. Regenerate it from
    /// `TestEncodeRequestLogWireOffsets` if the layout ever changes on purpose.
    #[test]
    fn parses_bytes_produced_by_the_go_encoder() {
        let hex = "50140000019b76daa87b0066290000070000002a013e010012d68710726573706f6e7365\
                   732e676f3a353339001e6e6f207365207075646f206f6274656e657220656c2072656769\
                   7374726f";
        let payload: Vec<u8> = (0..hex.len() / 2)
            .map(|index| u8::from_str_radix(&hex[index * 2..index * 2 + 2], 16).unwrap())
            .collect();

        let record = parse_request_log(&payload).expect("the Go encoder produced an unparsable frame");
        assert_eq!(record.date, 20_500);
        assert_eq!(record.request_id, 1_767_225_600_123);
        assert_eq!(record.route_id, 102);
        assert_eq!(record.frame, 41);
        assert_eq!(record.company_id, 7);
        assert_eq!(record.user_id, 42);
        assert_eq!(record.elapsed_ms, 318);
        assert_eq!(record.errors.len(), 1);
        assert_eq!(record.errors[0].id, 1_234_567);
        assert_eq!(record.errors[0].code_line, "responses.go:539");
        assert_eq!(record.errors[0].text, "no se pudo obtener el registro");
        // And the column the dashboard ranges over comes out of those bytes correctly.
        assert_eq!(
            record.frame_route_company_agg(),
            41_i64 << 40 | 102_i64 << 24 | 7
        );
    }

    #[test]
    fn a_short_payload_is_refused() {
        assert_eq!(
            parse_request_log(&[0_u8; REQUEST_LOG_HEADER_SIZE - 1]),
            Err(RequestLogError::TooShort(REQUEST_LOG_HEADER_SIZE - 1))
        );
    }

    #[test]
    fn a_frame_outside_the_day_is_refused() {
        let mut record = sample();
        record.frame = FRAMES_PER_DAY;
        record.errors.clear();
        assert_eq!(
            parse_request_log(&encode(&record)),
            Err(RequestLogError::InvalidFrame(FRAMES_PER_DAY))
        );
    }

    #[test]
    fn more_errors_than_the_cap_are_refused() {
        let mut payload = encode(&{
            let mut record = sample();
            record.errors.clear();
            record
        });
        payload[22] = MAX_ERRORS_PER_REQUEST as u8 + 1;
        assert_eq!(
            parse_request_log(&payload),
            Err(RequestLogError::TooManyErrors(
                MAX_ERRORS_PER_REQUEST as u8 + 1
            ))
        );
    }

    /// A length that runs past the buffer is the one thing a variable-length frame must never act
    /// on: believing it is how a parser reads memory it was not given.
    #[test]
    fn a_length_past_the_end_is_refused() {
        let full = encode(&sample());
        for truncated_at in REQUEST_LOG_HEADER_SIZE..full.len() {
            let outcome = parse_request_log(&full[..truncated_at]);
            assert!(
                outcome.is_err(),
                "a payload cut at {truncated_at} bytes parsed instead of failing"
            );
        }
    }

    #[test]
    fn an_oversized_field_is_refused() {
        let mut record = sample();
        record.errors.truncate(1);
        record.errors[0].code_line = "x".repeat(MAX_CODE_LINE_BYTES + 1);
        assert_eq!(
            parse_request_log(&encode(&record)),
            Err(RequestLogError::FieldTooLong(
                0,
                MAX_CODE_LINE_BYTES + 1,
                "code line"
            ))
        );
    }

    #[test]
    fn invalid_utf8_is_refused() {
        let mut record = sample();
        record.errors.truncate(1);
        let mut payload = encode(&record);
        // Overwrite the first byte of the code line with a lone continuation byte.
        payload[REQUEST_LOG_HEADER_SIZE + 5] = 0x80;
        assert_eq!(
            parse_request_log(&payload),
            Err(RequestLogError::InvalidUtf8(0, "code line"))
        );
    }

    #[test]
    fn trailing_bytes_are_refused() {
        let mut payload = encode(&sample());
        payload.push(0);
        assert_eq!(parse_request_log(&payload), Err(RequestLogError::TrailingBytes(1)));
    }

    #[test]
    fn the_widest_possible_record_fits_the_declared_ceiling() {
        let record = RequestLogRecord {
            date: i16::MAX,
            request_id: i64::MAX,
            route_id: i16::MAX,
            frame: FRAMES_PER_DAY - 1,
            company_id: 16_777_215,
            user_id: i32::MAX,
            elapsed_ms: i16::MAX,
            errors: (0..MAX_ERRORS_PER_REQUEST)
                .map(|index| ErrorEntry {
                    id: i32::MAX,
                    code_line: "c".repeat(MAX_CODE_LINE_BYTES),
                    text: format!("{}{}", index, "t".repeat(MAX_ERROR_TEXT_BYTES - 1)),
                })
                .collect(),
        };
        let payload = encode(&record);
        assert_eq!(payload.len(), REQUEST_LOG_MAX_PAYLOAD_SIZE);
        assert_eq!(parse_request_log(&payload).unwrap(), record);
    }
}
