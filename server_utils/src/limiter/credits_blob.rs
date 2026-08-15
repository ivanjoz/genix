//! Canonical compact representation of per-API-route absolute credit totals.
//!
//! One entry per route that spent anything:
//!
//! ```text
//! header:u16 = (route_id << 2) | width_code    route_id 0..=16383, width_code 0..=3
//! [cpu:width][inference:width]                 width = width_code + 1, one to four bytes
//! ```
//!
//! Entries ascend by route, an all-zero route is omitted, and every value uses the narrowest width
//! that holds it. Those three rules are what make the encoding canonical: one set of totals has
//! exactly one representation, so a byte difference is a real difference and `decode` can refuse
//! anything else as corruption rather than guess at it.

use std::collections::BTreeMap;

use thiserror::Error;

/// Fourteen bits of the two-byte header. The generated route table is two orders of magnitude
/// below this and route numbers are never reused, so the ceiling is a format statement rather than
/// a working limit — scripts/routes/route_ids_generator.go refuses to hand out a number past it.
pub const MAX_ROUTE_ID: u16 = 16_383;

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

pub type RoutedCredits = BTreeMap<u16, Credits>;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum CreditsBlobError {
    #[error("route {0} does not fit in fourteen bits")]
    InvalidRouteID(u16),
    #[error("credits exceed the persisted uint32 maximum")]
    CreditsOverflow,
    #[error("credit blob ends inside an entry")]
    Truncated,
    #[error("routes must be unique and strictly ascending")]
    NonCanonicalOrder,
    #[error("credit entry does not use the smallest possible width")]
    NonCanonicalWidth,
    #[error("an all-zero route must be omitted")]
    NonCanonicalZero,
}

pub fn encode(routes: &RoutedCredits) -> Result<Vec<u8>, CreditsBlobError> {
    let mut encoded = Vec::with_capacity(routes.len() * 6);
    for (&route_id, &credits) in routes {
        if route_id > MAX_ROUTE_ID {
            return Err(CreditsBlobError::InvalidRouteID(route_id));
        }
        if credits.cpu == 0 && credits.inference == 0 {
            continue;
        }
        let width = width_for(credits.cpu.max(credits.inference))?;
        let width_code = (width - 1) as u16;
        encoded.extend_from_slice(&((route_id << 2) | width_code).to_be_bytes());
        write_uint(&mut encoded, credits.cpu, width);
        write_uint(&mut encoded, credits.inference, width);
    }
    Ok(encoded)
}

pub fn decode(encoded: &[u8]) -> Result<RoutedCredits, CreditsBlobError> {
    let mut routes = RoutedCredits::new();
    let mut offset = 0;
    let mut previous_route = None;

    while offset < encoded.len() {
        // A lone trailing byte cannot be a header, and reading it as one would invent a route.
        if encoded.len() - offset < 2 {
            return Err(CreditsBlobError::Truncated);
        }
        let header = u16::from_be_bytes([encoded[offset], encoded[offset + 1]]);
        offset += 2;
        let route_id = header >> 2;
        let width = usize::from((header & 0b11) as u8 + 1);
        if previous_route.is_some_and(|previous| route_id <= previous) {
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
        routes.insert(route_id, Credits { cpu, inference });
        previous_route = Some(route_id);
    }

    Ok(routes)
}

pub fn sum(routes: &RoutedCredits) -> Result<Credits, CreditsBlobError> {
    routes
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
        let routes = RoutedCredits::from([(
            3,
            Credits {
                cpu: 300,
                inference: 25,
            },
        )]);
        // header = (3 << 2) | 1 = 0x000D over two bytes, then two-byte cpu and inference.
        assert_eq!(
            encode(&routes).unwrap(),
            [0x00, 0x0D, 0x01, 0x2C, 0x00, 0x19]
        );
        assert_eq!(decode(&encode(&routes).unwrap()).unwrap(), routes);
    }

    /// The whole reason the header grew a byte: the generated route table passed 63 long ago, and
    /// the ends of the range are where an off-by-one in the shift would show up.
    #[test]
    fn the_full_route_range_round_trips() {
        let routes = RoutedCredits::from([
            (
                0,
                Credits {
                    cpu: 1,
                    inference: 0,
                },
            ),
            (
                103,
                Credits {
                    cpu: 4_000,
                    inference: 12,
                },
            ),
            (
                MAX_ROUTE_ID,
                Credits {
                    cpu: 7,
                    inference: 9,
                },
            ),
        ]);
        assert_eq!(decode(&encode(&routes).unwrap()).unwrap(), routes);
        assert_eq!(
            encode(&RoutedCredits::from([(
                MAX_ROUTE_ID + 1,
                Credits {
                    cpu: 1,
                    inference: 1
                }
            )])),
            Err(CreditsBlobError::InvalidRouteID(MAX_ROUTE_ID + 1))
        );
    }

    #[test]
    fn width_boundaries_round_trip() {
        let routes = RoutedCredits::from([
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
        assert_eq!(decode(&encode(&routes).unwrap()).unwrap(), routes);
    }

    #[test]
    fn rejects_noncanonical_or_truncated_data() {
        // A header that promises one byte of each value and supplies one in total.
        assert_eq!(
            decode(&[0x00, 0x01, 0x00]),
            Err(CreditsBlobError::Truncated)
        );
        // A trailing byte that is half a header. Reading it alone would invent a route.
        assert_eq!(decode(&[0x00]), Err(CreditsBlobError::Truncated));
        assert_eq!(
            decode(&[0x00, 0x00, 0x00, 0x00]),
            Err(CreditsBlobError::NonCanonicalZero)
        );
        assert_eq!(
            decode(&[0x00, 0x01, 0x00, 0x01, 0x00, 0x01]),
            Err(CreditsBlobError::NonCanonicalWidth)
        );
        // Same route twice, and a route that goes backwards.
        assert_eq!(
            decode(&[0x00, 0x04, 0x01, 0x01, 0x00, 0x04, 0x01, 0x01]),
            Err(CreditsBlobError::NonCanonicalOrder)
        );
    }
}
