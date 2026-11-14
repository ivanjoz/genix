package dining

import (
	business "app/business/types"
	"app/core"
	"app/db"
	"app/dining/types"
	"encoding/json"
	"fmt"
	"slices"
)

// GetDiningLayouts is the delta of the company's active layouts; a deleted one arrives as a removed ID.
func GetDiningLayouts(req *core.HandlerArgs) core.HandlerResponse {
	layouts := []types.DiningLayout{}
	query := db.Query(&layouts)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("records"), int8(1))
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los salones:", err)
	}
	return req.MakeResponse(map[string]any{"records": &layouts})
}

// PostDiningLayout creates or updates one layout (docs/DINING_PLAN.md §1.2). The editor saves the
// whole plan: its new shapes get their IDs here, never from the browser. ss = 0 deletes it.
func PostDiningLayout(req *core.HandlerArgs) core.HandlerResponse {
	layout := types.DiningLayout{}
	if err := json.Unmarshal([]byte(*req.Body), &layout); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	layout.CompanyID = companyID
	layout.UpdatedBy = req.User.ID
	layout.Updated = core.SUnixTime()

	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var stored *types.DiningLayout
		if layout.ID > 0 {
			storedLayouts := []types.DiningLayout{{CompanyID: companyID, ID: layout.ID}}
			if err := db.LockByKeys(transaction, &storedLayouts); err != nil {
				return core.Err("Error al obtener el salón:", err)
			}
			if len(storedLayouts) == 0 || storedLayouts[0].Status == 0 {
				return core.Err("El salón no existe.")
			}
			stored = &storedLayouts[0]
		} else if layout.Status == 0 {
			return core.Err("No se puede eliminar un salón que no se ha guardado.")
		}

		// The QR version is only bumped by its own action (dining-layout-qr-regenerate).
		layout.QRVersion = 1
		if stored != nil {
			layout.QRVersion = stored.QRVersion
		}
		if layout.Status != 0 {
			layout.Status = 1
			types.NormalizeDiningLayout(&layout)
			if err := types.AssignLayoutShapeIDs(&layout, stored); err != nil {
				return err
			}
			if err := types.ValidateDiningLayout(layout); err != nil {
				return err
			}
			if err := validateLayoutWarehouse(companyID, layout.WarehouseID); err != nil {
				return err
			}
		} else {
			// A delete keeps the plan as it was: only its status changes.
			deletedLayout := *stored
			deletedLayout.Status, deletedLayout.Updated, deletedLayout.UpdatedBy = 0, layout.Updated, layout.UpdatedBy
			layout = deletedLayout
		}

		if stored != nil {
			if err := validateOpenTablesKept(layout, *stored); err != nil {
				return err
			}
		}

		layouts := []types.DiningLayout{layout}
		if stored == nil {
			if err := db.InsertIn(transaction, &layouts); err != nil {
				return core.Err("Error al registrar el salón:", err)
			}
		} else {
			layoutTable := db.TableOf[types.DiningLayout]()
			if err := db.UpdateIn(transaction, &layouts,
				layoutTable.Name, layoutTable.WarehouseID, layoutTable.GridWidth, layoutTable.GridHeight,
				layoutTable.Shapes, layoutTable.Stations, layoutTable.QRRequiresConfirmation, layoutTable.QRVersion,
				layoutTable.WaitWarnMinutes, layoutTable.WaitLateMinutes, layoutTable.NextShapeID,
				layoutTable.Status, layoutTable.Updated, layoutTable.UpdatedBy,
			); err != nil {
				return core.Err("Error al actualizar el salón:", err)
			}
		}
		layout = layouts[0]
		return nil
	})
	if err != nil {
		return req.MakeErr(err)
	}
	return req.MakeResponse(layout)
}

// validateOpenTablesKept refuses a save that deletes the layout, or one of its tables, while a table
// has an open order (§1.2): the order would point to a table that no longer exists.
func validateOpenTablesKept(layout types.DiningLayout, stored types.DiningLayout) error {
	openOrders, err := readOpenTableOrders(layout.CompanyID, layout.ID)
	if err != nil || len(openOrders) == 0 {
		return err
	}
	if layout.Status == 0 {
		return core.Err(fmt.Sprintf("El salón tiene %v mesa(s) con pedidos abiertos: ciérrelos antes de eliminarlo.", len(openOrders)))
	}
	for _, order := range openOrders {
		isKept := slices.ContainsFunc(layout.Shapes, func(shape types.LayoutShape) bool {
			return shape.ShapeID == order.TableShapeID && shape.Kind == types.ShapeTable
		})
		if !isKept {
			storedIndex := slices.IndexFunc(stored.Shapes, func(shape types.LayoutShape) bool { return shape.ShapeID == order.TableShapeID })
			tableLabel := fmt.Sprint(order.TableShapeID)
			if storedIndex >= 0 {
				tableLabel = stored.Shapes[storedIndex].Label
			}
			return core.Err(fmt.Sprintf(`La mesa "%s" tiene un pedido abierto: no se puede eliminar del salón.`, tableLabel))
		}
	}
	return nil
}

// validateLayoutWarehouse checks the layout's warehouse is an active warehouse of the company.
func validateLayoutWarehouse(companyID int32, warehouseID int32) error {
	warehouses := []business.Warehouse{}
	query := db.Query(&warehouses)
	query.Select(query.ID, query.Status).CompanyID.Equals(companyID).ID.Equals(warehouseID)
	if err := query.Exec(); err != nil {
		return core.Err("Error al obtener el almacén del salón:", err)
	}
	if len(warehouses) == 0 || warehouses[0].Status == 0 {
		return core.Err("El almacén del salón no existe o está inactivo.")
	}
	return nil
}
