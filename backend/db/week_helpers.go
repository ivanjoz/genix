package db

import "time"

var (
	weekCodeByUnixDay       = map[int16]int16{}
	mondayUnixDayByWeekCode = map[int16]int16{}
)

func makeWeekCodeFromUnixDay(fechaUnix int16) int16 {
	if weekCode, ok := weekCodeByUnixDay[fechaUnix]; ok {
		return weekCode
	}
	if fechaUnix == 0 {
		fechaUnix = int16(time.Now().Unix() / 60 / 60 / 24)
	}

	fechaUnixTime := int64(fechaUnix)*int64(24*60*60) + int64(8*60*60)
	fecha := time.Unix(fechaUnixTime, 0)
	fechaUnixBase := fechaUnix

	weekday := int(fecha.Weekday())
	if weekday == 0 {
		fecha = fecha.AddDate(0, 0, -6)
		fechaUnix -= 6
	} else if weekday != 1 {
		fecha = fecha.AddDate(0, 0, 1-weekday)
		fechaUnix += 1 - int16(weekday)
	}

	year, week := fecha.ISOWeek()
	weekCode := int16(year*100 + week - 200000)
	weekCodeByUnixDay[fechaUnixBase] = weekCode
	if _, ok := mondayUnixDayByWeekCode[weekCode]; !ok {
		mondayUnixDayByWeekCode[weekCode] = fechaUnix
	}
	return weekCode
}

func makeUnixDayFromWeekCode(weekCode int16) int16 {
	return mondayUnixDayByWeekCode[weekCode]
}
