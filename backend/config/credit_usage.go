package config

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"fmt"
	"math"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	creditUsageDaysCount = int32(15)
	dailyTimeFramePrefix = int32(200_000_000)
	secondsPerUTCDate    = int64(86_400)
	companyAggregateID   = int32(-1)
)

type creditUsageTotals struct {
	CPU       uint64
	Inference uint64
}

type creditUsageDay struct {
	Day       int16
	CPU       uint64
	Inference uint64
}

type creditUsageScope struct {
	CPU24hLimit       uint64
	Inference24hLimit uint64
	Days              []creditUsageDay
}

type creditUsageResponse struct {
	User    creditUsageScope
	Company creditUsageScope
}

func GetCreditUsage(req *core.HandlerArgs) core.HandlerResponse {
	currentUnixDay := int32(time.Now().UTC().Unix() / secondsPerUTCDate)
	firstUnixDay := currentUnixDay - creditUsageDaysCount + 1
	firstTimeFrame := dailyTimeFramePrefix + firstUnixDay
	lastTimeFrame := dailyTimeFramePrefix + currentUnixDay
	userRows, companyRows := []coreTypes.CreditUsage{}, []coreTypes.CreditUsage{}

	core.Log("credit usage query started::", " company::", req.User.CompanyID,
		" user::", req.User.ID, " first_day::", firstUnixDay, " last_day::", currentUnixDay)
	queries := errgroup.Group{}
	queries.Go(func() error {
		query := db.Query(&userRows)
		query.CompanyID.Equals(req.User.CompanyID).UserID.Equals(req.User.ID).TimeFrame.Between(firstTimeFrame, lastTimeFrame)
		return query.Exec()
	})
	queries.Go(func() error {
		query := db.Query(&companyRows)
		query.CompanyID.Equals(req.User.CompanyID).UserID.Equals(companyAggregateID).TimeFrame.Between(firstTimeFrame, lastTimeFrame)
		return query.Exec()
	})
	if err := queries.Wait(); err != nil {
		core.Log("credit usage query failed::", " company::", req.User.CompanyID, " user::", req.User.ID, " err::", err)
		return req.MakeErr("No se pudo obtener el uso de créditos.", err)
	}

	userUsage, err := makeCreditUsageScope(userRows, firstUnixDay,
		core.Env.RATE_LIMIT_USER_CPU_24H, core.Env.RATE_LIMIT_USER_INFERENCE_24H)
	if err != nil {
		core.Log("credit usage user blob invalid::", " company::", req.User.CompanyID, " user::", req.User.ID, " err::", err)
		return req.MakeErr("No se pudo interpretar el uso de créditos del usuario.", err)
	}
	companyUsage, err := makeCreditUsageScope(companyRows, firstUnixDay,
		core.Env.RATE_LIMIT_COMPANY_CPU_24H, core.Env.RATE_LIMIT_COMPANY_INFERENCE_24H)
	if err != nil {
		core.Log("credit usage company blob invalid::", " company::", req.User.CompanyID, " err::", err)
		return req.MakeErr("No se pudo interpretar el uso de créditos de la empresa.", err)
	}

	core.Log("credit usage query completed::", " company::", req.User.CompanyID, " user::", req.User.ID,
		" user_rows::", len(userRows), " company_rows::", len(companyRows))
	return req.MakeResponse(creditUsageResponse{User: userUsage, Company: companyUsage})
}

func makeCreditUsageScope(rows []coreTypes.CreditUsage, firstUnixDay int32, cpuLimit, inferenceLimit uint64) (creditUsageScope, error) {
	// Zero-fill the fixed UTC range so chart columns never shift when a day has no usage.
	days := make([]creditUsageDay, creditUsageDaysCount)
	for dayOffset := range creditUsageDaysCount {
		days[dayOffset].Day = int16(firstUnixDay + dayOffset)
	}
	for _, row := range rows {
		dayOffset := row.TimeFrame - dailyTimeFramePrefix - firstUnixDay
		if dayOffset < 0 || dayOffset >= creditUsageDaysCount {
			return creditUsageScope{}, fmt.Errorf("daily time frame %d is outside the requested range", row.TimeFrame)
		}
		totals, err := decodeCreditUsage(row.UsedCredits)
		if err != nil {
			return creditUsageScope{}, fmt.Errorf("invalid credit blob in time frame %d: %w", row.TimeFrame, err)
		}
		days[dayOffset].CPU = totals.CPU
		days[dayOffset].Inference = totals.Inference
	}
	return creditUsageScope{CPU24hLimit: cpuLimit, Inference24hLimit: inferenceLimit, Days: days}, nil
}

func decodeCreditUsage(encoded []byte) (creditUsageTotals, error) {
	totals := creditUsageTotals{}
	previousAPIGroup := -1
	for offset := 0; offset < len(encoded); {
		header := encoded[offset]
		offset++
		apiGroup := int(header >> 2)
		valueWidth := int(header&0b11) + 1
		if apiGroup <= previousAPIGroup {
			return creditUsageTotals{}, fmt.Errorf("API groups are not strictly ascending")
		}
		if len(encoded)-offset < valueWidth*2 {
			return creditUsageTotals{}, fmt.Errorf("credit blob ends inside API group %d", apiGroup)
		}

		cpuCredits := readBigEndianCredit(encoded[offset : offset+valueWidth])
		offset += valueWidth
		inferenceCredits := readBigEndianCredit(encoded[offset : offset+valueWidth])
		offset += valueWidth
		if cpuCredits == 0 && inferenceCredits == 0 {
			return creditUsageTotals{}, fmt.Errorf("API group %d contains only zero values", apiGroup)
		}
		if smallestCreditWidth(max(cpuCredits, inferenceCredits)) != valueWidth {
			return creditUsageTotals{}, fmt.Errorf("API group %d is not canonically encoded", apiGroup)
		}
		if math.MaxUint64-totals.CPU < cpuCredits || math.MaxUint64-totals.Inference < inferenceCredits {
			return creditUsageTotals{}, fmt.Errorf("summed credits overflow uint64")
		}
		totals.CPU += cpuCredits
		totals.Inference += inferenceCredits
		previousAPIGroup = apiGroup
	}
	return totals, nil
}

func readBigEndianCredit(encoded []byte) uint64 {
	var value uint64
	for _, encodedByte := range encoded {
		value = value<<8 | uint64(encodedByte)
	}
	return value
}

func smallestCreditWidth(value uint64) int {
	switch {
	case value <= math.MaxUint8:
		return 1
	case value <= math.MaxUint16:
		return 2
	case value <= 0xFF_FFFF:
		return 3
	default:
		return 4
	}
}
