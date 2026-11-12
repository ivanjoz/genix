package accounting

import (
	accountingModule "app/accounting"
	accountingTypes "app/accounting/types"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	financeTypes "app/finance/types"
	"app/production"
	productionTypes "app/production/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

//go:embed assets.psv
var assetsPSV string

const (
	// Acquisitions are spread over three years back, which is what makes the depreciation run
	// produce anything: it only posts periods that have already come due.
	assetMaxDaysBack = 1080
	assetMinDaysBack = 30
)

type assetSeed struct {
	materialSKU      string
	name             string
	description      string
	acquisitionValue int32
	quantity         int32
	serialNumber     string
}

func readAssetSeeds() ([]assetSeed, error) {
	rows, parseError := runner.ParsePSV(assetsPSV, 6)
	if parseError != nil {
		return nil, parseError
	}

	seeds := make([]assetSeed, 0, len(rows))
	for rowIndex, row := range rows {
		acquisitionValue, valueError := runner.ParsePriceToCents(row[3])
		if valueError != nil {
			return nil, core.Err("valor inválido en la fila", rowIndex+2, ":", valueError)
		}
		quantity, _ := strconv.Atoi(strings.TrimSpace(row[4]))
		if quantity < 1 {
			quantity = 1
		}

		seeds = append(seeds, assetSeed{
			materialSKU:      strings.TrimSpace(row[0]),
			name:             strings.TrimSpace(row[1]),
			description:      strings.TrimSpace(row[2]),
			acquisitionValue: acquisitionValue,
			quantity:         int32(quantity),
			serialNumber:     strings.TrimSpace(row[5]),
		})
	}
	return seeds, nil
}

// productIDBySKU indexes the supply catalog so the asset seeds can name their material by SKU.
func productIDBySKU(run *runner.Run) (map[string]int32, error) {
	request := run.MakeRequest("GET.supply-material", map[string]string{}, "")
	supplyRecords := []productionTypes.Product{}
	if err := runner.DecodeResponse(run.Send(request, production.GetSupplyMaterials), &supplyRecords); err != nil {
		return nil, err
	}

	idBySKU := map[string]int32{}
	for _, supplyRecord := range supplyRecords {
		if supplyRecord.Status != productionTypes.ProductStatusSupply || supplyRecord.SKU == "" {
			continue
		}
		// Keep the lowest ID when a SKU repeats, so a database seeded twice still resolves to the
		// first copy instead of drifting between runs.
		if existingID, exists := idBySKU[supplyRecord.SKU]; !exists || supplyRecord.ID < existingID {
			idBySKU[supplyRecord.SKU] = supplyRecord.ID
		}
	}
	return idBySKU, nil
}

// seedAssets acquires the static asset list: the first half bought, the second half donated. Then
// it settles part of what is owed and runs the depreciation so the schedules exist.
func seedAssets(run *runner.Run) error {
	existingAssets, countError := countAssets(run)
	if countError != nil {
		return countError
	}
	if existingAssets >= run.Config.Assets {
		core.Log("Seed:: activos ya existentes, no se generan", "total", existingAssets)
		return nil
	}

	seeds, parseError := readAssetSeeds()
	if parseError != nil {
		return core.Err("error al leer assets.psv:", parseError)
	}
	if len(seeds) > run.Config.Assets {
		seeds = seeds[:run.Config.Assets]
	}

	idBySKU, indexError := productIDBySKU(run)
	if indexError != nil {
		return indexError
	}

	warehouseID, warehouseError := resolveWarehouseID(run)
	if warehouseError != nil {
		return warehouseError
	}
	// Paid from the bank, whose currency becomes the assets' currency: PostAssetPayment refuses a
	// payment whose currency differs from the account's.
	cashBank, cashBankError := financeTypes.GetCaja(run.UserToken.CompanyID, run.BankCashBankID)
	if cashBankError != nil {
		return cashBankError
	}

	todayUnixDay := core.FechaUnix()
	// The first half is bought and the second half donated, so the two payment paths are always
	// both present regardless of how many seeds the file carries.
	purchasedCount := len(seeds) / 2
	createdAssets := []accountingTypes.Asset{}

	for seedIndex, seed := range seeds {
		productID, productExists := idBySKU[seed.materialSKU]
		if !productExists {
			return core.Err("el insumo", seed.materialSKU, "no existe; corre el script sin saltar los insumos")
		}

		acquisitionDate := todayUnixDay - int16(run.RandomInRange(assetMinDaysBack, assetMaxDaysBack))
		purchaseAmount := int32(0)
		if seedIndex < purchasedCount {
			// Bought at book value: the acquisition cost and what was owed are the same money.
			purchaseAmount = seed.acquisitionValue
		}

		serialNumbers := []string{}
		quantity := seed.quantity
		if seed.serialNumber != "" {
			serialNumbers = []string{seed.serialNumber}
			quantity = 1
		}

		payload := accountingModule.AssetAcquisitionPayload{
			ProductID:        productID,
			SerialNumbers:    serialNumbers,
			Quantity:         quantity,
			WarehouseID:      warehouseID,
			AcquisitionDate:  acquisitionDate,
			Name:             seed.name,
			Description:      seed.description,
			SupplierID:       run.SupplyProviderIDs[run.Random.Intn(len(run.SupplyProviderIDs))],
			CurrencyType:     cashBank.CurrencyType,
			AcquisitionValue: seed.acquisitionValue,
			PurchaseAmount:   purchaseAmount,
			DueDate:          acquisitionDate + 30,
		}

		bodyBytes, marshalError := json.Marshal(payload)
		if marshalError != nil {
			return marshalError
		}

		request := run.MakeRequest("POST.asset", nil, string(bodyBytes))
		assetsOfAcquisition := []accountingTypes.Asset{}
		if err := runner.DecodeResponse(run.Send(request, accountingModule.PostAsset), &assetsOfAcquisition); err != nil {
			return core.Err("error al registrar el activo", seed.name, ":", err)
		}
		createdAssets = append(createdAssets, assetsOfAcquisition...)
	}

	core.Log("Seed:: activos registrados", "total", len(createdAssets),
		"comprados", purchasedCount, "donados", len(seeds)-purchasedCount)

	if err := payAssets(run, createdAssets, cashBank); err != nil {
		return err
	}
	return runAssetDepreciation(run)
}

func countAssets(run *runner.Run) (int, error) {
	request := run.MakeRequest("GET.assets", map[string]string{}, "")
	assets := []accountingTypes.Asset{}
	if err := runner.DecodeResponse(run.Send(request, accountingModule.GetAssets), &assets); err != nil {
		return 0, err
	}
	// The delta response ends with its log's Metadata records, which are not assets.
	assetCount := 0
	for _, asset := range assets {
		if asset.Metadata == nil {
			assetCount++
		}
	}
	return assetCount, nil
}

// payAssets settles the purchased assets in three groups, so the Activos page shows every payment
// state: half paid in full, a third partially, and the rest untouched.
func payAssets(run *runner.Run,
	createdAssets []accountingTypes.Asset, cashBank financeTypes.CashBank,
) error {
	type plannedPayment struct {
		assetID int32
		amount  int32
		date    int16
	}

	payments := []plannedPayment{}
	payableIndex := 0
	for _, asset := range createdAssets {
		if asset.PurchaseAmount <= 0 {
			continue
		}

		paymentAmount := int32(0)
		switch payableIndex % 10 {
		case 0, 1, 2, 3, 4:
			paymentAmount = asset.PurchaseAmount
		case 5, 6, 7:
			// Between 30% and 70% settled, which is what leaves PaymentStatus pending with a
			// non-zero PaidAmount — the state neither "paid" nor "untouched" produces.
			paymentAmount = asset.PurchaseAmount * int32(run.RandomInRange(30, 70)) / 100
		default:
			paymentAmount = 0
		}
		payableIndex++

		if paymentAmount > 0 {
			payments = append(payments, plannedPayment{
				assetID: asset.ID, amount: paymentAmount, date: asset.AcquisitionDate + 15,
			})
		}
	}

	if len(payments) == 0 {
		return nil
	}

	totalToPay := int32(0)
	for _, payment := range payments {
		totalToPay += payment.amount
	}
	if err := run.EnsureCashBalance(cashBank.ID, totalToPay, core.Now()); err != nil {
		return err
	}

	for _, payment := range payments {
		bodyBytes, marshalError := json.Marshal(accountingModule.AssetPaymentPayload{
			AssetID:    payment.assetID,
			CashBankID: cashBank.ID,
			Amount:     payment.amount,
			Date:       payment.date,
		})
		if marshalError != nil {
			return marshalError
		}

		request := run.MakeRequest("POST.asset-payment", nil, string(bodyBytes))
		if response := run.Send(request, accountingModule.PostAssetPayment); response.StatusCode != 200 {
			return core.Err("error al pagar el activo", payment.assetID, ":", response.Error)
		}
	}

	core.Log("Seed:: pagos de activos registrados", "total", len(payments), "monto", totalToPay)
	return nil
}

// runAssetDepreciation materializes every period already due. Generation is lazy and keyed on each
// asset's own watermark, which is why the acquisitions are backdated: an asset bought today has no
// period to post and its schedule would come out empty.
func runAssetDepreciation(run *runner.Run) error {
	request := run.MakeRequest("POST.asset-depreciation-run", nil, "")
	depreciationResult := map[string]int{}
	if err := runner.DecodeResponse(run.Send(request, accountingModule.PostAssetDepreciationRun), &depreciationResult); err != nil {
		return core.Err("error al correr la depreciación:", err)
	}

	core.Log("Seed:: depreciación corrida",
		"asientos", depreciationResult["PostedEntries"], "activos", depreciationResult["UpdatedAssets"])
	return nil
}

// resolveWarehouseID picks the lowest active warehouse, so the script runs on a company whose
// warehouse IDs do not start at 1.
func resolveWarehouseID(run *runner.Run) (int32, error) {
	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status).
		CompanyID.Equals(run.UserToken.CompanyID).Status.Equals(1)

	if queryError := warehouseQuery.Exec(); queryError != nil {
		return 0, core.Err("error al consultar los almacenes:", queryError)
	}
	if len(warehouses) == 0 {
		return 0, core.Err("no hay almacenes activos en la company")
	}

	slices.SortFunc(warehouses, func(leftWarehouse, rightWarehouse businessTypes.Warehouse) int {
		return int(leftWarehouse.ID - rightWarehouse.ID)
	})
	return warehouses[0].ID, nil
}
