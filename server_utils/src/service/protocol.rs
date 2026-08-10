//! Opcode routing for the shared server-utilities TCP port.
//!
//! Every operation on this port is framed as `[opcode:1][payload][hmac:8]`. The opcode is a
//! routing header and nothing more: each operation owns its own payload layout, its own width,
//! and its own codec in its own module. What the operations share is only the socket, the
//! handshake nonce, and the frame sequence that binds each HMAC to that connection.

use crate::{
    limiter::protocol::CHARGE_PAYLOAD_SIZE,
    lock::protocol::{ACQUIRE_PAYLOAD_SIZE, RELEASE_PAYLOAD_SIZE},
};

pub const OPCODE_SIZE: usize = 1;
pub const AUTH_TAG_SIZE: usize = 8;
pub const REPLY_SIZE: usize = 5;

/// "I could not answer." Deliberately not a valid decision for any opcode: the charge decoder
/// rejects it because its top bits are set, and the lock decoder treats an unknown status as
/// unavailable. Both leave the client to apply its own policy — charges fail open, sign-up locks
/// fail closed — instead of mistaking it for a real verdict.
pub const UNAVAILABLE_STATUS: u8 = 0xFF;

/// Builds the reply frame: `[correlation:u16][status:u8][detail:u16]`.
///
/// `correlation` is the low 16 bits of the request's frame sequence. Nothing new travels in the
/// request to carry it — the sequence already exists, is already per-connection and monotonic,
/// and both sides already track it for the HMAC. It is what lets a client match a reply to the
/// caller that is waiting for it once several requests are in flight at once; truncating to 16
/// bits only becomes ambiguous past 65_535 concurrent requests on one connection.
///
/// `status` keeps its per-opcode meaning, and zero is still success everywhere. `detail` carries
/// the lock generation on a granted acquire and is zero otherwise.
pub fn encode_reply(sequence: u64, status: u8, detail: u16) -> [u8; REPLY_SIZE] {
    let mut reply = [0_u8; REPLY_SIZE];
    reply[0..2].copy_from_slice(&((sequence & 0xFFFF) as u16).to_be_bytes());
    reply[2] = status;
    reply[3..5].copy_from_slice(&detail.to_be_bytes());
    reply
}

/// Widest payload across every opcode, so one stack buffer serves them all.
const LARGEST_PAYLOAD_SIZE: usize = if CHARGE_PAYLOAD_SIZE > ACQUIRE_PAYLOAD_SIZE {
    CHARGE_PAYLOAD_SIZE
} else {
    ACQUIRE_PAYLOAD_SIZE
};
pub const MAX_FRAME_SIZE: usize = OPCODE_SIZE + LARGEST_PAYLOAD_SIZE + AUTH_TAG_SIZE;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum Opcode {
    ChargeCredits = 0x01,
    LockAcquire = 0x02,
    LockRelease = 0x03,
}

impl Opcode {
    /// 0x00 stays permanently unassigned, so an all-zero frame from a broken or misconfigured
    /// client cannot route to a real operation.
    pub fn from_byte(byte: u8) -> Option<Self> {
        match byte {
            0x01 => Some(Self::ChargeCredits),
            0x02 => Some(Self::LockAcquire),
            0x03 => Some(Self::LockRelease),
            _ => None,
        }
    }

    /// Bytes between the opcode and the authentication tag.
    pub fn payload_size(self) -> usize {
        match self {
            Self::ChargeCredits => CHARGE_PAYLOAD_SIZE,
            Self::LockAcquire => ACQUIRE_PAYLOAD_SIZE,
            Self::LockRelease => RELEASE_PAYLOAD_SIZE,
        }
    }

    /// Whole frame width: opcode, payload, and tag.
    pub fn frame_size(self) -> usize {
        OPCODE_SIZE + self.payload_size() + AUTH_TAG_SIZE
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_opcode_fits_the_shared_buffer() {
        for opcode in [
            Opcode::ChargeCredits,
            Opcode::LockAcquire,
            Opcode::LockRelease,
        ] {
            assert!(opcode.frame_size() <= MAX_FRAME_SIZE);
        }
    }

    #[test]
    fn the_reply_carries_the_truncated_sequence() {
        assert_eq!(encode_reply(0, 0, 0), [0x00, 0x00, 0x00, 0x00, 0x00]);
        assert_eq!(encode_reply(1, 27, 0), [0x00, 0x01, 0x1B, 0x00, 0x00]);
        assert_eq!(encode_reply(7, 0, 300), [0x00, 0x07, 0x00, 0x01, 0x2C]);
        // Only the low 16 bits travel, so a client correlates on the same truncation.
        assert_eq!(encode_reply(0x1_0002, 0, 0), [0x00, 0x02, 0x00, 0x00, 0x00]);
    }

    #[test]
    fn every_opcode_keeps_its_documented_width() {
        assert_eq!(Opcode::from_byte(0x01), Some(Opcode::ChargeCredits));
        assert_eq!(Opcode::from_byte(0x02), Some(Opcode::LockAcquire));
        assert_eq!(Opcode::from_byte(0x03), Some(Opcode::LockRelease));
        assert_eq!(Opcode::ChargeCredits.frame_size(), 20);
        assert_eq!(Opcode::LockAcquire.frame_size(), 24);
        // Release names the lock it ends: action, identifier, generation.
        assert_eq!(Opcode::LockRelease.frame_size(), 21);
        // Unassigned bytes must not resolve, or a garbage frame would be dispatched.
        assert_eq!(Opcode::from_byte(0x00), None);
        assert_eq!(Opcode::from_byte(0xFF), None);
    }
}
