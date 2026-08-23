package accounting

import (
	accountingTypes "app/accounting/types"
	businessTypes "app/business/types"
	"app/core"
	"app/db"
	"app/logistics"
	logisticsTypes "app/logistics/types"
	"encoding/json"
)

// GetAssets returns the asset register using the delta-cache protocol. Depreciation is not
// generated here — a list read does not write. The page calls PostAssetDepreciationRun first.
func GetAssets(req *core.HandlerArgs) core.HandlerResponse {
	// Delta syncs are watermarked by "upv", the write sequence number, not by a timestamp: two
	// writes in the same second are distinguishable, so nothing is re-sent and nothing is skipped.
	updatedSince := req.GetQueryInt("upv")

	assets := []accountingTypes.Asset{}
	assetQuery := db.Query(&assets).CompanyID.Equals(req.User.CompanyID)
	// Active, fully depreciated and disposed assets all stay on the register; only a removed
	// row (status 0) leaves it, and the delta carries it so the client can evict it.
	assetQuery.Delta(updatedSince,
		int64(accountingTypes.AssetStatusActive),
		int64(accountingTypes.AssetStatusFullyDepreciated),
		int64(accountingTypes.AssetStatusDisposed),
	)

	if queryError := assetQuery.Exec(); queryError != nil {
		core.Log("GetAssets query error:", queryError)
		return req.MakeErr("Error al obtener los activos.", queryError)
	}

	core.Log("GetAssets result_count:", len(assets))
	return req.MakeResponse(assets)
}

// AssetAcquisitionPayload registers one acquisition. It may produce several Asset rows: one
// per serial number when the units are tracked individually, or a single grouped row when
// they are not. Whether serials were entered is what decides the granularity.
type AssetAcquisitionPayload struct {
	ProductID     int32    `json:",omitempty"`
	SerialNumbers []string `json:",omitempty"`
	// Used only when SerialNumbers is empty — the size of the grouped acquisition lot.
	Quantity        int32  `json:",omitempty"`
	WarehouseID     int32  `json:",omitempty"`
	AcquisitionDate int16  `json:",omitempty"`
	Name            string `json:",omitempty"`
	Description     string `json:",omitempty"`
	SupplierID      int32  `json:",omitempty"`
	CurrencyType    int8   `json:",omitempty"`
	// Book value per unit, in cents. This is what depreciates.
	AcquisitionValue int32 `json:",omitempty"`
	// Cash owed per unit, in cents. 0 means the asset was donated or contributed: nothing is
	// owed and nothing is payable, but the asset still has a value and still depreciates.
	PurchaseAmount int32 `json:",omitempty"`
	DueDate        int16 `json:",omitempty"`
	// Overrides the product's own term. 0 inherits it.
	DepreciationMonths int16 `json:",omitempty"`
}

// PostAsset acquires one or more assets. It writes no Expense: buying a fixed asset is cash
// turning into a balance-sheet item, not a cost, so the money owed lives on the Asset and is
// settled through PostAssetPayment. The only expense an asset ever produces is depreciation.
func PostAsset(req *core.HandlerArgs) core.HandlerResponse {
	payload := AssetAcquisitionPayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &payload); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}

	// 1. The catalog row has to exist, be a supply, and be depreciable — an asset that does
	//    not depreciate is just stock, and belongs in an inventory expense instead.
	supplyProducts := []businessTypes.Product{}
	supplyQuery := db.Query(&supplyProducts)
	supplyQuery.Select(
		supplyQuery.ID, supplyQuery.Name, supplyQuery.DepreciationMonths, supplyQuery.Status,
	).CompanyID.Equals(req.User.CompanyID).ID.Equals(payload.ProductID)

	if queryError := supplyQuery.Exec(); queryError != nil {
		return req.MakeErr("Error al obtener el insumo.", queryError)
	}
	if len(supplyProducts) == 0 {
		return req.MakeErr("No se encontró el insumo indicado.")
	}
	supplyProduct := supplyProducts[0]
	if supplyProduct.Status != businessTypes.ProductStatusSupply {
		return req.MakeErr("El registro seleccionado no es un insumo o material.")
	}

	depreciationMonths := core.If(
		payload.DepreciationMonths > 0, payload.DepreciationMonths, supplyProduct.DepreciationMonths,
	)
	if depreciationMonths <= 0 {
		return req.MakeErr("El insumo no tiene un esquema de depreciación configurado.")
	}

	// 2. Serials decide the granularity: one asset per serial, otherwise one grouped row.
	serialNumbers := make([]string, 0, len(payload.SerialNumbers))
	for _, serialNumber := range payload.SerialNumbers {
		if len(serialNumber) > 0 {
			serialNumbers = append(serialNumbers, serialNumber)
		}
	}
	unitCount := int32(len(serialNumbers))
	if unitCount == 0 {
		unitCount = payload.Quantity
	}
	if unitCount <= 0 {
		return req.MakeErr("Debe indicar la cantidad o al menos un número de serie.")
	}
	if payload.WarehouseID <= 0 {
		return req.MakeErr("Debe seleccionar el almacén donde queda el activo.")
	}
	if payload.AcquisitionValue <= 0 {
		return req.MakeErr("El valor del activo debe ser mayor a 0.")
	}
	if payload.PurchaseAmount < 0 {
		return req.MakeErr("El monto pagado no puede ser negativo.")
	}
	if payload.CurrencyType != 1 && payload.CurrencyType != 2 {
		return req.MakeErr("Moneda inválida (debe ser 1=PEN o 2=USD).")
	}
	acquisitionDate := core.If(payload.AcquisitionDate > 0, payload.AcquisitionDate, core.FechaUnix())
	assetName := core.If(len(payload.Name) > 0, payload.Name, supplyProduct.Name)

	currentTimestamp := core.SUnixTime()

	// 3. The asset rows. They are inserted before the stock movement so each movement can
	//    carry its own asset as DocumentID — the ORM assigns the IDs during the insert.
	newAssets := []accountingTypes.Asset{}
	makeAsset := func(serialNumber string, quantity int32) accountingTypes.Asset {
		purchaseAmount := payload.PurchaseAmount * quantity
		return accountingTypes.Asset{
			CompanyID:    req.User.CompanyID,
			ProductID:    payload.ProductID,
			SerialNumber: serialNumber,
			Quantity:     quantity,
			WarehouseID:  payload.WarehouseID,
			Name:         assetName,
			Description:  payload.Description,
			SupplierID:   payload.SupplierID,
			// A donation owes nothing, so it is never payable; anything else starts pending.
			PurchaseAmount:     purchaseAmount,
			PaymentStatus:      core.If(purchaseAmount > 0, accountingTypes.AssetPaymentPending, accountingTypes.AssetPaymentNone),
			DueDate:            core.If(payload.DueDate > 0, payload.DueDate, acquisitionDate),
			AcquisitionDate:    acquisitionDate,
			AcquisitionValue:   payload.AcquisitionValue * quantity,
			DepreciationMonths: depreciationMonths,
			CurrencyType:       payload.CurrencyType,
			Status:             accountingTypes.AssetStatusActive,
			Updated:            currentTimestamp,
			UpdatedBy:          req.User.ID,
			Created:            currentTimestamp,
			CreatedBy:          req.User.ID,
		}
	}
	if len(serialNumbers) > 0 {
		for _, serialNumber := range serialNumbers {
			newAssets = append(newAssets, makeAsset(serialNumber, 1))
		}
	} else {
		newAssets = append(newAssets, makeAsset("", unitCount))
	}

	if insertError := db.Insert(&newAssets); insertError != nil {
		return req.MakeErr("Error al registrar el activo.", insertError)
	}

	// 4. The asset's existence is its stock, so the units move in through the same engine as
	//    anything else. A serial-tracked unit lands on its own ProductStockDetail row.
	movements := make([]logisticsTypes.InternalMovement, len(newAssets))
	for assetIndex := range newAssets {
		movements[assetIndex] = logisticsTypes.InternalMovement{
			ProductID:    payload.ProductID,
			WarehouseID:  payload.WarehouseID,
			SerialNumber: newAssets[assetIndex].SerialNumber,
			SupplierID:   payload.SupplierID,
			Quantity:     newAssets[assetIndex].Quantity,
			Price:        payload.AcquisitionValue,
			DocumentID:   int64(newAssets[assetIndex].ID),
		}
	}
	if movementError := logistics.ApplyMovimientos(req, movements); movementError != nil {
		return req.MakeErr("Error al ingresar el activo al almacén.", movementError)
	}

	core.Log("PostAsset created:", len(newAssets))
	return req.MakeResponse(newAssets)
}

// AssetDisposalPayload retires an asset from the register and takes its units out of stock.
type AssetDisposalPayload struct {
	AssetID      int32  `json:",omitempty"`
	DisposalDate int16  `json:",omitempty"`
	Notes        string `json:",omitempty"`
}

// PutAssetDisposal disposes an asset: it stops depreciating, leaves the warehouse, and stays
// on the register with its history intact — which is why the accounting row is not the stock row.
func PutAssetDisposal(req *core.HandlerArgs) core.HandlerResponse {
	payload := AssetDisposalPayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &payload); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if payload.AssetID <= 0 {
		return req.MakeErr("Debe indicar el activo a dar de baja.")
	}

	asset, loadError := loadAsset(req.User.CompanyID, payload.AssetID)
	if loadError != nil {
		return req.MakeErr(loadError)
	}
	if asset.DisposalDate > 0 {
		return req.MakeErr("El activo ya fue dado de baja.")
	}

	disposalDate := core.If(payload.DisposalDate > 0, payload.DisposalDate, core.FechaUnix())
	if disposalDate < asset.AcquisitionDate {
		return req.MakeErr("La fecha de baja no puede ser anterior a la fecha de adquisición.")
	}

	// Take the units out of the warehouse through the ordinary movement path.
	outboundMovement := logisticsTypes.InternalMovement{
		ProductID:    asset.ProductID,
		WarehouseID:  asset.WarehouseID,
		SerialNumber: asset.SerialNumber,
		Quantity:     -asset.Quantity,
		DocumentID:   int64(asset.ID),
	}
	if movementError := logistics.ApplyMovimientos(
		req, []logisticsTypes.InternalMovement{outboundMovement},
	); movementError != nil {
		return req.MakeErr("Error al retirar el activo del almacén.", movementError)
	}

	asset.DisposalDate = disposalDate
	asset.Status = ResolveAssetStatus(&asset)
	asset.Updated = core.SUnixTime()
	asset.UpdatedBy = req.User.ID

	assetTable := db.TableOf[accountingTypes.Asset]()
	assetRecords := []accountingTypes.Asset{asset}
	if updateError := db.Update(&assetRecords,
		assetTable.DisposalDate, assetTable.Status, assetTable.Updated, assetTable.UpdatedBy,
	); updateError != nil {
		return req.MakeErr("Error al dar de baja el activo.", updateError)
	}

	return req.MakeResponse(assetRecords[0])
}

// AssetTransferPayload moves an asset between warehouses. The accounting identity does not
// change — that is exactly why Asset has its own ID rather than being keyed by the stock row.
type AssetTransferPayload struct {
	AssetID           int32 `json:",omitempty"`
	TargetWarehouseID int32 `json:",omitempty"`
}

func PutAssetTransfer(req *core.HandlerArgs) core.HandlerResponse {
	payload := AssetTransferPayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &payload); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if payload.AssetID <= 0 || payload.TargetWarehouseID <= 0 {
		return req.MakeErr("Debe indicar el activo y el almacén de destino.")
	}

	asset, loadError := loadAsset(req.User.CompanyID, payload.AssetID)
	if loadError != nil {
		return req.MakeErr(loadError)
	}
	if asset.DisposalDate > 0 {
		return req.MakeErr("No se puede trasladar un activo dado de baja.")
	}
	if asset.WarehouseID == payload.TargetWarehouseID {
		return req.MakeErr("El activo ya está en ese almacén.")
	}

	transferMovement := logisticsTypes.InternalMovement{
		ProductID:       asset.ProductID,
		WarehouseID:     asset.WarehouseID,
		DestWarehouseID: payload.TargetWarehouseID,
		SerialNumber:    asset.SerialNumber,
		Quantity:        asset.Quantity,
		DocumentID:      int64(asset.ID),
	}
	if movementError := logistics.ApplyMovimientos(
		req, []logisticsTypes.InternalMovement{transferMovement},
	); movementError != nil {
		return req.MakeErr("Error al trasladar el activo.", movementError)
	}

	asset.WarehouseID = payload.TargetWarehouseID
	asset.Updated = core.SUnixTime()
	asset.UpdatedBy = req.User.ID

	assetTable := db.TableOf[accountingTypes.Asset]()
	assetRecords := []accountingTypes.Asset{asset}
	// Status rides along for the delta view's composite key, as in PostAssetPayment.
	if updateError := db.Update(&assetRecords,
		assetTable.WarehouseID, assetTable.Status, assetTable.Updated, assetTable.UpdatedBy,
	); updateError != nil {
		return req.MakeErr("Error al actualizar el activo.", updateError)
	}

	return req.MakeResponse(assetRecords[0])
}

func loadAsset(companyID, assetID int32) (accountingTypes.Asset, error) {
	assets := []accountingTypes.Asset{}
	assetQuery := db.Query(&assets)
	assetQuery.Select().CompanyID.Equals(companyID).ID.Equals(assetID)

	if queryError := assetQuery.Exec(); queryError != nil {
		return accountingTypes.Asset{}, core.Err("Error al obtener el activo.", queryError)
	}
	if len(assets) == 0 {
		return accountingTypes.Asset{}, core.Err("No se encontró el activo.")
	}
	return assets[0], nil
}
