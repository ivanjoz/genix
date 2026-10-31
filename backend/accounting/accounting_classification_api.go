// Classifying the operations of Consolidado & Cuentas: writing the nature account the user picks
// back onto the record it belongs to.
//
// Only the nature lines are classifiable (frontend/routes/accounting/accounts-consolidation/
// PLAN.md §1): an expense's account, a supply line of a purchase order, an asset's 33x, and the
// counter-account of a cash movement the user classifies. Everything else is derived and locked,
// so this handler refuses any other kind.

package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	logistics "app/logistics/types"
	"encoding/json"
	"fmt"
)

// The classifiable operation kinds. Same ids as OPERATION_* in accounts-consolidation.ts.
const (
	ClassificationKindPurchaseOrder int8 = 3
	ClassificationKindExpense       int8 = 4
	ClassificationKindAsset         int8 = 6
	ClassificationKindCash          int8 = 9
)

// AccountClassification sets one nature line's account.
type AccountClassification struct {
	Kind int8  `cb:"1"`
	ID   int64 `cb:"2"`
	// Line is the purchase order line's index; 0 for every other kind.
	Line        int16 `cb:"3"`
	AccountCode int32 `cb:"4"`
	// CashMovementType retypes an unspecified cash movement (type 1) into the type of the option
	// picked — a withdrawal, a loss or a collection. 0 keeps the movement's type.
	CashMovementType finance.CashMovementType `cb:"5"`
}

// PutAccountingClassification validates every classification first and writes only when all of
// them hold, so a bulk apply never lands half done.
func PutAccountingClassification(req *core.HandlerArgs) core.HandlerResponse {
	classifications := []AccountClassification{}
	if deserializeError := json.Unmarshal([]byte(*req.Body), &classifications); deserializeError != nil {
		return req.MakeErr("Error al deserializar el body.", deserializeError)
	}
	if len(classifications) == 0 {
		return req.MakeErr("No se enviaron clasificaciones.")
	}

	idsByKind := map[int8]core.SliceSet[int64]{}
	for _, classification := range classifications {
		if classification.ID <= 0 {
			return req.MakeErr("Una clasificación no indica su registro.")
		}
		ids := idsByKind[classification.Kind]
		ids.Add(classification.ID)
		idsByKind[classification.Kind] = ids
	}
	companyID := req.User.CompanyID
	now := core.SUnixTime()

	orders, expenses, assets, cashMovements := []logistics.PurchaseOrder{}, []finance.Expense{}, []types.Asset{}, []finance.CashBankMovement{}
	for kind, ids := range idsByKind {
		var loadError error
		switch kind {
		case ClassificationKindPurchaseOrder:
			query := db.Query(&orders)
			loadError = query.Select().CompanyID.Equals(companyID).ID.In(toInt32IDs(ids.Values)...).Exec()
		case ClassificationKindExpense:
			query := db.Query(&expenses)
			loadError = query.Select().CompanyID.Equals(companyID).ID.In(toInt32IDs(ids.Values)...).Exec()
		case ClassificationKindAsset:
			query := db.Query(&assets)
			loadError = query.Select().CompanyID.Equals(companyID).ID.In(toInt32IDs(ids.Values)...).Exec()
		case ClassificationKindCash:
			query := db.Query(&cashMovements)
			loadError = query.Select().CompanyID.Equals(companyID).ID.In(ids.Values...).Exec()
		default:
			return req.MakeErr(fmt.Sprintf("Las operaciones de tipo %v no se clasifican: sus cuentas son derivadas.", kind))
		}
		if loadError != nil {
			return req.MakeErr("Error al obtener los registros a clasificar:", loadError)
		}
	}

	orderByID := indexByID(orders, func(order *logistics.PurchaseOrder) int64 { return int64(order.ID) })
	expenseByID := indexByID(expenses, func(expense *finance.Expense) int64 { return int64(expense.ID) })
	assetByID := indexByID(assets, func(asset *types.Asset) int64 { return int64(asset.ID) })
	cashMovementByID := indexByID(cashMovements, func(movement *finance.CashBankMovement) int64 { return movement.ID })

	for _, classification := range classifications {
		if validationError := applyClassification(classification, orderByID, expenseByID, assetByID, cashMovementByID); validationError != nil {
			return req.MakeErr(validationError)
		}
	}

	// Every classification held: write each table once.
	if len(orders) > 0 {
		for index := range orders {
			orders[index].Updated, orders[index].UpdatedBy = now, req.User.ID
		}
		q := db.TableOf[logistics.PurchaseOrder]()
		if writeError := db.Update(&orders, q.Lines, q.Updated, q.UpdatedBy); writeError != nil {
			return req.MakeErr("Error al guardar las órdenes de compra:", writeError)
		}
		// A reception posts to the stock pair of its line's account, stamped on the movement.
		for index := range orders {
			order := &orders[index]
			if restampError := logistics.RestampReceptionAccounts(companyID, int64(order.ID),
				logistics.StockMovementTypePurchaseReception, order.ReceptionAccountOf); restampError != nil {
				return req.MakeErr(restampError)
			}
		}
	}
	if len(expenses) > 0 {
		for index := range expenses {
			expenses[index].Updated, expenses[index].UpdatedBy = now, req.User.ID
		}
		q := db.TableOf[finance.Expense]()
		if writeError := db.Update(&expenses, q.AccountCode, q.Updated, q.UpdatedBy); writeError != nil {
			return req.MakeErr("Error al guardar los gastos:", writeError)
		}
		// An inventory expense's reception posts to the stock pair of its account.
		for index := range expenses {
			expense := &expenses[index]
			if expense.Type != finance.ExpenseTypeInventory {
				continue
			}
			if restampError := logistics.RestampReceptionAccounts(companyID, int64(expense.ID),
				logistics.StockMovementTypeSuppliesPurchase,
				func(int32) int32 { return expense.AccountCode }); restampError != nil {
				return req.MakeErr(restampError)
			}
		}
	}
	if len(assets) > 0 {
		for index := range assets {
			assets[index].Updated, assets[index].UpdatedBy = now, req.User.ID
		}
		q := db.TableOf[types.Asset]()
		if writeError := db.Update(&assets, q.AccountCode, q.Updated, q.UpdatedBy); writeError != nil {
			return req.MakeErr("Error al guardar los activos:", writeError)
		}
	}
	if len(cashMovements) > 0 {
		// The period delta index packs Period, SameDayMovement and Type: they are written together.
		// A classified movement is never a document's payment, so SameDayMovement keeps its 0.
		for index := range cashMovements {
			cashMovements[index].Period = core.PeriodOfDay(cashMovements[index].Date)
		}
		q := db.TableOf[finance.CashBankMovement]()
		if writeError := db.Update(&cashMovements, q.Type, q.CounterAccountCode, q.Period, q.SameDayMovement); writeError != nil {
			return req.MakeErr("Error al guardar los movimientos de caja:", writeError)
		}
	}

	core.Log("PutAccountingClassification:", len(classifications), "orders:", len(orders),
		"expenses:", len(expenses), "assets:", len(assets), "cash:", len(cashMovements))
	return req.MakeResponse(map[string]int{"Classified": len(classifications)})
}

// applyClassification validates one classification against its record and sets the account on it.
func applyClassification(
	classification AccountClassification,
	orderByID map[int64]*logistics.PurchaseOrder, expenseByID map[int64]*finance.Expense,
	assetByID map[int64]*types.Asset, cashMovementByID map[int64]*finance.CashBankMovement,
) error {
	switch classification.Kind {
	case ClassificationKindPurchaseOrder:
		order := orderByID[classification.ID]
		if order == nil {
			return core.Err("No se encontró la orden de compra:", classification.ID)
		}
		if classification.Line < 0 || int(classification.Line) >= len(order.Lines) {
			return core.Err(fmt.Sprintf("La orden de compra %v no tiene la línea %v.", order.ID, classification.Line))
		}
		line := &order.Lines[classification.Line]
		// A product line always posts to 6011; only a supply line (stamped with its account) is classified.
		if line.AccountCode == 0 {
			return core.Err(fmt.Sprintf("La línea %v de la orden %v es de un producto: su cuenta es 6011.", classification.Line, order.ID))
		}
		if classification.AccountCode == 0 {
			return core.Err("Seleccione la cuenta de la línea de la orden de compra.")
		}
		if validationError := types.ValidatePCGEAccount(types.PCGEKindSupply, classification.AccountCode); validationError != nil {
			return validationError
		}
		line.AccountCode = classification.AccountCode

	case ClassificationKindExpense:
		expense := expenseByID[classification.ID]
		if expense == nil || expense.Status == 0 {
			return core.Err("No se encontró el gasto:", classification.ID)
		}
		if expense.Type == finance.ExpenseTypeDepreciation {
			return core.Err("La depreciación toma sus cuentas del activo; clasifique el activo.")
		}
		accountKind := types.PCGEKindExpense
		if expense.Type == finance.ExpenseTypeInventory {
			accountKind = types.PCGEKindSupply
		}
		if classification.AccountCode == 0 {
			return core.Err("Seleccione la cuenta del gasto.")
		}
		if validationError := types.ValidatePCGEAccount(accountKind, classification.AccountCode); validationError != nil {
			return validationError
		}
		expense.AccountCode = classification.AccountCode

	case ClassificationKindAsset:
		asset := assetByID[classification.ID]
		if asset == nil || asset.Status == types.AssetStatusRemoved {
			return core.Err("No se encontró el activo:", classification.ID)
		}
		if classification.AccountCode == 0 {
			return core.Err("Seleccione la cuenta del activo.")
		}
		if validationError := types.ValidatePCGEAccount(types.PCGEKindAsset, classification.AccountCode); validationError != nil {
			return validationError
		}
		asset.AccountCode = classification.AccountCode

	case ClassificationKindCash:
		movement := cashMovementByID[classification.ID]
		if movement == nil {
			return core.Err("No se encontró el movimiento de caja:", classification.ID)
		}
		targetType := movement.Type
		if classification.CashMovementType != 0 && classification.CashMovementType != movement.Type {
			// Only an unspecified movement changes type, and only into one the user could have
			// registered by hand with that sign.
			if movement.Type != finance.CashMovementTypeUnspecified ||
				classification.CashMovementType == finance.CashMovementTypeTransfer {
				return core.Err("Sólo un movimiento sin especificar puede cambiar de tipo.")
			}
			if validationError := finance.ValidateManualCashMovement(classification.CashMovementType, movement.Amount); validationError != nil {
				return validationError
			}
			targetType = classification.CashMovementType
		}
		if len(finance.CashMovementAccountOptionsFor(targetType, movement.Amount)) == 0 {
			return core.Err("Este movimiento de caja toma su cuenta de su documento; no se clasifica.")
		}
		accountCode, resolveError := finance.ResolveCashMovementAccount(targetType, movement.Amount, classification.AccountCode)
		if resolveError != nil {
			return resolveError
		}
		movement.Type, movement.CounterAccountCode = targetType, accountCode

	default:
		return core.Err(fmt.Sprintf("Las operaciones de tipo %v no se clasifican: sus cuentas son derivadas.", classification.Kind))
	}
	return nil
}

func toInt32IDs(ids []int64) []int32 {
	converted := make([]int32, len(ids))
	for index, id := range ids {
		converted[index] = int32(id)
	}
	return converted
}

func indexByID[T any](records []T, idOf func(*T) int64) map[int64]*T {
	byID := make(map[int64]*T, len(records))
	for index := range records {
		byID[idOf(&records[index])] = &records[index]
	}
	return byID
}
