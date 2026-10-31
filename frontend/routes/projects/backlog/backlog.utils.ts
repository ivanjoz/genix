// Generic helpers of the Backlog page, no business logic.

// packIntoLanes puts day spans (both days included) side by side in as few lanes as possible: by
// start, each span takes the first lane already free that day, else a new lane. Taking them in
// start order makes first-fit optimal: the lanes equal the most spans overlapping on one day.
export const packIntoLanes = <S extends { start: number, end: number }>(spans: S[]): S[][] => {
  const lanes: S[][] = []
  const laneEnds: number[] = []
  for (const span of [...spans].sort((left, right) => left.start - right.start || left.end - right.end)) {
    const freeLaneIndex = laneEnds.findIndex((laneEnd) => laneEnd < span.start)
    if (freeLaneIndex < 0) {
      lanes.push([span])
      laneEnds.push(span.end)
    } else {
      lanes[freeLaneIndex].push(span)
      laneEnds[freeLaneIndex] = span.end
    }
  }
  return lanes
}

// sUnixTimeToUnixMillis reads a SUnixTime ((unix - 1e9) / 2, the backend's datetime format).
export const sUnixTimeToUnixMillis = (sUnixTime: number): number => (sUnixTime * 2 + 1_000_000_000) * 1000
