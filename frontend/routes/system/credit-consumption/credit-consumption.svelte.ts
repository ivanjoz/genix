import { GET } from '#libs/ui-runtime.svelte.ts'
import type { ICreditConsumptionResponse } from './credit-consumption.model'

/**
 * Every company's last four hours from `GET.credit-consumption`. Not delta-cached: fareward's rows
 * are absolute snapshots rewritten every flush (15 s), so each refresh replaces the whole window.
 */
export const getCreditConsumption = async (): Promise<ICreditConsumptionResponse> => {
  const response: ICreditConsumptionResponse = await GET({ route: 'credit-consumption' })
  console.debug('[CreditConsumption] companies:', response?.Companies?.length || 0,
    'frames:', response?.FrameIDs?.length || 0)
  return response || {}
}
