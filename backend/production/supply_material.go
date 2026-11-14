package production

import (
	accounting "app/accounting/types"
	"app/core"
	"app/db"
	logistics "app/logistics/types"
	"app/production/types"
	"encoding/json"
	"fmt"
)

// GetSupplyMaterials returns the supply catalog using the delta-cache protocol. A first sync
// lists supplies (status 2) only; later syncs read the log, which names every status, so a row
// converted to a product comes back with ss=1 and the client filters it out of the supply list.
func GetSupplyMaterials(req *core.HandlerArgs) core.HandlerResponse {
	supplyRecords := []types.Product{}
	supplyQuery := db.Query(&supplyRecords).CompanyID.Equals(req.User.CompanyID)

	// Supplies carry none of the storefront payload; excluding it keeps the sync small.
	supplyQuery.Select(
		supplyQuery.ID, supplyQuery.Name, supplyQuery.Description, supplyQuery.SKU,
		supplyQuery.BrandID, supplyQuery.Price, supplyQuery.CurrencyID, supplyQuery.UnitID,
		supplyQuery.SbuQuantity, supplyQuery.SbuUnit,
		supplyQuery.DepreciationMonths, supplyQuery.AccountCode, supplyQuery.Status, supplyQuery.Updated,
		supplyQuery.Version, supplyQuery.Origin, supplyQuery.ProductionYieldPortions, supplyQuery.ProductionSupplies,
		supplyQuery.ProductionExtraCostPerUnit, supplyQuery.ProductionStandardCost, supplyQuery.ProductionLabor,
		supplyQuery.TracksLots, supplyQuery.ShelfLifeDays, supplyQuery.QualityHoldOnEntry,
	)
	supplyQuery.Delta(req.GetDeltaCursor(), types.ProductStatusSupply)

	if queryError := supplyQuery.Exec(); queryError != nil {
		core.Log("GetSupplyMaterials query error:", queryError)
		return req.MakeErr("Error al obtener los insumos.", queryError)
	}

	core.Log("GetSupplyMaterials result_count:", len(supplyRecords))
	return req.MakeResponse(supplyRecords)
}

// SupplyMaterialPayload is what the Supplies & Materials form sends: the catalog fields that
// live on Product plus the replenishment config that lives on ProductSupply. Both are saved
// in one call because the user sees a single record.
type SupplyMaterialPayload struct {
	ID                 int32                                `cb:"1"`
	Name               string                               `cb:"2"`
	Description        string                               `cb:"3"`
	SKU                string                               `cb:"4"`
	BrandID            int32                                `cb:"5"`
	Price              int32                                `cb:"6"`
	CurrencyID         int16                                `cb:"7"`
	UnitID             int16                                `cb:"8"`
	DepreciationMonths int16                                `cb:"9"`
	MinimunStock       int32                                `cb:"10"`
	ProviderSupply     []logistics.ProductSupplyProviderRow `cb:"11"`
	// Only 0 is honoured, as the soft-delete tombstone. Anything else saves as a supply;
	// a client cannot promote its own row into the product catalog.
	Status int8 `json:"ss" cb:"ss,12"`
	// The supply's default PCGE account. 0 takes accounting.DefaultSupplyAccountCode.
	AccountCode int32 `cb:"13"`
	// The sub-unit, as on a product: SbuQuantity sub-units make one unit (1000 g in a kg), so a
	// recipe can consume a fraction of the supply. Same refinement-only rule as a product.
	SbuQuantity int16  `cb:"14"`
	SbuUnit     string `cb:"15"`
	// Origin = ProductOriginMade makes the supply semi-finished (docs/MANUFACTURING_PLAN.md §2.1): the
	// company makes it with the recipe below, and its stock account is 231 by that origin. 0 otherwise.
	Origin                     int8                     `cb:"16"`
	ProductionYieldPortions    int16                    `cb:"17"`
	ProductionSupplies         []types.ProductionSupply `cb:"18"`
	ProductionExtraCostPerUnit int32                    `cb:"19"`
	ProductionStandardCost     int32                    `cb:"20"`
	ProductionLabor            []types.ProductionLabor  `cb:"21"`
	// The lot policy, as on a product (docs/MANUFACTURING_PLAN.md §10).
	TracksLots         bool  `cb:"22"`
	ShelfLifeDays      int16 `cb:"23"`
	QualityHoldOnEntry bool  `cb:"24"`
}

// supplyMaterialIDMapping is the shape the frontend GetHandler.postAndSync expects back: for
// each input record, the resolved server ID paired with the client-sent (possibly negative,
// temporary) ID so optimistic local rows can be reconciled.
type supplyMaterialIDMapping struct {
	ID     int32 `cb:"1"`
	TempID int32 `cb:"2"`
}

// validateStoredSupplyChanges checks the edit of stored supplies. The divisor follows the product
// rules: a valid, named divisor, refined only to a multiple of the stored one, since the supply's
// stock and the recipes that consume it are expressed in it. The origin may not change while the
// supply has stock: its stock account would move between 231 (semi-finished) and its purchase one.
func validateStoredSupplyChanges(companyID int32, supplyProducts []types.Product) error {
	storedIDs := core.SliceSet[int32]{}
	for _, supply := range supplyProducts {
		storedIDs.AddIf(supply.ID)
	}
	storedByID := map[int32]types.Product{}
	if !storedIDs.IsEmpty() {
		storedSupplies := []types.Product{}
		query := db.Query(&storedSupplies)
		query.Select(query.ID, query.Name, query.SbuQuantity, query.Origin).CompanyID.Equals(companyID).ID.In(storedIDs.Values...)
		if err := query.Exec(); err != nil {
			return core.Err("Error al obtener los insumos guardados:", err)
		}
		for _, stored := range storedSupplies {
			storedByID[stored.ID] = stored
		}
	}
	if err := validateProductSubUnitDivisors(supplyProducts, storedByID); err != nil {
		return err
	}
	for _, supply := range supplyProducts {
		stored, isStored := storedByID[supply.ID]
		if !isStored || stored.Origin == supply.Origin {
			continue
		}
		hasStock, err := productHasStock(companyID, stored)
		if err != nil {
			return err
		}
		if hasStock {
			return core.Err(fmt.Sprintf(`"%s" tiene stock: no puede cambiar entre semielaborado y comprado mientras lo tenga.`, stored.Name))
		}
	}
	return nil
}

// PostSupplyMaterial upserts a batch of supplies as Product rows with Status = 2, then writes
// each one's ProductSupply configuration. Response is []{ID, TempID}.
func PostSupplyMaterial(req *core.HandlerArgs) core.HandlerResponse {
	incomingSupplies := []SupplyMaterialPayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &incomingSupplies); deserializeError != nil {
		core.Log("PostSupplyMaterial deserialization error:", deserializeError)
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if len(incomingSupplies) == 0 {
		return req.MakeErr("No se recibieron insumos para guardar.")
	}

	currentTimestamp := core.SUnixTime()
	supplyProducts := make([]types.Product, len(incomingSupplies))
	// Preserve the client-sent ID per record so we can return ID mappings after the
	// autoincrement assigns real IDs to the inserts.
	clientSentIDs := make([]int32, len(incomingSupplies))

	for recordIndex := range incomingSupplies {
		supplyPayload := &incomingSupplies[recordIndex]
		clientSentIDs[recordIndex] = supplyPayload.ID

		if len(supplyPayload.Name) < 2 {
			return req.MakeErr("El nombre del insumo debe tener al menos 2 caracteres.")
		}
		if supplyPayload.Price < 0 {
			return req.MakeErr("El precio del insumo no puede ser negativo.")
		}
		if supplyPayload.MinimunStock < 0 {
			return req.MakeErr("El stock mínimo no puede ser negativo.")
		}
		// A depreciation term is what makes a supply an asset; a negative one is meaningless,
		// and anything past a century is a typo rather than an intent.
		if supplyPayload.DepreciationMonths < 0 || supplyPayload.DepreciationMonths > 1200 {
			return req.MakeErr("Los meses de depreciación deben estar entre 0 y 1200.")
		}
		if supplyPayload.ShelfLifeDays < 0 {
			return req.MakeErr("La vida útil del insumo no puede ser negativa.")
		}

		isSemiFinished := supplyPayload.Origin == types.ProductOriginMade
		if supplyPayload.Origin != 0 && !isSemiFinished {
			return req.MakeErr(fmt.Sprintf(`Insumo "%s": el origen %v no es válido.`, supplyPayload.Name, supplyPayload.Origin))
		}
		if isSemiFinished {
			// Made, not bought: its accounts follow from the origin (231 / 713), and it is an
			// inventory item, never an asset.
			if supplyPayload.DepreciationMonths != 0 {
				return req.MakeErr(fmt.Sprintf(`Insumo "%s": un semielaborado no se deprecia.`, supplyPayload.Name))
			}
			supplyPayload.AccountCode = 0
		} else {
			// Only a semi-finished supply has a recipe.
			supplyPayload.ProductionYieldPortions, supplyPayload.ProductionSupplies = 0, nil
			supplyPayload.ProductionExtraCostPerUnit, supplyPayload.ProductionStandardCost = 0, 0
			supplyPayload.ProductionLabor = nil
			if supplyPayload.AccountCode == 0 {
				supplyPayload.AccountCode = accounting.DefaultSupplyAccountCode(supplyPayload.DepreciationMonths)
			}
			if accountError := accounting.ValidatePCGEAccount(accounting.PCGEKindSupply, supplyPayload.AccountCode); accountError != nil {
				return req.MakeErr(accountError)
			}
		}

		supplyPayload.ProviderSupply = logistics.SanitizeProviderSupplyRows(supplyPayload.ProviderSupply)
		if validationError := logistics.ValidateProviderSupplyRows(req.User.CompanyID, supplyPayload.ProviderSupply); validationError != nil {
			return req.MakeErr(validationError)
		}

		supplyProducts[recordIndex] = types.Product{
			CompanyID:          req.User.CompanyID,
			ID:                 supplyPayload.ID,
			TempID:             supplyPayload.ID,
			Name:               supplyPayload.Name,
			Description:        supplyPayload.Description,
			SKU:                supplyPayload.SKU,
			BrandID:            supplyPayload.BrandID,
			Price:              supplyPayload.Price,
			CurrencyID:         supplyPayload.CurrencyID,
			UnitID:             supplyPayload.UnitID,
			SbuQuantity:        supplyPayload.SbuQuantity,
			SbuUnit:            supplyPayload.SbuUnit,
			DepreciationMonths: supplyPayload.DepreciationMonths,
			AccountCode:        supplyPayload.AccountCode,
			Origin:             supplyPayload.Origin,
			// The recipe: zeroed above unless the supply is semi-finished.
			ProductionYieldPortions:    supplyPayload.ProductionYieldPortions,
			ProductionSupplies:         supplyPayload.ProductionSupplies,
			ProductionExtraCostPerUnit: supplyPayload.ProductionExtraCostPerUnit,
			ProductionStandardCost:     supplyPayload.ProductionStandardCost,
			ProductionLabor:            supplyPayload.ProductionLabor,
			TracksLots:                 supplyPayload.TracksLots,
			ShelfLifeDays:              supplyPayload.ShelfLifeDays,
			QualityHoldOnEntry:         supplyPayload.QualityHoldOnEntry,
			// Deleting is only meaningful for a row that already exists; a new record with no
			// ss in the payload must not be born as a tombstone.
			Status:    core.If(supplyPayload.Status == 0 && supplyPayload.ID > 0, int8(0), types.ProductStatusSupply),
			Updated:   currentTimestamp,
			UpdatedBy: req.User.ID,
		}
		if supplyProducts[recordIndex].ID < 0 {
			// Negative IDs are the frontend's temporary keys; the autoincrement assigns the real one.
			supplyProducts[recordIndex].ID = 0
		}
	}

	if storedError := validateStoredSupplyChanges(req.User.CompanyID, supplyProducts); storedError != nil {
		return req.MakeErr(storedError)
	}
	if productionError := validateProductsProduction(req.User.CompanyID, supplyProducts); productionError != nil {
		return req.MakeErr(productionError)
	}

	// The merge writes BrandName, the brand half of the product text index.
	brandNamesByID, brandError := getProductBrandNames(req.User.CompanyID, supplyProducts)
	if brandError != nil {
		return req.MakeErr(brandError)
	}
	for recordIndex := range supplyProducts {
		supplyProducts[recordIndex].BrandName = brandNamesByID[supplyProducts[recordIndex].BrandID]
	}

	productTable := db.TableOf[types.Product]()
	// Merge resolves insert vs. update per key. The excluded columns are the storefront and
	// stock-derived ones a supply never sets — leaving them out stops a save from blanking
	// values the stock engine maintains. The recipe columns are written: a semi-finished supply
	// sends its recipe, any other supply clears it.
	mergeError := db.Merge(&supplyProducts,
		db.Cols(
			productTable.Stock, productTable.ReservedStock, productTable.StockStatus,
			productTable.CategoriesWithStock, productTable.Created, productTable.CreatedBy,
			productTable.ImageMain, productTable.ImageIDs, productTable.ImageDescriptions,
			productTable.Presentations, productTable.Properties, productTable.ContentHTML,
			productTable.CategoryIDs,
		),
		func(previousProduct, currentProduct *types.Product) bool {
			currentProduct.CompanyID = req.User.CompanyID
			currentProduct.Created = previousProduct.Created
			currentProduct.CreatedBy = previousProduct.CreatedBy
			return true
		},
		func(currentProduct *types.Product) {
			currentProduct.Created = currentTimestamp
			currentProduct.CreatedBy = req.User.ID
		},
	)
	if mergeError != nil {
		core.Log("PostSupplyMaterial merge error:", mergeError)
		return req.MakeErr("Error al guardar los insumos.", mergeError)
	}

	// ProductSupply is keyed by ProductID, so it can only be written once the inserts above
	// have real IDs. This is the same table the products page uses for replenishment config.
	supplyConfigs := make([]logistics.ProductSupply, len(incomingSupplies))
	for recordIndex := range incomingSupplies {
		supplyConfigs[recordIndex] = logistics.ProductSupply{
			CompanyID:      req.User.CompanyID,
			ProductID:      supplyProducts[recordIndex].ID,
			MinimunStock:   incomingSupplies[recordIndex].MinimunStock,
			ProviderSupply: incomingSupplies[recordIndex].ProviderSupply,
			Status:         core.If(supplyProducts[recordIndex].Status == 0, int8(0), int8(1)),
			Updated:        currentTimestamp,
			UpdatedBy:      req.User.ID,
		}
	}

	configMergeError := db.Merge(&supplyConfigs, nil,
		func(previousConfig, currentConfig *logistics.ProductSupply) bool {
			// SalesPerDayEstimated is owned by the products page; a supply save must not clear it.
			currentConfig.SalesPerDayEstimated = previousConfig.SalesPerDayEstimated
			return true
		},
		// Every column of supplyConfigs is already filled above from the payload plus the
		// server-owned metadata, so an insert has nothing left to complete.
		func(_ *logistics.ProductSupply) {},
	)
	if configMergeError != nil {
		core.Log("PostSupplyMaterial product_supply merge error:", configMergeError)
		return req.MakeErr("Error al guardar la configuración de abastecimiento del insumo.", configMergeError)
	}

	idMappings := make([]supplyMaterialIDMapping, len(supplyProducts))
	for recordIndex := range supplyProducts {
		idMappings[recordIndex] = supplyMaterialIDMapping{
			ID:     supplyProducts[recordIndex].ID,
			TempID: clientSentIDs[recordIndex],
		}
	}

	return req.MakeResponse(idMappings)
}
