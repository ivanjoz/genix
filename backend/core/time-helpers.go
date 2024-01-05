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
	FechaUnix   int16
	Nro         uint8
	Year        int16
	FechaString string
	Idx         int16
}

func (e *FecSemana) MakeName() string {
	s2 := strconv.Itoa(int(e.Nro))
	if len(s2) == 1 {
		s2 = "0" + s2
	}
	return Concat("-", e.Year, s2)
}

var fechaToSemana = map[int16]*FecSemana{}

// Si fechaUnix = 0 entonces se obtiene la semana actual
func MakeSemanaFromFechaUnix(fechaUnix int16, isMonday bool) *FecSemana {
	if fecSemana, ok := fechaToSemana[fechaUnix]; ok {
		return fecSemana
	}
	if fechaUnix == 0 {
		fechaUnix = TimeToFechaUnix(time.Now())
	}

	// convierte la fechaUnix en TimeStamp
	fechaUnixTime := int64(fechaUnix)*int64(24*60*60) + int64(8*60*60)
	fecha := time.Unix(fechaUnixTime, 0)

	fechaUnixBase := fechaUnix

	if !isMonday {
		weekday := int(fecha.Weekday())
		// Log("weekday:: ", weekday)
		if weekday == 0 {
			fecha = fecha.AddDate(0, 0, -6)
			fechaUnix = fechaUnix - 6
		} else if weekday != 1 {
			fecha = fecha.AddDate(0, 0, (1 - weekday))
			fechaUnix = fechaUnix + 1 - int16(weekday)
		}
	}

	year, week := fecha.ISOWeek()

	fechSemana := FecSemana{
		Year: int16(year), Nro: uint8(week), Id: year*100 + week,
		FechaUnix:   fechaUnix,
		FechaString: fecha.Format("2006-01-02"),
	}
	fechSemana.Code = int16(fechSemana.Id - 200000)

	fechaToSemana[fechaUnixBase] = &fechSemana
	return &fechSemana
}

// Crea un array de semanas basados en una fecha unix
func GetSemanasFromFecha(fechaUnix int16, incremento uint8, decremento uint8) []*FecSemana {

	if fechaUnix == 0 {
		fechaUnix = int16(time.Now().Unix() / 60 / 60 / 24)
	}

	// convierte la fechaUnix en TimeStamp
	fechaUnixTime := int64(fechaUnix)*int64(24*60*60) + int64(8*60*60)
	fecha := time.Unix(fechaUnixTime, 0)

	weekday := int(fecha.Weekday())
	if weekday == 0 {
		// fecha = fecha.AddDate(0, 0, -6)
		fechaUnix = fechaUnix - 6
	} else if weekday != 1 {
		// fecha = fecha.AddDate(0, 0, (weekday - 1))
		fechaUnix = fechaUnix - (int16(weekday) - 1)
	}

	fechasSemana := []*FecSemana{
		MakeSemanaFromFechaUnix(fechaUnix, true),
	}

	// Obtiene las semanas del incremento
	for i := 1; i <= int(incremento); i++ {
		fechaUnix2 := fechaUnix + int16(i*7)
		semana := MakeSemanaFromFechaUnix(fechaUnix2, true)
		fechasSemana = append(fechasSemana, semana)
	}

	// Obtiene las semanas del decremento
	for i := 1; i <= int(decremento); i++ {
		fechaUnix2 := fechaUnix - int16(i*7)
		semana := MakeSemanaFromFechaUnix(fechaUnix2, true)
		fechasSemana = append(fechasSemana, semana)
	}

	sort.Slice(fechasSemana, func(i, j int) bool {
		return fechasSemana[i].Id < fechasSemana[j].Id
	})

	return fechasSemana
}

type TimeHelper struct {
	init          bool
	fechaToSemana map[int16]*FecSemana
	codeToSemana  map[int16]*FecSemana
	fechaToString map[int16]string
	semanas       []*FecSemana
	idxToSemana   map[int16]*FecSemana
	semanaToIdx   map[int16]int16
	maxSemana     int16
	minSemana     int16
}

func (e *TimeHelper) Init() {
	if !e.init {
		e.fechaToSemana = map[int16]*FecSemana{}
		e.codeToSemana = map[int16]*FecSemana{}
		e.fechaToString = map[int16]string{}
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
	fechaUnix := int16((nowTime.Unix() + int64(offset)) / 60 / 60 / 24)
	return fechaUnix
}

func (e *TimeHelper) ExtendRangeToSemana(semanaCode, offset int16) {
	if offset == 0 || semanaCode == 0 {
		return
	}
	if semanaCode < 1001 || semanaCode > 3001 {
		panic(Concats("semana inválida:: ", semanaCode))
	}

	semanaFrom := e.SemanaFromCode(semanaCode)
	semanaTo := e.SemanaFromFecha(semanaFrom.FechaUnix + (offset * 7))
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
		semana := e.SemanaFromFecha(semanaFrom.FechaUnix + (it * 7))
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

	return Round(float32(semanaInicio.FechaUnix-semanaFin.FechaUnix) / 7)
}

func (e *TimeHelper) GetFechaUnixP() int16 {
	nowTime := time.Now()
	zone, offset := nowTime.Zone()
	nowUnix := nowTime.Unix()
	if zone == "UTC" {
		nowUnix -= 18000
	}
	fechaUnix := int16((nowUnix + int64(offset)) / 60 / 60 / 24)
	return fechaUnix
}

func (e *TimeHelper) GetFechaUnixI() int16 {
	nowTime := time.Now()
	zone, offset := nowTime.Zone()
	nowUnix := nowTime.Unix()
	if zone == "UTC" {
		nowUnix += 19800
	}
	fechaUnix := int16((nowUnix + int64(offset)) / 60 / 60 / 24)
	return fechaUnix
}

func (e *TimeHelper) TimeToFechaUnix(fechaT time.Time) int16 {
	return TimeToFechaUnix(fechaT)
}

func (e *TimeHelper) FechaToTime(fechaUnix int16) time.Time {
	return time.Unix(int64(fechaUnix)*24*60*60, 0)
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
	fechaLunes := WeekStartDate(year, semanaNro)
	// Log(fechaLunes)

	fecSemana := FecSemana{
		Id:          (year)*100 + semanaNro,
		Code:        semanaCode,
		FechaUnix:   TimeToFechaUnix(fechaLunes),
		Nro:         uint8(semanaNro),
		Year:        int16(year),
		FechaString: fechaLunes.Local().UTC().Format("2006-01-02"),
	}
	e.codeToSemana[semanaCode] = &fecSemana
	return &fecSemana
}

func (e *TimeHelper) FechaToTimeUTC(fechaUnix int16) time.Time {
	_, offset := (time.Now()).Zone()
	var unixTime int64 = int64(fechaUnix)*24*60*60 - int64(offset)
	return time.Unix(unixTime, 0)
}

func (e *TimeHelper) FechaUnixToStringU(fechaUnix int16, format ...int8) string {
	if fechaUnix == 0 {
		return ""
	}
	if e.fechaToString == nil {
		e.fechaToString = map[int16]string{}
	} else if _, ok := e.fechaToString[fechaUnix]; ok {
		return e.fechaToString[fechaUnix]
	}
	fechaTime := time.Unix(int64(fechaUnix)*24*60*60, 0)
	parseFormat := "2006-01-02"
	if len(format) > 0 && format[0] == 2 {
		parseFormat = "02-01-2006"
	}

	fechaString := fechaTime.Local().UTC().Format(parseFormat)
	e.fechaToString[fechaUnix] = fechaString
	return fechaString
}

func (e *TimeHelper) FechaUnixToString(fechaUnix int16, format ...int8) string {
	return e.FechaUnixToStringU(int16(fechaUnix), format...)
}

func (e *TimeHelper) FechaUnixToStringP(fechaUnix int16) string {
	e.Init()
	if _, ok := e.fechaToString[fechaUnix]; ok {
		return e.fechaToString[fechaUnix]
	}
	fechaTime := time.Unix(int64(fechaUnix)*24*60*60, 0)
	fechaString := fechaTime.Local().UTC().Format("2006-01-02")
	e.fechaToString[fechaUnix] = fechaString
	return fechaString
}

func (e *TimeHelper) SemanaFromFecha(fecha int16) *FecSemana {
	e.Init()
	if _, ok := e.fechaToSemana[fecha]; ok {
		return e.fechaToSemana[fecha]
	}

	semana := MakeSemanaFromFechaUnix(fecha, false)
	e.fechaToSemana[fecha] = semana

	return e.fechaToSemana[fecha]
}

func UnixTimeToFechaZone(unixTime int64, utcTimeZone int8) int16 {
	zoneTime := unixTime + int64(utcTimeZone)*60*60
	return int16(math.Floor(float64(zoneTime) / 60 / 60 / 24))
}

func TimeToFechaUnix(fechaT time.Time) int16 {
	_, offset := fechaT.Zone()
	fechaUnix := int16((fechaT.Unix() + int64(offset)) / 24 / 60 / 60)
	return fechaUnix
}

// Ejemplo: 2023-01-01
func FechaStringToUnix(h string) int16 {
	if len(h) > 10 {
		h = h[0:10]
	}
	fechaT, error := time.Parse("2006-01-02", h)
	if error != nil {
		Log(error)
		return 0
	}
	return TimeToFechaUnix(fechaT)
}

func FechaHoraUnixToFecha(fechaHora int64) int16 {
	tm := time.Unix(fechaHora, 0)
	fechaT, error := time.Parse("2006-01-02", tm.Format(TimeLayouts["D"]))

	if error != nil {
		Log(error)
		return 0
	}
	_, offset := fechaT.Zone()
	fechaUnix := int16((fechaT.Unix() + int64(offset)) / 24 / 60 / 60)
	return fechaUnix
}

func FechaHoraUnixToFechaP(fechaHora int64) int16 {
	fechaT := FechaUnixToTime(fechaHora)
	// zona, _ := time.Now().Zone()
	// if zona == "UTC" {
	// }
	fechaT = fechaT.Add(-5 * time.Hour)
	fechaUnix := int16((fechaT.Unix()) / 24 / 60 / 60)
	return fechaUnix
}

func FechaHoraUnixToFechaX(fechaHora int64) int16 {
	fechaUnix := int16((fechaHora) / 24 / 60 / 60)
	return fechaUnix
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

func FechaHoraStringToUnix(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}

	date, error := time.Parse(layout, h)
	if error != nil {
		Log(error)
		Log("fecha a parsear::", h, " | layout: ", layout)
		return 0
	}
	return date.Unix()
}

func FechaHoraStringToUnixP(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}

	date, error := time.ParseInLocation(layout, h, time.Now().Location())
	if error != nil {
		Log(error)
		Log("fecha a parsear::", h, " | layout: ", layout)
		return 0
	}

	//_, offset := date.Zone()

	dUnix := date.Unix() // + int64(offset)

	return dUnix
}

func FechaHoraStringToUnixI(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}

	date, error := time.Parse(layout, h)
	if error != nil {
		Log(error)
		Log("fecha a parsear::", h, " | layout: ", layout)
		return 0
	}

	/*zone, _ := date.Zone()
	fmt.Println("Zone", zone)
	if zone != "UTC" {
	}*/
	date = date.Add(5 * time.Hour)

	return date.Unix()
}

func FechaUnixToTime(fUnix int64) time.Time {
	tm := time.Unix(fUnix, 0)
	return tm
}

func FechaUnixToTimeP(fUnix int64) time.Time {
	tm := time.Unix(fUnix, 0)

	return tm
}

func FechaTimeToFormat(layout string, fechaHora time.Time) string {
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	return fechaHora.Format(layout)
}

func FechaTimeToFormatP(layout string, fechaHora time.Time) string {
	zona, _ := time.Now().Zone()
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	if zona == "UTC" {
		fechaHora = fechaHora.Add(-5 * time.Hour)
	}

	return fechaHora.Format(layout)
}

func FechaHoraStringToUnixUTCAdd05_30(layout, h string) int64 {
	if h == "" {
		return 0
	}
	if newLayout, ok := TimeLayouts[layout]; ok {
		layout = newLayout
	}
	date, error := time.Parse(layout, h)
	if error != nil {
		Log(error)
		Log("fecha a parsear::", h, " | layout: ", layout)
		return 0
	}
	return date.Unix()
}

// Ejemplo: 07:30:02
func HoraToUnix(h string) int {
	if h == "" {
		return 0
	}
	return int(FechaHoraStringToUnix("A", "1970-01-01 "+h))
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
