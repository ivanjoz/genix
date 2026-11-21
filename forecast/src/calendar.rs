use std::collections::BTreeMap;

use serde::{Deserialize, Serialize};

/// The event types of the calendar_event table (backend sales/types): 1 holiday, 2 payday,
/// 3 gratificación, 4 commercial date. A type's index here is its code − 1.
pub const EVENT_TYPES: usize = 4;
/// A day with several events counts as its strongest one only, in this order.
const STRONGEST_FIRST: [u8; EVENT_TYPES] = [1, 3, 4, 2];
/// k of the shrinkage n / (n + k) of an event factor toward 1, n being the events each series saw on
/// average. Also the events a site's category needs before its own factor is used over the
/// all-sites one.
pub const EVENT_SHRINK: f64 = 3.0;

fn strength_rank(event_type: u8) -> Option<usize> {
    STRONGEST_FIRST.iter().position(|&strongest| strongest == event_type)
}

/// The commercial calendar the factors learn from and forecast with: past and future days, at most
/// one event type per day (plan §6 scenario B).
#[derive(Serialize, Deserialize, Clone, Debug, Default, PartialEq)]
pub struct Calendar {
    days: Vec<i32>,
    types: Vec<u8>,
}

impl Calendar {
    /// Builds the calendar from an event list in any order, keeping the strongest type of each day.
    pub fn new(days: &[i32], types: &[u8]) -> Result<Calendar, String> {
        if days.len() != types.len() {
            return Err(format!("calendar has {} days and {} types", days.len(), types.len()));
        }
        let mut type_by_day: BTreeMap<i32, u8> = BTreeMap::new();
        for (&day, &event_type) in days.iter().zip(types) {
            let rank = strength_rank(event_type).ok_or_else(|| format!("event type {event_type} on day {day} is not 1 to {EVENT_TYPES}"))?;
            let kept_type = type_by_day.entry(day).or_insert(event_type);
            if strength_rank(*kept_type).is_some_and(|kept_rank| rank < kept_rank) {
                *kept_type = event_type;
            }
        }
        Ok(Calendar { days: type_by_day.keys().copied().collect(), types: type_by_day.values().copied().collect() })
    }

    /// The index (type − 1) of the event that counts on a day.
    pub fn event_on(&self, day: i32) -> Option<usize> {
        self.days.binary_search(&day).ok().map(|position| self.types[position] as usize - 1)
    }

    /// The latest day up to `last_day` whose event differs from `other`'s.
    pub fn latest_change_until(&self, other: &Calendar, last_day: i32) -> Option<i32> {
        self.days.iter().chain(&other.days).filter(|&&day| day <= last_day && self.event_on(day) != other.event_on(day)).max().copied()
    }
}

/// One series' uncensored event days by type: what it sold and what it was forecast to sell without
/// the event (the base forecast). Their ratio is the event's factor.
#[derive(Serialize, Deserialize, Clone, Copy, Debug, Default, PartialEq)]
pub struct EventSums {
    pub actual: [f32; EVENT_TYPES],
    pub base: [f32; EVENT_TYPES],
    pub days: [u16; EVENT_TYPES],
}

impl EventSums {
    pub fn add(&mut self, event: usize, actual: f64, base: f64) {
        self.actual[event] += actual as f32;
        self.base[event] += base as f32;
        self.days[event] = self.days[event].saturating_add(1);
    }
}

/// A category's event sums pooled over its series.
#[derive(Clone, Copy, Debug, Default)]
pub struct EventPool {
    actual: [f64; EVENT_TYPES],
    base: [f64; EVENT_TYPES],
    days: [u32; EVENT_TYPES],
    series: [u32; EVENT_TYPES],
}

impl EventPool {
    pub fn add(&mut self, sums: &EventSums) {
        for event in 0..EVENT_TYPES {
            if sums.days[event] > 0 {
                self.actual[event] += sums.actual[event] as f64;
                self.base[event] += sums.base[event] as f64;
                self.days[event] += sums.days[event] as u32;
                self.series[event] += 1;
            }
        }
    }

    /// An event type's factor: pooled sales over pooled base forecast — volume-weighted, so a big
    /// seller weighs more than a slow one — shrunk toward 1 by the events each series saw on average.
    /// None when the series saw fewer than `min_events` on average.
    pub fn factor(&self, event: usize, min_events: f64) -> Option<f64> {
        if self.series[event] == 0 || self.base[event] <= 0.0 {
            return None;
        }
        let events_per_series = self.days[event] as f64 / self.series[event] as f64;
        if events_per_series < min_events {
            return None;
        }
        let ratio = self.actual[event] / self.base[event];
        Some((1.0 + (ratio - 1.0) * events_per_series / (events_per_series + EVENT_SHRINK)).max(0.05))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn keeps_the_strongest_event_of_a_day() {
        // A payday (2) that is also a holiday (1), and a commercial date (4) with a gratificación (3).
        let calendar = Calendar::new(&[20, 10, 10, 20, 30], &[4, 2, 1, 3, 2]).unwrap();
        assert_eq!(calendar.event_on(10), Some(0));
        assert_eq!(calendar.event_on(20), Some(2));
        assert_eq!(calendar.event_on(30), Some(1));
        assert_eq!(calendar.event_on(11), None);
        assert!(Calendar::new(&[1], &[5]).unwrap_err().contains("not 1 to 4"));
    }

    #[test]
    fn finds_the_latest_past_change() {
        let before = Calendar::new(&[10, 20, 40], &[1, 2, 1]).unwrap();
        let after = Calendar::new(&[10, 25, 40], &[1, 2, 4]).unwrap();
        assert_eq!(before.latest_change_until(&after, 30), Some(25));
        assert_eq!(before.latest_change_until(&after, 15), None);
    }

    #[test]
    fn pooled_factor_is_volume_weighted_and_shrunk() {
        let mut pool = EventPool::default();
        let mut big = EventSums::default();
        let mut small = EventSums::default();
        for _ in 0..12 {
            big.add(0, 200.0, 100.0);
            small.add(0, 1.0, 1.0);
        }
        pool.add(&big);
        pool.add(&small);
        let factor = pool.factor(0, 1.0).unwrap();
        // Ratio 2412 / 1212 ≈ 1.99, shrunk by 12 / 15.
        assert!((factor - (1.0 + (2412.0 / 1212.0 - 1.0) * 0.8)).abs() < 1e-9, "{factor}");
        assert_eq!(pool.factor(0, 13.0), None);
        assert_eq!(pool.factor(1, 1.0), None);
    }
}
