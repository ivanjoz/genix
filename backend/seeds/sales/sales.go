package sales

import (
	"app/core"
	financeTypes "app/finance/types"
	invoicingTypes "app/invoicing/types"
	salesModule "app/sales"
	salesTypes "app/sales/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

func availableProductIDs(run *runner.Run, warehouseID int32) []int32 {
	availableProductIDs := []int32{}
	for productID, buckets := range run.BucketsByWarehouse[warehouseID] {
		for _, bucket := range buckets {
			if bucket.Remaining > 0 {
				availableProductIDs = append(availableProductIDs, productID)
				break
			}
		}
	}
	return availableProductIDs
}

func pickBucket(run *runner.Run, warehouseID int32, productID int32) *runner.StockBucket {
	candidateBuckets := []*runner.StockBucket{}
	for _, bucket := range run.BucketsByWarehouse[warehouseID][productID] {
		if bucket.Remaining > 0 {
			candidateBuckets = append(candidateBuckets, bucket)
		}
	}
	if len(candidateBuckets) == 0 {
		return nil
	}
	return candidateBuckets[run.Random.Intn(len(candidateBuckets))]
}

// ─── Day loop ──────────────────────────────────────────────────────────────────

// runDailySales draws the day's volume and its payment/delivery plan up front, so the requested
// counts are exact instead of probabilistic.
func runDailySales(run *runner.Run, dayStart time.Time) error {
	// Draw the plan only once per day: on a resumed day it comes back from the state file, so the
	// unpaid/undelivered counts the day started with still hold.
	if run.State.SalesPlanCount == 0 {
		salesCount := run.RandomInRange(run.Config.SalesMin, run.Config.SalesMax)
		// The two flags are drawn independently, so a sale can end up both unpaid and undelivered.
		run.State.SalesPlanCount = salesCount
		run.State.SalesUnpaidFlags = makeFlagPlan(run, salesCount,
			run.RandomInRange(run.Config.UnpaidMin, run.Config.UnpaidMax))
		run.State.SalesUndeliveredFlags = makeFlagPlan(run, salesCount,
			run.RandomInRange(run.Config.UndeliveredMin, run.Config.UndeliveredMax))
		run.State.SalesAnnulFlags = makeFlagPlan(run, salesCount,
			run.RandomInRange(run.Config.AnnulMin, run.Config.AnnulMax))
		if err := run.SaveState(); err != nil {
			return err
		}
	}

	salesCount := run.State.SalesPlanCount
	// Spread the sales across a working day so the hourly reports have a realistic shape.
	saleInterval := (9 * time.Hour) / time.Duration(salesCount)

	for saleIndex := run.State.CompletedSales; saleIndex < salesCount; saleIndex++ {
		isUnpaid := run.State.SalesUnpaidFlags[saleIndex]
		isUndelivered := run.State.SalesUndeliveredFlags[saleIndex]

		saleTime := dayStart.Add(11*time.Hour + time.Duration(saleIndex)*saleInterval)
		createdSale, err := CreateSaleWithRetry(run, saleTime, isUnpaid, isUndelivered)
		if err != nil {
			return err
		}

		// The month's taxes are computed from what was actually sold.
		run.PeriodTaxTotals(core.TimeToFechaUnix(saleTime)).SalesGross += int64(createdSale.TotalAmount)
		if createdSale.IssueSeriesID > 0 {
			run.State.Stats.SalesInvoiced++
		}
		if run.State.SalesAnnulFlags[saleIndex] {
			run.State.PendingAnnulments = append(run.State.PendingAnnulments,
				runner.PendingAnnulment{SaleID: createdSale.ID, CashBankID: createdSale.LastPaymentCajaID})
		}
		run.State.Stats.SalesCreated++
		if isUnpaid {
			run.State.Stats.SalesUnpaid++
		}
		if isUndelivered {
			run.State.Stats.SalesUndelivered++
		}
		// Checkpoint per sale: the resume point must never point at a sale already written.
		run.State.CompletedSales = saleIndex + 1
		if err := run.SaveState(); err != nil {
			return err
		}
	}

	core.Log("Seed:: ventas del día", "total", salesCount,
		"impagas", countFlagged(run.State.SalesUnpaidFlags),
		"no entregadas", countFlagged(run.State.SalesUndeliveredFlags))
	return nil
}

// CreateSaleWithRetry rebuilds the sale from freshly reloaded stock when the handler rejects it
// for insufficient stock, which is the only failure the ledger can drift into.
func CreateSaleWithRetry(run *runner.Run, saleTime time.Time, isUnpaid bool, isUndelivered bool) (salesTypes.SaleOrder, error) {
	const maxStockRetries = 3

	for attempt := 0; ; attempt++ {
		warehouseID := run.Config.WarehouseIDs[run.Random.Intn(len(run.Config.WarehouseIDs))]
		salePayload, reservedBuckets, err := makeSalePayload(run, warehouseID, isUnpaid, isUndelivered)
		if err != nil {
			if attempt >= maxStockRetries {
				return salesTypes.SaleOrder{}, err
			}
			run.State.Stats.StockRetries++
			if reloadError := run.ReloadWarehouseLedger(warehouseID); reloadError != nil {
				return salesTypes.SaleOrder{}, reloadError
			}
			continue
		}

		createdSale, createError := postSaleOrder(run, saleTime, salePayload)
		if createError == nil {
			// Only a delivered sale leaves the warehouse, so undelivered ones must not consume the
			// reservation — the units stay available for the following sales.
			if !isUndelivered {
				for bucketIndex, bucket := range reservedBuckets {
					// The line quantity is packed; the reservation is counted in whole units.
					bucket.Remaining -= core.UnpackQuantityLine(salePayload.Lines[bucketIndex].Quantity).Units
				}
			}
			return createdSale, nil
		}
		if !isStockShortageError(createError) || attempt >= maxStockRetries {
			return salesTypes.SaleOrder{}, createError
		}

		run.State.Stats.StockRetries++
		core.Log("Seed:: stock insuficiente, recargando ledger", createError.Error())
		if reloadError := run.ReloadWarehouseLedger(warehouseID); reloadError != nil {
			return salesTypes.SaleOrder{}, reloadError
		}
	}
}

// makeSalePayload picks distinct products from one warehouse and resolves each to a concrete
// bucket, because a line drawing from a lot or a serial must name it or the stock validation
// looks at the free bucket instead.
func makeSalePayload(run *runner.Run, warehouseID int32, isUnpaid bool, isUndelivered bool) (salesTypes.SaleOrder, []*runner.StockBucket, error) {
	availableProductIDs := availableProductIDs(run, warehouseID)
	if len(availableProductIDs) < run.Config.SaleLinesMin {
		return salesTypes.SaleOrder{}, nil, core.Err("el almacén", warehouseID, "no tiene productos con stock suficiente")
	}

	lineCount := run.RandomInRange(run.Config.SaleLinesMin, run.Config.SaleLinesMax)
	if lineCount > len(availableProductIDs) {
		lineCount = len(availableProductIDs)
	}
	run.Random.Shuffle(len(availableProductIDs), func(i, j int) {
		availableProductIDs[i], availableProductIDs[j] = availableProductIDs[j], availableProductIDs[i]
	})

	salePayload := salesTypes.SaleOrder{
		WarehouseID:       warehouseID,
		LastPaymentCajaID: run.TillByWarehouse[warehouseID],
		CurrencyType:      financeTypes.CurrencyPEN,
	}
	reservedBuckets := []*runner.StockBucket{}

	for _, productID := range availableProductIDs[:lineCount] {
		bucket := pickBucket(run, warehouseID, productID)
		if bucket == nil {
			continue
		}

		// A serialized unit is unique, so its line can only ever be one unit.
		quantity := int32(1)
		if bucket.Kind != runner.StockBucketSerial {
			quantity = int32(1 + run.Random.Intn(5))
			if quantity > bucket.Remaining {
				quantity = bucket.Remaining
			}
		}

		salePrice := run.SalePriceByProduct[productID]

		salePayload.Lines = append(salePayload.Lines, salesTypes.SaleOrderLine{
			ProductID: productID,
			// Sale-order lines are packed (Units*QuantityLineScale + Sub); the seeder emits whole
			// units only, so the sub-unit half is zero.
			Quantity:       quantity * core.QuantityLineScale,
			Price:          salePrice,
			PresentationID: bucket.PresentationID,
			LotID:          bucket.LotID,
			Sku:            bucket.SerialNumber,
		})
		salePayload.TotalAmount += salePrice * quantity
		reservedBuckets = append(reservedBuckets, bucket)
	}

	if len(reservedBuckets) == 0 {
		return salesTypes.SaleOrder{}, nil, core.Err("no se pudo armar ninguna línea con stock disponible")
	}

	// ActionsIncluded is what drives the sale status: 2 registers the payment in the cash
	// register, 3 moves the goods out of the warehouse. An unpaid sale needs no DebtAmount: the
	// handler books its whole total as owed.
	if !isUnpaid {
		salePayload.ActionsIncluded = append(salePayload.ActionsIncluded, 2)
	}
	if !isUndelivered {
		salePayload.ActionsIncluded = append(salePayload.ActionsIncluded, 3)
	}

	assignSaleBuyerAndDocument(run, &salePayload)
	return salePayload, reservedBuckets, nil
}

// assignSaleBuyerAndDocument decides who buys and what the sale is issued as. One sale in eight
// goes to a company and carries a factura (it needs the buyer's RUC); the rest are boletas, half
// of them to a known client. A boleta of S/ 700 or more must name its buyer, so a big anonymous
// one is given a client with a DNI.
func assignSaleBuyerAndDocument(run *runner.Run, salePayload *salesTypes.SaleOrder) {
	isBusinessSale := run.FacturaSeriesID > 0 && len(run.BusinessClientIDs) > 0 && run.Random.Intn(8) == 0
	if isBusinessSale {
		salePayload.ClientID = run.BusinessClientIDs[run.Random.Intn(len(run.BusinessClientIDs))]
		salePayload.IssueSeriesID = run.FacturaSeriesID
		return
	}

	if run.Random.Intn(2) == 0 && len(run.ClientIDs) > 0 {
		salePayload.ClientID = run.ClientIDs[run.Random.Intn(len(run.ClientIDs))]
	}
	salePayload.IssueSeriesID = run.BoletaSeriesID
	needsIdentifiedBuyer := salePayload.TotalAmount >= invoicingTypes.BoletaIdentifiedFrom
	if needsIdentifiedBuyer && !slices.Contains(run.DNIClientIDs, salePayload.ClientID) && len(run.DNIClientIDs) > 0 {
		salePayload.ClientID = run.DNIClientIDs[run.Random.Intn(len(run.DNIClientIDs))]
	}
}

func postSaleOrder(run *runner.Run, saleTime time.Time, payload salesTypes.SaleOrder) (salesTypes.SaleOrder, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return salesTypes.SaleOrder{}, err
	}
	core.SetHistoricalUnix(saleTime.Unix())
	request := run.MakeRequest("POST.sale-order", nil, string(bodyBytes))
	createdSale := salesTypes.SaleOrder{}
	if err := runner.DecodeResponse(run.Send(request, salesModule.PostSaleOrder), &createdSale); err != nil {
		return salesTypes.SaleOrder{}, err
	}
	return createdSale, nil
}

// makeFlagPlan marks exactly flaggedCount of totalCount positions, in random order.
func makeFlagPlan(run *runner.Run, totalCount int, flaggedCount int) []bool {
	if flaggedCount > totalCount {
		flaggedCount = totalCount
	}
	plan := make([]bool, totalCount)
	for planIndex := 0; planIndex < flaggedCount; planIndex++ {
		plan[planIndex] = true
	}
	run.Random.Shuffle(len(plan), func(i, j int) {
		plan[i], plan[j] = plan[j], plan[i]
	})
	return plan
}

func countFlagged(flags []bool) int {
	flaggedCount := 0
	for _, isFlagged := range flags {
		if isFlagged {
			flaggedCount++
		}
	}
	return flaggedCount
}

// isStockShortageError matches the message types.ValidateSaleStock builds when a line outruns its bucket.
func isStockShortageError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Se posee en stock")
}

// AnnulReasons are what a cashier writes when voiding a sale.
var AnnulReasons = []string{
	"Error al registrar la venta",
	"El cliente desistió de la compra",
	"Producto con defecto de fábrica",
	"Venta registrada por duplicado",
}

// runAnnulmentsStep voids the sales flagged for annulment, refunding into the till that collected
// them. The handler checks the annul sub-access, which a real request learns from the gate; the
// run grants it to itself for this call only.
func runAnnulmentsStep(run *runner.Run, dayStart time.Time) error {
	annulTime := dayStart.Add(20*time.Hour + 30*time.Minute)
	for len(run.State.PendingAnnulments) > 0 {
		annulment := run.State.PendingAnnulments[0]
		bodyBytes, err := json.Marshal(map[string]any{
			"ID":               annulment.SaleID,
			"RefundCashBankID": annulment.CashBankID,
			"Reason":           AnnulReasons[run.Random.Intn(len(AnnulReasons))],
		})
		if err != nil {
			return err
		}
		core.SetHistoricalUnix(annulTime.Unix())
		request := run.MakeRequest("POST.sale-order-annul", nil, string(bodyBytes))
		annulToken := run.UserToken
		annulToken.SubAccesos = map[int32]uint16{salesTypes.AccesoIDGestionVentas: 1 << (core.SubAccesoTodosID - 1)}
		request.User = &annulToken
		if response := run.Send(request, salesModule.PostSaleOrderAnnul); response.StatusCode != 200 {
			return core.Err("no se pudo anular la venta", annulment.SaleID, ":", response.Error)
		}

		run.State.Stats.SalesAnnulled++
		run.State.PendingAnnulments = run.State.PendingAnnulments[1:]
		if err := run.SaveState(); err != nil {
			return err
		}
		annulTime = annulTime.Add(time.Minute)
	}
	return nil
}
