package finance

import (
	"app/core"
	"app/db"
	"app/finance/types"
	"encoding/json"
	"fmt"
)

func GetExchangeRates(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetUpVersion()

	records := []types.ExchangeRate{}

	// Delta() keeps only active rows on a first sync and every status afterwards, so the client
	// can evict the months that were deleted.
	query := db.Query(&records)
	query.CompanyID.Equals(req.User.CompanyID).Delta(updatedSince, 1)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los tipos de cambio:", err)
	}

	return req.MakeResponse(records)
}

func PostExchangeRates(req *core.HandlerArgs) core.HandlerResponse {
	payload := []types.ExchangeRate{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar los tipos de cambio:", err)
	}

	for i := range payload {
		record := &payload[i]
		if !types.IsValidExchangeRateMonth(record.ID) {
			return req.MakeErr(fmt.Sprintf("El mes enviado no es válido (formato AAMM, ej. 2601): %v", record.ID))
		}
		if err := validateExchangeRateDays("compra", record.DetailBuyRate); err != nil {
			return req.MakeErr(err.Error())
		}
		if err := validateExchangeRateDays("venta", record.DetailSellRate); err != nil {
			return req.MakeErr(err.Error())
		}
		record.CompanyID = req.User.CompanyID
	}

	nowTime := core.SUnixTime()
	t := types.ExchangeRateTable{}
	err := db.Merge(&payload,
		db.Cols(t.Created),
		func(prev, current *types.ExchangeRate) bool {
			current.CompanyID = req.User.CompanyID
			current.Created = prev.Created
			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			return true
		},
		func(current *types.ExchangeRate) {
			current.CompanyID = req.User.CompanyID
			current.Created = nowTime
			current.Updated = nowTime
			current.UpdatedBy = req.User.ID
			if current.Status == 0 {
				current.Status = 1
			}
		},
	)
	if err != nil {
		return req.MakeErr("Error al guardar los tipos de cambio:", err)
	}

	return req.MakeResponse(payload)
}

// validateExchangeRateDays rejects a month that carries more days than a month has, or a rate
// that is negative or so large it can only be a misplaced decimal separator.
func validateExchangeRateDays(rateName string, rates []int32) error {
	if len(rates) > types.ExchangeRateDaysPerMonth {
		return fmt.Errorf("El tipo de cambio %v tiene %v días, el máximo es %v",
			rateName, len(rates), types.ExchangeRateDaysPerMonth)
	}
	for dayIndex, rate := range rates {
		if rate < 0 || rate > types.ExchangeRateMax {
			return fmt.Errorf("El tipo de cambio %v del día %v no es válido: %v",
				rateName, dayIndex+1, rate)
		}
	}
	return nil
}
