package types

import (
	"app/core"
	"app/db"
	"fmt"
	"math"
	"time"
)

const (
	// ExchangeRateDaysPerMonth is the longest month, which is how many rates a row can carry.
	ExchangeRateDaysPerMonth = 31
	// ExchangeRateScale is the factor a rate is stored with: 3 decimals as an integer,
	// so 3.752 travels and is stored as 3752.
	ExchangeRateScale = 1000
	// ExchangeRateMax is the ceiling a single rate may reach (1000.000). A rate above it is a
	// typo — the point is to reject a misplaced decimal separator before it reaches accounting.
	ExchangeRateMax = 1000 * ExchangeRateScale
)

// ExchangeRate holds one calendar month of USD → PEN rates: one row per month, per company.
//
// The day is the position inside the parallel arrays — index 0 is day 1 — and a rate is stored
// as rate × 1000. A 0 means the day has no rate: a weekend, a holiday, or a day nobody has
// filled in yet. Trailing empty days are trimmed by the client, so a month loaded up to the
// 12th stores 12 numbers, not 31.
type ExchangeRate struct {
	db.TableStruct[ExchangeRateTable, ExchangeRate]
	CompanyID int32 `json:",omitempty"`
	// ID is the month in YYMM form: 2601 is January 2026. One row per month means the month
	// itself is the identity, so nothing is autoincremented here.
	ID             int16   `json:",omitempty"`
	DetailBuyRate  []int32 `json:",omitempty" db:",list"`
	DetailSellRate []int32 `json:",omitempty" db:",list"`
	Status         int8    `json:"ss,omitempty"`
	Updated        int32   `json:"upd,omitempty"`
	UpdatedVersion int32   `json:"upv,omitempty"`
	UpdatedBy      int32   `json:",omitempty"`
	Created        int32   `json:",omitempty"`
}

type ExchangeRateTable struct {
	db.TableStruct[ExchangeRateTable, ExchangeRate]
	CompanyID      db.Col[*ExchangeRateTable, int32]
	ID             db.Col[*ExchangeRateTable, int16]
	DetailBuyRate  db.Col[*ExchangeRateTable, []int32]
	DetailSellRate db.Col[*ExchangeRateTable, []int32]
	Status         db.Col[*ExchangeRateTable, int8]
	Updated        db.Col[*ExchangeRateTable, int32]
	UpdatedVersion db.Col[*ExchangeRateTable, int32]
	UpdatedBy      db.Col[*ExchangeRateTable, int32]
	Created        db.Col[*ExchangeRateTable, int32]
}

func (e ExchangeRateTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        56,
		Name:      "exchange_rates",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID),
		// Delta() enumerates its filter column, so every Status value must be declared.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Values: []int64{0, 1}},
		},
		Indexes: []db.Index{
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// IsValidExchangeRateMonth reports whether an ID is a YYMM month key the table accepts:
// a year from 2000 on and a month between January and December.
func IsValidExchangeRateMonth(monthKey int16) bool {
	return monthKey >= 1 && monthKey%100 >= 1 && monthKey%100 <= 12
}

// The currency codes a cash-bank, a product and a sale share.
const (
	CurrencyPEN int8 = 1
	CurrencyUSD int8 = 2
)

// ExchangeRateMaxAgeDays is how old the latest sell rate may be before a sale that needs it is
// refused: converting at a rate from weeks ago would quietly misprice the sale.
const ExchangeRateMaxAgeDays = 7

// LatestSellRate returns the sell rate (× 1000) of the most recent day that has one, looking back
// no further than maxAgeDays from today. A day reads the way the exchange-rate calendar shows it:
// the company's own rate when it loaded one, the published BCRP interbank rate otherwise.
func LatestSellRate(companyID int32, today int16, maxAgeDays int16) (int32, error) {
	oldestDay := today - maxAgeDays
	monthKeys := core.SliceSet[int16]{}
	monthKeys.Add(exchangeRateMonthKeyOfDay(oldestDay))
	monthKeys.Add(exchangeRateMonthKeyOfDay(today))

	records := []ExchangeRate{}
	query := db.Query(&records)
	query.Select(query.ID, query.DetailSellRate, query.Status).
		CompanyID.Equals(companyID).
		ID.In(monthKeys.Values...)
	if err := query.Exec(); err != nil {
		return 0, core.Err("Error al obtener el tipo de cambio:", err)
	}

	companySellRatesByMonthKey := map[int16][]int32{}
	for _, record := range records {
		if record.Status > 0 {
			companySellRatesByMonthKey[record.ID] = record.DetailSellRate
		}
	}

	bcrpYears := core.SliceSet[int]{}
	bcrpYears.Add(unixDayToUTCDate(oldestDay).Year())
	bcrpYears.Add(unixDayToUTCDate(today).Year())
	bcrpSellRateByDay := bcrpSellRatesByDay(bcrpYears.Values...)

	if sellRate := latestSellRateInWindow(companySellRatesByMonthKey, bcrpSellRateByDay, today, oldestDay); sellRate > 0 {
		return sellRate, nil
	}
	return 0, fmt.Errorf("No hay un tipo de cambio de venta (propio ni BCRP) en los últimos %v días. "+
		"Regístrelo en Finanzas > Tipo de cambio.", maxAgeDays)
}

// latestSellRateInWindow walks back from today to oldestDay and returns the first day's sell
// rate — the company's when it loaded one, BCRP's otherwise — or 0 when the whole window is empty.
func latestSellRateInWindow(companySellRatesByMonthKey map[int16][]int32,
	bcrpSellRateByDay map[int16]int32, today int16, oldestDay int16) int32 {

	for day := today; day >= oldestDay; day-- {
		dayOfMonthIndex := unixDayToUTCDate(day).Day() - 1
		companySellRates := companySellRatesByMonthKey[exchangeRateMonthKeyOfDay(day)]
		if dayOfMonthIndex < len(companySellRates) && companySellRates[dayOfMonthIndex] > 0 {
			return companySellRates[dayOfMonthIndex]
		}
		if bcrpSellRateByDay[day] > 0 {
			return bcrpSellRateByDay[day]
		}
	}
	return 0
}

// EffectiveExchangeRate is the rate a conversion into targetCurrency uses. The spread always
// favours the store, so it moves the sell rate in opposite directions: charging in dollars
// divides by a lower rate (more dollars), charging in soles multiplies by a higher one (more
// soles). Both values are × 1000.
func EffectiveExchangeRate(sellRate int32, spread int32, targetCurrency int8) (int32, error) {
	effectiveRate := sellRate + spread
	if targetCurrency == CurrencyUSD {
		effectiveRate = sellRate - spread
	}
	if effectiveRate <= 0 {
		return 0, fmt.Errorf("El spread de tipo de cambio (%v) no puede ser mayor o igual al tipo de cambio (%v).",
			float64(spread)/ExchangeRateScale, float64(sellRate)/ExchangeRateScale)
	}
	return effectiveRate, nil
}

// ConvertUnitPrice restates one unit price (cents) from priceCurrency into saleCurrency with
// an effective rate (× 1000). It converts a UNIT price, rounded to the cent, so a line is
// always quantity × the price the ticket prints and the lines add up to the total.
func ConvertUnitPrice(price int32, priceCurrency int8, saleCurrency int8, effectiveRate int32) int32 {
	if priceCurrency == saleCurrency || price == 0 {
		return price
	}
	if saleCurrency == CurrencyUSD {
		return int32(math.Round(float64(price) * ExchangeRateScale / float64(effectiveRate)))
	}
	return int32(math.Round(float64(price) * float64(effectiveRate) / ExchangeRateScale))
}

// exchangeRateMonthKeyOfDay is the YYMM key of the row a UnixDay's rate is stored in.
func exchangeRateMonthKeyOfDay(day int16) int16 {
	date := unixDayToUTCDate(day)
	return int16((date.Year()-2000)*100 + int(date.Month()))
}

// unixDayToUTCDate reads a UnixDay as the calendar date it names. UTC on purpose: a UnixDay
// is already a local date, so applying a zone again would shift it back a day.
func unixDayToUTCDate(day int16) time.Time {
	return time.Unix(int64(day)*86400, 0).UTC()
}
