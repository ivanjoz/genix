//! Charge payload and one-byte decision codecs for opcode `0x01`.
//!
//! Transport concerns — the opcode byte, the authentication tag, the frame sequence — belong to
//! `service`, so this module never sees them: it decodes exactly the eleven bytes that describe
//! one charge.

use thiserror::Error;

use crate::limiter::credits_blob::Credits;

pub const CHARGE_PAYLOAD_SIZE: usize = 11;
pub const API_GROUP_COUNT: u8 = 6;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Request {
    pub company_id: i32,
    pub user_id: i32,
    pub api_group: u8,
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
    #[error("API group {0} is not configured")]
    InvalidApiGroup(u8),
    #[error("at least one credit amount must be positive")]
    EmptyCharge,
}

pub fn parse_charge(payload: &[u8; CHARGE_PAYLOAD_SIZE]) -> Result<Request, ProtocolError> {
    let company_id = read_u24(&payload[0..3]) as i32;
    let user_id = read_u24(&payload[3..6]) as i32;
    let api_group = payload[6];
    let cpu = u16::from_be_bytes([payload[7], payload[8]]) as u64;
    let inference = u16::from_be_bytes([payload[9], payload[10]]) as u64;

    if company_id <= 0 {
        return Err(ProtocolError::InvalidCompany);
    }
    if user_id <= 0 {
        return Err(ProtocolError::InvalidUser);
    }
    if api_group >= API_GROUP_COUNT {
        return Err(ProtocolError::InvalidApiGroup(api_group));
    }
    if cpu == 0 && inference == 0 {
        return Err(ProtocolError::EmptyCharge);
    }

    Ok(Request {
        company_id,
        user_id,
        api_group,
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

    #[test]
    fn parses_the_exact_wire_offsets() {
        let mut payload = [0_u8; CHARGE_PAYLOAD_SIZE];
        payload[0..3].copy_from_slice(&[0x12, 0x34, 0x56]);
        payload[3..6].copy_from_slice(&[0x00, 0x00, 0x2A]);
        payload[6] = 5;
        payload[7..9].copy_from_slice(&300_u16.to_be_bytes());
        payload[9..11].copy_from_slice(&25_u16.to_be_bytes());

        let request = parse_charge(&payload).unwrap();
        assert_eq!(request.company_id, 0x12_34_56);
        assert_eq!(request.user_id, 42);
        assert_eq!(
            request.credits,
            Credits {
                cpu: 300,
                inference: 25
            }
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
