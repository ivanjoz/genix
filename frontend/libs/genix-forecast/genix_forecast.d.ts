/* tslint:disable */
/* eslint-disable */

/**
 * An empty state ending on `lastDay`, for a caller with a calendar: `create`, `setCalendar`, then
 * `refit` with every series, so the first fit already learns the event factors.
 */
export function create(last_day: number, config: any): Uint8Array;

/**
 * Full fit: optimizes every series' parameters and builds the state. One series per (site,
 * product), given by the three id arrays. `sales`, `closingStock` and `inflow` are series-major
 * over the same days from `startDay`; `initialStock` is each series' stock the day before.
 * `config` is a partial EngineConfig in camelCase, or undefined for the defaults.
 */
export function fit(start_day: number, site_ids: Int32Array | number[], product_ids: Int32Array | number[], category_ids: Int32Array | number[], initial_stock: Float64Array | number[], sales: Float64Array | number[], closing_stock: Float64Array | number[], inflow: Float64Array | number[], config: any): Uint8Array;

/**
 * Point forecasts for `horizon` days after the state's last day: `{ firstDay, horizon, values }`,
 * values series-major in the state's order.
 */
export function forecast(state: Uint8Array | number[], horizon: number): any;

/**
 * The state's last day and series, in the order `update`, `forecast` and `metrics` use, with each
 * series' model, scenario and current empty-shelf run.
 */
export function keys(state: Uint8Array | number[]): any;

/**
 * Online accuracy of each series, carried across refits: `{ count, wape, bias, mase }`, each series-major
 * with horizons 1 to 7 per series; NaN where nothing was scored yet.
 */
export function metrics(state: Uint8Array | number[]): any;

/**
 * Series whose refit trigger fired: `{ series: positions, reasons }`, reasons one of "Scenario",
 * "Calendar", "ModelSwitch", "WorseThanNaive", "Period".
 */
export function pendingRefits(state: Uint8Array | number[]): any;

/**
 * Fits some series again, with the same arrays as `fit` over a history ending on the state's last
 * day: the series that `pendingRefits` names, those whose past data was corrected, and new
 * products, which are appended in the given order. Returns the new state.
 */
export function refit(state: Uint8Array | number[], start_day: number, site_ids: Int32Array | number[], product_ids: Int32Array | number[], category_ids: Int32Array | number[], initial_stock: Float64Array | number[], sales: Float64Array | number[], closing_stock: Float64Array | number[], inflow: Float64Array | number[]): Uint8Array;

/**
 * Replaces the commercial calendar: every event (UnixDay and type 1 holiday, 2 payday,
 * 3 gratificación, 4 commercial date), past and future. A change on a day already learned asks the
 * affected series for a refit (`pendingRefits` reason "Calendar"). Returns the new state.
 */
export function setCalendar(state: Uint8Array | number[], days: Int32Array | number[], types: Uint8Array | number[]): Uint8Array;

export function start(): void;

/**
 * Adds one or more days to every series, in the state's order (`keys`), series-major, the first
 * being the day after the state's last. Returns the new state.
 */
export function update(state: Uint8Array | number[], first_day: number, sales: Float64Array | number[], closing_stock: Float64Array | number[], inflow: Float64Array | number[]): Uint8Array;

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

export interface InitOutput {
    readonly memory: WebAssembly.Memory;
    readonly create: (a: number, b: any) => [number, number, number, number];
    readonly fit: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number, k: number, l: number, m: number, n: number, o: number, p: any) => [number, number, number, number];
    readonly forecast: (a: number, b: number, c: number) => [number, number, number];
    readonly keys: (a: number, b: number) => [number, number, number];
    readonly metrics: (a: number, b: number) => [number, number, number];
    readonly pendingRefits: (a: number, b: number) => [number, number, number];
    readonly refit: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number, k: number, l: number, m: number, n: number, o: number, p: number, q: number) => [number, number, number, number];
    readonly setCalendar: (a: number, b: number, c: number, d: number, e: number, f: number) => [number, number, number, number];
    readonly start: () => void;
    readonly update: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number) => [number, number, number, number];
    readonly __wbindgen_malloc: (a: number, b: number) => number;
    readonly __wbindgen_realloc: (a: number, b: number, c: number, d: number) => number;
    readonly __wbindgen_free: (a: number, b: number, c: number) => void;
    readonly __wbindgen_externrefs: WebAssembly.Table;
    readonly __externref_table_dealloc: (a: number) => void;
    readonly __wbindgen_start: () => void;
}

export type SyncInitInput = BufferSource | WebAssembly.Module;

/**
 * Instantiates the given `module`, which can either be bytes or
 * a precompiled `WebAssembly.Module`.
 *
 * @param {{ module: SyncInitInput }} module - Passing `SyncInitInput` directly is deprecated.
 *
 * @returns {InitOutput}
 */
export function initSync(module: { module: SyncInitInput } | SyncInitInput): InitOutput;

/**
 * If `module_or_path` is {RequestInfo} or {URL}, makes a request and
 * for everything else, calls `WebAssembly.instantiate` directly.
 *
 * @param {{ module_or_path: InitInput | Promise<InitInput> }} module_or_path - Passing `InitInput` directly is deprecated.
 *
 * @returns {Promise<InitOutput>}
 */
export default function __wbg_init (module_or_path?: { module_or_path: InitInput | Promise<InitInput> } | InitInput | Promise<InitInput>): Promise<InitOutput>;
