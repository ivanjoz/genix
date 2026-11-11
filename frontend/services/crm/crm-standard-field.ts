// Plain module so pure files can import it without the Svelte runtime. Mirrors types.StandardField*:
// the standard opportunity fields a pipeline stage can require (ICrmPipelineStage.RequiredStandardFields).
export const STANDARD_FIELD = { AMOUNT: 1, EXPECTED_CLOSE_DATE: 2 } as const
