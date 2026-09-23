// The tax period, and how it relates to the project's UnixDay.
//
// SIRE writes a period as YYYYMM (field 3 of Anexo 3 of RS 112-2021), six characters, where
// PLE used the eight-character AAAAMM00. Everything here works on the six-character form,
// because SIRE is what this ERP files against.

package types

import (
	"fmt"
	"time"
)

// secondsPerDay converts a UnixDay to the instant its day starts at.
const secondsPerDay = int64(24 * 60 * 60)

// PeriodOf is the tax period a date falls in.
func PeriodOf(unixDay int16) string {
	return dayToTime(unixDay).Format("200601")
}

// PeriodBounds is the first and last UnixDay of a period, which is what a book query reads
// between. The last day is the month's own last day, so a book never spills into the next
// period — the error every "add 30 days" implementation makes in February.
func PeriodBounds(period string) (int16, int16, error) {
	firstDay, err := time.Parse("200601", period)
	if err != nil {
		return 0, 0, fmt.Errorf("el periodo %q no tiene el formato YYYYMM", period)
	}
	lastDay := firstDay.AddDate(0, 1, -1)
	return timeToDay(firstDay), timeToDay(lastDay), nil
}

// FormatBookDate writes a date the way every SUNAT flat file wants it: DD/MM/AAAA.
func FormatBookDate(unixDay int16) string {
	if unixDay == 0 {
		return ""
	}
	return dayToTime(unixDay).Format("02/01/2006")
}

func dayToTime(unixDay int16) time.Time {
	return time.Unix(int64(unixDay)*secondsPerDay, 0).UTC()
}

func timeToDay(dateTime time.Time) int16 {
	return int16(dateTime.UTC().Unix() / secondsPerDay)
}
