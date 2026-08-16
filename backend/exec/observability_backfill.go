package exec

import (
	configTypes "app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	backfillDefaultHours      = int32(4)
	backfillMaxHours          = int32(24)
	backfillFramesPerHour     = int32(12)
	backfillFiveMinutePrefix  = int32(100_000_000)
	backfillCompanyUserID     = int32(-1)
	backfillPlatformCompanyID = int32(0)
)

type backfillCredits struct {
	cpu       uint64
	inference uint64
}

// BackfillObservabilityCredits builds the reserved platform rows from existing per-company
// absolute aggregates. Re-running it writes the same totals to the same keys.
func BackfillObservabilityCredits(args *core.ExecArgs) core.FuncResponse {
	hours, parseError := parseObservabilityBackfillHours(args.Message)
	if parseError != nil {
		return args.MakeErr(parseError)
	}

	companies := []configTypes.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(1).AllowFilter()
	if queryError := companyQuery.Exec(); queryError != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", queryError)
	}

	currentTimeFrame := backfillFiveMinutePrefix + int32(time.Now().UTC().Unix()/300)
	firstTimeFrame := currentTimeFrame - hours*backfillFramesPerHour + 1
	totalsByFrame := map[int32]map[int16]backfillCredits{}
	rowsRead := 0

	for _, company := range companies {
		rows := []coreTypes.CreditUsage{}
		query := db.Query(&rows)
		query.CompanyID.Equals(company.ID).UserID.Equals(backfillCompanyUserID).
			TimeFrame.Between(firstTimeFrame, currentTimeFrame)
		if queryError := query.Exec(); queryError != nil {
			return args.MakeErr(fmt.Sprintf("no se pudo leer credit_usage de CompanyID %d:", company.ID), queryError)
		}
		rowsRead += len(rows)
		for _, row := range rows {
			frameTotals := totalsByFrame[row.TimeFrame]
			if frameTotals == nil {
				frameTotals = map[int16]backfillCredits{}
				totalsByFrame[row.TimeFrame] = frameTotals
			}
			if decodeError := mergeBackfillCreditBlob(row.UsedCredits, frameTotals); decodeError != nil {
				return args.MakeErr(fmt.Sprintf("blob inválido en CompanyID %d frame %d:", company.ID, row.TimeFrame), decodeError)
			}
		}
	}

	platformRows := make([]coreTypes.CreditUsage, 0, len(totalsByFrame))
	for timeFrame, routeTotals := range totalsByFrame {
		encoded, encodeError := encodeBackfillCreditBlob(routeTotals)
		if encodeError != nil {
			return args.MakeErr(fmt.Sprintf("no se pudo codificar frame %d:", timeFrame), encodeError)
		}
		platformRows = append(platformRows, coreTypes.CreditUsage{
			CompanyID:   backfillPlatformCompanyID,
			UserID:      backfillCompanyUserID,
			TimeFrame:   timeFrame,
			UsedCredits: encoded,
		})
	}
	slices.SortFunc(platformRows, func(left, right coreTypes.CreditUsage) int {
		return int(left.TimeFrame - right.TimeFrame)
	})
	if len(platformRows) > 0 {
		if insertError := db.Insert(&platformRows); insertError != nil {
			return args.MakeErr("no se pudieron escribir los agregados de plataforma:", insertError)
		}
	}

	message := fmt.Sprintf(
		"Backfill observability completado: %d empresa(s), %d fila(s) leídas, %d frame(s) escritos.",
		len(companies), rowsRead, len(platformRows),
	)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

func parseObservabilityBackfillHours(rawArgument string) (int32, error) {
	trimmedArgument := strings.TrimSpace(rawArgument)
	if trimmedArgument == "" {
		return backfillDefaultHours, nil
	}
	hours64, parseError := strconv.ParseInt(trimmedArgument, 10, 32)
	if parseError != nil || hours64 <= 0 || hours64 > int64(backfillMaxHours) {
		return 0, fmt.Errorf("las horas deben ser un entero entre 1 y %d", backfillMaxHours)
	}
	return int32(hours64), nil
}

func mergeBackfillCreditBlob(encoded []byte, routeTotals map[int16]backfillCredits) error {
	previousRouteID := -1
	for offset := 0; offset < len(encoded); {
		if len(encoded)-offset < 2 {
			return fmt.Errorf("cabecera truncada")
		}
		header := binary.BigEndian.Uint16(encoded[offset : offset+2])
		offset += 2
		routeID := int(header >> 2)
		valueWidth := int(header&0b11) + 1
		if routeID <= previousRouteID || len(encoded)-offset < valueWidth*2 {
			return fmt.Errorf("entrada no canónica para ruta %d", routeID)
		}
		cpuCredits := readBackfillCredit(encoded[offset : offset+valueWidth])
		offset += valueWidth
		inferenceCredits := readBackfillCredit(encoded[offset : offset+valueWidth])
		offset += valueWidth
		if cpuCredits == 0 && inferenceCredits == 0 || backfillCreditWidth(max(cpuCredits, inferenceCredits)) != valueWidth {
			return fmt.Errorf("valores no canónicos para ruta %d", routeID)
		}
		current := routeTotals[int16(routeID)]
		if math.MaxUint64-current.cpu < cpuCredits || math.MaxUint64-current.inference < inferenceCredits {
			return fmt.Errorf("overflow sumando ruta %d", routeID)
		}
		current.cpu += cpuCredits
		current.inference += inferenceCredits
		routeTotals[int16(routeID)] = current
		previousRouteID = routeID
	}
	return nil
}

func encodeBackfillCreditBlob(routeTotals map[int16]backfillCredits) ([]byte, error) {
	routeIDs := make([]int16, 0, len(routeTotals))
	for routeID := range routeTotals {
		routeIDs = append(routeIDs, routeID)
	}
	slices.Sort(routeIDs)
	encoded := []byte{}
	for _, routeID := range routeIDs {
		totals := routeTotals[routeID]
		if routeID < 0 || routeID > 16_383 || totals.cpu > math.MaxUint32 || totals.inference > math.MaxUint32 {
			return nil, fmt.Errorf("ruta %d excede el formato persistido", routeID)
		}
		if totals.cpu == 0 && totals.inference == 0 {
			continue
		}
		valueWidth := backfillCreditWidth(max(totals.cpu, totals.inference))
		header := uint16(routeID)<<2 | uint16(valueWidth-1)
		encoded = binary.BigEndian.AppendUint16(encoded, header)
		encoded = appendBackfillCredit(encoded, totals.cpu, valueWidth)
		encoded = appendBackfillCredit(encoded, totals.inference, valueWidth)
	}
	return encoded, nil
}

func readBackfillCredit(encoded []byte) uint64 {
	var value uint64
	for _, encodedByte := range encoded {
		value = value<<8 | uint64(encodedByte)
	}
	return value
}

func appendBackfillCredit(encoded []byte, value uint64, width int) []byte {
	bytes := [4]byte{}
	binary.BigEndian.PutUint32(bytes[:], uint32(value))
	return append(encoded, bytes[4-width:]...)
}

func backfillCreditWidth(value uint64) int {
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
