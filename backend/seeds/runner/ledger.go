package runner

import (
	"app/logistics"
	logisticsTypes "app/logistics/types"
	_ "embed"
	"strconv"
)

// StockBucketKind separates the three physical places a unit can live, because each one is
// validated against a different row by validateSaleStock: the free bucket against
// ProductStockV2.Quantity, the other two against their ProductStockDetail row.
type StockBucketKind int8

const (
	StockBucketFree StockBucketKind = iota
	StockBucketLot
	StockBucketSerial
)

type StockBucket struct {
	Kind           StockBucketKind
	ProductID      int32
	PresentationID int16
	LotID          int32
	SerialNumber   string
	Remaining      int32
}

// reloadWarehouseLedger rebuilds the in-memory view of one warehouse from the same endpoint the
// frontend uses, so the run reserves against exactly what the API reports.
func (run *Run) ReloadWarehouseLedger(warehouseID int32) error {
	request := run.MakeRequest("GET.warehouse-product-stock", map[string]string{
		"warehouse-id": strconv.Itoa(int(warehouseID)),
		"updated":      "0",
	}, "")
	// The details travel as []any (they end with the replace Metadata); decode them typed. The
	// Metadata records decode as zero-Status rows, which the loops below skip.
	stockResult := struct {
		ProductStock       []logisticsTypes.ProductStock
		ProductStockDetail []logisticsTypes.ProductStockDetail
	}{}
	if err := DecodeResponse(run.Send(request, logistics.GetWarehouseProductStock), &stockResult); err != nil {
		return err
	}

	bucketsByProduct := map[int32][]*StockBucket{}
	presentationByStockID := map[int64]int16{}

	// Only the pool's products. A warehouse holds stock this run never put there — supplies
	// and fixed assets land in it too — and selling one of those would price its line at 0, which
	// the sale handler rejects outright.
	isInProductPool := func(productID int32) bool {
		_, hasPrice := run.SalePriceByProduct[productID]
		return hasPrice
	}

	for _, stock := range stockResult.ProductStock {
		presentationByStockID[stock.ID] = stock.PresentationID
		// ProductStock.Quantity is only the free bucket; the lot/serial units live in the details.
		if stock.Status == 0 || stock.Quantity <= 0 || !isInProductPool(stock.ProductID) {
			continue
		}
		bucketsByProduct[stock.ProductID] = append(bucketsByProduct[stock.ProductID], &StockBucket{
			Kind:           StockBucketFree,
			ProductID:      stock.ProductID,
			PresentationID: stock.PresentationID,
			Remaining:      stock.Quantity,
		})
	}

	for _, detail := range stockResult.ProductStockDetail {
		if detail.Status == 0 || detail.Quantity <= 0 || !isInProductPool(detail.ProductID) {
			continue
		}
		bucketKind := StockBucketLot
		if detail.SerialNumber != "" {
			bucketKind = StockBucketSerial
		}
		bucketsByProduct[detail.ProductID] = append(bucketsByProduct[detail.ProductID], &StockBucket{
			Kind:           bucketKind,
			ProductID:      detail.ProductID,
			PresentationID: presentationByStockID[detail.ProductStockID],
			LotID:          detail.LotID,
			SerialNumber:   detail.SerialNumber,
			Remaining:      detail.Quantity,
		})
	}

	run.BucketsByWarehouse[warehouseID] = bucketsByProduct
	return nil
}
