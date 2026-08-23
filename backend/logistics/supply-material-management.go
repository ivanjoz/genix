package logistics

import (
	business "app/business/types"
	"app/core"
	"app/db"
	"app/logistics/types"
	"encoding/json"
)

// GetSupplyMaterials returns the supply catalog using the delta-cache protocol. The status
// slot is pinned to 2, so a first sync (upv=0) returns supplies only, and later syncs also
// carry rows that left the bucket (deleted, or converted to a product) so the client evicts them.
func GetSupplyMaterials(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("upv")

	supplyRecords := []business.Product{}
	supplyQuery := db.Query(&supplyRecords).CompanyID.Equals(req.User.CompanyID)

	// Supplies carry none of the storefront payload; excluding it keeps the sync small.
	supplyQuery.Select(
		supplyQuery.ID, supplyQuery.Name, supplyQuery.Description, supplyQuery.SKU,
		supplyQuery.BrandID, supplyQuery.Price, supplyQuery.CurrencyID, supplyQuery.UnitID,
		supplyQuery.DepreciationMonths, supplyQuery.Status, supplyQuery.Updated,
		supplyQuery.UpdatedVersion,
	)
	supplyQuery.Delta(updatedSince, int64(business.ProductStatusSupply))

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
	ID                 int32                            `json:",omitempty"`
	Name               string                           `json:",omitempty"`
	Description        string                           `json:",omitempty"`
	SKU                string                           `json:",omitempty"`
	BrandID            int32                            `json:",omitempty"`
	Price              int32                            `json:",omitempty"`
	CurrencyID         int16                            `json:",omitempty"`
	UnitID             int16                            `json:",omitempty"`
	DepreciationMonths int16                            `json:",omitempty"`
	MinimunStock       int32                            `json:",omitempty"`
	ProviderSupply     []types.ProductSupplyProviderRow `json:",omitempty"`
	// Only 0 is honoured, as the soft-delete tombstone. Anything else saves as a supply;
	// a client cannot promote its own row into the product catalog.
	Status int8 `json:"ss"`
}

// supplyMaterialIDMapping is the shape the frontend GetHandler.postAndSync expects back: for
// each input record, the resolved server ID paired with the client-sent (possibly negative,
// temporary) ID so optimistic local rows can be reconciled.
type supplyMaterialIDMapping struct {
	ID     int32 `json:",omitempty"`
	TempID int32 `json:",omitempty"`
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
	supplyProducts := make([]business.Product, len(incomingSupplies))
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

		supplyPayload.ProviderSupply = sanitizeProviderSupplyRows(supplyPayload.ProviderSupply)
		if validationError := validateProviderSupplyRows(req, supplyPayload.ProviderSupply); validationError != nil {
			return req.MakeErr(validationError)
		}

		supplyProducts[recordIndex] = business.Product{
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
			DepreciationMonths: supplyPayload.DepreciationMonths,
			// Deleting is only meaningful for a row that already exists; a new record with no
			// ss in the payload must not be born as a tombstone.
			Status:    core.If(supplyPayload.Status == 0 && supplyPayload.ID > 0, int8(0), business.ProductStatusSupply),
			Updated:   currentTimestamp,
			UpdatedBy: req.User.ID,
		}
		if supplyProducts[recordIndex].ID < 0 {
			// Negative IDs are the frontend's temporary keys; the autoincrement assigns the real one.
			supplyProducts[recordIndex].ID = 0
		}
	}

	productTable := db.TableOf[business.Product]()
	// Merge resolves insert vs. update per key. The excluded columns are the storefront and
	// stock-derived ones a supply never sets — leaving them out stops a save from blanking
	// values the stock engine maintains.
	mergeError := db.Merge(&supplyProducts,
		db.Cols(
			productTable.Stock, productTable.ReservedStock, productTable.StockStatus,
			productTable.CategoriesWithStock, productTable.Created, productTable.CreatedBy,
			productTable.ImageMain, productTable.ImageIDs, productTable.ImageDescriptions,
			productTable.Presentations, productTable.Properties, productTable.ContentHTML,
			productTable.CategoryIDs,
		),
		func(previousProduct, currentProduct *business.Product) bool {
			currentProduct.CompanyID = req.User.CompanyID
			currentProduct.Created = previousProduct.Created
			currentProduct.CreatedBy = previousProduct.CreatedBy
			return true
		},
		func(currentProduct *business.Product) {
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
	supplyConfigs := make([]types.ProductSupply, len(incomingSupplies))
	for recordIndex := range incomingSupplies {
		supplyConfigs[recordIndex] = types.ProductSupply{
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
		func(previousConfig, currentConfig *types.ProductSupply) bool {
			// SalesPerDayEstimated is owned by the products page; a supply save must not clear it.
			currentConfig.SalesPerDayEstimated = previousConfig.SalesPerDayEstimated
			return true
		},
		nil,
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
