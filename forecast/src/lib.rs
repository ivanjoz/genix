//! Sales forecasting engine (plan/SALES_FORECAST_PLAN.md): per site and product, daily sales and
//! stock in, daily forecasts out. Damped Holt-Winters for products that sell most days, TSB for
//! intermittent ones. The math is plain Rust, tested natively; `wasm` only binds it for the browser.

pub mod censoring;
pub mod engine;
pub mod holt_winters;
pub mod nelder_mead;
pub mod tsb;
pub mod weekly_index;
mod wasm;

use censoring::DayCensoring;

/// The weekday of a UnixDay (days since 1970-01-01, a Thursday), Monday = 0.
pub fn weekday(day: i32) -> usize {
    (day + 3).rem_euclid(7) as usize
}

/// Days at the start of a history the models settle on before their errors count in the fit.
pub(crate) const WARMUP_DAYS: usize = 7;

/// One series' daily history as the models read it.
pub struct SeriesHistory<'a> {
    /// UnixDay of the first value.
    pub start_day: i32,
    pub sales: &'a [f64],
    pub censoring: &'a [DayCensoring],
}

#[cfg(test)]
mod tests {
    use super::weekday;

    #[test]
    fn weekday_is_monday_first() {
        assert_eq!(weekday(0), 3); // 1970-01-01, Thursday
        assert_eq!(weekday(19723), 0); // 2024-01-01, Monday
        assert_eq!(weekday(20414), 5); // 2025-11-22, Saturday
    }
}
