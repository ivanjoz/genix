package core

import (
	mrand "math/rand"
	"time"
)

// EffectiveNow resolves the request-scoped write time and falls back to the real clock.
func (req *HandlerArgs) EffectiveNow() time.Time {
	if req == nil || req.HistoricalUnix <= 0 {
		return time.Now()
	}
	return time.Unix(req.HistoricalUnix, 0).In(time.Now().Location())
}

// EffectiveFechaUnix keeps historical sample writes on the intended local day.
func (req *HandlerArgs) EffectiveFechaUnix() int16 {
	return TimeToFechaUnix(req.EffectiveNow())
}

// EffectiveSUnixTime keeps persisted audit fields aligned with the effective request time.
func (req *HandlerArgs) EffectiveSUnixTime() int32 {
	if req == nil || req.HistoricalUnix <= 0 {
		return SUnixTime()
	}
	return UnixToSunix(req.HistoricalUnix)
}

// EffectiveSUnixTimeUUID keeps generated IDs date-aligned for historical sample writes.
func (req *HandlerArgs) EffectiveSUnixTimeUUID() int64 {
	if req == nil || req.HistoricalUnix <= 0 {
		return SUnixTimeUUID()
	}
	effectiveUnixMilli := req.HistoricalUnix*1000 + int64(mrand.Intn(1000))
	effectiveSunixMilli := int64((effectiveUnixMilli - 1e12) / 2)
	return effectiveSunixMilli*1000 + int64(mrand.Intn(1000))
}
