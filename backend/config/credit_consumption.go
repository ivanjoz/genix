package config

import (
	"app/core"
	"cmp"
	"context"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
)

// Credit consumption (System → Observabilidad → Consumo Créditos): every company's last four hours
// in five-minute frames, read from fareward's own store. Two reports: the company usage rows
// (credits and the credit gates' refusals) and the device rows (the device gate's refusals, the
// distinct (IP, device) pairs served, and each frame's busiest pairs).

const (
	creditConsumptionHours = int32(4)
	// creditConsumptionTopPairs is how many pairs a company's detail table lists. The four-hour
	// figure of a pair is the sum of the frames where it made fareward's per-frame top twelve, so
	// it is exact for the busy pairs this table is about and a lower bound for the quiet ones.
	creditConsumptionTopPairs = 25
)

type creditConsumptionPair struct {
	IP string
	// Device is the browser signature in decimal: an int64 does not survive a JSON number in the
	// browser, so it travels as text and the page decodes it with BigInt.
	Device   string
	Requests uint64
	CPU      uint64
	// Frames is how many five-minute frames the pair appeared in.
	Frames int32
}

type creditConsumptionCompany struct {
	CompanyID int32
	Name      string
	// One value per frame of the response's FrameIDs.
	CPU            []uint64
	Inference      []uint64
	CreditRejected []uint32
	DeviceRejected []uint32
	Devices        []uint32
	TotalCPU       uint64
	TotalInference uint64
	TotalRejected  uint64
	// MaxDevices is the most distinct pairs any single frame saw.
	MaxDevices uint32
	Pairs      []creditConsumptionPair
}

type creditConsumptionResponse struct {
	// FrameIDs are the frames' start times as sunix, the unit the page formats times from.
	FrameIDs  []int32
	Companies []creditConsumptionCompany
}

type creditConsumptionPairKey struct {
	ip     [16]byte
	device int64
}

func GetCreditConsumption(req *core.HandlerArgs) core.HandlerResponse {
	currentFrame := unixToObservabilityTimeFrame(time.Now().UTC().Unix())
	firstFrame := currentFrame - creditConsumptionHours*12 + 1

	var usageRows []core.CompanyUsageRow
	var deviceRows []core.DeviceUsageRow
	queries := errgroup.Group{}
	queries.Go(func() error {
		var err error
		usageRows, err = core.QueryEveryCompanyUsage(context.Background(), firstFrame, currentFrame)
		return err
	})
	queries.Go(func() error {
		var err error
		deviceRows, err = core.QueryDeviceUsage(context.Background(), 0, firstFrame, currentFrame)
		return err
	})
	if err := queries.Wait(); err != nil {
		core.Log("credit consumption query failed::", " first_frame::", firstFrame,
			" current_frame::", currentFrame, " err::", err)
		return req.MakeErr("No se pudo obtener el consumo de créditos.", err)
	}

	response, err := makeCreditConsumption(firstFrame, currentFrame, usageRows, deviceRows)
	if err != nil {
		core.Log("credit consumption aggregation failed::", " usage_rows::", len(usageRows),
			" device_rows::", len(deviceRows), " err::", err)
		return req.MakeErr("No se pudo interpretar el consumo de créditos.", err)
	}
	for index := range response.Companies {
		company := &response.Companies[index]
		record, err := getCompanyByID(company.CompanyID)
		if err != nil {
			core.Log("credit consumption company name failed::", " company::", company.CompanyID, " err::", err)
			continue
		}
		if record != nil {
			company.Name = record.Name
		}
	}
	return req.MakeResponse(response)
}

func makeCreditConsumption(
	firstFrame, lastFrame int32,
	usageRows []core.CompanyUsageRow,
	deviceRows []core.DeviceUsageRow,
) (creditConsumptionResponse, error) {
	if firstFrame > lastFrame {
		return creditConsumptionResponse{}, fmt.Errorf("invalid frame range %d..%d", firstFrame, lastFrame)
	}
	frameCount := int(lastFrame - firstFrame + 1)
	response := creditConsumptionResponse{FrameIDs: make([]int32, frameCount)}
	for offset := range frameCount {
		response.FrameIDs[offset] = core.UnixToSunix(observabilityTimeFrameToUnix(firstFrame + int32(offset)))
	}

	companies := map[int32]*creditConsumptionCompany{}
	pairsByCompany := map[int32]map[creditConsumptionPairKey]*creditConsumptionPair{}
	company := func(companyID int32) *creditConsumptionCompany {
		if existing := companies[companyID]; existing != nil {
			return existing
		}
		created := &creditConsumptionCompany{
			CompanyID:      companyID,
			CPU:            make([]uint64, frameCount),
			Inference:      make([]uint64, frameCount),
			CreditRejected: make([]uint32, frameCount),
			DeviceRejected: make([]uint32, frameCount),
			Devices:        make([]uint32, frameCount),
		}
		companies[companyID] = created
		return created
	}
	offsetOf := func(timeFrame int32) (int, error) {
		offset := int(timeFrame - firstFrame)
		if offset < 0 || offset >= frameCount {
			return 0, fmt.Errorf("time frame %d is outside the requested range", timeFrame)
		}
		return offset, nil
	}

	for _, row := range usageRows {
		// Company zero is the platform aggregate, not a tenant.
		if row.CompanyID <= 0 {
			continue
		}
		offset, err := offsetOf(row.TimeFrame)
		if err != nil {
			return creditConsumptionResponse{}, err
		}
		totals, err := decodeCreditUsage(row.UsedCredits, nil)
		if err != nil {
			return creditConsumptionResponse{}, fmt.Errorf("invalid credit blob of company %d in time frame %d: %w",
				row.CompanyID, row.TimeFrame, err)
		}
		entry := company(row.CompanyID)
		entry.CPU[offset] = totals.CPU
		entry.Inference[offset] = totals.Inference
		entry.CreditRejected[offset] = row.Rejections.Short + row.Rejections.Long + row.Rejections.NoFunds
	}

	for _, row := range deviceRows {
		if row.CompanyID <= 0 {
			continue
		}
		offset, err := offsetOf(row.TimeFrame)
		if err != nil {
			return creditConsumptionResponse{}, err
		}
		entry := company(row.CompanyID)
		entry.DeviceRejected[offset] = row.Rejected
		entry.Devices[offset] = row.Distinct
		pairs := pairsByCompany[row.CompanyID]
		if pairs == nil {
			pairs = map[creditConsumptionPairKey]*creditConsumptionPair{}
			pairsByCompany[row.CompanyID] = pairs
		}
		for _, devicePair := range row.Top {
			key := creditConsumptionPairKey{ip: devicePair.IP, device: devicePair.Device}
			pair := pairs[key]
			if pair == nil {
				pair = &creditConsumptionPair{
					IP:     net.IP(devicePair.IP[:]).String(),
					Device: strconv.FormatInt(devicePair.Device, 10),
				}
				pairs[key] = pair
			}
			pair.Requests += uint64(devicePair.Requests)
			pair.CPU += devicePair.CPU
			pair.Frames++
		}
	}

	response.Companies = make([]creditConsumptionCompany, 0, len(companies))
	for companyID, entry := range companies {
		for offset := range frameCount {
			entry.TotalCPU += entry.CPU[offset]
			entry.TotalInference += entry.Inference[offset]
			entry.TotalRejected += uint64(entry.CreditRejected[offset]) + uint64(entry.DeviceRejected[offset])
			entry.MaxDevices = max(entry.MaxDevices, entry.Devices[offset])
		}
		entry.Pairs = make([]creditConsumptionPair, 0, len(pairsByCompany[companyID]))
		for _, pair := range pairsByCompany[companyID] {
			entry.Pairs = append(entry.Pairs, *pair)
		}
		slices.SortFunc(entry.Pairs, func(left, right creditConsumptionPair) int {
			return cmp.Or(
				cmp.Compare(right.Requests, left.Requests),
				cmp.Compare(right.CPU, left.CPU),
				cmp.Compare(left.IP, right.IP),
				cmp.Compare(left.Device, right.Device),
			)
		})
		if len(entry.Pairs) > creditConsumptionTopPairs {
			entry.Pairs = entry.Pairs[:creditConsumptionTopPairs]
		}
		response.Companies = append(response.Companies, *entry)
	}
	slices.SortFunc(response.Companies, func(left, right creditConsumptionCompany) int {
		return cmp.Or(
			cmp.Compare(right.TotalCPU, left.TotalCPU),
			cmp.Compare(right.TotalRejected, left.TotalRejected),
			cmp.Compare(left.CompanyID, right.CompanyID),
		)
	})
	return response, nil
}
