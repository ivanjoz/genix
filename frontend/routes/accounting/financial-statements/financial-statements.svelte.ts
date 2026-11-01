import { notifyFailure } from '@genix/ui/notify'
import { extractError, GET } from '#libs/ui-runtime.svelte.ts'
import { tr } from '#core/store.svelte.ts'
import type { ICashBankPeriodBalance } from './financial-statements'

// The cash banks' balances at the period's opening and closing, for the cash flow. Not cached:
// a balance moves with every new movement of the period, and the read is one row per cash bank.
export class CashBalancesService {
  balances: ICashBankPeriodBalance[] | null = $state(null)

  async fetch(period: string): Promise<void> {
    this.balances = null
    try {
      const response = await GET({ route: `accounting-cash-balances?period=${period}` })
      this.balances = response.CashBalances || []
    } catch (error) {
      notifyFailure(extractError(error) || tr("The cash balances could not be read.|No se pudieron leer los saldos de caja."))
    }
  }
}
