package datafunctions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// A period is a small notation the model writes from the user's words ("ayer" → D-1, "semana 23" →
// S=23): the text of PERIOD("…") in the sales tool, the period arguments of a data function. The
// model only translates; ResolvePeriod does the calendar math, which is where models slip (week
// numbers, month ends, days back across a month).

// PeriodNotation is the notation as the model reads it, in the sales tool's sql description.
const PeriodNotation = `Write it in this notation; never compute dates. Units: D day, S ISO week (Monday to Sunday), M month, Y year.
- D today, S this week, M this month, Y this year.
- D-1 yesterday, D-4 four days ago, S-1 last week, M-1 last month, Y-1 last year.
- D=15 the 15th, S=23 week 23, M=12 December, Y=2024 the year 2024. Combine with commas: M=12,Y=2024; S=23,Y=2025; D=15,M=3. Without the year (or month) it is the latest one that has started.
- 2025-03-05 a full date.
- A..B from the start of A to the end of B: D-6..D the last 7 days; M=1..M=3 January to March; D=5,M=3..D since March 5.
- "la última semana", "los últimos 7 días" → D-6..D; "la semana pasada" → S-1. "el último mes", "los últimos 30 días" → D-29..D; "el mes pasado" → M-1.
- The user gave no period → D-13..D (the last 14 days), unless this field says otherwise. Never a period that just started (S, M) for a request without one.`

// DefaultPeriod is the period of a request that gives none: the last 14 days.
const DefaultPeriod = "D-13..D"

var (
	relativePeriodPattern = regexp.MustCompile(`^([DSMY])(?:-(\d{1,5}))?$`)
	absolutePeriodPattern = regexp.MustCompile(`^[DSMY]=\d{1,4}(?:,[DSMY]=\d{1,4})*$`)
)

// maxYearsBack bounds the search for the latest occurrence: a February 29 or a week 53 can be 8
// years back.
const maxYearsBack = 8

// ResolvePeriod turns a period into UnixDays against todayUnixDay (core.FechaUnix, the effective
// clock). The end is clipped to today: reports read the past, so "este mes" runs from the 1st to
// today. Errors are written for the model, in the turn's language, so it fixes its call.
func ResolvePeriod(period string, todayUnixDay int16, maxDays int, language string) (int16, int16, error) {
	today := time.Unix(int64(todayUnixDay)*86400, 0).UTC()
	startText, endText, isRange := strings.Cut(period, "..")
	if !isRange {
		endText = startText
	}
	periodStart, _, err := resolvePeriodPart(strings.TrimSpace(startText), today, language)
	if err != nil {
		return 0, 0, err
	}
	_, periodEnd, err := resolvePeriodPart(strings.TrimSpace(endText), today, language)
	if err != nil {
		return 0, 0, err
	}
	if periodStart.After(today) {
		return 0, 0, fmt.Errorf(Localized(language, "el periodo %q empieza después de hoy (%s): los reportes leen el pasado",
			"the period %q starts after today (%s): reports read the past"), period, today.Format(time.DateOnly))
	}
	if periodStart.Year() < 1970 {
		return 0, 0, fmt.Errorf(Localized(language, "el periodo %q empieza antes de 1970", "the period %q starts before 1970"), period)
	}
	if periodEnd.After(today) {
		periodEnd = today
	}
	if periodEnd.Before(periodStart) {
		return 0, 0, fmt.Errorf(Localized(language, "el periodo %q termina antes de empezar", "the period %q ends before it starts"), period)
	}
	fromDay, toDay := int16(periodStart.Unix()/86400), int16(periodEnd.Unix()/86400)
	if int(toDay-fromDay)+1 > maxDays {
		return 0, 0, fmt.Errorf(Localized(language, "el periodo %q abarca %d días; este reporte lee como máximo %d",
			"the period %q spans %d days; this report reads at most %d"), period, int(toDay-fromDay)+1, maxDays)
	}
	return fromDay, toDay, nil
}

// resolvePeriodPart reads one side of a range: its first and last day.
func resolvePeriodPart(text string, today time.Time, language string) (time.Time, time.Time, error) {
	if fullDate, err := time.Parse(time.DateOnly, text); err == nil {
		return fullDate, fullDate, nil
	}
	if match := relativePeriodPattern.FindStringSubmatch(text); match != nil {
		stepsBack := 0
		if match[2] != "" {
			stepsBack, _ = strconv.Atoi(match[2])
		}
		switch match[1] {
		case "D":
			day := today.AddDate(0, 0, -stepsBack)
			return day, day, nil
		case "S":
			monday := isoWeekMonday(today).AddDate(0, 0, -7*stepsBack)
			return monday, monday.AddDate(0, 0, 6), nil
		case "M":
			firstDay := time.Date(today.Year(), today.Month()-time.Month(stepsBack), 1, 0, 0, 0, 0, time.UTC)
			return firstDay, firstDay.AddDate(0, 1, -1), nil
		default:
			firstDay := time.Date(today.Year()-stepsBack, 1, 1, 0, 0, 0, 0, time.UTC)
			return firstDay, firstDay.AddDate(1, 0, -1), nil
		}
	}
	if absolutePeriodPattern.MatchString(text) {
		return resolveAbsolutePeriod(text, today, language)
	}
	return time.Time{}, time.Time{}, fmt.Errorf(Localized(language,
		"%q no es un periodo: usa D, S, M o Y solos, con -n o con =n (M=12,Y=2024), una fecha AAAA-MM-DD, o A..B",
		"%q is not a period: use D, S, M or Y alone, with -n or with =n (M=12,Y=2024), a YYYY-MM-DD date, or A..B"), text)
}

// resolveAbsolutePeriod reads "U=n" parts (M=12,Y=2024). Without the year, or the month for a day,
// it is the latest one that has started, skipping what the calendar lacks: on October 3, D=31 is
// August 31.
func resolveAbsolutePeriod(text string, today time.Time, language string) (time.Time, time.Time, error) {
	valueRanges := Localized(language, "D 1-31, S 1-53, M 1-12, Y desde 1970", "D 1-31, S 1-53, M 1-12, Y from 1970")
	values := map[string]int{}
	for _, assignment := range strings.Split(text, ",") {
		unit, number, _ := strings.Cut(assignment, "=")
		if _, isRepeated := values[unit]; isRepeated {
			return time.Time{}, time.Time{}, fmt.Errorf(Localized(language, "%q da %s dos veces", "%q gives %s twice"), text, unit)
		}
		values[unit], _ = strconv.Atoi(number)
		if values[unit] == 0 {
			return time.Time{}, time.Time{}, fmt.Errorf(Localized(language, "%q: %s=0 no es un valor (%s)", "%q: %s=0 is not a value (%s)"), text, unit, valueRanges)
		}
	}
	// 0 below means "not given".
	day, week, month, year := values["D"], values["S"], values["M"], values["Y"]
	switch {
	case week > 0 && (day > 0 || month > 0):
		return time.Time{}, time.Time{}, fmt.Errorf(Localized(language, "%q: S solo se combina con Y (S=23,Y=2025)", "%q: S combines only with Y (S=23,Y=2025)"), text)
	case day > 0 && month == 0 && year > 0:
		return time.Time{}, time.Time{}, fmt.Errorf(Localized(language, "%q: un día con año necesita su mes (D=15,M=3,Y=2025)",
			"%q: a day with a year needs its month (D=15,M=3,Y=2025)"), text)
	case day > 31 || week > 53 || month > 12 || (year > 0 && year < 1970):
		return time.Time{}, time.Time{}, fmt.Errorf(Localized(language, "%q tiene un valor fuera de rango (%s)", "%q has a value out of range (%s)"), text, valueRanges)
	}

	// D alone steps back by months, the rest by years.
	stepsLimit := maxYearsBack * 12
	if year > 0 {
		stepsLimit = 0
	}
	for stepsBack := 0; stepsBack <= stepsLimit; stepsBack++ {
		periodStart, periodEnd, exists := absolutePeriodAt(day, week, month, year, stepsBack, today)
		// With the year given there is no stepping back: a future period is the caller's error.
		if exists && (year > 0 || !periodStart.After(today)) {
			return periodStart, periodEnd, nil
		}
	}
	return time.Time{}, time.Time{}, fmt.Errorf(Localized(language, "%q no está en el calendario", "%q is not in the calendar"), text)
}

// absolutePeriodAt is the candidate stepsBack periods (years, or months for a day alone) before the
// current one. exists is false for a date the calendar lacks (D=31,M=2; S=53 in a 52-week year).
func absolutePeriodAt(day, week, month, year, stepsBack int, today time.Time) (time.Time, time.Time, bool) {
	calendarYear := year
	if calendarYear == 0 {
		calendarYear = today.Year() - stepsBack
	}
	switch {
	case week > 0:
		isoYear, _ := today.ISOWeek()
		if year > 0 {
			isoYear = year
		} else {
			isoYear -= stepsBack
		}
		// January 4 is always in ISO week 1.
		monday := isoWeekMonday(time.Date(isoYear, 1, 4, 0, 0, 0, 0, time.UTC)).AddDate(0, 0, 7*(week-1))
		mondayYear, mondayWeek := monday.ISOWeek()
		return monday, monday.AddDate(0, 0, 6), mondayYear == isoYear && mondayWeek == week
	case day > 0 && month == 0:
		monthStart := time.Date(today.Year(), today.Month()-time.Month(stepsBack), 1, 0, 0, 0, 0, time.UTC)
		date := time.Date(monthStart.Year(), monthStart.Month(), day, 0, 0, 0, 0, time.UTC)
		return date, date, date.Month() == monthStart.Month()
	case day > 0:
		date := time.Date(calendarYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return date, date, date.Month() == time.Month(month)
	case month > 0:
		monthStart := time.Date(calendarYear, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		return monthStart, monthStart.AddDate(0, 1, -1), true
	default:
		yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		return yearStart, yearStart.AddDate(1, 0, -1), true
	}
}

func isoWeekMonday(day time.Time) time.Time {
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

// spanishMonths names the months in the messages of a data answer.
var spanishMonths = [...]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

// spanishPeriodWords says a resolved period the way a message reads it, always after "de":
// "del 27 de noviembre de 2026", "de octubre de 2026", "de 2025", "del 17 al 23 de noviembre de 2026",
// "del 28 de octubre al 27 de noviembre de 2026".
func spanishPeriodWords(fromDay, toDay int16) string {
	from, to := time.Unix(int64(fromDay)*86400, 0).UTC(), time.Unix(int64(toDay)*86400, 0).UTC()
	monthName := func(day time.Time) string { return spanishMonths[day.Month()-1] }
	isWholeMonth := from.Day() == 1 && to.AddDate(0, 0, 1).Day() == 1 && from.Year() == to.Year() && from.Month() == to.Month()
	isWholeYear := from.YearDay() == 1 && to.Month() == 12 && to.Day() == 31 && from.Year() == to.Year()
	switch {
	case fromDay == toDay:
		return fmt.Sprintf("del %d de %s de %d", from.Day(), monthName(from), from.Year())
	case isWholeYear:
		return fmt.Sprintf("de %d", from.Year())
	case isWholeMonth:
		return fmt.Sprintf("de %s de %d", monthName(from), from.Year())
	case from.Year() != to.Year():
		return fmt.Sprintf("del %d de %s de %d al %d de %s de %d", from.Day(), monthName(from), from.Year(), to.Day(), monthName(to), to.Year())
	case from.Month() != to.Month():
		return fmt.Sprintf("del %d de %s al %d de %s de %d", from.Day(), monthName(from), to.Day(), monthName(to), to.Year())
	}
	return fmt.Sprintf("del %d al %d de %s de %d", from.Day(), to.Day(), monthName(to), to.Year())
}
