//! Sales forecasting engine (plan/SALES_FORECAST_PLAN.md): per site and product, daily sales and
//! stock in, daily forecasts out. Damped Holt-Winters for products that sell most days, TSB for
//! intermittent ones, with yearly and calendar factors once a product has the history (scenario B).
//! The math is plain Rust, tested natively; `wasm` only binds it for the browser.

pub mod calendar;
pub mod censoring;
pub mod engine;
pub mod holt_winters;
pub mod nelder_mead;
pub mod synthetic;
pub mod tsb;
pub mod weekly_index;
pub mod yearly_profile;
mod wasm;

use censoring::DayCensoring;
use yearly_profile::YEAR_WEEKS;

/// The weekday of a UnixDay (days since 1970-01-01, a Thursday), Monday = 0.
pub fn weekday(day: i32) -> usize {
    (day + 3).rem_euclid(7) as usize
}

/// The UnixDay of a civil date (Howard Hinnant's days_from_civil).
pub fn days_from_civil(year: i32, month: u32, day: u32) -> i32 {
    let year = if month <= 2 { year - 1 } else { year };
    let era = year.div_euclid(400);
    let year_of_era = year - era * 400;
    let month_from_march = (month as i32 + 9) % 12;
    let day_of_year = (153 * month_from_march + 2) / 5 + day as i32 - 1;
    let day_of_era = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year;
    era * 146097 + day_of_era - 719468
}

/// The week-of-year bucket of a UnixDay: day of year / 7, with the last day or two of a year folded
/// into the last week.
pub fn week_of_year(day: i32) -> usize {
    let mut year = 1970 + day.div_euclid(365);
    while days_from_civil(year, 1, 1) > day {
        year -= 1;
    }
    while days_from_civil(year + 1, 1, 1) <= day {
        year += 1;
    }
    (((day - days_from_civil(year, 1, 1)) / 7) as usize).min(YEAR_WEEKS - 1)
}

/// Days at the start of a history the models settle on before their errors count in the fit.
pub(crate) const WARMUP_DAYS: usize = 7;

/// Half-life, in days, of a one-step error's weight in the parameter fit. Set on the synthetic
/// benchmark: 45 to 91 days all cut the dying products' error by 10–15% at no cost elsewhere; 60 is
/// the middle, less tuned to synthetic data than the best one.
const FIT_ERROR_HALF_LIFE_DAYS: f64 = 60.0;

/// The weight of day `day_index`'s squared error in a fit over `days` days: recent days weigh more
/// (discounted least squares). Fitted on equal weights, a long stable history picks slow smoothing
/// that a product whose demand is changing now — fading out, taking off — cannot follow, refit after
/// refit.
pub(crate) fn fit_error_weight(day_index: usize, days: usize) -> f64 {
    0.5f64.powf((days - 1 - day_index) as f64 / FIT_ERROR_HALF_LIFE_DAYS)
}

/// One series' daily history as the models read it.
pub struct SeriesHistory<'a> {
    /// UnixDay of the first value.
    pub start_day: i32,
    /// Sales as the model sees them: divided by the yearly factor once the series has one.
    pub sales: &'a [f64],
    /// Which days the model learns from: event days are skipped once the series has calendar factors.
    pub censoring: &'a [DayCensoring],
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn weekday_is_monday_first() {
        assert_eq!(weekday(0), 3); // 1970-01-01, Thursday
        assert_eq!(weekday(19723), 0); // 2024-01-01, Monday
        assert_eq!(weekday(20414), 5); // 2025-11-22, Saturday
    }

    #[test]
    fn civil_dates_and_weeks_of_year() {
        assert_eq!(days_from_civil(1970, 1, 1), 0);
        assert_eq!(days_from_civil(2024, 1, 1), 19723);
        assert_eq!(days_from_civil(2024, 3, 1), 19723 + 60); // 2024 is a leap year
        assert_eq!(week_of_year(19723), 0);
        assert_eq!(week_of_year(19723 + 7), 1);
        assert_eq!(week_of_year(19723 + 365), 51); // 2024-12-31, day 366 of a leap year
        assert_eq!(week_of_year(19722), 51); // 2023-12-31
        assert_eq!(week_of_year(19723 + 366), 0); // 2025-01-01
    }
}
