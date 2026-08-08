//! Decimal period keys shared by aggregation and Scylla persistence.

use thiserror::Error;

pub const FIVE_MINUTE_PREFIX: i64 = 100_000_000;
pub const DAILY_PREFIX: i64 = 200_000_000;
pub const FIVE_MINUTE_SECONDS: i64 = 300;
pub const DAY_SECONDS: i64 = 86_400;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum TimeFrameError {
    #[error("unix seconds cannot be negative")]
    NegativeUnixTime,
    #[error("time frame {0} is outside its reserved int32 range")]
    OutOfRange(i64),
}

pub fn five_minute(unix_seconds: i64) -> Result<i32, TimeFrameError> {
    encode(FIVE_MINUTE_PREFIX, unix_seconds, FIVE_MINUTE_SECONDS)
}

pub fn daily(unix_seconds: i64) -> Result<i32, TimeFrameError> {
    encode(DAILY_PREFIX, unix_seconds, DAY_SECONDS)
}

pub fn hour_five_minute_range(unix_seconds: i64) -> Result<(i32, i32), TimeFrameError> {
    if unix_seconds < 0 {
        return Err(TimeFrameError::NegativeUnixTime);
    }
    // Twelve five-minute records exactly cover the fixed UTC hour.
    let hour_start = unix_seconds / 3_600 * 3_600;
    Ok((five_minute(hour_start)?, five_minute(hour_start + 3_599)?))
}

pub fn is_five_minute(value: i32) -> bool {
    (100_000_000..200_000_000).contains(&value)
}

pub fn is_daily(value: i32) -> bool {
    (200_000_000..300_000_000).contains(&value)
}

fn encode(prefix: i64, unix_seconds: i64, divisor: i64) -> Result<i32, TimeFrameError> {
    if unix_seconds < 0 {
        return Err(TimeFrameError::NegativeUnixTime);
    }
    let encoded = prefix + unix_seconds / divisor;
    if encoded >= prefix + 100_000_000 || encoded > i64::from(i32::MAX) {
        return Err(TimeFrameError::OutOfRange(encoded));
    }
    Ok(encoded as i32)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn requested_examples_match() {
        // Reverse the examples into seconds to pin the agreed decimal representation.
        assert_eq!(five_minute(5_954_061 * 300).unwrap(), 105_954_061);
        assert_eq!(daily(54_123 * 86_400).unwrap(), 200_054_123);
    }

    #[test]
    fn hour_range_has_twelve_buckets() {
        let unix_seconds = 1_800_123_456;
        let (start, end) = hour_five_minute_range(unix_seconds).unwrap();
        assert_eq!(end - start, 11);
    }
}
