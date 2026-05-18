package core

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"
)

type FecSemana struct {
	Id          int
	Code        int16
	DateUnix   int16
	Nro         uint8
	Year        int16
	DateString string
	Idx         int16
}

func (e *FecSemana) MakeName() string {
	s2 := strconv.Itoa(int(e.Nro))
	if len(s2) == 1 {
		s2 = "0" + s2
	}
	return Concat("-", e.Year, s2)
}

var dateToSemana = map[int16]*FecSemana{}

// Si dateUnix = 0 entonces se obtiene la semana actual
func MakeSemanaFromFechaUnix(dateUnix int16, isMonday bool) *FecSemana {
	if fecSemana, ok := dateToSemana[dateUnix]; ok {
		return fecSemana
	}
	if dateUnix == 0 {
		dateUnix = TimeToFechaUnix(time.Now())
	}

	// convierte la dateUnix en TimeStamp
	dateUnixTime := int64(dateUnix)*int64(24*60*60) + int64(8*60*60)
	date := time.Unix(dateUnixTime, 0)

	dateUnixBase := dateUnix

	if !isMonday {
		weekday := int(date.Weekday())
		// Log("weekday:: ", weekday)
		if weekday == 0 {
			date = date.AddDate(0, 0, -6)
			dateUnix = dateUnix - 6
		} else if weekday != 1 {
			date = date.AddDate(0, 0, (1 - weekday))
			dateUnix = dateUnix + 1 - int16(weekday)
		}
	}

	year, week := date.ISOWeek()

	fechSemana := FecSemana{
		Year: int16(year), Nro: uint8(week), Id: year*100 + week,
		DateUnix:   dateUnix,
		DateString: date.Format("2006-01-02"),
	}
	fechSemana.Code = int16(fechSemana.Id - 200000)

	dateToSemana[dateUnixBase] = &fechSemana
	return &fechSemana
}

// Crea un array de semanas basados en una date unix
func GetSemanasFromFecha(dateUnix int16, incremento uint8, decremento uint8) []*FecSemana {

	if dateUnix == 0 {
		dateUnix = int16(time.Now().Unix() / 60 / 60 / 24)
	}

	// convierte la dateUnix en TimeStamp
	dateUnixTime := int64(dateUnix)*int64(24*60*60) + int64(8*60*60)
	date := time.Unix(dateUnixTime, 0)

	weekday := int(date.Weekday())
	if weekday == 0 {
		// date = date.AddDate(0, 0, -6)
		dateUnix = dateUnix - 6
	} else if weekday != 1 {
		// date = date.AddDate(0, 0, (weekday - 1))
		dateUnix = dateUnix - (int16(weekday) - 1)
	}

	datesSemana := []*FecSemana{
		MakeSemanaFromFechaUnix(dateUnix, true),
	}

	// Obtiene las semanas del incremento
	for i := 1; i <= int(incremento); i++ {
		dateUnix2 := dateUnix + int16(i*7)
		semana := MakeSemanaFromFechaUnix(dateUnix2, true)
		datesSemana = append(datesSemana, semana)
	}

	// Obtiene las semanas del decremento
	for i := 1; i <= int(decremento); i++ {
		dateUnix2 := dateUnix - int16(i*7)
		semana := MakeSemanaFromFechaUnix(dateUnix2, true)
		datesSemana = append(datesSemana, semana)
	}

	sort.Slice(datesSemana, func(i, j int) bool {
		return datesSemana[i].Id < datesSemana[j].Id
	})

	return datesSemana
}

type TimeHelper struct {
	init          bool
	dateToSemana map[int16]*FecSemana
	codeToSemana  map[int16]*FecSemana
	dateToString map[int16]string
	semanas       []*FecSemana
	idxToSemana   map[int16]*FecSemana
	semanaToIdx   map[int16]int16
	maxSemana     int16
	minSemana     int16
}

func (e *TimeHelper) Init() {
	if !e.init {
		e.dateToSemana = map[int16]*FecSemana{}
		e.codeToSemana = map[int16]*FecSemana{}
		e.dateToString = map[int16]string{}
		e.semanas = []*FecSemana{}
		e.idxToSemana = map[int16]*FecSemana{}
		e.semanaToIdx = map[int16]int16{}
		e.init = true
	}
}

func (e *TimeHelper) Print() {
	slice1 := [][]int16{}
	for idx, sem := range e.idxToSemana {
		slice1 = append(slice1, []int16{idx, sem.Code})
	}
	sort.Slice(slice1, func(i, j int) bool {
		return slice1[j][0] > slice1[i][0]
	})

	for _, e := range slice1 {
		fmt.Println("idx: ", e[0], " | sem: ", e[1])
	}
}

func (e *TimeHelper) GetFechaUnix() int16 {
	nowTime := time.Now()
	_, offset := nowTime.Zone()
	// core.Log("offset zone:", offset, " |  now Time", nowTime.Unix())
	dateUnix := int16((nowTime.Unix() + int64(offset)) / 60 / 60 / 24)
	return dateUnix
}

func (e *TimeHelper) ExtendRangeToSemana(semanaCode, offset int16) {
	if offset == 0 || semanaCode == 0 {
		return
	}
	if semanaCode < 1001 || semanaCode > 3001 {
		panic(Concats("semana inválida:: ", semanaCode))
	}

	semanaFrom := e.SemanaFromCode(semanaCode)
	semanaTo := e.SemanaFromFecha(semanaFrom.DateUnix + (offset * 7))
	semanaToIdx := int16(0)
	semanaFromIdx := int16(0)

	if semanaFrom.Code > semanaTo.Code {
		semanaToNew := semanaFrom
		semanaFrom = semanaTo
		semanaTo = semanaToNew
	}

	if e.maxSemana > 0 && semanaTo.Code > e.maxSemana {
		semanaFrom = e.codeToSemana[e.maxSemana]
		semanaFromIdx = e.semanaToIdx[e.maxSemana]
	} else if e.minSemana > 0 && semanaTo.Code < e.minSemana {
		semanaTo = e.codeToSemana[e.minSemana]
		semanaToIdx = e.semanaToIdx[e.minSemana]
	} else if idx, ok := e.semanaToIdx[semanaFrom.Code]; ok {
		semanaFromIdx = idx
	} else if idx, ok := e.semanaToIdx[semanaTo.Code]; ok {
		semanaToIdx = idx
	} else if e.maxSemana == 0 && e.minSemana == 0 {
		semanaFromIdx = 1
	} else {
		panic("No se reconocio la semana")
	}

	newSemanas := []*FecSemana{semanaFrom}
	currentSemana := int16(0)
	it := int16(1)

	if semanaFromIdx != 0 {
		semanaFrom.Idx = semanaFromIdx
	}

	for currentSemana <= semanaTo.Code {
		semana := e.SemanaFromFecha(semanaFrom.DateUnix + (it * 7))
		newSemanas = append(newSemanas, semana)
		currentSemana = semana.Code
		if semanaFromIdx != 0 {
			semana.Idx = semanaFromIdx + it
			if prevSem, ok := e.idxToSemana[semana.Idx]; ok {
				if prevSem.Code != semana.Code {
					e.Print()
					panic(Concats("Hay una diferencia de semanas:: ", prevSem.Code, semana.Code, " | Idx: ", semana.Idx))
				}
			}
		}
		if it > 800 {
			e.Print()
			fmt.Println("semana from:: ", semanaFrom.Code, " | semana to: ", semanaTo.Code)
			panic("algo raro paso aqui:: ")
		}
		it++
	}

	if semanaToIdx != 0 {
		for i, semana := range newSemanas {
			semana.Idx = semanaToIdx - int16(len(newSemanas)-1-i) + 1
			if prevSem, ok := e.idxToSemana[semana.Idx]; ok {
				if prevSem.Code != semana.Code {
					e.Print()
					panic(Concats("Hay una diferencia de semanas:: ", prevSem.Code, semana.Code, " | Idx: ", semana.Idx))
				}
			}
		}
	}

	for _, sem := range newSemanas {
		if sem.Code > e.maxSemana {
			e.maxSemana = sem.Code
		}
		if sem.Code < e.minSemana || e.minSemana == 0 {
			e.minSemana = sem.Code
		}
		e.idxToSemana[sem.Idx] = sem
		e.semanaToIdx[sem.Code] = sem.Idx
		e.codeToSemana[sem.Code] = sem
	}
}

func (e *TimeHelper) AddToSemana(semanaCode, nroToAdd int16) *FecSemana {
	e.Init()
	if _, ok := e.semanaToIdx[semanaCode]; !ok {
		e.ExtendRangeToSemana(semanaCode, nroToAdd)
	}

	semanaBaseIdx := e.semanaToIdx[semanaCode]
	semanaTargetIdx := semanaBaseIdx + nroToAdd

	if _, ok := e.idxToSemana[semanaTargetIdx]; !ok {
		e.ExtendRangeToSemana(semanaCode, nroToAdd)
	}

	return e.idxToSemana[semanaTargetIdx]
}

func (e *TimeHelper) SemanaDiference(semanaInicioCode, semanaFinCode int16) int32 {
	e.Init()
	semanaInicio := e.SemanaFromCode(semanaInicioCode)
	semanaFin := e.SemanaFromCode(semanaFinCode)

	return Round(float32(semanaInicio.DateUnix-semanaFin.DateUnix) / 7)
}

func (e *TimeHelper) GetFechaUnixP() int16 {
	nowTime := time.Now()
	zone, offset := nowTime.Zone()
	nowUnix := nowTime.Unix()
	if zone == "UTC" {
		nowUnix -= 18000
	}
	dateUnix := int16((nowUnix + int64(offset)) / 60 / 60 / 24)
	return dateUnix
}

func (e *TimeHelper) GetFechaUnixI() int16 {
	nowTime := time.Now()
	zone, offset := nowTime.Zone()
	nowUnix := nowTime.Unix()
	if zone == "UTC" {
		nowUnix += 19800
	}
	dateUnix := int16((nowUnix + int64(offset)) / 60 / 60 / 24)
	return dateUnix
}

func (e *TimeHelper) TimeToFechaUnix(dateT time.Time) int16 {
	return TimeToFechaUnix(dateT)
}

func (e *TimeHelper) DateToTime(dateUnix int16) time.Time {
	return time.Unix(int64(dateUnix)*24*60*60, 0)
}

func (e *TimeHelper) SemanaFromCode(semanaCode int16) *FecSemana {
	e.Init()
	if _, ok := e.codeToSemana[semanaCode]; ok {
		return e.codeToSemana[semanaCode]
	}

	year1 := int(float64(semanaCode)/100 + 0.001)
	semanaNro := int(semanaCode) - year1*100
	// Log("year:: ", year1, "  |  semana:: ", semanaNro)
	year := year1 + 2000
	dateLunes := WeekStartDate(year, semanaNro)
	// Log(dateLunes)

	fecSemana := FecSemana{
		Id:          (year)*100 + semanaNro,
		Code:        semanaCode,
		DateUnix:   TimeToFechaUnix(dateLunes),
		Nro:         uint8(semanaNro),
		Year:        int16(year),
		DateString: dateLunes.Local().UTC().Format("2006-01-02"),
	}
	e.codeToSemana[semanaCode] = &fecSemana
	return &fecSemana
}

func (e *TimeHelper) DateToTimeUTC(dateUnix int16) time.Time {
	_, offset := (time.Now()).Zone()
	var unixTime int64 = int64(dateUnix)*24*60*60 - int64(offset)
	return time.Unix(unixTime, 0)
}

func (e *TimeHelper) DateUnixToStringU(dateUnix int16, format ...int8) string {
	if dateUnix == 0 {
		return ""
	}
	if e.dateToString == nil {
		e.dateToString = map[int16]string{}
	} else if _, ok := e.dateToString[dateUnix]; ok {
		return e.dateToString[dateUnix]
	}
	dateTime := time.Unix(int64(dateUnix)*24*60*60, 0)
	parseFormat := "2006-01-02"
	if len(format) > 0 && format[0] == 2 {
		parseFormat = "02-01-2006"
	}

	dateString := dateTime.Local().UTC().Format(parseFormat)
	e.dateToString[dateUnix] = dateString
	return dateString
}

func (e *TimeHelper) DateUnixToString(dateUnix int16, format ...int8) string {
	return e.DateUnixToStringU(int16(dateUnix), format...)
}

func (e *TimeHelper) DateUnixToStringP(dateUnix int16) string {
	e.Init()
	if _, ok := e.dateToString[dateUnix]; ok {
		return e.dateToString[dateUnix]
	}
	dateTime := time.Unix(int64(dateUnix)*24*60*60, 0)
	dateString := dateTime.Local().UTC().Format("2006-01-02")
	e.dateToString[dateUnix] = dateString
	return dateString
}

func (e *TimeHelper) SemanaFromFecha(date int16) *FecSemana {
	e.Init()
	if _, ok := e.dateToSemana[date]; ok {
		return e.dateToSemana[date]
	}

	semana := MakeSemanaFromFechaUnix(date, false)
	e.dateToSemana[date] = semana

	return e.dateToSemana[date]
}

func UnixTimeToFechaZone(unixTime int64, utcTimeZone int8) int16 {
	zoneTime := unixTime + int64(utcTimeZone)*60*60
	return int16(math.Floor(float64(zoneTime) / 60 / 60 / 24))
}

func TimeToFechaUnix(dateT time.Time) int16 {
	_, offset := dateT.Zone()
	dateUnix := int16((dateT.Unix() + int64(offset)) / 24 / 60 / 60)
	return dateUnix
}

// Ejemplo: 2023-01-01
func DateStringToUnix(h string) int16 {
	if len(h) > 10 {
		h = h[0:10]
	}
	dateT, error := time.Parse("2006-01-02", h)
	if error != nil {
		Log(error)
		return 0
	}
	return TimeToFechaUnix(dateT)
}

func DateTimeUnixToFecha(dateTime int64) int16 {
	tm := time.Unix(dateTime, 0)
	dateT, error := time.Parse("2006-01-02", tm.Format(TimeLayouts["D"]))

	if error != nil {
		Log(error)
		return 0
	}
	_, offset := dateT.Zone()
	dateUnix := int16((dateT.Unix() + int64(offset)) / 24 / 60 / 60)
	return dateUnix
}

func DateTimeUnixToFechaP(dateTime int64) int16 {
	dateT := DateUnixToTime(dateTime)
	// zona, _ := time.Now().Zone()
	// if zona == "UTC" {
	// }
	dateT = dateT.Add(-5 * time.Hour)
	dateUnix := int16((dateT.Unix()) / 24 / 60 / 60)
	return dateUnix
}

func DateTimeUnixToFechaX(dateTime int64) int16 {
	dateUnix := int16((dateTime) / 24 / 60 / 60)
	return dateUnix
}

func UnixTimeToFormat(unixTime int64, layout string) string {
	if unixTime == 0 {
		return ""
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	inputTime := time.Unix(unixTime, 0)
	return inputTime.UTC().Format(layout)
}

var TimeLayouts map[string]string = map[string]string{
	"A": "2006-01-02 15:04:05",
	"B": "2006-01-02 15:04",
	"C": time.RFC1123Z,
	"D": "2006-01-02",
	"E": "2006-01-02 15",
	"F": "2006/01/02 15:04:05",
	"G": "02/01/06",
	"H": "Mon, 2 Jan 2006 15:04:05 -0700",
	"I": "15:04:05",
}

func DateTimeStringToUnix(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}

	date, error := time.Parse(layout, h)
	if error != nil {
		Log(error)
		Log("date a parsear::", h, " | layout: ", layout)
		return 0
	}
	return date.Unix()
}

func DateTimeStringToUnixP(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}

	date, error := time.ParseInLocation(layout, h, time.Now().Location())
	if error != nil {
		Log(error)
		Log("date a parsear::", h, " | layout: ", layout)
		return 0
	}

	//_, offset := date.Zone()

	dUnix := date.Unix() // + int64(offset)

	return dUnix
}

func DateTimeStringToUnixI(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}

	date, error := time.Parse(layout, h)
	if error != nil {
		Log(error)
		Log("date a parsear::", h, " | layout: ", layout)
		return 0
	}

	/*zone, _ := date.Zone()
	fmt.Println("Zone", zone)
	if zone != "UTC" {
	}*/
	date = date.Add(5 * time.Hour)

	return date.Unix()
}

func DateUnixToTime(fUnix int64) time.Time {
	tm := time.Unix(fUnix, 0)
	return tm
}

func DateUnixToTimeP(fUnix int64) time.Time {
	tm := time.Unix(fUnix, 0)

	return tm
}

func DateTimeToFormat(layout string, dateTime time.Time) string {
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	return dateTime.Format(layout)
}

func DateTimeToFormatP(layout string, dateTime time.Time) string {
	zona, _ := time.Now().Zone()
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	if zona == "UTC" {
		dateTime = dateTime.Add(-5 * time.Hour)
	}

	return dateTime.Format(layout)
}

func DateTimeStringToUnixUTCAdd05_30(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	date, error := time.Parse(layout, h)
	if error != nil {
		Log(error)
		Log("date a parsear::", h, " | layout: ", layout)
		return 0
	}
	return date.Unix()
}

// Ejemplo: 07:30:02
func HoraToUnix(h string) int {
	if h == "" {
		return 0
	}
	return int(DateTimeStringToUnix("A", "1970-01-01 "+h))
}

func HoraToUnixTime(h string) int64 {
	date, error := time.Parse("2006-01-02 15:04:05", h)
	if error != nil {
		Log(error)
		return 0
	}
	return date.Unix()
}

// Basado en: https://github.com/snabb/isoweek/
// startOffset returns the offset (in days) from the start of a year to
// Monday of the given week. Offset may be negative
// StartDate returns the starting date (Monday) of the given ISO 8601 week.
func WeekStartDate(wyear, week int) time.Time {
	year, month, day := JulianToDate(DateToJulian(wyear, 1, 1) + WeekStartOffset(wyear, week))
	dayTime := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	return dayTime
}

func WeekStartOffset(y, week int) (offset int) {
	// This is optimized version of the following:
	// return week*7 - ISOWeekday(y, 1, 4) - 3
	// Uses Tomohiko Sakamoto's algorithm for calculating the weekday.
	y = y - 1
	return week*7 - (y+y/4-y/100+y/400+3)%7 - 4
}

// DateToJulian converts a date to a Julian day number.
func DateToJulian(year int, month time.Month, day int) (dayNo int) {
	// Claus Tøndering's Calendar FAQ
	if month < 3 {
		year = year - 1
		month = month + 12
	}
	year = year + 4800

	return day + (153*(int(month)-3)+2)/5 + 365*year +
		year/4 - year/100 + year/400 - 32045
}

// JulianToDate converts a Julian day number to a date.
func JulianToDate(dayNo int) (year int, month time.Month, day int) {
	// Richards, E. G. (2013) pp. 585–624
	e := 4*(dayNo+1401+(4*dayNo+274277)/146097*3/4-38) + 3
	h := e%1461/4*5 + 2

	day = h%153/5 + 1
	month = time.Month((h/153+2)%12 + 1)
	year = e/1461 - 4716 + (14-int(month))/12

	return year, month, day
}
