use serde::{Deserialize, Serialize};

use crate::censoring::DayCensoring;
use crate::nelder_mead::{minimize, to_bounds};
use crate::weekly_index::WeeklyIndex;
use crate::{SeriesHistory, WARMUP_DAYS, fit_error_weight, weekday};

/// Smoothing of the demand size (α_z) and of the probability of a sale (α_p).
#[derive(Serialize, Deserialize, Clone, Copy, Debug, PartialEq)]
pub struct TsbParams {
    pub alpha_size: f64,
    pub alpha_probability: f64,
}

/// Teunter-Syntetos-Babai for intermittent demand (plan §4.2): `probability` that a day sells, and
/// the `size` of a day that does. The probability decays on every zero day, so an obsolete product
/// fades out instead of keeping its last forecast (Croston's flaw).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct TsbState {
    pub probability: f64,
    pub size: f64,
}

impl TsbState {
    /// The forecast for a day on `weekday`, shaped by the category's weekly index (an intermittent
    /// product is too sparse for its own).
    pub fn forecast(&self, category_week: &WeeklyIndex, weekday: usize) -> f64 {
        self.probability * self.size * category_week[weekday]
    }

    /// Incorporates one day by its censoring. A partially censored day that sold proves the day
    /// would sell (probability up) but its size only bounds demand from below, so the size moves
    /// only upward. A fully censored day teaches nothing (a partial day always sold: censoring.rs).
    pub fn step(&mut self, params: &TsbParams, sales: f64, censoring: DayCensoring) {
        match censoring {
            DayCensoring::Normal if sales > 0.0 => {
                self.size += params.alpha_size * (sales - self.size);
                self.probability += params.alpha_probability * (1.0 - self.probability);
            }
            DayCensoring::Normal => self.probability -= params.alpha_probability * self.probability,
            DayCensoring::Partial => {
                if sales > self.size {
                    self.size += params.alpha_size * (sales - self.size);
                }
                self.probability += params.alpha_probability * (1.0 - self.probability);
            }
            DayCensoring::Full => {}
        }
    }
}

/// The state before the first day: the probability and size of the whole history's uncensored days.
fn initial_state(history: &SeriesHistory) -> TsbState {
    let (mut observed_days, mut sale_days, mut sales_sum) = (0, 0, 0.0);
    for (&sales, &censoring) in history.sales.iter().zip(history.censoring) {
        if censoring == DayCensoring::Normal {
            observed_days += 1;
            if sales > 0.0 {
                sale_days += 1;
                sales_sum += sales;
            }
        }
    }
    if sale_days == 0 {
        return TsbState { probability: 0.0, size: 0.0 };
    }
    TsbState { probability: sale_days as f64 / observed_days as f64, size: sales_sum / sale_days as f64 }
}

/// Runs the model over the whole history, returning the final state and the sum of squared
/// one-step errors on the uncensored days after the warm-up. `one_step` collects every day's
/// one-step forecast when given.
fn run(history: &SeriesHistory, params: &TsbParams, initial: &TsbState, category_week: &WeeklyIndex, mut one_step: Option<&mut Vec<f64>>) -> (TsbState, f64) {
    let mut state = initial.clone();
    let mut squared_error_sum = 0.0;
    for (day_index, (&sales, &censoring)) in history.sales.iter().zip(history.censoring).enumerate() {
        let expected = state.forecast(category_week, weekday(history.start_day + day_index as i32));
        if let Some(one_step) = one_step.as_deref_mut() {
            one_step.push(expected);
        }
        if censoring == DayCensoring::Normal && day_index >= WARMUP_DAYS {
            squared_error_sum += fit_error_weight(day_index, history.sales.len()) * (sales - expected).powi(2);
        }
        state.step(params, sales, censoring);
    }
    (state, squared_error_sum)
}

/// Optimizes the parameters over the history and returns them with the state after the last day and
/// each day's one-step forecast under them.
pub fn fit(history: &SeriesHistory, category_week: &WeeklyIndex, max_evaluations: u32) -> (TsbParams, TsbState, Vec<f64>) {
    let initial = initial_state(history);
    let params_of = |coordinates: &[f64; 2]| TsbParams {
        alpha_size: to_bounds(coordinates[0], 0.01, 0.5),
        alpha_probability: to_bounds(coordinates[1], 0.01, 0.5),
    };
    let best = minimize(|coordinates| run(history, &params_of(coordinates), &initial, category_week, None).1, [0.0; 2], 1.0, max_evaluations);
    let params = params_of(&best);
    let mut one_step = Vec::with_capacity(history.sales.len());
    let (state, _) = run(history, &params, &initial, category_week, Some(&mut one_step));
    (params, state, one_step)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::weekly_index::FLAT_WEEK;

    #[test]
    fn forecasts_the_mean_rate_of_intermittent_demand() {
        // Sells 6 units every fourth day: a daily mean of 1.5.
        let sales: Vec<f64> = (0..120).map(|day_index| if day_index % 4 == 0 { 6.0 } else { 0.0 }).collect();
        let censoring = vec![DayCensoring::Normal; sales.len()];
        let history = SeriesHistory { start_day: 19723, sales: &sales, censoring: &censoring };
        let (_, state, _) = fit(&history, &FLAT_WEEK, 200);
        let daily_forecast = state.forecast(&FLAT_WEEK, 0);
        assert!((daily_forecast - 1.5).abs() < 0.4, "{daily_forecast}");
    }

    #[test]
    fn an_obsolete_product_fades_out() {
        let mut sales: Vec<f64> = (0..120).map(|day_index| if day_index % 3 == 0 { 3.0 } else { 0.0 }).collect();
        sales[60..].fill(0.0);
        let censoring = vec![DayCensoring::Normal; sales.len()];
        let history = SeriesHistory { start_day: 19723, sales: &sales, censoring: &censoring };
        let (_, state, _) = fit(&history, &FLAT_WEEK, 200);
        assert!(state.forecast(&FLAT_WEEK, 0) < 0.2);
    }
}
