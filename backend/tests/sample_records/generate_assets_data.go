// Supplies (insumos) and fixed assets for the Insumos & Materiales and Activos pages. Kept apart
// from generate_supply_data.go because it is a different domain: that file configures how products
// are replenished, this one seeds a catalog of materials and the assets acquired from it.
package sample_records

import (
	"app/accounting"
	accountingTypes "app/accounting/types"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"app/finance"
	financeTypes "app/finance/types"
	"app/production"
	productionTypes "app/production/types"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
)

//go:embed supply_materials.psv
var supplyMaterialsPSV string

//go:embed assets.psv
var assetsPSV string

const (
	// Padding added on top of what the payments need, so the till does not land on exactly zero.
	assetCashPadding int32 = 500000
	// Acquisitions are spread over three years back, which is what makes the depreciation run
	// produce anything: it only posts periods that have already come due.
	assetMaxDaysBack = 1080
	assetMinDaysBack = 30
)

type supplyMaterialSeed struct {
	name               string
	sku                string
	price              int32
	unitID             int16
	currencyID         int16
	depreciationMonths int16
}

type assetSeed struct {
	materialSKU      string
	name             string
	description      string
	acquisitionValue int32
	quantity         int32
	serialNumber     string
}

// parsePSV reads a pipe-separated file with a header row and returns the data rows.
func parsePSV(content string, expectedColumns int) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = '|'
	reader.FieldsPerRecord = expectedColumns

	records, readError := reader.ReadAll()
	if readError != nil {
		return nil, readError
	}
	if len(records) < 2 {
		return nil, core.Err("el archivo .psv no tiene filas de datos")
	}
	return records[1:], nil
}

// parsePriceToCents reads a decimal amount written in currency units and returns cents, which is
// how every monetary column is persisted.
func parsePriceToCents(rawValue string) (int32, error) {
	parsedValue, parseError := strconv.ParseFloat(strings.TrimSpace(rawValue), 64)
	if parseError != nil {
		return 0, parseError
	}
	return int32(math.Round(parsedValue * 100)), nil
}

func readSupplyMaterialSeeds() ([]supplyMaterialSeed, error) {
	rows, parseError := parsePSV(supplyMaterialsPSV, 6)
	if parseError != nil {
		return nil, parseError
	}

	seeds := make([]supplyMaterialSeed, 0, len(rows))
	for rowIndex, row := range rows {
		price, priceError := parsePriceToCents(row[2])
		if priceError != nil {
			return nil, core.Err("precio inválido en la fila", rowIndex+2, ":", priceError)
		}
		unitID, _ := strconv.Atoi(strings.TrimSpace(row[3]))
		currencyID, _ := strconv.Atoi(strings.TrimSpace(row[4]))
		depreciationMonths, _ := strconv.Atoi(strings.TrimSpace(row[5]))

		seeds = append(seeds, supplyMaterialSeed{
			name:               strings.TrimSpace(row[0]),
			sku:                strings.TrimSpace(row[1]),
			price:              price,
			unitID:             int16(unitID),
			currencyID:         int16(currencyID),
			depreciationMonths: int16(depreciationMonths),
		})
	}
	return seeds, nil
}

func readAssetSeeds() ([]assetSeed, error) {
	rows, parseError := parsePSV(assetsPSV, 6)
	if parseError != nil {
		return nil, parseError
	}

	seeds := make([]assetSeed, 0, len(rows))
	for rowIndex, row := range rows {
		acquisitionValue, valueError := parsePriceToCents(row[3])
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

// seedSupplyMaterials writes the static supply catalog, unless the company already has enough.
// `POST.supply-material` inserts rather than matching on SKU, so without this count a rerun would
// stack a second hundred on top of the first.
func (generator *supplyDataGenerator) seedSupplyMaterials() error {
	existingSupplies, countError := generator.countSupplyMaterials()
	if countError != nil {
		return countError
	}
	if existingSupplies >= generator.config.supplyMaterialsCount {
		core.Log("GenerateSupplyData:: insumos ya existentes, no se generan", "total", existingSupplies)
		return nil
	}

	seeds, parseError := readSupplyMaterialSeeds()
	if parseError != nil {
		return core.Err("error al leer supply_materials.psv:", parseError)
	}
	if len(seeds) > generator.config.supplyMaterialsCount {
		seeds = seeds[:generator.config.supplyMaterialsCount]
	}

	payload := make([]production.SupplyMaterialPayload, 0, len(seeds))
	for _, seed := range seeds {
		payload = append(payload, production.SupplyMaterialPayload{
			Name:               seed.name,
			SKU:                seed.sku,
			Price:              seed.price,
			CurrencyID:         seed.currencyID,
			UnitID:             seed.unitID,
			DepreciationMonths: seed.depreciationMonths,
			// A supply that depreciates is equipment and is not replenished to a threshold; the
			// consumables are, which is what the Stock Mínimo column on that page is for.
			MinimunStock: core.If(seed.depreciationMonths > 0, int32(0),
				int32(generator.randomInRange(generator.config.minStockMin, generator.config.minStockMax))),
			Status: productionTypes.ProductStatusSupply,
		})
	}

	bodyBytes, marshalError := json.Marshal(payload)
	if marshalError != nil {
		return marshalError
	}

	request := generator.makeRequest("POST.supply-material", nil, string(bodyBytes))
	if response := production.PostSupplyMaterial(&request); response.StatusCode != 200 {
		return core.Err(response.Error)
	}

	core.Log("GenerateSupplyData:: insumos generados", "total", len(payload))
	return nil
}

func (generator *supplyDataGenerator) countSupplyMaterials() (int, error) {
	request := generator.makeRequest("GET.supply-material", map[string]string{"upv": "0"}, "")
	supplyRecords := []productionTypes.Product{}
	if err := decodeResponse(production.GetSupplyMaterials(&request), &supplyRecords); err != nil {
		return 0, err
	}

	activeSupplies := 0
	for _, supplyRecord := range supplyRecords {
		if supplyRecord.Status == productionTypes.ProductStatusSupply {
			activeSupplies++
		}
	}
	return activeSupplies, nil
}

// productIDBySKU indexes the supply catalog so the asset seeds can name their material by SKU.
func (generator *supplyDataGenerator) productIDBySKU() (map[string]int32, error) {
	request := generator.makeRequest("GET.supply-material", map[string]string{"upv": "0"}, "")
	supplyRecords := []productionTypes.Product{}
	if err := decodeResponse(production.GetSupplyMaterials(&request), &supplyRecords); err != nil {
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
func (generator *supplyDataGenerator) seedAssets() error {
	existingAssets, countError := generator.countAssets()
	if countError != nil {
		return countError
	}
	if existingAssets >= generator.config.assetsCount {
		core.Log("GenerateSupplyData:: activos ya existentes, no se generan", "total", existingAssets)
		return nil
	}

	seeds, parseError := readAssetSeeds()
	if parseError != nil {
		return core.Err("error al leer assets.psv:", parseError)
	}
	if len(seeds) > generator.config.assetsCount {
		seeds = seeds[:generator.config.assetsCount]
	}

	idBySKU, indexError := generator.productIDBySKU()
	if indexError != nil {
		return indexError
	}

	warehouseID, warehouseError := generator.resolveWarehouseID()
	if warehouseError != nil {
		return warehouseError
	}
	cashBank, cashBankError := generator.resolveCashBank()
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

		acquisitionDate := todayUnixDay - int16(generator.randomInRange(assetMinDaysBack, assetMaxDaysBack))
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

		payload := accounting.AssetAcquisitionPayload{
			ProductID:        productID,
			SerialNumbers:    serialNumbers,
			Quantity:         quantity,
			WarehouseID:      warehouseID,
			AcquisitionDate:  acquisitionDate,
			Name:             seed.name,
			Description:      seed.description,
			SupplierID:       generator.providerIDs[generator.random.Intn(len(generator.providerIDs))],
			CurrencyType:     cashBank.CurrencyType,
			AcquisitionValue: seed.acquisitionValue,
			PurchaseAmount:   purchaseAmount,
			DueDate:          acquisitionDate + 30,
		}

		bodyBytes, marshalError := json.Marshal(payload)
		if marshalError != nil {
			return marshalError
		}

		request := generator.makeRequest("POST.asset", nil, string(bodyBytes))
		assetsOfAcquisition := []accountingTypes.Asset{}
		if err := decodeResponse(accounting.PostAsset(&request), &assetsOfAcquisition); err != nil {
			return core.Err("error al registrar el activo", seed.name, ":", err)
		}
		createdAssets = append(createdAssets, assetsOfAcquisition...)
	}

	core.Log("GenerateSupplyData:: activos registrados", "total", len(createdAssets),
		"comprados", purchasedCount, "donados", len(seeds)-purchasedCount)

	if err := generator.payAssets(createdAssets, cashBank); err != nil {
		return err
	}
	return generator.runAssetDepreciation()
}

func (generator *supplyDataGenerator) countAssets() (int, error) {
	request := generator.makeRequest("GET.assets", map[string]string{"upv": "0"}, "")
	assets := []accountingTypes.Asset{}
	if err := decodeResponse(accounting.GetAssets(&request), &assets); err != nil {
		return 0, err
	}
	return len(assets), nil
}

// payAssets settles the purchased assets in three groups, so the Activos page shows every payment
// state: half paid in full, a third partially, and the rest untouched.
func (generator *supplyDataGenerator) payAssets(
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
			paymentAmount = asset.PurchaseAmount * int32(generator.randomInRange(30, 70)) / 100
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
	if err := generator.ensureCashBalance(cashBank, totalToPay); err != nil {
		return err
	}

	for _, payment := range payments {
		bodyBytes, marshalError := json.Marshal(accounting.AssetPaymentPayload{
			AssetID:    payment.assetID,
			CashBankID: cashBank.ID,
			Amount:     payment.amount,
			Date:       payment.date,
		})
		if marshalError != nil {
			return marshalError
		}

		request := generator.makeRequest("POST.asset-payment", nil, string(bodyBytes))
		if response := accounting.PostAssetPayment(&request); response.StatusCode != 200 {
			return core.Err("error al pagar el activo", payment.assetID, ":", response.Error)
		}
	}

	core.Log("GenerateSupplyData:: pagos de activos registrados", "total", len(payments), "monto", totalToPay)
	return nil
}

// ensureCashBalance tops the till up when it cannot cover the payments, with one type 7 movement
// instead of one per asset.
func (generator *supplyDataGenerator) ensureCashBalance(cashBank financeTypes.CashBank, amountNeeded int32) error {
	if cashBank.CurrentAmount >= amountNeeded {
		return nil
	}

	amountToInject := amountNeeded - cashBank.CurrentAmount + assetCashPadding
	bodyBytes, marshalError := json.Marshal(financeTypes.CashBankMovement{
		CashBankID: cashBank.ID,
		Type:       7,
		Amount:     amountToInject,
		// The handler rejects the write unless FinalAmount - Amount matches the stored balance,
		// so it is sent pre-computed exactly like the frontend does.
		FinalAmount: cashBank.CurrentAmount + amountToInject,
	})
	if marshalError != nil {
		return marshalError
	}

	request := generator.makeRequest("POST.cash-banks-movement", nil, string(bodyBytes))
	if response := finance.PostCashBankMovement(&request); response.StatusCode != 200 {
		return core.Err("error al inyectar efectivo:", response.Error)
	}

	core.Log("GenerateSupplyData:: efectivo inyectado", "monto", amountToInject, "saldo previo", cashBank.CurrentAmount)
	return nil
}

// runAssetDepreciation materializes every period already due. Generation is lazy and keyed on each
// asset's own watermark, which is why the acquisitions are backdated: an asset bought today has no
// period to post and its schedule would come out empty.
func (generator *supplyDataGenerator) runAssetDepreciation() error {
	request := generator.makeRequest("POST.asset-depreciation-run", nil, "")
	depreciationResult := map[string]int{}
	if err := decodeResponse(accounting.PostAssetDepreciationRun(&request), &depreciationResult); err != nil {
		return core.Err("error al correr la depreciación:", err)
	}

	core.Log("GenerateSupplyData:: depreciación corrida",
		"asientos", depreciationResult["PostedEntries"], "activos", depreciationResult["UpdatedAssets"])
	return nil
}

// resolveWarehouseID picks the lowest active warehouse, so the script runs on a company whose
// warehouse IDs do not start at 1.
func (generator *supplyDataGenerator) resolveWarehouseID() (int32, error) {
	warehouses := []businessTypes.Warehouse{}
	warehouseQuery := db.Query(&warehouses)
	warehouseQuery.Select(warehouseQuery.ID, warehouseQuery.Status).
		CompanyID.Equals(supplyCompanyID).Status.Equals(1)

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

// resolveCashBank picks the lowest active till. Its currency becomes the assets' currency, because
// PostAssetPayment refuses a payment whose currency differs from the till's.
func (generator *supplyDataGenerator) resolveCashBank() (financeTypes.CashBank, error) {
	cashBanks := []financeTypes.CashBank{}
	cashBankQuery := db.Query(&cashBanks)
	cashBankQuery.Select(cashBankQuery.ID, cashBankQuery.Status, cashBankQuery.CurrencyType,
		cashBankQuery.CurrentAmount).
		CompanyID.Equals(supplyCompanyID).Status.Equals(1)

	if queryError := cashBankQuery.Exec(); queryError != nil {
		return financeTypes.CashBank{}, core.Err("error al consultar las cajas:", queryError)
	}
	if len(cashBanks) == 0 {
		return financeTypes.CashBank{}, core.Err("no hay cajas activas en la company")
	}

	slices.SortFunc(cashBanks, func(leftCashBank, rightCashBank financeTypes.CashBank) int {
		return int(leftCashBank.ID - rightCashBank.ID)
	})
	return cashBanks[0], nil
}
