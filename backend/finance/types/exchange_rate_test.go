package types

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"testing"
)

// 20718 is 2026-09-22 and 20698 is 2026-09-02 as UnixDays.
const (
	testDaySep22 = int16(20718)
	testDaySep02 = int16(20698)
)

func TestLatestSellRateInWindowTakesTheMostRecentFilledDay(t *testing.T) {
	sellRates := make([]int32, 22)
	sellRates[17] = 3740 // Sep 18
	sellRates[19] = 3752 // Sep 20; 21 and 22 are empty (a weekend nobody loaded)
	rate := latestSellRateInWindow(map[int16][]int32{2609: sellRates}, nil, testDaySep22, testDaySep22-7)
	if rate != 3752 {
		t.Fatalf("rate = %v, want 3752", rate)
	}
}

func TestLatestSellRateInWindowCrossesIntoThePreviousMonth(t *testing.T) {
	augustRates := make([]int32, 31)
	augustRates[30] = 3701 // Aug 31
	rate := latestSellRateInWindow(map[int16][]int32{2608: augustRates, 2609: {0, 0}}, nil, testDaySep02, testDaySep02-7)
	if rate != 3701 {
		t.Fatalf("rate = %v, want 3701", rate)
	}
}

func TestLatestSellRateInWindowRefusesARateOlderThanTheWindow(t *testing.T) {
	sellRates := make([]int32, 22)
	sellRates[13] = 3740 // Sep 14: eight days before the 22nd
	if rate := latestSellRateInWindow(map[int16][]int32{2609: sellRates}, nil, testDaySep22, testDaySep22-7); rate != 0 {
		t.Fatalf("rate = %v, want 0", rate)
	}
}

func TestLatestSellRateInWindowFallsBackToBcrpDayByDay(t *testing.T) {
	sellRates := make([]int32, 22)
	sellRates[19] = 3752 // the company loaded Sep 20
	bcrpRates := map[int16]int32{testDaySep22: 3371, testDaySep22 - 2: 3360}

	// Sep 22 has no company rate, so BCRP's is the newest day with any rate.
	if rate := latestSellRateInWindow(map[int16][]int32{2609: sellRates}, bcrpRates, testDaySep22, testDaySep22-7); rate != 3371 {
		t.Fatalf("rate = %v, want 3371", rate)
	}
	// On a day the company loaded, its rate wins over BCRP's.
	if rate := latestSellRateInWindow(map[int16][]int32{2609: sellRates}, bcrpRates, testDaySep22-2, testDaySep22-9); rate != 3752 {
		t.Fatalf("rate = %v, want 3752", rate)
	}
}

func TestDecodeBcrpYearReadsLittleEndianRecords(t *testing.T) {
	payload := make([]byte, 0, 20)
	for _, record := range []struct {
		day       int16
		buy, sell int32
	}{{testDaySep02, 3356, 3358}, {testDaySep22, 3369, 3371}} {
		payload = binary.LittleEndian.AppendUint16(payload, uint16(record.day))
		payload = binary.LittleEndian.AppendUint32(payload, uint32(record.buy))
		payload = binary.LittleEndian.AppendUint32(payload, uint32(record.sell))
	}
	var gzipped bytes.Buffer
	writer := gzip.NewWriter(&gzipped)
	writer.Write(payload)
	writer.Close()

	sellRateByDay, err := decodeBcrpYear(gzipped.Bytes())
	if err != nil || sellRateByDay[testDaySep22] != 3371 || sellRateByDay[testDaySep02] != 3358 {
		t.Fatalf("decodeBcrpYear = %v, %v", sellRateByDay, err)
	}
}

func TestEffectiveExchangeRateAlwaysFavoursTheStore(t *testing.T) {
	toUSD, _ := EffectiveExchangeRate(3750, 20, CurrencyUSD)
	toPEN, _ := EffectiveExchangeRate(3750, 20, CurrencyPEN)
	if toUSD != 3730 || toPEN != 3770 {
		t.Fatalf("toUSD = %v, toPEN = %v, want 3730 and 3770", toUSD, toPEN)
	}
	if _, err := EffectiveExchangeRate(3750, 3750, CurrencyUSD); err == nil {
		t.Fatal("a spread as large as the rate must be refused")
	}
}

func TestConvertUnitPriceRoundsEachUnitToTheCent(t *testing.T) {
	cases := []struct {
		price         int32
		priceCurrency int8
		saleCurrency  int8
		rate          int32
		want          int32
	}{
		{1000, CurrencyPEN, CurrencyUSD, 3730, 268}, // S/ 10.00 / 3.730 = 2.6809
		{268, CurrencyUSD, CurrencyPEN, 3770, 1010}, // $ 2.68 × 3.770 = 10.1036
		{1000, CurrencyUSD, CurrencyUSD, 3730, 1000},
		{1000, CurrencyPEN, CurrencyPEN, 3770, 1000},
	}
	for _, testCase := range cases {
		got := ConvertUnitPrice(testCase.price, testCase.priceCurrency, testCase.saleCurrency, testCase.rate)
		if got != testCase.want {
			t.Errorf("ConvertUnitPrice(%v, %v→%v @ %v) = %v, want %v", testCase.price,
				testCase.priceCurrency, testCase.saleCurrency, testCase.rate, got, testCase.want)
		}
	}
}
