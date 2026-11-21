//! The browser API (plan §7). Arrays come in as typed arrays; the state goes out as postcard bytes
//! the caller stores and hands back to every call.

use serde::Serialize;
use wasm_bindgen::prelude::*;

use crate::engine::{self, DayBatch, EngineConfig, EngineState, RefitReason, STATE_VERSION, SeriesBatch, SeriesKey};

#[wasm_bindgen(start)]
fn start() {
    console_error_panic_hook::set_once();
}

fn decode(state: &[u8]) -> Result<EngineState, JsError> {
    let state: EngineState = postcard::from_bytes(state).map_err(|error| JsError::new(&format!("unreadable state, refit: {error}")))?;
    if state.version != STATE_VERSION {
        return Err(JsError::new(&format!("state version {} is not {STATE_VERSION}, refit", state.version)));
    }
    Ok(state)
}

fn encode(state: &EngineState) -> Result<Vec<u8>, JsError> {
    postcard::to_allocvec(state).map_err(|error| JsError::new(&error.to_string()))
}

fn to_js<T: Serialize>(value: &T) -> Result<JsValue, JsError> {
    serde_wasm_bindgen::to_value(value).map_err(|error| JsError::new(&error.to_string()))
}

fn series_keys(site_ids: &[i32], product_ids: &[i32], category_ids: &[i32]) -> Result<Vec<SeriesKey>, JsError> {
    if site_ids.len() != product_ids.len() || site_ids.len() != category_ids.len() {
        return Err(JsError::new("siteIds, productIds and categoryIds must have the same length"));
    }
    Ok((0..site_ids.len()).map(|index| SeriesKey { site_id: site_ids[index], product_id: product_ids[index], category_id: category_ids[index] }).collect())
}

/// Full fit: optimizes every series' parameters and builds the state. One series per (site,
/// product), given by the three id arrays. `sales`, `closingStock` and `inflow` are series-major
/// over the same days from `startDay`; `initialStock` is each series' stock the day before.
/// `config` is a partial EngineConfig in camelCase, or undefined for the defaults.
#[wasm_bindgen]
#[allow(clippy::too_many_arguments)]
pub fn fit(
    start_day: i32,
    site_ids: &[i32],
    product_ids: &[i32],
    category_ids: &[i32],
    initial_stock: &[f64],
    sales: &[f64],
    closing_stock: &[f64],
    inflow: &[f64],
    config: JsValue,
) -> Result<Vec<u8>, JsError> {
    let keys = series_keys(site_ids, product_ids, category_ids)?;
    let config: EngineConfig = if config.is_undefined() || config.is_null() {
        EngineConfig::default()
    } else {
        serde_wasm_bindgen::from_value(config).map_err(|error| JsError::new(&format!("invalid config: {error}")))?
    };
    let batch = SeriesBatch { start_day, keys: &keys, initial_stock, sales, closing_stock, inflow };
    encode(&engine::fit(&batch, config).map_err(|error| JsError::new(&error))?)
}

/// Fits some series again, with the same arrays as `fit` over a history ending on the state's last
/// day: the series that `pendingRefits` names, those whose past data was corrected, and new
/// products, which are appended in the given order. Returns the new state.
#[wasm_bindgen]
#[allow(clippy::too_many_arguments)]
pub fn refit(
    state: &[u8],
    start_day: i32,
    site_ids: &[i32],
    product_ids: &[i32],
    category_ids: &[i32],
    initial_stock: &[f64],
    sales: &[f64],
    closing_stock: &[f64],
    inflow: &[f64],
) -> Result<Vec<u8>, JsError> {
    let mut state = decode(state)?;
    let keys = series_keys(site_ids, product_ids, category_ids)?;
    let batch = SeriesBatch { start_day, keys: &keys, initial_stock, sales, closing_stock, inflow };
    engine::refit(&mut state, &batch).map_err(|error| JsError::new(&error))?;
    encode(&state)
}

/// Adds one or more days to every series, in the state's order (`keys`), series-major, the first
/// being the day after the state's last. Returns the new state.
#[wasm_bindgen]
pub fn update(state: &[u8], first_day: i32, sales: &[f64], closing_stock: &[f64], inflow: &[f64]) -> Result<Vec<u8>, JsError> {
    let mut state = decode(state)?;
    engine::update(&mut state, &DayBatch { first_day, sales, closing_stock, inflow }).map_err(|error| JsError::new(&error))?;
    encode(&state)
}

/// Point forecasts for `horizon` days after the state's last day: `{ firstDay, horizon, values }`,
/// values series-major in the state's order.
#[wasm_bindgen]
pub fn forecast(state: &[u8], horizon: u32) -> Result<JsValue, JsError> {
    let state = decode(state)?;
    to_js(&engine::forecast(&state, horizon).map_err(|error| JsError::new(&error))?)
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct SeriesKeysOutput {
    last_day: i32,
    site_ids: Vec<i32>,
    product_ids: Vec<i32>,
    category_ids: Vec<i32>,
}

/// The state's last day and series, in the order `update`, `forecast` and `metrics` use.
#[wasm_bindgen]
pub fn keys(state: &[u8]) -> Result<JsValue, JsError> {
    let state = decode(state)?;
    to_js(&SeriesKeysOutput {
        last_day: state.last_day,
        site_ids: state.series.iter().map(|series| series.key.site_id).collect(),
        product_ids: state.series.iter().map(|series| series.key.product_id).collect(),
        category_ids: state.series.iter().map(|series| series.key.category_id).collect(),
    })
}

#[derive(Serialize)]
struct PendingRefitsOutput {
    series: Vec<u32>,
    reasons: Vec<RefitReason>,
}

/// Series whose refit trigger fired: `{ series: positions, reasons }`, reasons one of "Scenario",
/// "ModelSwitch", "WorseThanNaive", "Period".
#[wasm_bindgen(js_name = pendingRefits)]
pub fn pending_refits(state: &[u8]) -> Result<JsValue, JsError> {
    let state = decode(state)?;
    let (series, reasons) = engine::pending_refits(&state).into_iter().unzip();
    to_js(&PendingRefitsOutput { series, reasons })
}

/// Online accuracy since each series' last refit: `{ count, wape, bias, mase }`, each series-major
/// with horizons 1 to 7 per series; NaN where nothing was scored yet.
#[wasm_bindgen]
pub fn metrics(state: &[u8]) -> Result<JsValue, JsError> {
    to_js(&engine::metrics(&decode(state)?))
}
