use serde::{Deserialize, Serialize};

/// How much a day's sales say about demand, judged from the stock around it (plan §3.1).
#[derive(Serialize, Deserialize, Clone, Copy, Debug, PartialEq, Eq)]
pub enum DayCensoring {
    /// Stock was left at the close: sales are the demand.
    Normal,
    /// The shelf emptied during the day: sales are only a lower bound of demand.
    Partial,
    /// Nothing on hand and nothing received: the day says nothing about demand.
    Full,
}

/// Classifies one day of a product whose stock is tracked. The opening stock is the previous day's
/// closing; inflow is what the ledger received that day (purchases, transfers, adjustments).
pub fn classify_day(opening_stock: f64, inflow: f64, closing_stock: f64, sales: f64, partial_threshold: f64) -> DayCensoring {
    let available = opening_stock + inflow;
    if available < 0.0 {
        // Recorded stock below zero means the ledger lags behind the shelf (the company sells into
        // negative): availability is not what limits sales, so every such day is demand, with or
        // without sales. Censoring only the days without sales would keep the selling days alone
        // and inflate the forecast.
        return DayCensoring::Normal;
    }
    if available == 0.0 {
        // An empty shelf with nothing received: a sale still proves demand (it lagged the ledger),
        // a day without one says nothing.
        return if sales > 0.0 { DayCensoring::Normal } else { DayCensoring::Full };
    }
    if sales == 0.0 {
        // Stock was on hand and nothing sold: zero demand, however low the closing stock. Read as
        // partial it would teach nothing, and a slow mover sitting on its last units would never
        // fade out.
        return DayCensoring::Normal;
    }
    if closing_stock <= partial_threshold { DayCensoring::Partial } else { DayCensoring::Normal }
}

/// A product whose stock never existed in the window (never received, never above zero) is not
/// stock-tracked — a made-to-order dish, a service — so none of its days are censored.
pub fn is_stock_tracked(initial_stock: f64, inflow: &[f64], closing_stock: &[f64]) -> bool {
    initial_stock > 0.0 || inflow.iter().any(|&units| units > 0.0) || closing_stock.iter().any(|&units| units > 0.0)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn classifies_days_by_stock() {
        assert_eq!(classify_day(10.0, 0.0, 4.0, 6.0, 0.0), DayCensoring::Normal);
        assert_eq!(classify_day(10.0, 0.0, 0.0, 10.0, 0.0), DayCensoring::Partial);
        assert_eq!(classify_day(0.0, 0.0, 0.0, 0.0, 0.0), DayCensoring::Full);
        // A receipt on an empty shelf makes the day observable again.
        assert_eq!(classify_day(0.0, 20.0, 12.0, 8.0, 0.0), DayCensoring::Normal);
        // Sold into negative stock: demand was served.
        assert_eq!(classify_day(-3.0, 0.0, -5.0, 2.0, 0.0), DayCensoring::Normal);
        // Negative stock and no sales: still observed, a day of zero demand.
        assert_eq!(classify_day(-5.0, 0.0, -5.0, 0.0, 0.0), DayCensoring::Normal);
        // A threshold above zero flags low closing stock as partial on a day that sold...
        assert_eq!(classify_day(10.0, 0.0, 2.0, 8.0, 2.0), DayCensoring::Partial);
        // ...but units left on the shelf and nothing sold is zero demand.
        assert_eq!(classify_day(1.0, 0.0, 1.0, 0.0, 2.0), DayCensoring::Normal);
    }

    #[test]
    fn detects_untracked_stock() {
        assert!(!is_stock_tracked(0.0, &[0.0, 0.0], &[0.0, -4.0]));
        assert!(is_stock_tracked(0.0, &[0.0, 5.0], &[0.0, 3.0]));
    }
}
