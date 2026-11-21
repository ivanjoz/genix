use serde::{Deserialize, Serialize};

/// Week-of-year buckets: day of year / 7, the last day or two of a year folded into the last.
pub const YEAR_WEEKS: usize = 52;

/// Multiplicative weight of each week of the year, averaging 1.
pub type YearProfile = [f64; YEAR_WEEKS];

/// One series' sales and observed days per week of the year, over its uncensored days without
/// events. Vecs of YEAR_WEEKS (serde does not derive arrays this long).
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq)]
pub struct YearBuckets {
    pub sales: Vec<f32>,
    pub days: Vec<u16>,
}

impl Default for YearBuckets {
    fn default() -> Self {
        YearBuckets { sales: vec![0.0; YEAR_WEEKS], days: vec![0; YEAR_WEEKS] }
    }
}

impl YearBuckets {
    pub fn add(&mut self, week: usize, sales: f64) {
        self.sales[week] += sales as f32;
        self.days[week] = self.days[week].saturating_add(1);
    }

    fn mean_daily_sales(&self) -> f64 {
        let observed_days: u32 = self.days.iter().map(|&days| days as u32).sum();
        if observed_days == 0 {
            return 0.0;
        }
        self.sales.iter().map(|&sales| sales as f64).sum::<f64>() / observed_days as f64
    }
}

/// A category's week-of-year buckets pooled over its series: for each week, every series' daily
/// sales that week as a ratio to its own daily mean, weighted by the volume it expected.
#[derive(Clone, Debug)]
pub struct YearPool {
    /// `(ratio, weight)` per series that observed the week.
    ratios_by_week: Vec<Vec<(f64, f64)>>,
}

impl Default for YearPool {
    fn default() -> Self {
        YearPool { ratios_by_week: vec![Vec::new(); YEAR_WEEKS] }
    }
}

impl YearPool {
    pub fn add(&mut self, buckets: &YearBuckets) {
        let mean_daily_sales = buckets.mean_daily_sales();
        if mean_daily_sales <= 0.0 {
            return;
        }
        for week in 0..YEAR_WEEKS {
            let observed_days = buckets.days[week] as f64;
            if observed_days > 0.0 {
                let expected = observed_days * mean_daily_sales;
                self.ratios_by_week[week].push((buckets.sales[week] as f64 / expected, expected));
            }
        }
    }

    /// The yearly profile: each week's volume-weighted median ratio across the series, smoothed with
    /// its neighbors (circular 1-2-1, plan §6 scenario B), averaging 1. None until every week of the
    /// year has been observed.
    ///
    /// A median, not a pooled sum: a product's life cycle inside the history reads as a season — a
    /// product dying in its second year sits far above its mean in the first year's weeks and at zero
    /// in the second's. Summed, a few of those bend the category's profile, and every product of the
    /// category forecasts with it (the synthetic benchmark measured +5% on flat demand). The median
    /// keeps the shape most of the category shares.
    pub fn profile(&self) -> Option<YearProfile> {
        if self.ratios_by_week.iter().any(|ratios| ratios.is_empty()) {
            return None;
        }
        let raw: YearProfile = std::array::from_fn(|week| weighted_median(&self.ratios_by_week[week]));
        let smoothed: YearProfile = std::array::from_fn(|week| {
            let previous = raw[(week + YEAR_WEEKS - 1) % YEAR_WEEKS];
            let next = raw[(week + 1) % YEAR_WEEKS];
            (0.25 * previous + 0.5 * raw[week] + 0.25 * next).max(0.05)
        });
        let mean = smoothed.iter().sum::<f64>() / YEAR_WEEKS as f64;
        Some(smoothed.map(|weight| weight / mean))
    }
}

/// The value at which half the total weight lies on each side.
fn weighted_median(values_and_weights: &[(f64, f64)]) -> f64 {
    let mut sorted = values_and_weights.to_vec();
    sorted.sort_by(|left, right| left.0.total_cmp(&right.0));
    let half_weight = sorted.iter().map(|(_, weight)| weight).sum::<f64>() / 2.0;
    let mut cumulative_weight = 0.0;
    for (value, weight) in &sorted {
        cumulative_weight += weight;
        if cumulative_weight >= half_weight {
            return *value;
        }
    }
    sorted[sorted.len() - 1].0
}

#[cfg(test)]
mod tests {
    use super::*;

    fn buckets_of(daily_sales_by_week: impl Fn(usize) -> f64) -> YearBuckets {
        let mut buckets = YearBuckets::default();
        for week in 0..YEAR_WEEKS {
            for _ in 0..7 {
                buckets.add(week, daily_sales_by_week(week));
            }
        }
        buckets
    }

    #[test]
    fn profile_reads_the_season_and_averages_one() {
        let mut pool = YearPool::default();
        // A high season in weeks 20-29 at 3× and a slow product with the same shape.
        pool.add(&buckets_of(|week| if (20..30).contains(&week) { 30.0 } else { 10.0 }));
        pool.add(&buckets_of(|week| if (20..30).contains(&week) { 0.3 } else { 0.1 }));
        let profile = pool.profile().unwrap();
        assert!((profile.iter().sum::<f64>() - YEAR_WEEKS as f64).abs() < 1e-9);
        // The buckets hold f32 sales, so 0.3 is not exact.
        assert!((profile[25] / profile[5] - 3.0).abs() < 1e-6);
        // The smoothing softens the season's edges.
        assert!(profile[20] < profile[25] && profile[20] > profile[5]);
    }

    #[test]
    fn a_dying_product_does_not_bend_the_category_profile() {
        let mut pool = YearPool::default();
        for _ in 0..6 {
            pool.add(&buckets_of(|_| 10.0));
        }
        // A big seller that died at mid-year: twice its mean in the first half, nothing after. Summed
        // into the pool it would tilt the profile ×1.6 toward the first half.
        pool.add(&buckets_of(|week| if week < 26 { 60.0 } else { 0.0 }));
        let profile = pool.profile().unwrap();
        assert!(profile.iter().all(|&weight| (weight - 1.0).abs() < 1e-9), "{profile:?}");
    }

    #[test]
    fn profile_needs_every_week() {
        let mut pool = YearPool::default();
        let mut buckets = YearBuckets::default();
        buckets.add(3, 5.0);
        pool.add(&buckets);
        assert_eq!(pool.profile(), None);
    }
}
