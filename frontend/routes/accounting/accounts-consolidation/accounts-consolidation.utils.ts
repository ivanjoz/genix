// splitByWeights divides an integer total in proportion to the weights, rounding each share and
// putting the remainder on the last one, so the shares always add up to the total exactly. With
// no positive weight the whole total goes to the last share.
export function splitByWeights(total: number, weights: number[]): number[] {
  if (weights.length === 0) { return [] }
  const weightSum = weights.reduce((sum, weight) => sum + Math.max(0, weight), 0)
  const shares = weights.map(weight => weightSum > 0 ? Math.round(total * Math.max(0, weight) / weightSum) : 0)
  const assigned = shares.slice(0, -1).reduce((sum, share) => sum + share, 0)
  shares[shares.length - 1] = total - assigned
  return shares
}

// convertUSD restates dollar cents in soles with a rate × 1000.
export const convertUSD = (cents: number, rate: number): number => Math.round(cents * rate / 1000)
