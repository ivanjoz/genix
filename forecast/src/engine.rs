use std::collections::HashMap;

use serde::{Deserialize, Serialize};

use crate::calendar::{Calendar, EVENT_SHRINK, EVENT_TYPES, EventPool, EventSums};
use crate::censoring::{DayCensoring, classify_day, is_stock_tracked};
use crate::holt_winters::{self, DAMPING_LOW, HwParams, HwState};
use crate::tsb::{self, TsbParams, TsbState};
use crate::weekly_index::{FLAT_WEEK, WeekdayTotals, WeeklyIndex, shrink};
use crate::yearly_profile::{YearBuckets, YearPool, YearProfile};
use crate::{SeriesHistory, week_of_year, weekday};

/// Format of the serialized EngineState. Pre-alpha: a change refits instead of migrating.
pub const STATE_VERSION: u16 = 4;
/// The site of a category's pooled factors over every site: the fallback of a thin site.
pub const ALL_SITES: i32 = -1;
/// Forecasts for 1 to this many days ahead are kept until their day arrives and scored (plan §5.4).
pub const TRACKED_HORIZONS: usize = 7;
/// The last days of a refit's history go through the daily `advance` instead of the optimization:
/// they seed the pending forecasts, the seasonal-naive week and the first online errors with days the
/// parameters never saw.
const REPLAYED_DAYS: usize = TRACKED_HORIZONS;
const MIN_HISTORY_DAYS: usize = 21;
const MAX_HORIZON: u32 = 366;
/// Half-life, in observed days, of the recent demand counters that choose the model.
const DEMAND_HALF_LIFE_DAYS: f64 = 56.0;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
#[serde(default, rename_all = "camelCase")]
pub struct EngineConfig {
    /// A closing stock at or below this marks the day as partially censored (plan §3.1). To be set
    /// per category with real data (plan §9).
    pub partial_censor_threshold: f64,
    /// Average days between sales (ADI) from which a product is intermittent and uses TSB.
    pub intermittent_adi: f64,
    /// k of the shrinkage weight n / (n + k) of a product's weekly shape, in weeks of history.
    pub shrink_weeks: f64,
    /// Sale days a site's category needs before its own weekly shape is used over the all-sites one.
    pub min_category_sale_days: u32,
    /// Upper bound of the Holt-Winters damping: kept low in scenario A, where 90 days don't justify
    /// trusting a trend.
    pub max_damping: f64,
    /// Cost evaluations of the parameter optimization, per product.
    pub max_evaluations: u32,
    /// Days after a refit when a series asks for the next one.
    pub refit_period_days: u32,
    /// Consecutive weeks of online error above the seasonal naive's that ask for a refit.
    pub worse_than_naive_weeks: u32,
    /// Consecutive weeks the demand pattern sits on the other model's side before a refit switches
    /// it (hysteresis, plan §3.2).
    pub model_switch_weeks: u32,
    /// Days of empty shelf (nothing on hand, nothing received, nothing sold) the forecast holds
    /// through; after them it halves every as many days. An empty shelf hides demand, so a short
    /// stockout keeps the forecast that drives the reorder; a shelf left empty for weeks means the
    /// product is no longer sold, and its forecast fades instead of freezing for good.
    pub empty_shelf_half_life_days: u32,
}

impl Default for EngineConfig {
    fn default() -> Self {
        EngineConfig {
            partial_censor_threshold: 0.0,
            intermittent_adi: 1.32,
            shrink_weeks: 8.0,
            min_category_sale_days: 28,
            max_damping: 0.9,
            max_evaluations: 200,
            refit_period_days: 28,
            worse_than_naive_weeks: 2,
            model_switch_weeks: 4,
            empty_shelf_half_life_days: 28,
        }
    }
}

impl EngineConfig {
    fn validate(&self) -> Result<(), String> {
        if !(self.max_damping > DAMPING_LOW && self.max_damping <= 0.98) {
            return Err(format!("maxDamping must be in ({DAMPING_LOW}, 0.98], got {}", self.max_damping));
        }
        if self.intermittent_adi < 1.0 {
            return Err(format!("intermittentAdi must be at least 1, got {}", self.intermittent_adi));
        }
        if self.shrink_weeks < 0.0 || self.partial_censor_threshold < 0.0 {
            return Err("shrinkWeeks and partialCensorThreshold cannot be negative".into());
        }
        if self.max_evaluations < 10 {
            return Err(format!("maxEvaluations must be at least 10, got {}", self.max_evaluations));
        }
        if self.refit_period_days == 0 || self.worse_than_naive_weeks == 0 || self.model_switch_weeks == 0 || self.empty_shelf_half_life_days == 0 {
            return Err("refitPeriodDays, worseThanNaiveWeeks, modelSwitchWeeks and emptyShelfHalfLifeDays must be at least 1".into());
        }
        Ok(())
    }
}

/// What a series forecasts: one product in one site, and the category its factors pool with.
#[derive(Serialize, Deserialize, Clone, Copy, Debug, PartialEq)]
pub struct SeriesKey {
    pub site_id: i32,
    pub product_id: i32,
    pub category_id: i32,
}

impl SeriesKey {
    /// A series is one site and product; its category may change between fits.
    fn is_same_series(&self, other: &SeriesKey) -> bool {
        self.site_id == other.site_id && self.product_id == other.product_id
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub enum Model {
    HoltWinters { params: HwParams, state: HwState },
    Tsb { params: TsbParams, state: TsbState },
}

/// The factor set a series' history supports (plan §6). A: the models alone. B and up: the
/// category's yearly profile and calendar factors multiply in (C's per-product factors come in P4).
#[derive(Serialize, Deserialize, Clone, Copy, Debug, PartialEq, Eq)]
pub enum Scenario {
    A,
    B,
    C,
}

impl Scenario {
    fn of_history_days(days: i32) -> Scenario {
        if days >= 730 {
            Scenario::C
        } else if days >= 180 {
            Scenario::B
        } else {
            Scenario::A
        }
    }

    fn of_history(first_active_day: Option<i32>, day: i32) -> Scenario {
        first_active_day.map_or(Scenario::A, |first_day| Scenario::of_history_days(day - first_day + 1))
    }
}

#[derive(Serialize, Deserialize, Clone, Copy, Debug, PartialEq, Eq)]
pub enum RefitReason {
    Scenario,
    /// An event was added, moved or removed on a day the series already learned from.
    Calendar,
    ModelSwitch,
    WorseThanNaive,
    Period,
}

/// Uncensored days and sale days, decayed by DEMAND_HALF_LIFE_DAYS: the recent ADI that picks the
/// model at a fit and drives the switch afterwards, so both read the same measure.
#[derive(Serialize, Deserialize, Clone, Copy, Debug, Default, PartialEq)]
pub struct RecentDemand {
    pub observed_days: f64,
    pub sale_days: f64,
}

impl RecentDemand {
    fn add(&mut self, sales: f64) {
        let decay = 0.5f64.powf(1.0 / DEMAND_HALF_LIFE_DAYS);
        self.observed_days = self.observed_days * decay + 1.0;
        self.sale_days = self.sale_days * decay + if sales > 0.0 { 1.0 } else { 0.0 };
    }

    /// Syntetos-Boylan (plan §3.2): sparse sales go to TSB.
    fn is_intermittent(&self, intermittent_adi: f64) -> bool {
        self.sale_days <= 0.0 || self.observed_days / self.sale_days >= intermittent_adi
    }
}

/// Running sums of one horizon's online errors, over uncensored days whose seasonal naive exists
/// (plan §5.4). Kept in f32: they feed ratios, not money.
#[derive(Serialize, Deserialize, Clone, Copy, Debug, Default, PartialEq)]
pub struct HorizonErrors {
    pub count: u32,
    pub actual_sum: f32,
    pub abs_error_sum: f32,
    pub error_sum: f32,
    pub squared_error_sum: f32,
    /// Σ |naive − actual|: the MASE scale, the seasonal naive's error on the same days.
    pub naive_abs_error_sum: f32,
}

impl HorizonErrors {
    fn add(&mut self, forecast: f32, actual: f32, naive: f32) {
        let error = forecast - actual;
        self.count += 1;
        self.actual_sum += actual;
        self.abs_error_sum += error.abs();
        self.error_sum += error;
        self.squared_error_sum += error * error;
        self.naive_abs_error_sum += (naive - actual).abs();
    }
}

/// What a series contributes to its category's pooled factors.
#[derive(Serialize, Deserialize, Clone, Debug, Default, PartialEq)]
pub struct SeriesStats {
    /// Uncensored days without events per weekday: the weekly shape.
    pub observed: WeekdayTotals,
    /// Uncensored days without events per week of the year: the yearly profile.
    pub year: YearBuckets,
    /// Uncensored event days by type: the calendar factors.
    pub events: EventSums,
}

/// The saved state of one series (plan §5.2).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct SeriesState {
    pub key: SeriesKey,
    /// False while the product's stock is not tracked: none of its days are censored.
    pub stock_tracked: bool,
    /// Closing stock of the last day: tomorrow's opening, which classifies its censoring.
    pub last_stock: f64,
    /// Consecutive fully censored days (empty shelf) up to the last day.
    pub empty_shelf_days: u32,
    /// First day with a sale or stock, which measures the history for the scenario.
    pub first_active_day: Option<i32>,
    /// The scenario of the last fit: from B the yearly and calendar factors apply.
    pub fitted_scenario: Scenario,
    pub stats: SeriesStats,
    pub recent_demand: RecentDemand,
    pub model: Model,
    /// `issued[age][h − 1]`: the forecast for `h` days ahead issued `age` days before the last day,
    /// NaN when none was. Those with `h > age` are still pending.
    pub issued: [[f32; TRACKED_HORIZONS]; TRACKED_HORIZONS],
    /// Sales of the last 7 days by weekday, NaN on a censored day: the seasonal naive forecast of
    /// the coming week, at every horizon up to 7.
    pub last_week_sales: [f32; 7],
    /// Online errors per horizon, 1 to 7 days ahead, carried across refits.
    pub errors: [HorizonErrors; TRACKED_HORIZONS],
    pub days_since_refit: u32,
    /// This week's squared errors of the model and of the seasonal naive, every horizon together.
    /// Squared, as the fit minimizes: on intermittent demand an absolute error favors the naive's
    /// zeros over any mean forecast and would ask for refits forever.
    pub week_squared_error: f32,
    pub week_naive_squared_error: f32,
    pub weeks_worse_than_naive: u32,
    pub weeks_on_other_model: u32,
    /// Set by the first trigger that fires; the series keeps forecasting until the caller refits it.
    pub refit_reason: Option<RefitReason>,
}

impl SeriesState {
    fn uses_factors(&self) -> bool {
        self.fitted_scenario != Scenario::A
    }

    /// The model's forecast, before the yearly and calendar factors.
    fn model_forecast(&self, days_ahead: u32, target_weekday: usize, category_week: &WeeklyIndex) -> f64 {
        match &self.model {
            Model::HoltWinters { params, state } => state.forecast(params, days_ahead, target_weekday),
            Model::Tsb { state, .. } => state.forecast(category_week, target_weekday),
        }
    }

    /// The forecast for `target_day`, `days_ahead` after the last day incorporated.
    fn forecast(&self, days_ahead: u32, target_day: i32, factors: &SeriesFactors, calendar: &Calendar, config: &EngineConfig) -> f64 {
        let multiplier = if self.uses_factors() { factors.yearly(target_day) * factors.event(target_day, calendar) } else { 1.0 };
        self.model_forecast(days_ahead, weekday(target_day), &factors.week) * multiplier * self.empty_shelf_fade(config)
    }

    /// 1 through the first `empty_shelf_half_life_days` of an empty shelf, then halving every as many
    /// days (EngineConfig.empty_shelf_half_life_days).
    fn empty_shelf_fade(&self, config: &EngineConfig) -> f64 {
        let half_life = config.empty_shelf_half_life_days as f64;
        0.5f64.powf((self.empty_shelf_days as f64 - half_life).max(0.0) / half_life)
    }

    /// Incorporates one day (plan §5.3): classify it, score the forecasts issued for it, update the
    /// counters and the model, issue the next week's forecasts, check the refit triggers.
    #[allow(clippy::too_many_arguments)]
    fn advance(&mut self, day: i32, sales: f64, closing_stock: f64, inflow: f64, config: &EngineConfig, factors: &SeriesFactors, calendar: &Calendar) {
        let day_weekday = weekday(day);
        if !self.stock_tracked && (inflow > 0.0 || closing_stock > 0.0) {
            self.stock_tracked = true;
        }
        let censoring = if self.stock_tracked {
            classify_day(self.last_stock, inflow, closing_stock, sales, config.partial_censor_threshold)
        } else {
            DayCensoring::Normal
        };
        if self.first_active_day.is_none() && (sales > 0.0 || inflow > 0.0 || closing_stock > 0.0) {
            self.first_active_day = Some(day);
        }
        self.empty_shelf_days = if censoring == DayCensoring::Full { self.empty_shelf_days + 1 } else { 0 };
        let event = calendar.event_on(day);
        let yearly = if self.uses_factors() { factors.yearly(day) } else { 1.0 };

        // Score today's forecasts beside the seasonal naive. A censored day has no actual demand to
        // score against, and a censored day last week has no naive to compare with.
        let naive = self.last_week_sales[day_weekday];
        if censoring == DayCensoring::Normal && naive.is_finite() {
            let actual = sales as f32;
            for horizon_index in 0..TRACKED_HORIZONS {
                let issued = self.issued[horizon_index][horizon_index];
                if issued.is_finite() {
                    self.errors[horizon_index].add(issued, actual, naive);
                    self.week_squared_error += (issued - actual).powi(2);
                    self.week_naive_squared_error += (naive - actual).powi(2);
                }
            }
        }
        self.last_week_sales[day_weekday] = if censoring == DayCensoring::Normal { sales as f32 } else { f32::NAN };

        // An event day feeds its event factor, compared with the forecast without it; the other days
        // feed the weekly and yearly shapes.
        if censoring == DayCensoring::Normal {
            self.recent_demand.add(sales);
            match event {
                Some(event) => {
                    let base_forecast = self.model_forecast(1, day_weekday, &factors.week) * yearly;
                    self.stats.events.add(event, sales, base_forecast);
                }
                None => {
                    self.stats.observed.add(day_weekday, sales);
                    self.stats.year.add(week_of_year(day), sales);
                }
            }
        }
        // With factors, an event day teaches only its factor: its spike never reaches the level.
        let model_censoring = if self.uses_factors() && event.is_some() { DayCensoring::Full } else { censoring };
        match &mut self.model {
            Model::HoltWinters { params, state } => {
                state.step(params, day_weekday, sales / yearly, model_censoring);
            }
            Model::Tsb { params, state } => state.step(params, sales / yearly, model_censoring),
        }
        self.last_stock = closing_stock;

        self.issued.copy_within(0..TRACKED_HORIZONS - 1, 1);
        for days_ahead in 1..=TRACKED_HORIZONS as u32 {
            self.issued[0][days_ahead as usize - 1] = self.forecast(days_ahead, day + days_ahead as i32, factors, calendar, config) as f32;
        }

        self.days_since_refit += 1;
        self.check_refit_triggers(day, config);
    }

    fn check_refit_triggers(&mut self, day: i32, config: &EngineConfig) {
        // The weekly comparisons close every 7 days after the refit.
        if self.days_since_refit.is_multiple_of(7) {
            let week_was_scored = self.week_squared_error > 0.0 || self.week_naive_squared_error > 0.0;
            if week_was_scored {
                self.weeks_worse_than_naive = if self.week_squared_error > self.week_naive_squared_error { self.weeks_worse_than_naive + 1 } else { 0 };
            }
            self.week_squared_error = 0.0;
            self.week_naive_squared_error = 0.0;

            let wants_tsb = self.recent_demand.is_intermittent(config.intermittent_adi);
            let is_tsb = matches!(self.model, Model::Tsb { .. });
            self.weeks_on_other_model = if wants_tsb != is_tsb { self.weeks_on_other_model + 1 } else { 0 };
        }
        if self.refit_reason.is_some() {
            return;
        }
        self.refit_reason = if Scenario::of_history(self.first_active_day, day) != self.fitted_scenario {
            Some(RefitReason::Scenario)
        } else if self.weeks_on_other_model >= config.model_switch_weeks {
            Some(RefitReason::ModelSwitch)
        } else if self.weeks_worse_than_naive >= config.worse_than_naive_weeks {
            Some(RefitReason::WorseThanNaive)
        } else if self.days_since_refit >= config.refit_period_days {
            Some(RefitReason::Period)
        } else {
            None
        };
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct EngineState {
    pub version: u16,
    /// The last day incorporated (UnixDay).
    pub last_day: i32,
    pub config: EngineConfig,
    pub calendar: Calendar,
    pub series: Vec<SeriesState>,
}

/// One category's factors pooled over its series, in one site or over every site.
#[derive(Default)]
struct CategoryPool {
    week: WeekdayTotals,
    year: YearPool,
    /// The pool's yearly profile, computed once its series are all in (a median per week is too
    /// costly to recompute for every series that reads it).
    year_profile: Option<YearProfile>,
    events: EventPool,
}

/// The categories' pooled factors, summed from the series' stats. Rebuilt per call rather than
/// stored, so they always equal what the series hold and a refit cannot leave them stale.
pub(crate) struct CategoryFactors {
    pool_by_site_category: HashMap<(i32, i32), CategoryPool>,
    min_sale_days: u32,
}

/// The factors one series forecasts with.
pub(crate) struct SeriesFactors {
    pub(crate) week: WeeklyIndex,
    pub(crate) year: Option<YearProfile>,
    pub(crate) events: [f64; EVENT_TYPES],
}

impl SeriesFactors {
    fn yearly(&self, day: i32) -> f64 {
        self.year.map_or(1.0, |profile| profile[week_of_year(day)])
    }

    fn event(&self, day: i32, calendar: &Calendar) -> f64 {
        calendar.event_on(day).map_or(1.0, |event| self.events[event])
    }
}

impl CategoryFactors {
    pub(crate) fn of<'a>(series: impl Iterator<Item = (&'a SeriesKey, &'a SeriesStats)>, min_sale_days: u32) -> CategoryFactors {
        let mut pool_by_site_category: HashMap<(i32, i32), CategoryPool> = HashMap::new();
        for (key, stats) in series {
            for site_id in [key.site_id, ALL_SITES] {
                let pool = pool_by_site_category.entry((site_id, key.category_id)).or_default();
                pool.week.merge(&stats.observed);
                pool.year.add(&stats.year);
                pool.events.add(&stats.events);
            }
        }
        for pool in pool_by_site_category.values_mut() {
            pool.year_profile = pool.year.profile();
        }
        CategoryFactors { pool_by_site_category, min_sale_days }
    }

    fn of_state(state: &EngineState) -> CategoryFactors {
        CategoryFactors::of(state.series.iter().map(|series| (&series.key, &series.stats)), state.config.min_category_sale_days)
    }

    /// A series' factors: its site's category where that has enough data, else the category over
    /// every site, else neutral (flat week, no yearly profile, event factors of 1).
    pub(crate) fn for_series(&self, key: &SeriesKey) -> SeriesFactors {
        let site_pool = self.pool_by_site_category.get(&(key.site_id, key.category_id));
        let all_sites_pool = self.pool_by_site_category.get(&(ALL_SITES, key.category_id));
        let week = site_pool
            .filter(|pool| pool.week.sale_days >= self.min_sale_days)
            .and_then(|pool| pool.week.index())
            .or_else(|| all_sites_pool.and_then(|pool| pool.week.index()))
            .unwrap_or(FLAT_WEEK);
        let year = site_pool.and_then(|pool| pool.year_profile).or_else(|| all_sites_pool.and_then(|pool| pool.year_profile));
        let events = std::array::from_fn(|event| {
            site_pool
                .and_then(|pool| pool.events.factor(event, EVENT_SHRINK))
                .or_else(|| all_sites_pool.and_then(|pool| pool.events.factor(event, 1.0)))
                .unwrap_or(1.0)
        });
        SeriesFactors { week, year, events }
    }
}

/// The history of a batch of series over the same days. Daily arrays are series-major: series `i`
/// holds days `[i × days, (i + 1) × days)`. A product that started selling later has zeros before,
/// which its empty stock marks as censored.
pub struct SeriesBatch<'a> {
    pub start_day: i32,
    pub keys: &'a [SeriesKey],
    /// Each series' stock at the close of the day before `start_day`.
    pub initial_stock: &'a [f64],
    pub sales: &'a [f64],
    pub closing_stock: &'a [f64],
    pub inflow: &'a [f64],
}

impl SeriesBatch<'_> {
    /// Validates the shapes and returns the number of days.
    fn days(&self) -> Result<usize, String> {
        let series_count = self.keys.len();
        if series_count == 0 {
            return Err("the batch needs at least one series".into());
        }
        for (index, key) in self.keys.iter().enumerate() {
            if self.keys[..index].iter().any(|earlier| earlier.is_same_series(key)) {
                return Err(format!("site {} product {} appears twice", key.site_id, key.product_id));
            }
        }
        if self.initial_stock.len() != series_count {
            return Err(format!("initialStock has {} values for {series_count} series", self.initial_stock.len()));
        }
        if !self.sales.len().is_multiple_of(series_count) {
            return Err(format!("sales has {} values, not a multiple of {series_count} series", self.sales.len()));
        }
        let days = self.sales.len() / series_count;
        if days < MIN_HISTORY_DAYS {
            return Err(format!("a fit needs at least {MIN_HISTORY_DAYS} days of history, got {days}"));
        }
        if self.closing_stock.len() != self.sales.len() || self.inflow.len() != self.sales.len() {
            return Err("closingStock and inflow must have as many values as sales".into());
        }
        if let Some(series_index) = self.initial_stock.iter().position(|stock| !stock.is_finite()) {
            return Err(format!("initialStock of series {series_index} is not a number"));
        }
        check_day_values(self.sales, self.closing_stock, self.inflow, days)?;
        Ok(days)
    }

    fn series_days(&self, series_index: usize, days: usize) -> (&[f64], &[f64], &[f64]) {
        let range = series_index * days..(series_index + 1) * days;
        (&self.sales[range.clone()], &self.closing_stock[range.clone()], &self.inflow[range])
    }
}

/// Rejects day values the models cannot learn from. A NaN or an infinity (an `undefined + n` in the
/// caller) would sit in the saved weekday totals for good and spread through the category's pooled
/// factors to every series in it; negative sales or inflow do not describe a real day. Closing stock
/// may be negative: the ledger can lag behind the shelf.
fn check_day_values(sales: &[f64], closing_stock: &[f64], inflow: &[f64], days: usize) -> Result<(), String> {
    let position = |value_index: usize| format!("series {} day {}", value_index / days, value_index % days);
    for (field, values, may_be_negative) in [("sales", sales, false), ("closingStock", closing_stock, true), ("inflow", inflow, false)] {
        if let Some(value_index) = values.iter().position(|&value| !value.is_finite() || (!may_be_negative && value < 0.0)) {
            return Err(format!("{field} of {} is {}: it must be a finite number{}", position(value_index), values[value_index], if may_be_negative { "" } else { " ≥ 0" }));
        }
    }
    Ok(())
}

/// One series of a batch, classified and counted over the days the optimization sees.
struct PreparedSeries {
    key: SeriesKey,
    censoring: Vec<DayCensoring>,
    stock_tracked: bool,
    first_active_day: Option<i32>,
    scenario: Scenario,
    stats: SeriesStats,
    recent_demand: RecentDemand,
    last_week_sales: [f32; 7],
}

fn prepare_series(batch: &SeriesBatch, series_index: usize, days: usize, fit_days: usize, config: &EngineConfig, calendar: &Calendar) -> PreparedSeries {
    let (sales, closing_stock, inflow) = batch.series_days(series_index, days);
    let initial_stock = batch.initial_stock[series_index];
    // Tracking and activity look at the whole window: a product first stocked during the replayed
    // days still has its earlier empty days censored, not read as zero demand.
    let stock_tracked = is_stock_tracked(initial_stock, inflow, closing_stock);
    let first_active_day = (0..days)
        .find(|&day_index| sales[day_index] > 0.0 || inflow[day_index] > 0.0 || closing_stock[day_index] > 0.0)
        .map(|day_index| batch.start_day + day_index as i32);

    let mut prepared = PreparedSeries {
        key: batch.keys[series_index],
        censoring: Vec::with_capacity(fit_days),
        stock_tracked,
        first_active_day,
        scenario: Scenario::of_history(first_active_day, batch.start_day + days as i32 - 1),
        stats: SeriesStats::default(),
        recent_demand: RecentDemand::default(),
        last_week_sales: [f32::NAN; 7],
    };
    for day_index in 0..fit_days {
        let day = batch.start_day + day_index as i32;
        let censoring = if stock_tracked {
            let opening_stock = if day_index == 0 { initial_stock } else { closing_stock[day_index - 1] };
            classify_day(opening_stock, inflow[day_index], closing_stock[day_index], sales[day_index], config.partial_censor_threshold)
        } else {
            DayCensoring::Normal
        };
        if censoring == DayCensoring::Normal {
            prepared.recent_demand.add(sales[day_index]);
            // Event days are counted once the model has run (fit_series), against its base forecast.
            if calendar.event_on(day).is_none() {
                prepared.stats.observed.add(weekday(day), sales[day_index]);
                prepared.stats.year.add(week_of_year(day), sales[day_index]);
            }
        }
        prepared.last_week_sales[weekday(day)] = if censoring == DayCensoring::Normal { sales[day_index] as f32 } else { f32::NAN };
        prepared.censoring.push(censoring);
    }
    prepared
}

/// Picks the model by the recent demand pattern and optimizes it (plan §5.1) over the fit days, then
/// adds the history's event days to the series' event sums against the model's one-step forecasts.
/// From scenario B the model sees sales over the yearly profile and skips event days; in A it sees
/// them as they are and the factors are only learned, for the category.
fn fit_series(prepared: &mut PreparedSeries, batch: &SeriesBatch, series_index: usize, days: usize, factors: &SeriesFactors, config: &EngineConfig, calendar: &Calendar) -> Model {
    let fit_days = prepared.censoring.len();
    let (sales, _, _) = batch.series_days(series_index, days);
    let uses_factors = prepared.scenario != Scenario::A;
    let day_of = |day_index: usize| batch.start_day + day_index as i32;
    let yearly_by_day: Vec<f64> = (0..fit_days).map(|day_index| if uses_factors { factors.yearly(day_of(day_index)) } else { 1.0 }).collect();
    let model_sales: Vec<f64> = (0..fit_days).map(|day_index| sales[day_index] / yearly_by_day[day_index]).collect();
    let model_censoring: Vec<DayCensoring> = (0..fit_days)
        .map(|day_index| if uses_factors && calendar.event_on(day_of(day_index)).is_some() { DayCensoring::Full } else { prepared.censoring[day_index] })
        .collect();
    let history = SeriesHistory { start_day: batch.start_day, sales: &model_sales, censoring: &model_censoring };

    let (model, one_step) = if prepared.recent_demand.is_intermittent(config.intermittent_adi) {
        let (params, state, one_step) = tsb::fit(&history, &factors.week, config.max_evaluations);
        (Model::Tsb { params, state }, one_step)
    } else {
        // Scenario A: the weekly indices start at the product's shape shrunk toward its category's.
        let observed_weeks = prepared.stats.observed.observed_days() as f64 / 7.0;
        let product_weight = observed_weeks / (observed_weeks + config.shrink_weeks);
        let product_week = prepared.stats.observed.index().unwrap_or(factors.week);
        let initial_season = shrink(product_week, factors.week, product_weight);
        let (params, state, one_step) = holt_winters::fit(&history, initial_season, config.max_damping, config.max_evaluations);
        (Model::HoltWinters { params, state }, one_step)
    };

    for day_index in 0..fit_days {
        if prepared.censoring[day_index] != DayCensoring::Normal {
            continue;
        }
        if let Some(event) = calendar.event_on(day_of(day_index)) {
            prepared.stats.events.add(event, sales[day_index], one_step[day_index] * yearly_by_day[day_index]);
        }
    }
    model
}

/// An empty state ending on `last_day`. `fit` is `create` + `refit`; a caller with a calendar runs
/// `create`, `set_calendar` and then `refit`, so the first fit already learns the event factors.
pub fn create(last_day: i32, config: EngineConfig) -> Result<EngineState, String> {
    config.validate()?;
    Ok(EngineState { version: STATE_VERSION, last_day, config, calendar: Calendar::default(), series: Vec::new() })
}

/// Full fit (plan §7 `fit`): a state over the batch's days, every series fitted, no calendar.
pub fn fit(batch: &SeriesBatch, config: EngineConfig) -> Result<EngineState, String> {
    let days = batch.days()?;
    let mut state = create(batch.start_day + days as i32 - 1, config)?;
    refit(&mut state, batch)?;
    Ok(state)
}

/// Replaces the calendar. A change on a day a series already learned from (up to the state's last
/// day, from its first active day) asks that series for a refit; future-only changes take effect on
/// the next forecast.
pub fn set_calendar(state: &mut EngineState, calendar: Calendar) {
    if let Some(latest_changed_day) = state.calendar.latest_change_until(&calendar, state.last_day) {
        for series in &mut state.series {
            if series.refit_reason.is_none() && series.first_active_day.is_some_and(|first_day| first_day <= latest_changed_day) {
                series.refit_reason = Some(RefitReason::Calendar);
            }
        }
    }
    state.calendar = calendar;
}

/// Fits the batch's series again over their history, which must end on the state's last day:
/// replaces those already in the state (same site and product) in place and appends the new ones in
/// batch order. Refit triggers (`pending_refits`), corrected past data, calendar changes and new
/// products all come through here. A series' stats are rebuilt from the batch's history, so a
/// scenario B series should get the longest history the caller has (up to 400 days).
pub fn refit(state: &mut EngineState, batch: &SeriesBatch) -> Result<(), String> {
    let days = batch.days()?;
    let end_day = batch.start_day + days as i32 - 1;
    if end_day != state.last_day {
        return Err(format!("the history must end on the state's last day {}, it ends on {end_day}", state.last_day));
    }
    let fit_days = days - REPLAYED_DAYS;
    let config = state.config.clone();
    let calendar = state.calendar.clone();
    let mut prepared: Vec<PreparedSeries> =
        (0..batch.keys.len()).map(|series_index| prepare_series(batch, series_index, days, fit_days, &config, &calendar)).collect();
    let position_by_batch_index: Vec<Option<usize>> =
        batch.keys.iter().map(|key| state.series.iter().position(|series| series.key.is_same_series(key))).collect();
    // The history a series has is older than a refit window: keep the earliest active day the state
    // knows, or a 400-day series refitted on 91 days would fall back to scenario A and lose its
    // yearly and calendar factors.
    for (series, position) in prepared.iter_mut().zip(&position_by_batch_index) {
        let stored_first_active_day = position.and_then(|position| state.series[position].first_active_day);
        series.first_active_day = [series.first_active_day, stored_first_active_day].into_iter().flatten().min();
        series.scenario = Scenario::of_history(series.first_active_day, end_day);
    }
    let kept_series = |state: &EngineState| -> Vec<(SeriesKey, SeriesStats)> {
        state
            .series
            .iter()
            .enumerate()
            .filter(|(position, _)| !position_by_batch_index.contains(&Some(*position)))
            .map(|(_, series)| (series.key, series.stats.clone()))
            .collect()
    };
    let kept = kept_series(state);

    // The weekly and yearly shapes pool the kept series with the batch's fresh counts; the models
    // are fitted with them. Their event sums come out of the fit, so the factors are pooled again
    // before the replayed days, which issue forecasts with them.
    let shapes = CategoryFactors::of(kept.iter().map(|(key, stats)| (key, stats)).chain(prepared.iter().map(|series| (&series.key, &series.stats))), config.min_category_sale_days);
    let models: Vec<Model> = prepared
        .iter_mut()
        .enumerate()
        .map(|(series_index, series)| {
            let factors = shapes.for_series(&series.key);
            fit_series(series, batch, series_index, days, &factors, &config, &calendar)
        })
        .collect();
    let factors = CategoryFactors::of(kept.iter().map(|(key, stats)| (key, stats)).chain(prepared.iter().map(|series| (&series.key, &series.stats))), config.min_category_sale_days);

    for (series_index, (prepared, model)) in prepared.into_iter().zip(models).enumerate() {
        let (sales, closing_stock, inflow) = batch.series_days(series_index, days);
        let series_factors = factors.for_series(&prepared.key);
        let mut series = SeriesState {
            key: prepared.key,
            stock_tracked: prepared.stock_tracked,
            last_stock: closing_stock[fit_days - 1],
            // The empty-shelf run the fit days end on; the replayed days continue it.
            empty_shelf_days: prepared.censoring.iter().rev().take_while(|&&censoring| censoring == DayCensoring::Full).count() as u32,
            first_active_day: prepared.first_active_day,
            fitted_scenario: prepared.scenario,
            stats: prepared.stats,
            recent_demand: prepared.recent_demand,
            model,
            issued: [[f32::NAN; TRACKED_HORIZONS]; TRACKED_HORIZONS],
            last_week_sales: prepared.last_week_sales,
            errors: [HorizonErrors::default(); TRACKED_HORIZONS],
            days_since_refit: 0,
            week_squared_error: 0.0,
            week_naive_squared_error: 0.0,
            weeks_worse_than_naive: 0,
            weeks_on_other_model: 0,
            refit_reason: None,
        };
        for day_index in fit_days..days {
            series.advance(batch.start_day + day_index as i32, sales[day_index], closing_stock[day_index], inflow[day_index], &config, &series_factors, &calendar);
        }
        // A new series keeps the replayed days' scores as its first record. A refitted one keeps its
        // live record instead: it already scored those days, and its accuracy reads continuously
        // across refits. The triggers start counting from here either way.
        if let Some(position) = position_by_batch_index[series_index] {
            series.errors = state.series[position].errors;
        }
        series.days_since_refit = 0;
        series.week_squared_error = 0.0;
        series.week_naive_squared_error = 0.0;
        series.weeks_worse_than_naive = 0;
        series.weeks_on_other_model = 0;
        series.refit_reason = None;

        match position_by_batch_index[series_index] {
            Some(position) => state.series[position] = series,
            None => state.series.push(series),
        }
    }
    Ok(())
}

/// New days for every series of the state, in its order, from the day after its last one.
/// Series-major like SeriesBatch: series `i` holds days `[i × days, (i + 1) × days)`.
pub struct DayBatch<'a> {
    pub first_day: i32,
    pub sales: &'a [f64],
    pub closing_stock: &'a [f64],
    pub inflow: &'a [f64],
}

/// Incremental: adds one or more new days to the state (plan §7 `update`). With fixed parameters
/// the result is what rerunning the whole history gives, at one step per day and series.
pub fn update(state: &mut EngineState, batch: &DayBatch) -> Result<(), String> {
    let series_count = state.series.len();
    if series_count == 0 {
        return Err("the state has no series".into());
    }
    if batch.first_day != state.last_day + 1 {
        return Err(format!("the update must start on day {}, the day after the state's last, got {}", state.last_day + 1, batch.first_day));
    }
    if batch.sales.is_empty() || !batch.sales.len().is_multiple_of(series_count) {
        return Err(format!("sales has {} values, not a positive multiple of {series_count} series", batch.sales.len()));
    }
    if batch.closing_stock.len() != batch.sales.len() || batch.inflow.len() != batch.sales.len() {
        return Err("closingStock and inflow must have as many values as sales".into());
    }
    let days = batch.sales.len() / series_count;
    check_day_values(batch.sales, batch.closing_stock, batch.inflow, days)?;

    // Day by day across every series, pooling the category factors again before each day: the
    // series' stats change as days are added, and a multi-day catch-up must equal the same days sent
    // one call at a time.
    for day_index in 0..days {
        let category_factors = CategoryFactors::of_state(state);
        let factors_by_series: Vec<SeriesFactors> = state.series.iter().map(|series| category_factors.for_series(&series.key)).collect();
        for (series_index, series) in state.series.iter_mut().enumerate() {
            let value_index = series_index * days + day_index;
            series.advance(
                state.last_day + 1,
                batch.sales[value_index],
                batch.closing_stock[value_index],
                batch.inflow[value_index],
                &state.config,
                &factors_by_series[series_index],
                &state.calendar,
            );
        }
        state.last_day += 1;
    }
    Ok(())
}

/// Series whose refit trigger fired, by position, with the trigger (plan §7 `pending_refits`). The
/// caller spreads them over days if they are many and refits them with `refit`.
pub fn pending_refits(state: &EngineState) -> Vec<(u32, RefitReason)> {
    state.series.iter().enumerate().filter_map(|(position, series)| series.refit_reason.map(|reason| (position as u32, reason))).collect()
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
#[serde(rename_all = "camelCase")]
pub struct ForecastOutput {
    /// The UnixDay of the first forecast day: the day after the last one incorporated.
    pub first_day: i32,
    pub horizon: u32,
    /// Series-major, in the state's order: series `i` holds days `[i × horizon, (i + 1) × horizon)`.
    pub values: Vec<f64>,
}

/// Point forecasts from the current state, which is not modified (plan §7 `forecast`). Intervals
/// come in P4.
pub fn forecast(state: &EngineState, horizon: u32) -> Result<ForecastOutput, String> {
    if horizon == 0 || horizon > MAX_HORIZON {
        return Err(format!("horizon must be in [1, {MAX_HORIZON}], got {horizon}"));
    }
    let category_factors = CategoryFactors::of_state(state);
    let mut values = Vec::with_capacity(state.series.len() * horizon as usize);
    for series in &state.series {
        let factors = category_factors.for_series(&series.key);
        for days_ahead in 1..=horizon {
            values.push(series.forecast(days_ahead, state.last_day + days_ahead as i32, &factors, &state.calendar, &state.config));
        }
    }
    Ok(ForecastOutput { first_day: state.last_day + 1, horizon, values })
}

/// Online accuracy per series and horizon (plan §5.4). Series-major: series `i` holds horizons 1 to
/// 7 at `[i × 7, (i + 1) × 7)`. NaN where nothing was scored yet.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
#[serde(rename_all = "camelCase")]
pub struct MetricsOutput {
    pub count: Vec<u32>,
    /// Σ |error| / Σ actual.
    pub wape: Vec<f64>,
    /// Σ error / Σ actual: positive over-forecasts.
    pub bias: Vec<f64>,
    /// Σ |error| / Σ |seasonal naive error|: below 1 beats the naive.
    pub mase: Vec<f64>,
}

pub fn metrics(state: &EngineState) -> MetricsOutput {
    let ratio = |numerator: f32, denominator: f32| if denominator > 0.0 { (numerator / denominator) as f64 } else { f64::NAN };
    let mut output = MetricsOutput { count: Vec::new(), wape: Vec::new(), bias: Vec::new(), mase: Vec::new() };
    for errors in state.series.iter().flat_map(|series| series.errors.iter()) {
        output.count.push(errors.count);
        output.wape.push(ratio(errors.abs_error_sum, errors.actual_sum));
        output.bias.push(ratio(errors.error_sum, errors.actual_sum));
        output.mase.push(ratio(errors.abs_error_sum, errors.naive_abs_error_sum));
    }
    output
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::days_from_civil;

    // 2024-01-01, a Monday.
    const START_DAY: i32 = 19723;
    const DAYS: usize = 91;
    const HOLIDAY: u8 = 1;

    struct SeriesData {
        key: SeriesKey,
        initial_stock: f64,
        sales: Vec<f64>,
        closing_stock: Vec<f64>,
        inflow: Vec<f64>,
    }

    impl SeriesData {
        fn days(&self, range: std::ops::Range<usize>) -> SeriesData {
            SeriesData {
                key: self.key,
                initial_stock: if range.start == 0 { self.initial_stock } else { self.closing_stock[range.start - 1] },
                sales: self.sales[range.clone()].to_vec(),
                closing_stock: self.closing_stock[range.clone()].to_vec(),
                inflow: self.inflow[range].to_vec(),
            }
        }
    }

    /// Plenty of stock all along: a receipt of 100000 on day 0, never running out.
    fn well_stocked(key: SeriesKey, sales: Vec<f64>) -> SeriesData {
        let mut inflow = vec![0.0; sales.len()];
        inflow[0] = 100000.0;
        let mut closing_stock = Vec::with_capacity(sales.len());
        let mut stock = 0.0;
        for day_index in 0..sales.len() {
            stock += inflow[day_index] - sales[day_index];
            closing_stock.push(stock);
        }
        SeriesData { key, initial_stock: 0.0, sales, closing_stock, inflow }
    }

    fn flatten(series: &[SeriesData], field: fn(&SeriesData) -> &Vec<f64>) -> Vec<f64> {
        series.iter().flat_map(|data| field(data).iter().copied()).collect()
    }

    fn refit_with(state: &mut EngineState, start_day: i32, series: &[SeriesData]) -> Result<(), String> {
        let keys: Vec<SeriesKey> = series.iter().map(|data| data.key).collect();
        let initial_stock: Vec<f64> = series.iter().map(|data| data.initial_stock).collect();
        let (sales, closing_stock, inflow) = (flatten(series, |data| &data.sales), flatten(series, |data| &data.closing_stock), flatten(series, |data| &data.inflow));
        refit(state, &SeriesBatch { start_day, keys: &keys, initial_stock: &initial_stock, sales: &sales, closing_stock: &closing_stock, inflow: &inflow })
    }

    /// create → set_calendar → refit, the path of a caller with a calendar.
    fn fit_with_calendar(series: &[SeriesData], config: EngineConfig, calendar: Calendar) -> EngineState {
        let days = series[0].sales.len();
        let mut state = create(START_DAY + days as i32 - 1, config).unwrap();
        set_calendar(&mut state, calendar);
        refit_with(&mut state, START_DAY, series).unwrap();
        state
    }

    fn fit_with(series: &[SeriesData], config: EngineConfig) -> EngineState {
        fit_with_calendar(series, config, Calendar::default())
    }

    fn update_with(state: &mut EngineState, series: &[SeriesData]) -> Result<(), String> {
        let (sales, closing_stock, inflow) = (flatten(series, |data| &data.sales), flatten(series, |data| &data.closing_stock), flatten(series, |data| &data.inflow));
        update(state, &DayBatch { first_day: state.last_day + 1, sales: &sales, closing_stock: &closing_stock, inflow: &inflow })
    }

    fn key(site_id: i32, product_id: i32, category_id: i32) -> SeriesKey {
        SeriesKey { site_id, product_id, category_id }
    }

    fn weekend_heavy(days: usize) -> Vec<f64> {
        (0..days).map(|day_index| if day_index % 7 >= 5 { 20.0 } else { 10.0 }).collect()
    }

    fn encoded(state: &EngineState) -> Vec<u8> {
        postcard::to_allocvec(state).unwrap()
    }

    fn quiet_triggers() -> EngineConfig {
        EngineConfig { refit_period_days: 365, worse_than_naive_weeks: 52, ..EngineConfig::default() }
    }

    #[test]
    fn picks_the_model_by_demand_pattern_and_forecasts_both() {
        let intermittent: Vec<f64> = (0..DAYS).map(|day_index| if day_index % 5 == 0 { 5.0 } else { 0.0 }).collect();
        let state = fit_with(&[well_stocked(key(1, 10, 3), weekend_heavy(DAYS)), well_stocked(key(1, 11, 3), intermittent)], EngineConfig::default());
        assert!(matches!(state.series[0].model, Model::HoltWinters { .. }));
        assert!(matches!(state.series[1].model, Model::Tsb { .. }));

        let output = forecast(&state, 14).unwrap();
        assert_eq!(output.first_day, START_DAY + DAYS as i32);
        assert_eq!(output.values.len(), 28);
        // 91 days end on a Sunday, so the first forecast day is a Monday (10) and day 6 a Saturday (20).
        assert!((output.values[0] - 10.0).abs() < 1.0, "{}", output.values[0]);
        assert!((output.values[5] - 20.0).abs() < 2.0, "{}", output.values[5]);
        let intermittent_week: f64 = output.values[14..21].iter().sum();
        assert!((intermittent_week - 7.0).abs() < 2.5, "{intermittent_week}");
    }

    #[test]
    fn untracked_stock_is_never_censored() {
        let untracked = SeriesData { key: key(2, 20, 5), initial_stock: 0.0, sales: vec![4.0; DAYS], closing_stock: vec![0.0; DAYS], inflow: vec![0.0; DAYS] };
        let state = fit_with(&[untracked], EngineConfig::default());
        assert!(!state.series[0].stock_tracked);
        assert_eq!(state.series[0].stats.observed.observed_days(), DAYS as u32);
        let output = forecast(&state, 7).unwrap();
        assert!(output.values.iter().all(|&value| (value - 4.0).abs() < 0.5), "{:?}", output.values);
    }

    #[test]
    fn a_thin_site_falls_back_to_the_all_sites_category() {
        let weekend_tripled: Vec<f64> = (0..DAYS).map(|day_index| if day_index % 7 >= 5 { 30.0 } else { 10.0 }).collect();
        let mut thin = vec![0.0; DAYS];
        thin[0] = 2.0;
        let state = fit_with(&[well_stocked(key(1, 30, 7), weekend_tripled), well_stocked(key(2, 31, 7), thin)], EngineConfig::default());
        let category_factors = CategoryFactors::of_state(&state);
        let all_sites_week = category_factors.pool_by_site_category[&(ALL_SITES, 7)].week.index().unwrap();
        assert_eq!(category_factors.for_series(&key(2, 31, 7)).week, all_sites_week);
        assert!(all_sites_week[5] > all_sites_week[0]);
    }

    #[test]
    fn several_days_at_once_equal_one_day_at_a_time() {
        // Two intermittent products share category 3 with a regular one: the TSB forecasts read the
        // category week, which every added day changes.
        let every_nth = |nth: usize, units: f64| -> Vec<f64> { (0..DAYS + 14).map(|day_index| if day_index % nth == 0 { units } else { 0.0 }).collect() };
        let whole = [well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 14)), well_stocked(key(1, 11, 3), every_nth(3, 4.0)), well_stocked(key(1, 12, 3), every_nth(5, 7.0))];
        let window = |range: std::ops::Range<usize>| -> Vec<SeriesData> { whole.iter().map(|series| series.days(range.clone())).collect() };
        let mut batched = fit_with(&window(0..DAYS), EngineConfig::default());
        assert!(matches!(batched.series[1].model, Model::Tsb { .. }));
        let mut daily = batched.clone();
        update_with(&mut batched, &window(DAYS..DAYS + 14)).unwrap();
        for day_index in DAYS..DAYS + 14 {
            update_with(&mut daily, &window(day_index..day_index + 1)).unwrap();
        }
        assert_eq!(encoded(&batched), encoded(&daily));
        assert_eq!(batched.last_day, START_DAY + DAYS as i32 + 13);
    }

    #[test]
    fn selling_into_negative_stock_reads_zero_days_as_demand() {
        // One receipt of 5, then 5 units every 4th day: the ledger sinks below zero and stays there.
        // True demand is 1.25 a day; reading only the selling days would forecast 5.
        let sales: Vec<f64> = (0..DAYS).map(|day_index| if day_index % 4 == 0 { 5.0 } else { 0.0 }).collect();
        let mut inflow = vec![0.0; DAYS];
        inflow[0] = 5.0;
        let mut stock = 0.0;
        let closing_stock: Vec<f64> = (0..DAYS)
            .map(|day_index| {
                stock += inflow[day_index] - sales[day_index];
                stock
            })
            .collect();
        let state = fit_with(&[SeriesData { key: key(1, 10, 3), initial_stock: 0.0, sales, closing_stock, inflow }], EngineConfig::default());
        assert!(matches!(state.series[0].model, Model::Tsb { .. }));
        let mean = forecast(&state, 28).unwrap().values.iter().sum::<f64>() / 28.0;
        assert!((mean - 1.25).abs() < 0.3, "mean {mean}");
    }

    #[test]
    fn a_slow_mover_on_its_last_units_fades_out() {
        // A unit received and sold every 3rd day for a month, then one unit left unsold for 60 days,
        // with a partial-censoring threshold of 2 units.
        let sales: Vec<f64> = (0..DAYS).map(|day_index| if day_index < 31 && day_index % 3 == 0 { 1.0 } else { 0.0 }).collect();
        let inflow = sales.clone();
        let data = SeriesData { key: key(1, 10, 3), initial_stock: 1.0, sales, closing_stock: vec![1.0; DAYS], inflow };
        let state = fit_with(&[data], EngineConfig { partial_censor_threshold: 2.0, ..EngineConfig::default() });
        let daily_forecast = forecast(&state, 7).unwrap().values;
        assert!(daily_forecast.iter().all(|&value| value < 0.1), "{daily_forecast:?}");
    }

    #[test]
    fn a_shelf_left_empty_fades_the_forecast_after_its_half_life() {
        let mut whole = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 84));
        // The shelf is emptied at the close of the fit window's last day and nothing is received
        // again: every day after it is fully censored.
        whole.closing_stock[DAYS - 1] = 0.0;
        for day_index in DAYS..DAYS + 84 {
            whole.sales[day_index] = 0.0;
            whole.closing_stock[day_index] = 0.0;
        }
        let mut state = fit_with(&[whole.days(0..DAYS)], quiet_triggers());
        let week_forecast = |state: &EngineState| forecast(state, 7).unwrap().values.iter().sum::<f64>();
        let stocked_week = week_forecast(&state);
        // Through the first 28 empty days the forecast holds: a stockout still needs the reorder.
        update_with(&mut state, &[whole.days(DAYS..DAYS + 28)]).unwrap();
        assert_eq!(state.series[0].empty_shelf_days, 28);
        assert!((week_forecast(&state) / stocked_week - 1.0).abs() < 1e-9);
        // 28 days later it is halved, and 28 more halve it again.
        update_with(&mut state, &[whole.days(DAYS + 28..DAYS + 56)]).unwrap();
        assert!((week_forecast(&state) / stocked_week - 0.5).abs() < 1e-9);
        update_with(&mut state, &[whole.days(DAYS + 56..DAYS + 84)]).unwrap();
        assert!((week_forecast(&state) / stocked_week - 0.25).abs() < 1e-9);
        // A refit counts the run from its history instead of starting over.
        refit_with(&mut state, START_DAY + 84, &[whole.days(84..DAYS + 84)]).unwrap();
        assert_eq!(state.series[0].empty_shelf_days, 84);
    }

    #[test]
    fn rejects_values_that_are_not_finite_or_negative() {
        let mut broken = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 1));
        broken.sales[50] = f64::NAN;
        let keys = [broken.key];
        let batch = SeriesBatch { start_day: START_DAY, keys: &keys, initial_stock: &[0.0], sales: &broken.sales, closing_stock: &broken.closing_stock, inflow: &broken.inflow };
        let error = fit(&batch, EngineConfig::default()).unwrap_err();
        assert!(error.contains("sales of series 0 day 50"), "{error}");

        let whole = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 1));
        let mut state = fit_with(&[whole.days(0..DAYS)], EngineConfig::default());
        let state_before = encoded(&state);
        let mut next = whole.days(DAYS..DAYS + 1);
        next.inflow[0] = -1.0;
        let error = update_with(&mut state, &[next]).unwrap_err();
        assert!(error.contains("inflow of series 0 day 0"), "{error}");
        // A rejected update leaves the state untouched.
        assert_eq!(encoded(&state), state_before);
    }

    #[test]
    fn a_refit_on_a_short_window_keeps_the_history_scenario() {
        let days = 400;
        let series = well_stocked(key(1, 10, 3), yearly_wave(START_DAY, days));
        let mut state = fit_with(&[series.days(0..days)], quiet_triggers());
        assert_eq!(state.series[0].fitted_scenario, Scenario::B);
        refit_with(&mut state, START_DAY + (days - DAYS) as i32, &[series.days(days - DAYS..days)]).unwrap();
        assert_eq!(state.series[0].first_active_day, Some(START_DAY));
        assert_eq!(state.series[0].fitted_scenario, Scenario::B);
    }

    #[test]
    fn scores_every_tracked_horizon_once_its_day_arrives() {
        let whole = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 14));
        let mut state = fit_with(&[whole.days(0..DAYS)], EngineConfig::default());
        // The 7 replayed days: horizon h was issued on 7 − h of them before the last.
        let counts_after_fit: Vec<u32> = state.series[0].errors.iter().map(|errors| errors.count).collect();
        assert_eq!(counts_after_fit, vec![6, 5, 4, 3, 2, 1, 0]);

        update_with(&mut state, &[whole.days(DAYS..DAYS + 14)]).unwrap();
        let output = metrics(&state);
        assert_eq!(output.count, vec![20, 19, 18, 17, 16, 15, 14]);
        assert!(output.wape.iter().all(|&wape| wape < 0.05), "{:?}", output.wape);
    }

    #[test]
    fn censored_days_are_not_scored() {
        let mut whole = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 14));
        // The shelf empties at the close of the fit window and stays empty for a week.
        for day_index in DAYS - 1..DAYS + 7 {
            whole.sales[day_index] = 0.0;
            whole.closing_stock[day_index] = 0.0;
        }
        whole.inflow[DAYS + 7] = 500.0;
        whole.closing_stock[DAYS + 7] = 500.0 - whole.sales[DAYS + 7];
        for day_index in DAYS + 8..DAYS + 14 {
            whole.closing_stock[day_index] = whole.closing_stock[day_index - 1] - whole.sales[day_index];
        }
        let mut state = fit_with(&[whole.days(0..DAYS)], EngineConfig::default());
        let count_before = state.series[0].errors[0].count;
        update_with(&mut state, &[whole.days(DAYS..DAYS + 7)]).unwrap();
        assert_eq!(state.series[0].errors[0].count, count_before);
        // After the receipt the days are observed again, but last week's naive is still missing.
        update_with(&mut state, &[whole.days(DAYS + 7..DAYS + 14)]).unwrap();
        assert_eq!(state.series[0].errors[0].count, count_before);
    }

    #[test]
    fn the_period_trigger_asks_for_a_refit_and_refit_clears_it() {
        // A noise-free weekly pattern is the seasonal naive's best case: keep that trigger out of the way.
        let config = EngineConfig { worse_than_naive_weeks: 52, ..EngineConfig::default() };
        let whole = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 28));
        let mut state = fit_with(&[whole.days(0..DAYS)], config);
        update_with(&mut state, &[whole.days(DAYS..DAYS + 27)]).unwrap();
        assert!(pending_refits(&state).is_empty());
        update_with(&mut state, &[whole.days(DAYS + 27..DAYS + 28)]).unwrap();
        assert_eq!(pending_refits(&state), vec![(0, RefitReason::Period)]);

        let errors_before_refit = state.series[0].errors;
        refit_with(&mut state, START_DAY + 28, &[whole.days(28..DAYS + 28)]).unwrap();
        assert!(pending_refits(&state).is_empty());
        assert_eq!(state.series.len(), 1);
        assert_eq!(state.series[0].errors, errors_before_refit);
    }

    #[test]
    fn two_weeks_worse_than_the_naive_ask_for_a_refit() {
        let mut state = fit_with(&[well_stocked(key(1, 10, 3), weekend_heavy(DAYS))], EngineConfig::default());
        let config = state.config.clone();
        let series = &mut state.series[0];
        for week in 1..=2 {
            series.week_squared_error = 10.0;
            series.week_naive_squared_error = 1.0;
            series.days_since_refit = 7 * week;
            series.check_refit_triggers(state.last_day, &config);
        }
        assert_eq!(series.weeks_worse_than_naive, 2);
        assert_eq!(series.refit_reason, Some(RefitReason::WorseThanNaive));
    }

    #[test]
    fn demand_turning_intermittent_switches_the_model_after_the_hysteresis() {
        let mut sales = weekend_heavy(DAYS);
        sales.extend((0..84).map(|day_index| if day_index % 5 == 0 { 8.0 } else { 0.0 }));
        let whole = well_stocked(key(1, 10, 3), sales);
        let mut state = fit_with(&[whole.days(0..DAYS)], quiet_triggers());
        assert!(matches!(state.series[0].model, Model::HoltWinters { .. }));
        // A month in, the decayed ADI has only just crossed the threshold: no switch yet.
        update_with(&mut state, &[whole.days(DAYS..DAYS + 28)]).unwrap();
        assert!(pending_refits(&state).is_empty());
        update_with(&mut state, &[whole.days(DAYS + 28..DAYS + 84)]).unwrap();
        assert_eq!(pending_refits(&state), vec![(0, RefitReason::ModelSwitch)]);

        refit_with(&mut state, START_DAY + 84, &[whole.days(84..DAYS + 84)]).unwrap();
        assert!(matches!(state.series[0].model, Model::Tsb { .. }));
    }

    #[test]
    fn crossing_180_days_of_history_asks_for_a_refit() {
        let whole = well_stocked(key(1, 10, 3), weekend_heavy(180));
        let mut state = fit_with(&[whole.days(0..DAYS)], quiet_triggers());
        update_with(&mut state, &[whole.days(DAYS..179)]).unwrap();
        assert!(pending_refits(&state).is_empty());
        update_with(&mut state, &[whole.days(179..180)]).unwrap();
        assert_eq!(pending_refits(&state), vec![(0, RefitReason::Scenario)]);
    }

    #[test]
    fn refit_replaces_in_place_and_appends_new_products() {
        let first = well_stocked(key(1, 10, 3), weekend_heavy(DAYS));
        let second = well_stocked(key(1, 11, 3), vec![5.0; DAYS]);
        let mut state = fit_with(&[first.days(0..DAYS), second.days(0..DAYS)], EngineConfig::default());

        // Product 11 changed category and product 12 is new: one refit call does both.
        let recategorized = SeriesData { key: key(1, 11, 4), ..second.days(0..DAYS) };
        let new_product = well_stocked(key(1, 12, 3), vec![3.0; DAYS]);
        refit_with(&mut state, START_DAY, &[recategorized, new_product]).unwrap();
        let product_ids: Vec<i32> = state.series.iter().map(|series| series.key.product_id).collect();
        assert_eq!(product_ids, vec![10, 11, 12]);
        assert_eq!(state.series[1].key.category_id, 4);

        let error = refit_with(&mut state, START_DAY + 1, &[first.days(0..DAYS)]).unwrap_err();
        assert!(error.contains("must end on the state's last day"), "{error}");
    }

    #[test]
    fn update_refuses_a_gap() {
        let whole = well_stocked(key(1, 10, 3), weekend_heavy(DAYS + 2));
        let mut state = fit_with(&[whole.days(0..DAYS)], EngineConfig::default());
        let next = whole.days(DAYS + 1..DAYS + 2);
        let skipped_day = state.last_day + 2;
        let error = update(&mut state, &DayBatch { first_day: skipped_day, sales: &next.sales, closing_stock: &next.closing_stock, inflow: &next.inflow }).unwrap_err();
        assert!(error.contains("must start on day"), "{error}");
    }

    #[test]
    fn state_survives_serialization() {
        let calendar = Calendar::new(&[START_DAY + 40], &[HOLIDAY]).unwrap();
        let state = fit_with_calendar(&[well_stocked(key(1, 10, 3), vec![6.0; DAYS])], EngineConfig::default(), calendar);
        let bytes = encoded(&state);
        let restored: EngineState = postcard::from_bytes(&bytes).unwrap();
        // NaN (forecasts never issued) is not equal to itself, so compare the encodings.
        assert_eq!(encoded(&restored), bytes);
        assert_eq!(forecast(&restored, 7).unwrap(), forecast(&state, 7).unwrap());
    }

    #[test]
    fn rejects_inconsistent_input() {
        let keys = [key(1, 1, 1)];
        let short = vec![1.0; 10];
        let batch = SeriesBatch { start_day: START_DAY, keys: &keys, initial_stock: &[0.0], sales: &short, closing_stock: &short, inflow: &short };
        let error = fit(&batch, EngineConfig::default()).unwrap_err();
        assert!(error.contains("at least 21 days"), "{error}");
    }

    // ---- P3: scenario B (yearly profile and calendar factors) ----

    /// Daily demand with a yearly wave (±50%, peak in early April) on a flat week.
    fn yearly_wave(start_day: i32, days: usize) -> Vec<f64> {
        (0..days)
            .map(|day_index| {
                let day_of_year = (start_day + day_index as i32 - days_from_civil(2024, 1, 1)).rem_euclid(365) as f64;
                (20.0 * (1.0 + 0.5 * (2.0 * std::f64::consts::PI * day_of_year / 365.0).sin())).round()
            })
            .collect()
    }

    #[test]
    fn scenario_b_follows_the_yearly_profile_where_a_cannot() {
        // 400 days from 2024-01-01 end on 2025-02-03, on the wave's rising flank.
        let days = 400;
        let series = well_stocked(key(1, 10, 3), yearly_wave(START_DAY, days));
        let state = fit_with(&[series.days(0..days)], quiet_triggers());
        assert_eq!(state.series[0].fitted_scenario, Scenario::B);
        let output = forecast(&state, 56).unwrap();
        let truth = yearly_wave(START_DAY + days as i32, 56);
        // Eight weeks ahead demand is ~30% higher; the profile carries the forecast up with it.
        let (forecast_rise, true_rise) = (output.values[55] / output.values[0], truth[55] / truth[0]);
        assert!(forecast_rise > 1.15 && (forecast_rise - true_rise).abs() < 0.12, "forecast ×{forecast_rise:.2}, truth ×{true_rise:.2}");

        // The same last 120 days alone are scenario A: no profile, the damped trend stays near flat.
        let recent_only = well_stocked(key(1, 10, 3), series.sales[days - 120..].to_vec());
        let mut recent_state = create(state.last_day, quiet_triggers()).unwrap();
        refit_with(&mut recent_state, state.last_day - 119, &[recent_only]).unwrap();
        assert_eq!(recent_state.series[0].fitted_scenario, Scenario::A);
        let recent_output = forecast(&recent_state, 56).unwrap();
        assert!(recent_output.values[55] / recent_output.values[0] < forecast_rise);
    }

    #[test]
    fn scenario_b_learns_a_holiday_factor_and_keeps_the_spike_out_of_the_level() {
        let days = 400;
        // Every 30th day is a holiday that doubles a flat demand of 10.
        let holidays: Vec<i32> = (0..days as i32 + 60).filter(|day_index| day_index % 30 == 15).map(|day_index| START_DAY + day_index).collect();
        let sales: Vec<f64> = (0..days).map(|day_index| if day_index % 30 == 15 { 20.0 } else { 10.0 }).collect();
        let calendar = Calendar::new(&holidays, &vec![HOLIDAY; holidays.len()]).unwrap();
        let state = fit_with_calendar(&[well_stocked(key(1, 10, 3), sales.clone())], quiet_triggers(), calendar.clone());

        let output = forecast(&state, 60).unwrap();
        let next_holiday = (1..=60).find(|&days_ahead| calendar.event_on(state.last_day + days_ahead).is_some()).unwrap();
        let holiday_value = output.values[next_holiday as usize - 1];
        let ordinary_value = output.values[(next_holiday as usize) % 60];
        // 13 holidays seen: factor 2 shrunk by 13 / 16 ≈ 1.81.
        assert!((ordinary_value - 10.0).abs() < 0.5, "ordinary {ordinary_value}");
        assert!((holiday_value / ordinary_value - 1.81).abs() < 0.1, "holiday ×{:.2}", holiday_value / ordinary_value);

        // In scenario A the factor is learned for the category but not applied.
        let short = well_stocked(key(1, 11, 3), sales[days - 120..].to_vec());
        let mut short_state = create(state.last_day, quiet_triggers()).unwrap();
        set_calendar(&mut short_state, calendar);
        refit_with(&mut short_state, state.last_day - 119, &[short]).unwrap();
        let short_output = forecast(&short_state, 60).unwrap();
        assert!(short_output.values[next_holiday as usize - 1] < 12.0);
        assert!(short_state.series[0].stats.events.days[0] > 0);
    }

    #[test]
    fn a_past_calendar_change_asks_for_a_refit_and_a_future_one_does_not() {
        let mut state = fit_with(&[well_stocked(key(1, 10, 3), weekend_heavy(DAYS))], quiet_triggers());
        let last_day = state.last_day;
        set_calendar(&mut state, Calendar::new(&[last_day + 5], &[HOLIDAY]).unwrap());
        assert!(pending_refits(&state).is_empty());
        set_calendar(&mut state, Calendar::new(&[last_day + 5, last_day - 10], &[HOLIDAY, HOLIDAY]).unwrap());
        assert_eq!(pending_refits(&state), vec![(0, RefitReason::Calendar)]);
    }
}
