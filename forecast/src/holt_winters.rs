use serde::{Deserialize, Serialize};

use crate::censoring::DayCensoring;
use crate::nelder_mead::{minimize, to_bounds};
use crate::weekly_index::{WeeklyIndex, normalize};
use crate::{SeriesHistory, WARMUP_DAYS, fit_error_weight, weekday};

/// Smoothing of the level (α), trend (β) and weekly indices (γ), and the trend's damping (φ).
#[derive(Serialize, Deserialize, Clone, Copy, Debug, PartialEq)]
pub struct HwParams {
    pub alpha: f64,
    pub beta: f64,
    pub gamma: f64,
    pub phi: f64,
}

/// Damped Holt-Winters with multiplicative weekly seasonality (plan §4.1), after the last day
/// incorporated.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct HwState {
    pub level: f64,
    pub trend: f64,
    pub season: WeeklyIndex,
}

// Multiplicative seasonality divides by the level and by the indices: the floors keep a run of zero
// days from driving either to 0, which would blow up the next update (plan §9).
const LEVEL_FLOOR: f64 = 1e-3;
const SEASON_FLOOR: f64 = 0.05;
// The damping's lower bound; the upper one is EngineConfig.max_damping.
pub const DAMPING_LOW: f64 = 0.85;

impl HwState {
    /// The forecast `horizon` days after the last day incorporated, which falls on `weekday`.
    pub fn forecast(&self, params: &HwParams, horizon: u32, weekday: usize) -> f64 {
        let mut damped_trend_sum = 0.0;
        let mut damping_power = 1.0;
        for _ in 0..horizon {
            damping_power *= params.phi;
            damped_trend_sum += damping_power;
        }
        ((self.level + damped_trend_sum * self.trend) * self.season[weekday]).max(0.0)
    }

    fn update(&mut self, params: &HwParams, weekday: usize, sales: f64) {
        let previous_level = self.level;
        let damped_trend = params.phi * self.trend;
        self.level = (params.alpha * sales / self.season[weekday] + (1.0 - params.alpha) * (previous_level + damped_trend)).max(LEVEL_FLOOR);
        self.trend = params.beta * (self.level - previous_level) + (1.0 - params.beta) * damped_trend;
        self.season[weekday] = (params.gamma * sales / self.level + (1.0 - params.gamma) * self.season[weekday]).max(SEASON_FLOOR);
        self.season = normalize(self.season);
    }

    /// A day without a demand signal: the level follows its damped trend and nothing is learned.
    fn propagate(&mut self, params: &HwParams) {
        self.level = (self.level + params.phi * self.trend).max(LEVEL_FLOOR);
        self.trend *= params.phi;
    }

    /// Incorporates one day by its censoring, returning the one-step forecast it was compared with.
    /// A partially censored day teaches only when it sold more than expected: then demand was at
    /// least that high.
    pub fn step(&mut self, params: &HwParams, weekday: usize, sales: f64, censoring: DayCensoring) -> f64 {
        let expected = self.forecast(params, 1, weekday);
        match censoring {
            DayCensoring::Normal => self.update(params, weekday, sales),
            DayCensoring::Partial if sales > expected => self.update(params, weekday, sales),
            _ => self.propagate(params),
        }
        expected
    }
}

/// The state before the first day: the level is the deseasonalized mean of the uncensored days of
/// the first four weeks (all of them if there are fewer), no trend, and the given weekly shape.
fn initial_state(history: &SeriesHistory, season: WeeklyIndex) -> HwState {
    let (mut deseasonalized_sum, mut observed_days) = (0.0, 0);
    for (day_index, (&sales, &censoring)) in history.sales.iter().zip(history.censoring).enumerate() {
        if censoring != DayCensoring::Normal {
            continue;
        }
        if day_index >= 28 && observed_days > 0 {
            break;
        }
        deseasonalized_sum += sales / season[weekday(history.start_day + day_index as i32)];
        observed_days += 1;
    }
    let level = if observed_days == 0 { LEVEL_FLOOR } else { (deseasonalized_sum / observed_days as f64).max(LEVEL_FLOOR) };
    HwState { level, trend: 0.0, season }
}

/// Runs the model over the whole history, returning the final state and the sum of squared
/// one-step errors on the uncensored days after the warm-up. `one_step` collects every day's
/// one-step forecast when given.
fn run(history: &SeriesHistory, params: &HwParams, initial: &HwState, mut one_step: Option<&mut Vec<f64>>) -> (HwState, f64) {
    let mut state = initial.clone();
    let mut squared_error_sum = 0.0;
    for (day_index, (&sales, &censoring)) in history.sales.iter().zip(history.censoring).enumerate() {
        let expected = state.step(params, weekday(history.start_day + day_index as i32), sales, censoring);
        if let Some(one_step) = one_step.as_deref_mut() {
            one_step.push(expected);
        }
        if censoring == DayCensoring::Normal && day_index >= WARMUP_DAYS {
            squared_error_sum += fit_error_weight(day_index, history.sales.len()) * (sales - expected).powi(2);
        }
    }
    (state, squared_error_sum)
}

/// Optimizes the parameters over the history (plan §5.1, the expensive layer) and returns them with
/// the state they leave after the last day and each day's one-step forecast under them.
pub fn fit(history: &SeriesHistory, initial_season: WeeklyIndex, max_damping: f64, max_evaluations: u32) -> (HwParams, HwState, Vec<f64>) {
    let initial = initial_state(history, initial_season);
    let params_of = |coordinates: &[f64; 4]| HwParams {
        alpha: to_bounds(coordinates[0], 0.01, 0.6),
        beta: to_bounds(coordinates[1], 0.001, 0.2),
        gamma: to_bounds(coordinates[2], 0.01, 0.5),
        phi: to_bounds(coordinates[3], DAMPING_LOW, max_damping),
    };
    let best = minimize(|coordinates| run(history, &params_of(coordinates), &initial, None).1, [0.0; 4], 1.0, max_evaluations);
    let params = params_of(&best);
    let mut one_step = Vec::with_capacity(history.sales.len());
    let (state, _) = run(history, &params, &initial, Some(&mut one_step));
    (params, state, one_step)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::weekly_index::FLAT_WEEK;

    #[test]
    fn learns_a_weekly_shape() {
        let shape = [1.0, 1.0, 1.0, 1.0, 1.5, 2.0, 0.5];
        // 2024-01-01 is a Monday: day_index 0 is weekday 0.
        let start_day = 19723;
        let sales: Vec<f64> = (0..112).map(|day_index| 10.0 * shape[day_index % 7]).collect();
        let censoring = vec![DayCensoring::Normal; sales.len()];
        let history = SeriesHistory { start_day, sales: &sales, censoring: &censoring };
        let (params, state, _) = fit(&history, FLAT_WEEK, 0.9, 300);
        let mean = shape.iter().sum::<f64>() / 7.0;
        for horizon in 1..=7u32 {
            let target_weekday = weekday(start_day + 111 + horizon as i32);
            let expected = 10.0 * shape[target_weekday];
            let forecast = state.forecast(&params, horizon, target_weekday);
            assert!((forecast - expected).abs() < 0.05 * 10.0 * mean, "h{horizon}: {forecast} vs {expected}");
        }
    }

    #[test]
    fn censored_days_do_not_drag_the_level_down() {
        let start_day = 19723;
        let mut sales = vec![10.0; 84];
        let mut censoring = vec![DayCensoring::Normal; 84];
        // A three-week stockout at the end: zero sales that say nothing about demand.
        for day_index in 63..84 {
            sales[day_index] = 0.0;
            censoring[day_index] = DayCensoring::Full;
        }
        let history = SeriesHistory { start_day, sales: &sales, censoring: &censoring };
        let (params, state, _) = fit(&history, FLAT_WEEK, 0.9, 300);
        assert!(state.forecast(&params, 1, weekday(start_day + 84)) > 9.0);
    }
}
