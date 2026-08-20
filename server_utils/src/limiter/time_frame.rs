//! Decimal period keys shared by aggregation and Scylla persistence.

use thiserror::Error;

pub const FIVE_MINUTE_PREFIX: i64 = 100_000_000;
pub const DAILY_PREFIX: i64 = 200_000_000;
pub const FIVE_MINUTE_SECONDS: i64 = 300;
pub const DAY_SECONDS: i64 = 86_400;
/// A "day" here is the Lima business day (UTC-5), not a UTC day: the reports built on these frames
/// are read in that timezone, and the project's UnixDay convention is local everywhere else.
///
/// A fixed offset rather than a zone lookup, because Peru has no DST and because the two processes
/// that must agree on the boundary — this daemon and the Go backend — can run on hosts in any
/// timezone, including a Lambda that is always UTC. Both sides carry the same constant.
///
/// Five-minute frames are deliberately left on the raw unix division: they round-trip back to a
/// wall-clock time, and the offset is an exact multiple of 300 anyway, so shifting them would
/// renumber every bucket without moving a single boundary.
pub const DAY_ZONE_OFFSET_SECONDS: i64 = -5 * 60 * 60;

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
    encode(DAILY_PREFIX, local_day_seconds(unix_seconds)?, DAY_SECONDS)
}

/// The local business day as a UnixDay, which is the day both the daily frame and the daily quota
/// window are bucketed by. Public because the quota window has to be counted on the same day the
/// usage row it is recovered from was written on: on the raw UTC division the two disagree for the
/// last five hours of every day, and the daily cap would reset at 19:00 local time.
pub fn local_unix_day(unix_seconds: i64) -> Result<i64, TimeFrameError> {
    Ok(local_day_seconds(unix_seconds)? / DAY_SECONDS)
}

/// Shifts unix seconds into the local day the frames are bucketed by.
fn local_day_seconds(unix_seconds: i64) -> Result<i64, TimeFrameError> {
    unix_seconds
        .checked_add(DAY_ZONE_OFFSET_SECONDS)
        .filter(|shifted| *shifted >= 0)
        .ok_or(TimeFrameError::NegativeUnixTime)
}

pub fn month_start_day(unix_seconds: i64) -> Result<i16, TimeFrameError> {
    if unix_seconds < 0 {
        return Err(TimeFrameError::NegativeUnixTime);
    }
    let unix_day = local_unix_day(unix_seconds)?;
    // Civil-date conversion identifies the day of month without adding a date dependency.
    let shifted_day = unix_day + 719_468;
    let era = shifted_day.div_euclid(146_097);
    let day_of_era = shifted_day - era * 146_097;
    let year_of_era =
        (day_of_era - day_of_era / 1_460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let month_part = (5 * day_of_year + 2) / 153;
    let day_of_month = day_of_year - (153 * month_part + 2) / 5 + 1;
    i16::try_from(unix_day - day_of_month + 1).map_err(|_| TimeFrameError::OutOfRange(unix_day))
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
        // Reverse the examples into seconds to pin the agreed decimal representation. The daily
        // example is stated at local midnight, which is the boundary the frame counts from.
        assert_eq!(five_minute(5_954_061 * 300).unwrap(), 105_954_061);
        assert_eq!(
            daily(54_123 * 86_400 - DAY_ZONE_OFFSET_SECONDS).unwrap(),
            200_054_123
        );
    }

    // The whole point of the offset: 21:00 in Lima is already tomorrow in UTC, and the frame has to
    // stay on today. This is what silently emptied the reports every evening when it was UTC.
    #[test]
    fn the_evening_stays_on_the_local_day() {
        let local_midnight = 20_683 * 86_400 - DAY_ZONE_OFFSET_SECONDS;
        assert_eq!(daily(local_midnight).unwrap(), 200_020_683);
        // 23:59:59 local — past midnight UTC, still the same local day.
        assert_eq!(daily(local_midnight + 86_399).unwrap(), 200_020_683);
        // 00:00:00 local the next day.
        assert_eq!(daily(local_midnight + 86_400).unwrap(), 200_020_684);
    }

    #[test]
    fn hour_range_has_twelve_buckets() {
        let unix_seconds = 1_800_123_456;
        let (start, end) = hour_five_minute_range(unix_seconds).unwrap();
        assert_eq!(end - start, 11);
    }

    #[test]
    fn month_start_uses_local_unix_days() {
        // 2026-08-16 and 2024-03-15 (leap year), stated at local noon so the offset cannot move
        // them off their own day, map to the first day of their month.
        assert_eq!(month_start_day(1_786_838_400 + 43_200).unwrap(), 20_666);
        assert_eq!(month_start_day(1_710_460_800 + 43_200).unwrap(), 19_783);
    }
}
