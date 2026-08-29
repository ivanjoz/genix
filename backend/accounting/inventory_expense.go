package accounting

import (
	"app/core"
	"app/db"
	finance "app/finance/types"
	logistics "app/logistics/types"
	production "app/production/types"
	"encoding/json"
)

// InventoryExpensePayload buys a supply and puts it in a warehouse in one step: the money
// side becomes a Type-2 Expense, the goods side becomes an ordinary inbound stock movement.
//
// This lives in accounting rather than finance because it needs both sides, and logistics
// already imports finance for cash movements — the reverse edge would be a cycle.
type InventoryExpensePayload struct {
	Name         string `json:",omitempty"`
	Description  string `json:",omitempty"`
	CategoryID   int8   `json:",omitempty"`
	SupplierID   int32  `json:",omitempty"`
	CurrencyType int8   `json:",omitempty"`
	Date         int16  `json:",omitempty"`
	DueDate      int16  `json:",omitempty"`
	ProductID    int32  `json:",omitempty"`
	WarehouseID  int32  `json:",omitempty"`
	Quantity     int32  `json:",omitempty"`
	// Unit price in cents. The expense total is this times the quantity.
	UnitPrice int32 `json:",omitempty"`
}

// PostInventoryExpense registers a purchase that lands in stock. Unlike a Simple expense it
// does not go straight to the P&L: the money became inventory, and only becomes a cost when
// the stock is consumed or sold. Type 2 is what records that distinction.
func PostInventoryExpense(req *core.HandlerArgs) core.HandlerResponse {
	payload := InventoryExpensePayload{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &payload); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}

	if len(payload.Name) < 2 {
		return req.MakeErr("El nombre del gasto debe tener al menos 2 caracteres.")
	}
	if payload.ProductID <= 0 {
		return req.MakeErr("Debe seleccionar el insumo o material comprado.")
	}
	if payload.WarehouseID <= 0 {
		return req.MakeErr("Debe seleccionar el almacén de destino.")
	}
	if payload.Quantity <= 0 {
		return req.MakeErr("La cantidad debe ser mayor a 0.")
	}
	if payload.UnitPrice <= 0 {
		return req.MakeErr("El precio unitario debe ser mayor a 0.")
	}
	if payload.CurrencyType != 1 && payload.CurrencyType != 2 {
		return req.MakeErr("Moneda inválida (debe ser 1=PEN o 2=USD).")
	}

	// The catalog row has to be a supply. Buying a sellable product is a purchase order,
	// which has its own reception flow with lots and differences.
	supplyProducts := []production.Product{}
	supplyQuery := db.Query(&supplyProducts)
	supplyQuery.Select(supplyQuery.ID, supplyQuery.Name, supplyQuery.Status).
		CompanyID.Equals(req.User.CompanyID).ID.Equals(payload.ProductID)

	if queryError := supplyQuery.Exec(); queryError != nil {
		return req.MakeErr("Error al obtener el insumo.", queryError)
	}
	if len(supplyProducts) == 0 {
		return req.MakeErr("No se encontró el insumo indicado.")
	}
	if supplyProducts[0].Status != production.ProductStatusSupply {
		return req.MakeErr("El registro seleccionado no es un insumo o material.")
	}

	currentTimestamp := core.SUnixTime()
	expenseDate := core.If(payload.Date > 0, payload.Date, core.FechaUnix())

	inventoryExpense := finance.Expense{
		CompanyID:    req.User.CompanyID,
		Name:         payload.Name,
		Description:  payload.Description,
		Type:         finance.ExpenseTypeInventory,
		CategoryID:   core.If(payload.CategoryID > 0, payload.CategoryID, int8(4)),
		SupplierID:   payload.SupplierID,
		ProductID:    payload.ProductID,
		WarehouseID:  payload.WarehouseID,
		Quantity:     payload.Quantity,
		CurrencyType: payload.CurrencyType,
		Date:         expenseDate,
		DueDate:      core.If(payload.DueDate > 0, payload.DueDate, expenseDate),
		Amount:       payload.UnitPrice * payload.Quantity,
		Status:       1,
		Updated:      currentTimestamp,
		UpdatedBy:    req.User.ID,
		Created:      currentTimestamp,
		CreatedBy:    req.User.ID,
	}

	expenseRecords := []finance.Expense{inventoryExpense}
	if insertError := db.Insert(&expenseRecords); insertError != nil {
		return req.MakeErr("Error al registrar el gasto de inventario.", insertError)
	}

	// The stock movement carries the expense as its DocumentID, so the ledger points back
	// at what paid for it.
	inboundMovement := logistics.InternalMovement{
		ProductID:   payload.ProductID,
		WarehouseID: payload.WarehouseID,
		SupplierID:  payload.SupplierID,
		Quantity:    payload.Quantity,
		Price:       payload.UnitPrice,
		DocumentID:  int64(expenseRecords[0].ID),
	}
	if movementError := logistics.ApplyMovimientos(
		req, []logistics.InternalMovement{inboundMovement},
	); movementError != nil {
		return req.MakeErr("Error al ingresar el insumo al almacén.", movementError)
	}

	core.Log("PostInventoryExpense expense:", expenseRecords[0].ID, "product:", payload.ProductID)
	return req.MakeResponse(expenseRecords[0])
}
