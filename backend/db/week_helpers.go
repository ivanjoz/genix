package db

import "time"

var (
	weekCodeByUnixDay       = map[int16]int16{}
	mondayUnixDayByWeekCode = map[int16]int16{}
)

func makeWeekCodeFromUnixDay(dateUnix int16) int16 {
	if weekCode, ok := weekCodeByUnixDay[dateUnix]; ok {
		return weekCode
	}
	if dateUnix == 0 {
		dateUnix = int16(time.Now().Unix() / 60 / 60 / 24)
	}

	dateUnixTime := int64(dateUnix)*int64(24*60*60) + int64(8*60*60)
	date := time.Unix(dateUnixTime, 0)
	dateUnixBase := dateUnix

	weekday := int(date.Weekday())
	if weekday == 0 {
		date = date.AddDate(0, 0, -6)
		dateUnix -= 6
	} else if weekday != 1 {
		date = date.AddDate(0, 0, 1-weekday)
		dateUnix += 1 - int16(weekday)
	}

	year, week := date.ISOWeek()
	weekCode := int16(year*100 + week - 200000)
	weekCodeByUnixDay[dateUnixBase] = weekCode
	if _, ok := mondayUnixDayByWeekCode[weekCode]; !ok {
		mondayUnixDayByWeekCode[weekCode] = dateUnix
	}
	return weekCode
}

func makeUnixDayFromWeekCode(weekCode int16) int16 {
	if mondayUnixDay, ok := mondayUnixDayByWeekCode[weekCode]; ok {
		return mondayUnixDay
	}
	if weekCode == 0 {
		return 0
	}

	year := int(weekCode/100) + 2000
	week := int(weekCode % 100)
	if week < 1 || week > 53 {
		return 0
	}

	// ISO week 1 is the week containing January 4th. Move back to Monday from that anchor.
	januaryFourth := time.Date(year, time.January, 4, 8, 0, 0, 0, time.UTC)
	weekday := int(januaryFourth.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	firstMonday := januaryFourth.AddDate(0, 0, 1-weekday)
	targetMonday := firstMonday.AddDate(0, 0, (week-1)*7)
	mondayUnixDay := int16(targetMonday.Unix() / (24 * 60 * 60))
	mondayUnixDayByWeekCode[weekCode] = mondayUnixDay
	return mondayUnixDay
}
