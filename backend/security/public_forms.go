package security

import (
	"app/core"
	"regexp"
	"strings"
	"time"
)

// What the two unauthenticated forms on /welcome — public sign-up and the contact form — have in
// common: both are keyed by an email address, both are partitioned by ISO week, and both are
// throttled per client IP because neither carries a token and so neither passes through the credit
// rate limiter (main-handlers.go). The address handling and the week arithmetic live here rather
// than being written once per form.

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

// normalizeEmail is what gets stored and compared, so "A@B.com" and "a@b.com " are the same
// address — which is what makes sign-up's "one company per email" rule mean anything.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// weekCodeAt is the [year][week] partition code for a moment in time: year*100 + isoWeek - 200000,
// so 2026-W32 is 2632. The helper is called with an explicit day (never 0) because
// MakeSemanaFromFechaUnix memoizes by its argument, and 0 would pin "the current week" forever in
// a long-running server process.
func weekCodeAt(moment time.Time) int32 {
	return int32(core.MakeSemanaFromFechaUnix(core.TimeToFechaUnix(moment), true).Code)
}

// recentWeekCodes are the partitions a lookup has to touch. Both forms only ever look back over a
// short window — 2 hours for a live sign-up request, minutes for the contact rate limit — so the
// current week and the one before it are always enough, and the previous one matters only for a
// window that started late on a Sunday and is read after midnight.
func recentWeekCodes() []int32 {
	now := core.Now()
	currentWeekCode := weekCodeAt(now)
	previousWeekCode := weekCodeAt(now.AddDate(0, 0, -7))
	if previousWeekCode == currentWeekCode {
		return []int32{currentWeekCode}
	}
	return []int32{currentWeekCode, previousWeekCode}
}
