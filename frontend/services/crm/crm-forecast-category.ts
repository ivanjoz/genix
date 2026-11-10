// Plain module so pure files can import it without the Svelte runtime. Mirrors types.ForecastCategory*;
// Won and Lost are outcomes, not stages, so they are not categories.
export const FORECAST_CATEGORY = { PIPELINE: 1, BEST_CASE: 2, COMMIT: 3 } as const
