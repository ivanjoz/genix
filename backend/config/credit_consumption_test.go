package config

import (
	"app/core"
	"slices"
	"testing"
)

func TestMakeCreditConsumptionJoinsCreditsRefusalsAndDevices(t *testing.T) {
	firstFrame := unixToObservabilityTimeFrame(1_800_000_000)
	office := [16]byte{10: 0xFF, 11: 0xFF, 12: 200, 13: 48, 14: 1, 15: 9}
	usageRows := []core.CompanyUsageRow{
		// Route 34, one-byte values, 10 CPU and 3 inference credits; two credit refusals.
		{CompanyID: 7, TimeFrame: firstFrame, UsedCredits: []byte{0x00, 0x88, 0x0A, 0x03},
			Rejections: core.Rejections{Short: 1, NoFunds: 1}},
		{CompanyID: 7, TimeFrame: firstFrame + 2, UsedCredits: []byte{0x00, 0x88, 0x0A, 0x03}},
		// The platform aggregate is not a tenant.
		{CompanyID: 0, TimeFrame: firstFrame, UsedCredits: []byte{0x00, 0x88, 0x0A, 0x03}},
		{CompanyID: 9, TimeFrame: firstFrame + 1, UsedCredits: []byte{0x00, 0x88, 0x01, 0x00}},
	}
	deviceRows := []core.DeviceUsageRow{
		{CompanyID: 7, TimeFrame: firstFrame, Rejected: 4, Distinct: 3, Top: []core.DevicePair{
			{IP: office, Device: -42, Requests: 5, CPU: 10},
			{IP: office, Device: 8, Requests: 7, CPU: 3},
		}},
		{CompanyID: 7, TimeFrame: firstFrame + 2, Distinct: 1, Top: []core.DevicePair{
			{IP: office, Device: -42, Requests: 6, CPU: 10},
		}},
	}

	response, err := makeCreditConsumption(firstFrame, firstFrame+2, usageRows, deviceRows)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.FrameIDs) != 3 || len(response.Companies) != 2 {
		t.Fatalf("unexpected shape: %#v", response)
	}
	busiest := response.Companies[0]
	if busiest.CompanyID != 7 || busiest.TotalCPU != 20 || busiest.TotalInference != 6 {
		t.Fatalf("companies are not ranked by CPU or totals are wrong: %#v", busiest)
	}
	if !slices.Equal(busiest.CPU, []uint64{10, 0, 10}) ||
		!slices.Equal(busiest.CreditRejected, []uint32{2, 0, 0}) ||
		!slices.Equal(busiest.DeviceRejected, []uint32{4, 0, 0}) ||
		!slices.Equal(busiest.Devices, []uint32{3, 0, 1}) {
		t.Fatalf("unexpected per-frame values: %#v", busiest)
	}
	if busiest.TotalRejected != 6 || busiest.MaxDevices != 3 {
		t.Fatalf("unexpected refusals or devices: %#v", busiest)
	}
	// The pair seen in two frames adds up to 11 requests and ranks above the one seen once.
	if len(busiest.Pairs) != 2 {
		t.Fatalf("unexpected pairs: %#v", busiest.Pairs)
	}
	first := busiest.Pairs[0]
	if first.IP != "200.48.1.9" || first.Device != "-42" || first.Requests != 11 || first.CPU != 20 ||
		first.Frames != 2 {
		t.Fatalf("unexpected top pair: %#v", first)
	}
	if response.Companies[1].CompanyID != 9 || response.Companies[1].Pairs == nil {
		t.Fatalf("a company without device rows still needs an empty pair list: %#v", response.Companies[1])
	}
}

func TestMakeCreditConsumptionRefusesAFrameOutsideTheWindow(t *testing.T) {
	firstFrame := unixToObservabilityTimeFrame(1_800_000_000)
	_, err := makeCreditConsumption(firstFrame, firstFrame+1, nil, []core.DeviceUsageRow{
		{CompanyID: 7, TimeFrame: firstFrame + 5},
	})
	if err == nil {
		t.Fatal("a device row outside the window was accepted")
	}
}
