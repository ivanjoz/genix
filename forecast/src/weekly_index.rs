use serde::{Deserialize, Serialize};

/// Multiplicative weight of each weekday, Monday first, averaging 1.
pub type WeeklyIndex = [f64; 7];

pub const FLAT_WEEK: WeeklyIndex = [1.0; 7];

/// Rescales an index to average 1. An index with nothing in it is flat.
pub fn normalize(index: WeeklyIndex) -> WeeklyIndex {
    let mean = index.iter().sum::<f64>() / 7.0;
    if mean <= 0.0 {
        return FLAT_WEEK;
    }
    index.map(|weight| weight / mean)
}

/// Pulls a product's weekly shape toward its category's: `weight` is the share the product's own
/// shape keeps, `n / (n + k)` in weeks of history (plan §6, scenario A).
pub fn shrink(product: WeeklyIndex, category: WeeklyIndex, weight: f64) -> WeeklyIndex {
    let mut blended = [0.0; 7];
    for weekday in 0..7 {
        blended[weekday] = weight * product[weekday] + (1.0 - weight) * category[weekday];
    }
    normalize(blended)
}

/// Sales and observed days per weekday, over uncensored days only.
#[derive(Serialize, Deserialize, Clone, Copy, Debug, Default, PartialEq)]
pub struct WeekdayTotals {
    pub sales: [f64; 7],
    pub days: [u32; 7],
    /// Observed days that sold something: what decides whether a shape is trustworthy.
    pub sale_days: u32,
}

impl WeekdayTotals {
    pub fn add(&mut self, weekday: usize, sales: f64) {
        self.sales[weekday] += sales;
        self.days[weekday] += 1;
        if sales > 0.0 {
            self.sale_days += 1;
        }
    }

    pub fn merge(&mut self, other: &WeekdayTotals) {
        for weekday in 0..7 {
            self.sales[weekday] += other.sales[weekday];
            self.days[weekday] += other.days[weekday];
        }
        self.sale_days += other.sale_days;
    }

    pub fn observed_days(&self) -> u32 {
        self.days.iter().sum()
    }

    /// The mean sales of each weekday over their average. None when a weekday was never observed or
    /// nothing sold: there is no shape to read.
    pub fn index(&self) -> Option<WeeklyIndex> {
        if self.sale_days == 0 || self.days.contains(&0) {
            return None;
        }
        Some(normalize(std::array::from_fn(|weekday| self.sales[weekday] / self.days[weekday] as f64)))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn index_reads_the_weekday_shape() {
        let mut totals = WeekdayTotals::default();
        for week in 0..4 {
            for weekday in 0..7 {
                // Weekends sell double; one week has an extra Monday sale that averages out.
                let sales = if weekday >= 5 { 20.0 } else { 10.0 } + if week == 0 && weekday == 0 { 4.0 } else { 0.0 };
                totals.add(weekday, sales);
            }
        }
        let index = totals.index().unwrap();
        assert!((index.iter().sum::<f64>() - 7.0).abs() < 1e-9);
        assert!((index[6] / index[2] - 2.0).abs() < 1e-9);
    }

    #[test]
    fn index_needs_every_weekday() {
        let mut totals = WeekdayTotals::default();
        totals.add(0, 5.0);
        assert_eq!(totals.index(), None);
    }

    #[test]
    fn shrink_blends_and_normalizes() {
        let product = normalize([2.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0]);
        let blended = shrink(product, FLAT_WEEK, 0.5);
        assert!((blended.iter().sum::<f64>() - 7.0).abs() < 1e-9);
        assert!(blended[0] > 1.0 && blended[0] < product[0]);
    }
}
