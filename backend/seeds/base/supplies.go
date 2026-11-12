package base

import (
	"app/core"
	"app/crm"
	crmTypes "app/crm/types"
	"app/production"
	productionTypes "app/production/types"
	"app/seeds/runner"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
)

//go:embed supply_providers.json
var supplyProviderSeedJSON []byte

// seedSupplyProviders writes the 100 static providers and keeps the IDs the handler resolved. The
// save deduplicates on RegistryNumber, so a second run matches the existing rows instead of creating
// a second set, and the response already carries their IDs.
func seedSupplyProviders(run *runner.Run) error {
	providerSeeds := []runner.PartySeed{}
	if err := json.Unmarshal(supplyProviderSeedJSON, &providerSeeds); err != nil {
		return core.Err("error al leer el JSON de proveedores:", err)
	}
	if len(providerSeeds) == 0 {
		return core.Err("el JSON de proveedores está vacío")
	}

	payload := make([]crmTypes.ClientProvider, 0, len(providerSeeds))
	for seedIndex, seed := range providerSeeds {
		if strings.TrimSpace(seed.Name) == "" {
			return core.Err("el proveedor en posición", seedIndex, "no tiene Name")
		}
		payload = append(payload, crmTypes.ClientProvider{
			Type:           crmTypes.ClientProviderTypeProvider,
			Name:           seed.Name,
			PersonType:     seed.PersonType,
			RegistryNumber: strings.TrimSpace(seed.RegistryNumber),
			CountryID:      seed.CountryID,
			CityID:         seed.CityID,
		})
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request := run.MakeRequest("POST.client-provider", nil, string(bodyBytes))
	savedProviders := []crmTypes.ClientProvider{}
	if err := runner.DecodeResponse(run.Send(request, crm.PostClientProviders), &savedProviders); err != nil {
		return err
	}

	run.SupplyProviderIDs = make([]int32, 0, len(savedProviders))
	for _, savedProvider := range savedProviders {
		if savedProvider.ID > 0 {
			run.SupplyProviderIDs = append(run.SupplyProviderIDs, savedProvider.ID)
		}
	}
	if len(run.SupplyProviderIDs) < run.Config.ProvidersPerProduct {
		return core.Err("se resolvieron sólo", len(run.SupplyProviderIDs), "proveedores")
	}

	core.Log("Seed:: proveedores listos", "total", len(run.SupplyProviderIDs))
	return nil
}

//go:embed supply_materials.psv
var supplyMaterialsPSV string

type supplyMaterialSeed struct {
	name               string
	sku                string
	price              int32
	unitID             int16
	currencyID         int16
	depreciationMonths int16
}

func readSupplyMaterialSeeds() ([]supplyMaterialSeed, error) {
	rows, parseError := runner.ParsePSV(supplyMaterialsPSV, 6)
	if parseError != nil {
		return nil, parseError
	}

	seeds := make([]supplyMaterialSeed, 0, len(rows))
	for rowIndex, row := range rows {
		price, priceError := runner.ParsePriceToCents(row[2])
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

// seedSupplyMaterials writes the static supply catalog, unless the company already has enough.
// `POST.supply-material` inserts rather than matching on SKU, so without this count a rerun would
// stack a second hundred on top of the first.
func seedSupplyMaterials(run *runner.Run) error {
	existingSupplies, countError := countSupplyMaterials(run)
	if countError != nil {
		return countError
	}
	if existingSupplies >= run.Config.SupplyMaterials {
		core.Log("Seed:: insumos ya existentes, no se generan", "total", existingSupplies)
		return nil
	}

	seeds, parseError := readSupplyMaterialSeeds()
	if parseError != nil {
		return core.Err("error al leer supply_materials.psv:", parseError)
	}
	if len(seeds) > run.Config.SupplyMaterials {
		seeds = seeds[:run.Config.SupplyMaterials]
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
				int32(run.RandomInRange(run.Config.MinStockMin, run.Config.MinStockMax))),
			Status: productionTypes.ProductStatusSupply,
		})
	}

	bodyBytes, marshalError := json.Marshal(payload)
	if marshalError != nil {
		return marshalError
	}

	request := run.MakeRequest("POST.supply-material", nil, string(bodyBytes))
	if response := run.Send(request, production.PostSupplyMaterial); response.StatusCode != 200 {
		return core.Err(response.Error)
	}

	core.Log("Seed:: insumos generados", "total", len(payload))
	return nil
}

func countSupplyMaterials(run *runner.Run) (int, error) {
	request := run.MakeRequest("GET.supply-material", map[string]string{}, "")
	supplyRecords := []productionTypes.Product{}
	if err := runner.DecodeResponse(run.Send(request, production.GetSupplyMaterials), &supplyRecords); err != nil {
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
