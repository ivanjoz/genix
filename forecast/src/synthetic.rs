//! Synthetic sales with a known demand, to measure the engine's accuracy
//! (examples/synthetic_benchmark.rs) and to test it.
//!
//! Every series draws its demand from a known expected value (base rate × site × weekday × yearly
//! wave × calendar lift × trend), so a forecast can be scored against the truth instead of against
//! sales. A replenishment policy then turns demand into what the ERP records: sales capped by the
//! stock on hand, receipts and closing stock. Each product kind stresses one part of the engine.
//! Same config, same data: the random numbers come from a seeded SplitMix64.

use crate::calendar::{Calendar, EVENT_TYPES};
use crate::engine::SeriesKey;
use crate::{days_from_civil, weekday};

/// What each kind of product tests.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum ProductKind {
    /// Sells every day, well stocked: the Holt-Winters case.
    Regular,
    /// Sells a few days a week in lumps: the TSB case.
    Intermittent,
    /// Under-stocked, so the shelf empties often: sales understate demand (censoring).
    StockoutProne,
    /// The company sells without stock on hand: the ledger sinks below zero.
    SellsIntoNegative,
    /// No stock is ever recorded (a dish, a service): never censored.
    MadeToOrder,
    /// Demand fades to zero in the second half: the forecast must follow it down.
    Dying,
    /// No demand until a launch day, then a 14-day ramp: a short history.
    Launched,
}

impl ProductKind {
    pub const ALL: [ProductKind; 7] = [
        ProductKind::Regular,
        ProductKind::Intermittent,
        ProductKind::StockoutProne,
        ProductKind::SellsIntoNegative,
        ProductKind::MadeToOrder,
        ProductKind::Dying,
        ProductKind::Launched,
    ];

    /// Stock kept by the replenishment policy, in days of expected demand per review cycle.
    /// Below 1 the shelf runs out before the next receipt.
    fn stock_cover(self) -> f64 {
        match self {
            ProductKind::Regular | ProductKind::Dying | ProductKind::Launched => 1.5,
            ProductKind::Intermittent => 2.0,
            ProductKind::StockoutProne => 0.75,
            ProductKind::SellsIntoNegative => 0.5,
            ProductKind::MadeToOrder => 0.0,
        }
    }
}

pub struct SyntheticConfig {
    pub seed: u64,
    /// UnixDay of the first day.
    pub start_day: i32,
    pub days: usize,
    pub sites: i32,
    pub categories: i32,
    /// Products per category; their kinds cycle through ProductKind::ALL.
    pub products_per_category: i32,
}

impl Default for SyntheticConfig {
    fn default() -> Self {
        // Two years from 2024-01-01, 2 sites × 4 categories × 14 products = 112 series.
        SyntheticConfig { seed: 20240101, start_day: days_from_civil(2024, 1, 1), days: 760, sites: 2, categories: 4, products_per_category: 14 }
    }
}

/// One site and product: the truth and what the ERP records.
pub struct SyntheticSeries {
    pub key: SeriesKey,
    pub kind: ProductKind,
    /// The expected demand of each day: what a perfect forecast says.
    pub expected_demand: Vec<f64>,
    /// Units customers wanted each day, sold or not.
    pub demand: Vec<f64>,
    /// Closing stock of the day before the first.
    pub initial_stock: f64,
    pub sales: Vec<f64>,
    pub inflow: Vec<f64>,
    pub closing_stock: Vec<f64>,
}

pub struct SyntheticDataset {
    pub start_day: i32,
    pub days: usize,
    pub series: Vec<SyntheticSeries>,
    /// Holidays, paydays, gratificaciones and commercial dates over the days and 60 more, so
    /// forecasts past the last day see them too.
    pub calendar: Calendar,
}

/// Days of calendar past the last day of data.
const CALENDAR_LOOKAHEAD_DAYS: i32 = 60;
/// Days between replenishment reviews and from an order to its receipt.
const REVIEW_DAYS: usize = 7;
const LEAD_DAYS: usize = 3;
/// Mean units of an intermittent product's sale day.
const INTERMITTENT_SALE_SIZE: f64 = 2.5;
/// Day-to-day noise of a regular product's demand rate (lognormal σ): overdispersion over Poisson.
const RATE_NOISE: f64 = 0.3;

/// A category's demand shape: shared by its products, which is what the engine pools.
struct CategoryProfile {
    week: [f64; 7],
    yearly_amplitude: f64,
    yearly_peak_day: f64,
    /// Demand multiplier on each event type's days, by type code − 1.
    event_lift: [f64; EVENT_TYPES],
}

impl CategoryProfile {
    fn draw(random: &mut SplitMix64, category_index: i32) -> CategoryProfile {
        let mut week: [f64; 7] = std::array::from_fn(|_| (1.0 + 0.25 * random.normal()).clamp(0.4, 1.8));
        if category_index == 0 {
            // One clearly weekend-heavy category.
            week = [0.8, 0.8, 0.85, 0.9, 1.1, 1.6, 1.3];
        }
        let week_mean = week.iter().sum::<f64>() / 7.0;
        week.iter_mut().for_each(|factor| *factor /= week_mean);
        CategoryProfile {
            week,
            yearly_amplitude: random.uniform_between(0.0, 0.5),
            yearly_peak_day: random.uniform_between(0.0, 365.0),
            // Holiday (some categories sell less: closed offices), payday, gratificación, commercial date.
            event_lift: [random.uniform_between(0.5, 2.0), random.uniform_between(1.0, 1.3), random.uniform_between(1.2, 1.7), random.uniform_between(1.0, 1.8)],
        }
    }

    fn yearly(&self, day: i32) -> f64 {
        1.0 + self.yearly_amplitude * (2.0 * std::f64::consts::PI * (day as f64 - self.yearly_peak_day) / 365.25 + std::f64::consts::FRAC_PI_2).sin()
    }
}

pub fn generate(config: &SyntheticConfig) -> SyntheticDataset {
    let mut random = SplitMix64(config.seed);
    let end_day = config.start_day + config.days as i32 - 1;
    let event_type_by_day = commercial_calendar(config.start_day, end_day + CALENDAR_LOOKAHEAD_DAYS);
    let event_on = |day: i32| event_type_by_day.iter().find(|(event_day, _)| *event_day == day).map(|(_, event_type)| *event_type);
    let events_in_data: Vec<Option<u8>> = (0..config.days).map(|day_index| event_on(config.start_day + day_index as i32)).collect();

    let profiles: Vec<CategoryProfile> = (0..config.categories).map(|category_index| CategoryProfile::draw(&mut random, category_index)).collect();
    let mut series = Vec::new();
    for (category_index, profile) in profiles.iter().enumerate() {
        for product_index in 0..config.products_per_category {
            let kind = ProductKind::ALL[product_index as usize % ProductKind::ALL.len()];
            let base_rate = match kind {
                ProductKind::Intermittent => random.uniform_between(0.1, 0.6),
                _ => (6.0f64.ln() + 0.8 * random.normal()).exp().clamp(0.8, 60.0),
            };
            let yearly_growth = random.uniform_between(-0.2, 0.3);
            let change_day_index = match kind {
                ProductKind::Dying => random.uniform_between(0.45, 0.75) * config.days as f64,
                ProductKind::Launched => random.uniform_between(0.2, 0.85) * config.days as f64,
                _ => 0.0,
            };
            for site_id in 1..=config.sites {
                let site_share = if site_id == 1 { 1.0 } else { random.uniform_between(0.4, 0.9) };
                let expected_demand: Vec<f64> = (0..config.days)
                    .map(|day_index| {
                        let day = config.start_day + day_index as i32;
                        let trend = match kind {
                            // Linear fade to zero over 60 days.
                            ProductKind::Dying => (1.0 - (day_index as f64 - change_day_index).max(0.0) / 60.0).max(0.0),
                            ProductKind::Launched => ((day_index as f64 - change_day_index) / 14.0).clamp(0.0, 1.0),
                            _ => (1.0 + yearly_growth).powf(day_index as f64 / 365.0),
                        };
                        let event_lift = events_in_data[day_index].map_or(1.0, |event_type| profile.event_lift[event_type as usize - 1]);
                        base_rate * site_share * profile.week[weekday(day)] * profile.yearly(day) * event_lift * trend
                    })
                    .collect();
                let demand: Vec<f64> = expected_demand.iter().map(|&expected| draw_demand(&mut random, kind, expected)).collect();
                let key = SeriesKey { site_id, product_id: (category_index as i32 + 1) * 1000 + product_index, category_id: category_index as i32 + 1 };
                series.push(record_in_erp(&mut random, key, kind, expected_demand, demand));
            }
        }
    }
    let (calendar_days, calendar_types): (Vec<i32>, Vec<u8>) = event_type_by_day.into_iter().unzip();
    SyntheticDataset { start_day: config.start_day, days: config.days, series, calendar: Calendar::new(&calendar_days, &calendar_types).expect("valid calendar") }
}

/// One day's demand around its expected value: lumps of ~2.5 units for intermittent products,
/// overdispersed Poisson for the rest.
fn draw_demand(random: &mut SplitMix64, kind: ProductKind, expected: f64) -> f64 {
    if expected <= 0.0 {
        return 0.0;
    }
    if kind == ProductKind::Intermittent {
        let sale_probability = (expected / INTERMITTENT_SALE_SIZE).min(1.0);
        return if random.uniform() < sale_probability { 1.0 + random.poisson(INTERMITTENT_SALE_SIZE - 1.0) } else { 0.0 };
    }
    // Mean-one lognormal noise on the rate.
    let noisy_rate = expected * (RATE_NOISE * random.normal() - RATE_NOISE * RATE_NOISE / 2.0).exp();
    random.poisson(noisy_rate)
}

/// Runs the replenishment policy over the demand: a review every 7 days orders up to the stock cover
/// of the next review cycle plus lead time (planned on the true expected demand: a good planner), and
/// the order arrives 3 days later, available from the start of that day. Sales are the demand capped
/// by the stock on hand, except where the company sells into negative stock or keeps none.
fn record_in_erp(random: &mut SplitMix64, key: SeriesKey, kind: ProductKind, expected_demand: Vec<f64>, demand: Vec<f64>) -> SyntheticSeries {
    let days = demand.len();
    let cover = kind.stock_cover();
    let order_up_to = |day_index: usize| -> f64 {
        let cycle_demand: f64 = expected_demand[day_index..(day_index + REVIEW_DAYS + LEAD_DAYS).min(days)].iter().sum();
        if cycle_demand <= 0.0 { 0.0 } else { (cover * cycle_demand).ceil().max(if kind == ProductKind::Intermittent { 3.0 } else { 1.0 }) }
    };
    let review_offset = random.below(REVIEW_DAYS as u64) as usize;
    let initial_stock = if kind == ProductKind::MadeToOrder { 0.0 } else { order_up_to(0) };

    let (mut sales, mut inflow, mut closing_stock) = (vec![0.0; days], vec![0.0; days], vec![0.0; days]);
    let mut stock = initial_stock;
    for day_index in 0..days {
        if kind != ProductKind::MadeToOrder && day_index % REVIEW_DAYS == review_offset {
            let on_order: f64 = inflow[day_index..(day_index + LEAD_DAYS).min(days)].iter().sum();
            let order = order_up_to(day_index) - stock - on_order;
            if order > 0.0 && day_index + LEAD_DAYS < days {
                inflow[day_index + LEAD_DAYS] += order;
            }
        }
        let available = stock + inflow[day_index];
        sales[day_index] = match kind {
            ProductKind::SellsIntoNegative | ProductKind::MadeToOrder => demand[day_index],
            _ => demand[day_index].min(available.max(0.0)),
        };
        stock = if kind == ProductKind::MadeToOrder { 0.0 } else { available - sales[day_index] };
        closing_stock[day_index] = stock;
    }
    SyntheticSeries { key, kind, expected_demand, demand, initial_stock, sales, inflow, closing_stock }
}

/// Peru's commercial calendar from `first_day` to `last_day`, one event type per day, the strongest
/// first as the engine reads it (holiday 1 > gratificación 3 > commercial date 4 > payday 2).
fn commercial_calendar(first_day: i32, last_day: i32) -> Vec<(i32, u8)> {
    const FIXED_HOLIDAYS: [(u32, u32); 14] = [(1, 1), (5, 1), (6, 7), (6, 29), (7, 23), (7, 28), (7, 29), (8, 6), (8, 30), (10, 8), (11, 1), (12, 8), (12, 9), (12, 25)];
    const STRENGTH: [u8; EVENT_TYPES] = [1, 3, 4, 2];
    let mut events: Vec<(i32, u8)> = Vec::new();
    let first_year = 1970 + first_day / 366 - 1;
    let last_year = 1970 + last_day / 365 + 1;
    for year in first_year..=last_year {
        for (month, day) in FIXED_HOLIDAYS {
            events.push((days_from_civil(year, month, day), 1));
        }
        let easter = easter_sunday(year);
        // Maundy Thursday and Good Friday.
        events.extend([(easter - 3, 1), (easter - 2, 1)]);
        events.extend([(days_from_civil(year, 7, 15), 3), (days_from_civil(year, 12, 15), 3)]);
        // Mother's Day (second Sunday of May) and Christmas Eve.
        let may_first = days_from_civil(year, 5, 1);
        events.push((may_first + ((6 + 7 - weekday(may_first)) % 7) as i32 + 7, 4));
        events.push((days_from_civil(year, 12, 24), 4));
        for month in 1..=12 {
            let next_month_first = if month == 12 { days_from_civil(year + 1, 1, 1) } else { days_from_civil(year, month + 1, 1) };
            events.extend([(days_from_civil(year, month, 15), 2), (next_month_first - 1, 2)]);
        }
    }
    events.retain(|(day, _)| (first_day..=last_day).contains(day));
    let strength_rank = |event_type: u8| STRENGTH.iter().position(|&strongest| strongest == event_type).unwrap();
    events.sort_by_key(|&(day, event_type)| (day, strength_rank(event_type)));
    events.dedup_by_key(|(day, _)| *day);
    events
}

/// UnixDay of Easter Sunday (anonymous Gregorian algorithm).
fn easter_sunday(year: i32) -> i32 {
    let (a, b, c) = (year % 19, year / 100, year % 100);
    let (d, e) = (b / 4, b % 4);
    let g = (8 * b + 13) / 25;
    let h = (19 * a + b - d - g + 15) % 30;
    let (i, k) = (c / 4, c % 4);
    let l = (32 + 2 * e + 2 * i - h - k) % 7;
    let m = (a + 11 * h + 19 * l) / 433;
    let month = (h + l - 7 * m + 90) / 25;
    let day = (h + l - 7 * m + 33 * month + 19) % 32;
    days_from_civil(year, month as u32, day as u32)
}

/// SplitMix64: small, seedable, good enough for test data.
struct SplitMix64(u64);

impl SplitMix64 {
    fn next(&mut self) -> u64 {
        self.0 = self.0.wrapping_add(0x9E37_79B9_7F4A_7C15);
        let mut mixed = self.0;
        mixed = (mixed ^ (mixed >> 30)).wrapping_mul(0xBF58_476D_1CE4_E5B9);
        mixed = (mixed ^ (mixed >> 27)).wrapping_mul(0x94D0_49BB_1331_11EB);
        mixed ^ (mixed >> 31)
    }

    /// Uniform in (0, 1).
    fn uniform(&mut self) -> f64 {
        ((self.next() >> 11) as f64 + 0.5) / (1u64 << 53) as f64
    }

    fn uniform_between(&mut self, low: f64, high: f64) -> f64 {
        low + (high - low) * self.uniform()
    }

    fn below(&mut self, bound: u64) -> u64 {
        self.next() % bound
    }

    /// Standard normal (Box-Muller).
    fn normal(&mut self) -> f64 {
        (-2.0 * self.uniform().ln()).sqrt() * (2.0 * std::f64::consts::PI * self.uniform()).cos()
    }

    /// Poisson: Knuth's product of uniforms for small means, a rounded normal for large ones.
    fn poisson(&mut self, mean: f64) -> f64 {
        if mean <= 0.0 {
            return 0.0;
        }
        if mean > 30.0 {
            return (mean + mean.sqrt() * self.normal()).round().max(0.0);
        }
        let limit = (-mean).exp();
        let (mut count, mut product) = (0.0, self.uniform());
        while product > limit {
            count += 1.0;
            product *= self.uniform();
        }
        count
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn small_dataset(seed: u64) -> SyntheticDataset {
        generate(&SyntheticConfig { seed, days: 200, products_per_category: 7, ..SyntheticConfig::default() })
    }

    #[test]
    fn the_ledger_adds_up_and_sales_never_exceed_demand() {
        let dataset = small_dataset(7);
        assert_eq!(dataset.series.len(), 2 * 4 * 7);
        for series in &dataset.series {
            let mut opening = series.initial_stock;
            for day_index in 0..dataset.days {
                let closing = opening + series.inflow[day_index] - series.sales[day_index];
                assert!((series.closing_stock[day_index] - closing).abs() < 1e-9 || series.kind == ProductKind::MadeToOrder);
                assert!(series.sales[day_index] <= series.demand[day_index]);
                if series.kind != ProductKind::SellsIntoNegative && series.kind != ProductKind::MadeToOrder {
                    assert!(series.closing_stock[day_index] >= 0.0, "{:?} day {day_index}", series.kind);
                }
                opening = series.closing_stock[day_index];
            }
        }
    }

    #[test]
    fn each_kind_shows_its_trait() {
        let dataset = small_dataset(7);
        let of_kind = |kind: ProductKind| dataset.series.iter().filter(move |series| series.kind == kind);
        let lost_share = |kind: ProductKind| {
            let (sold, wanted) = of_kind(kind).fold((0.0, 0.0), |(sold, wanted), series| (sold + series.sales.iter().sum::<f64>(), wanted + series.demand.iter().sum::<f64>()));
            1.0 - sold / wanted
        };
        assert!(lost_share(ProductKind::StockoutProne) > 0.1, "{}", lost_share(ProductKind::StockoutProne));
        assert!(lost_share(ProductKind::Regular) < 0.05, "{}", lost_share(ProductKind::Regular));
        assert!(of_kind(ProductKind::SellsIntoNegative).all(|series| series.closing_stock.iter().any(|&stock| stock < 0.0)));
        assert!(of_kind(ProductKind::MadeToOrder).all(|series| series.inflow.iter().all(|&units| units == 0.0)));
        assert!(of_kind(ProductKind::Intermittent).all(|series| series.demand.iter().filter(|&&units| units > 0.0).count() < dataset.days / 2));
    }

    #[test]
    fn same_seed_same_data() {
        let (first, second) = (small_dataset(11), small_dataset(11));
        assert!(first.series.iter().zip(&second.series).all(|(left, right)| left.sales == right.sales && left.inflow == right.inflow));
        assert!(small_dataset(12).series[0].sales != first.series[0].sales);
    }

    #[test]
    fn the_calendar_keeps_the_strongest_event_of_a_day() {
        let start = days_from_civil(2024, 1, 1);
        let calendar = commercial_calendar(start, start + 365);
        // 2024-12-15 is a gratificación (3) and a payday (2): the gratificación wins.
        assert_eq!(calendar.iter().find(|(day, _)| *day == days_from_civil(2024, 12, 15)), Some(&(days_from_civil(2024, 12, 15), 3)));
        // Good Friday 2024 fell on March 29.
        assert!(calendar.contains(&(days_from_civil(2024, 3, 29), 1)));
        // Mother's Day 2024: May 12.
        assert!(calendar.contains(&(days_from_civil(2024, 5, 12), 4)));
    }
}
