//! Charge payload and one-byte decision codecs for opcode `0x01`.
//!
//! Transport concerns — the opcode byte, the authentication tag, the frame sequence — belong to
//! `service`, so this module never sees them: it decodes exactly the twelve bytes that describe
//! one charge.

use thiserror::Error;

use crate::limiter::credits_blob::{Credits, MAX_ROUTE_ID};

pub const CHARGE_PAYLOAD_SIZE: usize = 12;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Request {
    pub company_id: i32,
    pub user_id: i32,
    /// The generated number of the API route being charged, from
    /// backend/core/api_routes.generated.go. Zero is the unknown-route bucket: the Go side hands
    /// out zero for a path that matched no generated entry, and those credits are still real.
    pub route_id: u16,
    pub credits: Credits,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Scope {
    Company = 0,
    User = 1,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum Window {
    TenSeconds = 0,
    Hour = 1,
    Day = 2,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct LimitViolation {
    pub scope: Scope,
    pub window: Window,
    pub cpu: bool,
    pub inference: bool,
}

impl LimitViolation {
    pub fn response_byte(self) -> u8 {
        // Resource flags make every rejection nonzero while zero remains the allow response.
        (self.scope as u8)
            | ((self.window as u8) << 1)
            | (u8::from(self.inference) << 3)
            | (u8::from(self.cpu) << 4)
    }
}

#[derive(Debug, Error, PartialEq, Eq)]
pub enum ProtocolError {
    #[error("company_id must be positive")]
    InvalidCompany,
    #[error("user_id must be positive")]
    InvalidUser,
    #[error("route {0} does not fit the {MAX_ROUTE_ID}-route encoding")]
    InvalidRouteID(u16),
    #[error("at least one credit amount must be positive")]
    EmptyCharge,
}

/// Decodes one charge. The route is range-checked against what the blob can hold and against
/// nothing else — deliberately.
///
/// The tempting check is "does this route exist", but this daemon must not know the route table.
/// Routes are numbered by a Go generator and a new one appears whenever a handler is added; a
/// daemon that refused every number above the highest it was built with would reject exactly the
/// newest routes. Charging fails open on the Go side, so the refusal would not surface as an error
/// — those routes would simply stop being counted, quietly and for as long as the daemon ran.
pub fn parse_charge(payload: &[u8; CHARGE_PAYLOAD_SIZE]) -> Result<Request, ProtocolError> {
    let company_id = read_u24(&payload[0..3]) as i32;
    let user_id = read_u24(&payload[3..6]) as i32;
    let route_id = u16::from_be_bytes([payload[6], payload[7]]);
    let cpu = u16::from_be_bytes([payload[8], payload[9]]) as u64;
    let inference = u16::from_be_bytes([payload[10], payload[11]]) as u64;

    if company_id <= 0 {
        return Err(ProtocolError::InvalidCompany);
    }
    if user_id <= 0 {
        return Err(ProtocolError::InvalidUser);
    }
    if route_id > MAX_ROUTE_ID {
        return Err(ProtocolError::InvalidRouteID(route_id));
    }
    if cpu == 0 && inference == 0 {
        return Err(ProtocolError::EmptyCharge);
    }

    Ok(Request {
        company_id,
        user_id,
        route_id,
        credits: Credits { cpu, inference },
    })
}

fn read_u24(bytes: &[u8]) -> u32 {
    // The wire ID is network-order and expands into the positive internal int32 range.
    (u32::from(bytes[0]) << 16) | (u32::from(bytes[1]) << 8) | u32::from(bytes[2])
}

#[cfg(test)]
mod tests {
    use super::*;

    fn charge_payload(route_id: u16) -> [u8; CHARGE_PAYLOAD_SIZE] {
        let mut payload = [0_u8; CHARGE_PAYLOAD_SIZE];
        payload[0..3].copy_from_slice(&[0x12, 0x34, 0x56]);
        payload[3..6].copy_from_slice(&[0x00, 0x00, 0x2A]);
        payload[6..8].copy_from_slice(&route_id.to_be_bytes());
        payload[8..10].copy_from_slice(&300_u16.to_be_bytes());
        payload[10..12].copy_from_slice(&25_u16.to_be_bytes());
        payload
    }

    #[test]
    fn parses_the_exact_wire_offsets() {
        let request = parse_charge(&charge_payload(103)).unwrap();
        assert_eq!(request.company_id, 0x12_34_56);
        assert_eq!(request.user_id, 42);
        assert_eq!(request.route_id, 103);
        assert_eq!(
            request.credits,
            Credits {
                cpu: 300,
                inference: 25
            }
        );
    }

    /// A route this daemon has never heard of must still be charged. The Go side numbers routes,
    /// and anything stricter here silently stops counting whatever was added most recently.
    #[test]
    fn an_unknown_route_is_charged_and_an_unencodable_one_is_not() {
        assert_eq!(parse_charge(&charge_payload(0)).unwrap().route_id, 0);
        assert_eq!(
            parse_charge(&charge_payload(MAX_ROUTE_ID))
                .unwrap()
                .route_id,
            MAX_ROUTE_ID
        );
        assert_eq!(
            parse_charge(&charge_payload(MAX_ROUTE_ID + 1)),
            Err(ProtocolError::InvalidRouteID(MAX_ROUTE_ID + 1))
        );
    }

    #[test]
    fn response_bits_follow_the_contract() {
        let response = LimitViolation {
            scope: Scope::User,
            window: Window::Hour,
            cpu: true,
            inference: true,
        };
        assert_eq!(response.response_byte(), 0b1_1011);
    }
}
