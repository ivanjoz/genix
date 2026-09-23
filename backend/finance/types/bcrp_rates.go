package types

// The BCRP interbank USD→PEN series, the default rate of every day the company did not load
// itself — the same source the exchange-rate calendar shows in its empty cells. It is public
// data published as static files (github.com/ivanjoz/public-business-data), one gzipped file per
// year, so the server reads it the way the browser does and a sale is priced at the rate the
// operator saw.

import (
	"app/core"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	bcrpYearURL = "https://public-business-data.un.pe/bcrp-interbancario-usd-pen/%d.gz"
	// bcrpRecordSize is one published day: int16 unixDay + int32 buy + int32 sell, little endian.
	bcrpRecordSize = 10
	// A past year never changes, but the current one gains a day every working day.
	bcrpCurrentYearTTL = time.Hour
)

var bcrpHTTPClient = &http.Client{Timeout: 5 * time.Second}

type bcrpYearCacheEntry struct {
	sellRateByDay map[int16]int32
	fetchedAt     time.Time
}

var bcrpYearCache = struct {
	sync.Mutex
	byYear map[int]bcrpYearCacheEntry
}{byYear: map[int]bcrpYearCacheEntry{}}

// bcrpSellRatesByDay returns the published sell rates (× 1000) by UnixDay for the given years.
// A year that cannot be read is logged and skipped: the series is a default, and the company's
// own rates must keep working when the source is down.
func bcrpSellRatesByDay(years ...int) map[int16]int32 {
	sellRateByDay := map[int16]int32{}
	for _, year := range years {
		yearRates, err := loadBcrpYear(year)
		if err != nil {
			core.Log("BCRP: no se pudo leer el tipo de cambio del año", year, ":", err)
			continue
		}
		for day, sellRate := range yearRates {
			sellRateByDay[day] = sellRate
		}
	}
	return sellRateByDay
}

func loadBcrpYear(year int) (map[int16]int32, error) {
	isCurrentYear := year == core.Now().Year()

	bcrpYearCache.Lock()
	cached, found := bcrpYearCache.byYear[year]
	bcrpYearCache.Unlock()
	if found && (!isCurrentYear || time.Since(cached.fetchedAt) < bcrpCurrentYearTTL) {
		return cached.sellRateByDay, nil
	}

	response, err := bcrpHTTPClient.Get(fmt.Sprintf(bcrpYearURL, year))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("respuesta %v", response.StatusCode)
	}
	gzipped, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	sellRateByDay, err := decodeBcrpYear(gzipped)
	if err != nil {
		return nil, err
	}

	bcrpYearCache.Lock()
	bcrpYearCache.byYear[year] = bcrpYearCacheEntry{sellRateByDay: sellRateByDay, fetchedAt: time.Now()}
	bcrpYearCache.Unlock()
	return sellRateByDay, nil
}

// decodeBcrpYear unpacks one published year file. The host serves it as raw gzip bytes (no
// Content-Encoding), so it is decompressed here.
func decodeBcrpYear(gzipped []byte) (map[int16]int32, error) {
	reader, err := gzip.NewReader(bytes.NewReader(gzipped))
	if err != nil {
		return nil, err
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(payload)%bcrpRecordSize != 0 {
		return nil, fmt.Errorf("tamaño %v no es múltiplo de %v bytes", len(payload), bcrpRecordSize)
	}

	sellRateByDay := make(map[int16]int32, len(payload)/bcrpRecordSize)
	for offset := 0; offset < len(payload); offset += bcrpRecordSize {
		day := int16(binary.LittleEndian.Uint16(payload[offset:]))
		sellRate := int32(binary.LittleEndian.Uint32(payload[offset+6:]))
		sellRateByDay[day] = sellRate
	}
	return sellRateByDay, nil
}
