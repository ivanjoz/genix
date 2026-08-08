//! Fixed-size TCP frame and one-byte decision codecs.

use thiserror::Error;

use crate::credits_blob::Credits;

pub const REQUEST_SIZE: usize = 19;
pub const AUTHENTICATED_PAYLOAD_SIZE: usize = 11;
pub const API_GROUP_COUNT: u8 = 6;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Request {
    pub company_id: i32,
    pub user_id: i32,
    pub api_group: u8,
    pub credits: Credits,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ParsedFrame {
    pub request: Request,
    pub auth_hash: [u8; 8],
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

pub fn parse_frame(frame: &[u8; REQUEST_SIZE]) -> Result<ParsedFrame, ProtocolError> {
    let company_id = read_u24(&frame[0..3]) as i32;
    let user_id = read_u24(&frame[3..6]) as i32;
    let api_group = frame[6];
    let cpu = u16::from_be_bytes([frame[7], frame[8]]) as u64;
    let inference = u16::from_be_bytes([frame[9], frame[10]]) as u64;

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

    let mut auth_hash = [0_u8; 8];
    auth_hash.copy_from_slice(&frame[11..19]);
    Ok(ParsedFrame {
        request: Request {
            company_id,
            user_id,
            api_group,
            credits: Credits { cpu, inference },
        },
        auth_hash,
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
        let mut frame = [0_u8; REQUEST_SIZE];
        frame[0..3].copy_from_slice(&[0x12, 0x34, 0x56]);
        frame[3..6].copy_from_slice(&[0x00, 0x00, 0x2A]);
        frame[6] = 5;
        frame[7..9].copy_from_slice(&300_u16.to_be_bytes());
        frame[9..11].copy_from_slice(&25_u16.to_be_bytes());
        frame[11..19].copy_from_slice(&[1, 2, 3, 4, 5, 6, 7, 8]);

        let parsed = parse_frame(&frame).unwrap();
        assert_eq!(parsed.request.company_id, 0x12_34_56);
        assert_eq!(parsed.request.user_id, 42);
        assert_eq!(
            parsed.request.credits,
            Credits {
                cpu: 300,
                inference: 25
            }
        );
        assert_eq!(parsed.auth_hash, [1, 2, 3, 4, 5, 6, 7, 8]);
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
