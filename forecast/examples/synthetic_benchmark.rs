//! Accuracy of the engine on synthetic sales with a known demand (src/synthetic.rs).
//!
//! Runs the engine the way the browser will: fit on the first `--fit-days`, then one `update` per
//! day, refitting the series whose trigger fired on the last 400 days. Every 7 days it forecasts 28
//! days ahead and scores each forecast against the true demand (what customers wanted, sold or not),
//! next to two baselines fed the same recorded sales — the seasonal naive and a 28-day mean — and the
//! oracle (the expected demand itself: the noise floor no forecast can beat).
//!
//!   cargo run --release --example synthetic_benchmark -- [--days 760] [--fit-days 400]
//!     [--products-per-category 14] [--seed 20240101]

use std::collections::HashMap;
use std::time::Instant;

use genix_forecast::engine::{self, DayBatch, EngineConfig, EngineState, Model, RefitReason, Scenario, SeriesBatch, SeriesKey};
use genix_forecast::synthetic::{ProductKind, SyntheticConfig, SyntheticDataset, SyntheticSeries, generate};

/// Days ahead each forecast covers, and days between forecasts.
const FORECAST_HORIZON: usize = 28;
const FORECAST_EVERY_DAYS: usize = 7;
/// History a refit gets: the longest the browser keeps (one frame read).
const REFIT_WINDOW_DAYS: usize = 400;
/// Horizon buckets reported: days 1–7 and 8–28.
const BUCKETS: [(usize, usize, &str); 2] = [(1, 7, "1-7 days"), (8, 28, "8-28 days")];

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
enum Method {
    Engine,
    SeasonalNaive,
    Mean28,
    Oracle,
}

const METHODS: [Method; 4] = [Method::Engine, Method::SeasonalNaive, Method::Mean28, Method::Oracle];

/// Σ|forecast − demand|, Σ(forecast − demand) and Σ demand of one group.
#[derive(Default, Clone, Copy)]
struct ErrorSums {
    absolute: f64,
    signed: f64,
    demand: f64,
}

impl ErrorSums {
    fn wape(&self) -> f64 {
        self.absolute / self.demand
    }
    fn bias(&self) -> f64 {
        self.signed / self.demand
    }
}

/// Sums by (product kind or None for all, horizon bucket, method).
type Scoreboard = HashMap<(Option<ProductKind>, usize, Method), ErrorSums>;

fn main() {
    let arguments: Vec<String> = std::env::args().collect();
    let argument = |name: &str, default: usize| -> usize {
        arguments.iter().position(|argument| argument == name).and_then(|index| arguments.get(index + 1)).map_or(default, |value| value.parse().expect("a number"))
    };
    let config = SyntheticConfig {
        days: argument("--days", 760),
        products_per_category: argument("--products-per-category", 14) as i32,
        seed: argument("--seed", 20240101) as u64,
        ..SyntheticConfig::default()
    };
    let fit_days = argument("--fit-days", 400);
    assert!(fit_days >= 60 && fit_days + FORECAST_HORIZON < config.days, "--fit-days must leave at least {FORECAST_HORIZON} days to score");

    let dataset = generate(&config);
    println!("{} series ({} sites × {} categories × {} products), {} days, fit on {fit_days}, seed {}", dataset.series.len(), config.sites, config.categories, config.products_per_category, config.days, config.seed);

    let started = Instant::now();
    let mut state = engine::create(dataset.start_day + fit_days as i32 - 1, EngineConfig::default()).unwrap();
    engine::set_calendar(&mut state, dataset.calendar.clone());
    let all_series: Vec<usize> = (0..dataset.series.len()).collect();
    refit_window(&mut state, &dataset, &all_series, fit_days);
    let fit_seconds = started.elapsed().as_secs_f64();
    describe_models(&state, "after the fit");

    print_hidden_demand(&dataset, fit_days);

    let mut scoreboard = Scoreboard::new();
    let mut refits_by_reason: HashMap<String, usize> = HashMap::new();
    let started = Instant::now();
    for day_index in fit_days..config.days {
        // The state holds days up to day_index − 1: forecast from there.
        if (day_index - fit_days).is_multiple_of(FORECAST_EVERY_DAYS) && day_index + FORECAST_HORIZON <= config.days {
            score_forecasts(&state, &dataset, day_index, &mut scoreboard);
        }
        let day_values = |field: fn(&SyntheticSeries) -> &Vec<f64>| -> Vec<f64> { dataset.series.iter().map(|series| field(series)[day_index]).collect() };
        let (sales, closing_stock, inflow) = (day_values(|series| &series.sales), day_values(|series| &series.closing_stock), day_values(|series| &series.inflow));
        engine::update(&mut state, &DayBatch { first_day: dataset.start_day + day_index as i32, sales: &sales, closing_stock: &closing_stock, inflow: &inflow }).unwrap();

        let pending = engine::pending_refits(&state);
        if !pending.is_empty() {
            for (position, reason) in &pending {
                let kind = dataset.series[*position as usize].kind;
                *refits_by_reason.entry(format!("{} {kind:?}", reason_name(*reason))).or_default() += 1;
            }
            let positions: Vec<usize> = pending.iter().map(|(position, _)| *position as usize).collect();
            refit_window(&mut state, &dataset, &positions, day_index + 1);
        }
    }
    let rolling_seconds = started.elapsed().as_secs_f64();
    describe_models(&state, "at the end");
    let mut refit_counts: Vec<(String, usize)> = refits_by_reason.into_iter().collect();
    refit_counts.sort();
    println!("refits by trigger and kind:");
    for (trigger_and_kind, count) in refit_counts {
        println!("  {trigger_and_kind:<40} {count}");
    }
    println!("time: fit {fit_seconds:.2}s, {} daily updates with refits {rolling_seconds:.2}s\n", config.days - fit_days);

    print_scoreboard(&scoreboard);
}

/// Refits `positions` (state positions equal dataset indexes: the first fit took every series in
/// order) on the last REFIT_WINDOW_DAYS of the days before `end_day_index`.
fn refit_window(state: &mut EngineState, dataset: &SyntheticDataset, positions: &[usize], end_day_index: usize) {
    let first_day_index = end_day_index.saturating_sub(REFIT_WINDOW_DAYS);
    let picked: Vec<&SyntheticSeries> = positions.iter().map(|&position| &dataset.series[position]).collect();
    let keys: Vec<SeriesKey> = picked.iter().map(|series| series.key).collect();
    let initial_stock: Vec<f64> = picked.iter().map(|series| if first_day_index == 0 { series.initial_stock } else { series.closing_stock[first_day_index - 1] }).collect();
    let window = |field: fn(&SyntheticSeries) -> &Vec<f64>| -> Vec<f64> { picked.iter().flat_map(|series| field(series)[first_day_index..end_day_index].iter().copied()).collect() };
    let (sales, closing_stock, inflow) = (window(|series| &series.sales), window(|series| &series.closing_stock), window(|series| &series.inflow));
    let batch = SeriesBatch { start_day: dataset.start_day + first_day_index as i32, keys: &keys, initial_stock: &initial_stock, sales: &sales, closing_stock: &closing_stock, inflow: &inflow };
    engine::refit(state, &batch).unwrap();
}

/// Forecasts FORECAST_HORIZON days from `origin_day_index` (the first day the state has not seen)
/// with every method and adds the errors against the true demand.
fn score_forecasts(state: &EngineState, dataset: &SyntheticDataset, origin_day_index: usize, scoreboard: &mut Scoreboard) {
    let output = engine::forecast(state, FORECAST_HORIZON as u32).unwrap();
    assert_eq!(output.first_day, dataset.start_day + origin_day_index as i32);
    for (series_index, series) in dataset.series.iter().enumerate() {
        let last_week = &series.sales[origin_day_index - 7..origin_day_index];
        let mean_28 = series.sales[origin_day_index - 28..origin_day_index].iter().sum::<f64>() / 28.0;
        for days_ahead in 1..=FORECAST_HORIZON {
            let target_index = origin_day_index + days_ahead - 1;
            let bucket = BUCKETS.iter().position(|(from, to, _)| (*from..=*to).contains(&days_ahead)).unwrap();
            let demand = series.demand[target_index];
            for method in METHODS {
                let forecast = match method {
                    Method::Engine => output.values[series_index * FORECAST_HORIZON + days_ahead - 1],
                    // The same weekday of the last recorded week.
                    Method::SeasonalNaive => last_week[(days_ahead - 1) % 7],
                    Method::Mean28 => mean_28,
                    Method::Oracle => series.expected_demand[target_index],
                };
                assert!(forecast.is_finite(), "{method:?} forecast is not finite for {:?}", series.key);
                for kind in [Some(series.kind), None] {
                    let sums = scoreboard.entry((kind, bucket, method)).or_default();
                    sums.absolute += (forecast - demand).abs();
                    sums.signed += forecast - demand;
                    sums.demand += demand;
                }
            }
        }
    }
}

fn describe_models(state: &EngineState, moment: &str) {
    let holt_winters = state.series.iter().filter(|series| matches!(series.model, Model::HoltWinters { .. })).count();
    let in_scenario = |scenario: Scenario| state.series.iter().filter(|series| series.fitted_scenario == scenario).count();
    println!(
        "{moment}: {holt_winters} Holt-Winters, {} TSB; scenarios A {} / B {} / C {}",
        state.series.len() - holt_winters,
        in_scenario(Scenario::A),
        in_scenario(Scenario::B),
        in_scenario(Scenario::C)
    );
}

/// Over the scored days, per kind: the demand the shelf could not serve, and the days the ERP cannot
/// read at all (empty shelf, nothing received, nothing sold) — no forecast can learn demand there.
fn print_hidden_demand(dataset: &SyntheticDataset, fit_days: usize) {
    println!("{:<18} {:>12} {:>18}", "kind", "lost sales", "empty-shelf days");
    for kind in ProductKind::ALL {
        let (mut sold, mut wanted, mut empty_days, mut days) = (0.0, 0.0, 0, 0);
        for series in dataset.series.iter().filter(|series| series.kind == kind) {
            for day_index in fit_days..dataset.days {
                let opening = series.closing_stock[day_index - 1];
                sold += series.sales[day_index];
                wanted += series.demand[day_index];
                days += 1;
                if kind != ProductKind::MadeToOrder && opening + series.inflow[day_index] == 0.0 && series.sales[day_index] == 0.0 {
                    empty_days += 1;
                }
            }
        }
        println!("{:<18} {:>11.1}% {:>17.1}%", format!("{kind:?}"), 100.0 * (1.0 - sold / wanted), 100.0 * empty_days as f64 / days as f64);
    }
    println!();
}

fn reason_name(reason: RefitReason) -> &'static str {
    match reason {
        RefitReason::Scenario => "scenario",
        RefitReason::Calendar => "calendar",
        RefitReason::ModelSwitch => "model switch",
        RefitReason::WorseThanNaive => "worse than naive",
        RefitReason::Period => "period",
    }
}

fn print_scoreboard(scoreboard: &Scoreboard) {
    let kinds: Vec<Option<ProductKind>> = ProductKind::ALL.iter().copied().map(Some).chain([None]).collect();
    for (bucket, (_, _, bucket_name)) in BUCKETS.iter().enumerate() {
        println!("horizon {bucket_name}: WAPE against the true demand (bias in parentheses; oracle = noise floor)");
        println!("{:<18} {:>16} {:>16} {:>16} {:>16}", "kind", "engine", "seasonal naive", "28-day mean", "oracle");
        for kind in &kinds {
            let cells: Vec<String> = METHODS
                .iter()
                .map(|method| {
                    let sums = scoreboard[&(*kind, bucket, *method)];
                    format!("{:.3} ({:+.3})", sums.wape(), sums.bias())
                })
                .collect();
            let kind_name = kind.map_or("ALL".to_string(), |kind| format!("{kind:?}"));
            println!("{kind_name:<18} {:>16} {:>16} {:>16} {:>16}", cells[0], cells[1], cells[2], cells[3]);
        }
        println!();
    }
}
