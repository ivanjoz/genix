package logistics

import (
	"app/core"
	"app/db"
	logisticsModule "app/logistics"
	logisticsTypes "app/logistics/types"
	production "app/production/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"slices"
)

// loadReplenishedProducts takes the first N active products by ID, so a rerun configures the same
// products and the page shows a stable block of configured rows.
func loadReplenishedProducts(run *runner.Run) ([]production.Product, error) {
	products := []production.Product{}
	productQuery := db.Query(&products)
	productQuery.Select(productQuery.ID, productQuery.Name, productQuery.Price,
		productQuery.FinalPrice, productQuery.Status).
		CompanyID.Equals(run.UserToken.CompanyID).Status.Equals(1)

	if err := productQuery.Exec(); err != nil {
		return nil, err
	}

	activeProducts := make([]production.Product, 0, len(products))
	for _, productRecord := range products {
		if productRecord.ID > 0 {
			activeProducts = append(activeProducts, productRecord)
		}
	}
	if len(activeProducts) == 0 {
		return nil, core.Err("no hay productos activos en la empresa")
	}

	slices.SortFunc(activeProducts, func(leftProduct, rightProduct production.Product) int {
		return int(leftProduct.ID - rightProduct.ID)
	})
	if len(activeProducts) > run.Config.SupplyProducts {
		activeProducts = activeProducts[:run.Config.SupplyProducts]
	}

	return activeProducts, nil
}

// makeSupplyRecords draws the replenishment configuration for every selected product. Provider
// prices derive from the product's own sale price so the two quotes of a product stay in the same
// order of magnitude as what it sells for.
func makeSupplyRecords(run *runner.Run, products []production.Product) []logisticsTypes.ProductSupply {
	supplyRecords := make([]logisticsTypes.ProductSupply, 0, len(products))

	for _, productRecord := range products {
		salePrice := productRecord.FinalPrice
		if salePrice <= 0 {
			salePrice = productRecord.Price
		}
		if salePrice <= 0 {
			salePrice = 1000
		}

		// Base cost between 55% and 75% of the sale price: the margin a distributor would leave.
		costPrice := salePrice * int32(55+run.Random.Intn(21)) / 100

		providerRows := make([]logisticsTypes.ProductSupplyProviderRow, 0, run.Config.ProvidersPerProduct)
		for _, providerID := range pickDistinctProviders(run, run.Config.ProvidersPerProduct) {
			// Each provider quotes within ±10% of the base cost, so one of them is always cheaper.
			quotedPrice := costPrice * int32(90+run.Random.Intn(21)) / 100
			if quotedPrice < 1 {
				quotedPrice = 1
			}
			providerRows = append(providerRows, logisticsTypes.ProductSupplyProviderRow{
				ProviderID:   providerID,
				Capacity:     int32(50 + run.Random.Intn(451)),
				DeliveryTime: int16(1 + run.Random.Intn(15)),
				Price:        quotedPrice,
			})
		}

		supplyRecords = append(supplyRecords, logisticsTypes.ProductSupply{
			ProductID:            productRecord.ID,
			MinimunStock:         int32(run.RandomInRange(run.Config.MinStockMin, run.Config.MinStockMax)),
			SalesPerDayEstimated: int32(run.RandomInRange(run.Config.SalesPerDayMin, run.Config.SalesPerDayMax)),
			ProviderSupply:       providerRows,
		})
	}

	return supplyRecords
}

// pickDistinctProviders draws providers without repetition, because the handler rejects the same
// provider twice on one product.
func pickDistinctProviders(run *runner.Run, count int) []int32 {
	if count > len(run.SupplyProviderIDs) {
		count = len(run.SupplyProviderIDs)
	}

	shuffledProviderIDs := slices.Clone(run.SupplyProviderIDs)
	run.Random.Shuffle(len(shuffledProviderIDs), func(i, j int) {
		shuffledProviderIDs[i], shuffledProviderIDs[j] = shuffledProviderIDs[j], shuffledProviderIDs[i]
	})
	return shuffledProviderIDs[:count]
}

func postSupplyBatches(run *runner.Run, supplyRecords []logisticsTypes.ProductSupply) error {
	for batchStart := 0; batchStart < len(supplyRecords); batchStart += runner.PostBatchSize {
		batchEnd := min(batchStart+runner.PostBatchSize, len(supplyRecords))
		currentBatch := supplyRecords[batchStart:batchEnd]

		bodyBytes, err := json.Marshal(currentBatch)
		if err != nil {
			return err
		}

		request := run.MakeRequest("POST.product-supply", nil, string(bodyBytes))
		if response := run.Send(request, logisticsModule.PostProductSupply); response.StatusCode != 200 {
			return core.Err(response.Error)
		}

		core.Log("Seed:: lote guardado", "desde", batchStart, "hasta", batchEnd)
	}
	return nil
}
