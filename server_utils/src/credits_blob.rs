//! Canonical compact representation of per-API-group absolute credit totals.

use std::collections::BTreeMap;

use thiserror::Error;

pub const MAX_API_GROUP: u8 = 63;

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct Credits {
    pub cpu: u64,
    pub inference: u64,
}

impl Credits {
    pub fn checked_add(self, increment: Self) -> Option<Self> {
        Some(Self {
            cpu: self.cpu.checked_add(increment.cpu)?,
            inference: self.inference.checked_add(increment.inference)?,
        })
    }
}

pub type GroupedCredits = BTreeMap<u8, Credits>;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum CreditsBlobError {
    #[error("API group {0} does not fit in six bits")]
    InvalidApiGroup(u8),
    #[error("credits exceed the persisted uint32 maximum")]
    CreditsOverflow,
    #[error("credit blob ends inside an entry")]
    Truncated,
    #[error("API groups must be unique and strictly ascending")]
    NonCanonicalOrder,
    #[error("credit entry does not use the smallest possible width")]
    NonCanonicalWidth,
    #[error("an all-zero API group must be omitted")]
    NonCanonicalZero,
}

pub fn encode(groups: &GroupedCredits) -> Result<Vec<u8>, CreditsBlobError> {
    let mut encoded = Vec::with_capacity(groups.len() * 5);
    for (&api_group, &credits) in groups {
        if api_group > MAX_API_GROUP {
            return Err(CreditsBlobError::InvalidApiGroup(api_group));
        }
        if credits.cpu == 0 && credits.inference == 0 {
            continue;
        }
        let width = width_for(credits.cpu.max(credits.inference))?;
        let width_code = (width - 1) as u8;
        encoded.push((api_group << 2) | width_code);
        write_uint(&mut encoded, credits.cpu, width);
        write_uint(&mut encoded, credits.inference, width);
    }
    Ok(encoded)
}

pub fn decode(encoded: &[u8]) -> Result<GroupedCredits, CreditsBlobError> {
    let mut groups = GroupedCredits::new();
    let mut offset = 0;
    let mut previous_group = None;

    while offset < encoded.len() {
        let header = encoded[offset];
        offset += 1;
        let api_group = header >> 2;
        let width = usize::from((header & 0b11) + 1);
        if previous_group.is_some_and(|previous| api_group <= previous) {
            return Err(CreditsBlobError::NonCanonicalOrder);
        }
        if encoded.len().saturating_sub(offset) < width * 2 {
            return Err(CreditsBlobError::Truncated);
        }
        let cpu = read_uint(&encoded[offset..offset + width]);
        offset += width;
        let inference = read_uint(&encoded[offset..offset + width]);
        offset += width;
        if cpu == 0 && inference == 0 {
            return Err(CreditsBlobError::NonCanonicalZero);
        }
        if width_for(cpu.max(inference))? != width {
            return Err(CreditsBlobError::NonCanonicalWidth);
        }
        groups.insert(api_group, Credits { cpu, inference });
        previous_group = Some(api_group);
    }

    Ok(groups)
}

pub fn sum(groups: &GroupedCredits) -> Result<Credits, CreditsBlobError> {
    groups
        .values()
        .try_fold(Credits::default(), |total, value| {
            total
                .checked_add(*value)
                .ok_or(CreditsBlobError::CreditsOverflow)
        })
}

fn width_for(value: u64) -> Result<usize, CreditsBlobError> {
    if value <= u8::MAX.into() {
        Ok(1)
    } else if value <= u16::MAX.into() {
        Ok(2)
    } else if value <= 0xFF_FFFF {
        Ok(3)
    } else if value <= u32::MAX.into() {
        Ok(4)
    } else {
        Err(CreditsBlobError::CreditsOverflow)
    }
}

fn write_uint(target: &mut Vec<u8>, value: u64, width: usize) {
    // Slice the big-endian u32 representation to produce the requested 1-4 bytes.
    let bytes = (value as u32).to_be_bytes();
    target.extend_from_slice(&bytes[4 - width..]);
}

fn read_uint(bytes: &[u8]) -> u64 {
    bytes
        .iter()
        .fold(0_u64, |value, byte| (value << 8) | u64::from(*byte))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn requested_example_matches() {
        let groups = GroupedCredits::from([(
            3,
            Credits {
                cpu: 300,
                inference: 25,
            },
        )]);
        assert_eq!(encode(&groups).unwrap(), [0x0D, 0x01, 0x2C, 0x00, 0x19]);
        assert_eq!(decode(&encode(&groups).unwrap()).unwrap(), groups);
    }

    #[test]
    fn width_boundaries_round_trip() {
        let groups = GroupedCredits::from([
            (
                0,
                Credits {
                    cpu: 255,
                    inference: 1,
                },
            ),
            (
                1,
                Credits {
                    cpu: 256,
                    inference: 65_535,
                },
            ),
            (
                2,
                Credits {
                    cpu: 65_536,
                    inference: 0xFF_FFFF,
                },
            ),
            (
                3,
                Credits {
                    cpu: 0x1_000000,
                    inference: u32::MAX as u64,
                },
            ),
        ]);
        assert_eq!(decode(&encode(&groups).unwrap()).unwrap(), groups);
    }

    #[test]
    fn rejects_noncanonical_or_truncated_data() {
        assert_eq!(decode(&[0x01, 0x00]), Err(CreditsBlobError::Truncated));
        assert_eq!(
            decode(&[0x00, 0x00, 0x00]),
            Err(CreditsBlobError::NonCanonicalZero)
        );
        assert_eq!(
            decode(&[0x01, 0x00, 0x01, 0x00, 0x01]),
            Err(CreditsBlobError::NonCanonicalWidth)
        );
    }
}
