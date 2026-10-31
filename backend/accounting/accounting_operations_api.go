// Reading every accounting source of a period, for the Consolidado & Cuentas page.
//
// Like the books, the handler only reads and sends numbers. The entries — which account each
// line posts to, whether the operation enters the official Diario, its status — are built in
// the browser by frontend/routes/accounting/accounts-consolidation/accounts-consolidation.ts,
// so the page, the Diario de gestión and the official 5.2 read the same entries. See that
// folder's PLAN.md §3 for the rules.
//
// Each response key is one table's period delta log (the TypeDelta index pinned by Period), so a
// reopened period only travels what changed since (DELTA_CACHE_PLAN.md in that folder). What the
// period names outside itself — an older sale a credit note corrects, an older purchase a
// reception names, the products — the browser reads through the by-IDs cache. A payment that
// settled its document on the document's own date (SameDayMovement = 1) is not read at all: the
// browser infers it from the document's SameDayCashBankID.

package accounting

import (
	"app/accounting/types"
	"app/core"
	"app/db"
	finance "app/finance/types"
	invoicing "app/invoicing/types"
	logistics "app/logistics/types"
	sales "app/sales/types"

	"golang.org/x/sync/errgroup"
)

// OperationSaleLine is a sale line's charged amount, in the sale's currency. The browser splits
// the revenue between 7012 and 7022 by the product's Origin in these proportions.
type OperationSaleLine struct {
	ProductID int32 `cb:"1"`
	Amount    int32 `cb:"2"`
}

type OperationSale struct {
	ID                int64               `cb:"1"`
	Date              int16               `cb:"2"`
	Status            int8                `json:"ss" cb:"ss,3"`
	CurrencyType      int8                `cb:"4"`
	ExchangeRate      int32               `cb:"5"`
	TotalAmount       int32               `cb:"6"`
	TaxAmount         int32               `cb:"7"`
	Lines             []OperationSaleLine `cb:"8"`
	SameDayCashBankID int32               `cb:"9"`
	Version           int16               `json:"ver" cb:"ver,10"`
	Metadata          *string             `json:"mdt" cb:"mdt,11"`
}

// OperationStockMovement is a stock movement the entries post, valued in PEN net of IGV: a
// reception of a purchase order (type 3) or of an inventory expense (4) at its cost, and a sale
// delivery (8) or its annulment return (9) at the cost stamped on it (docs/SALE_COST_PLAN.md).
// A reception carries the purchase account its stock and variation pair follow from.
type OperationStockMovement struct {
	ID                  int64                       `cb:"1"`
	Date                int16                       `cb:"2"`
	Type                logistics.StockMovementType `cb:"3"`
	DocumentID          int64                       `cb:"4"`
	ProductID           int32                       `cb:"5"`
	MonetaryValue       int32                       `cb:"6"`
	PurchaseAccountCode int32                       `cb:"7"`
	Version             int16                       `json:"ver" cb:"ver,8"`
	Metadata            *string                     `json:"mdt" cb:"mdt,9"`
}

// OperationCashMovement is a cash movement no document settles same-day. A document payment
// carries the counter-account it clears, stamped when it was written.
type OperationCashMovement struct {
	ID                 int64                    `cb:"1"`
	CashBankID         int32                    `cb:"2"`
	DocumentID         int64                    `cb:"3"`
	Date               int16                    `cb:"4"`
	Type               finance.CashMovementType `cb:"5"`
	CounterAccountCode int32                    `cb:"6"`
	Amount             int32                    `cb:"7"` // signed, in the cash bank's currency
	CashBankRefID      int32                    `cb:"8"` // a transfer's other cash bank
	Version            int16                    `json:"ver" cb:"ver,9"`
	Metadata           *string                  `json:"mdt" cb:"mdt,10"`
}

// AccountingOperationsResponse is one key per period delta log. Each key travels its own cursor
// (?<Key>=…), so the browser merges each key on its own.
type AccountingOperationsResponse struct {
	SalesDocuments []SalesBookDocument `cb:"1"`
	Sales          []OperationSale     `cb:"2"`
	// Orders by DocIssueDate; expenses by their accounting date, depreciation included.
	PurchaseOrders []PurchaseBookDocument   `cb:"3"`
	Expenses       []PurchaseBookDocument   `cb:"4"`
	StockMovements []OperationStockMovement `cb:"5"`
	CashMovements  []OperationCashMovement  `cb:"6"`
}

// What a first sync lists of each log. An annulled sale, a canceled order and a removed expense
// are logged as deletes, so they never reach the page; a voided comprobante does (its State is
// not its Status) and the page skips it.
var (
	operationSaleStatuses = []any{sales.OrderStatusPending, sales.OrderStatusPaid,
		sales.OrderStatusDelivered, sales.OrderStatusCompleted}
	operationOrderStatuses = []any{logistics.PurchaseOrderStatusPending,
		logistics.PurchaseOrderStatusConfirmed, logistics.PurchaseOrderStatusFulfilled}
	operationExpenseStatuses = []any{int8(1), int8(2), finance.ExpenseStatusPosted}
	operationStockTypes      = []any{logistics.StockMovementTypePurchaseReception,
		logistics.StockMovementTypeSuppliesPurchase, logistics.StockMovementTypeSaleDelivery,
		logistics.StockMovementTypeSaleAnnulmentReturn}
	operationCashTypes = []any{finance.CashMovementTypeUnspecified, finance.CashMovementTypePhysicalCount,
		finance.CashMovementTypeTransfer, finance.CashMovementTypeWithdrawal, finance.CashMovementTypeLoss,
		finance.CashMovementTypeSupplierPayment, finance.CashMovementTypeCollection,
		finance.CashMovementTypeSaleCollection, finance.CashMovementTypeExpensePayment,
		finance.CashMovementTypeAssetPayment, finance.CashMovementTypeSaleRefund}
)

// GetAccountingOperations reads a whole period. Unlike the books, never a single day: the
// period is the unit the page classifies and the statements close.
func GetAccountingOperations(req *core.HandlerArgs) core.HandlerResponse {
	period := req.Query["period"]
	firstDay, _, boundsError := types.PeriodBounds(period)
	if boundsError != nil {
		return req.MakeErr(boundsError.Error())
	}
	companyID := req.User.CompanyID
	periodKey := core.PeriodOfDay(firstDay)
	response := AccountingOperationsResponse{}

	readGroup := errgroup.Group{}
	readGroup.Go(func() error {
		documents := []invoicing.InvoiceDocument{}
		query := db.Query(&documents)
		query.Select().CompanyID.Equals(companyID).Period.Equals(periodKey).
			Delta(req.GetDeltaCursor("SalesDocuments"), core.ToAny(invoicing.AllInvoiceStates)...)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener los comprobantes del periodo.", queryError)
		}
		response.SalesDocuments = projectRecords(documents, func(document invoicing.InvoiceDocument) SalesBookDocument {
			return makeBookDocument(&document)
		})
		return nil
	})
	readGroup.Go(func() error {
		orders := []sales.SaleOrder{}
		query := db.Query(&orders)
		query.Select().CompanyID.Equals(companyID).Period.Equals(periodKey).
			Delta(req.GetDeltaCursor("Sales"), operationSaleStatuses...)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener las ventas del periodo.", queryError)
		}
		response.Sales = projectRecords(orders, func(order sales.SaleOrder) OperationSale { return makeOperationSale(&order) })
		return nil
	})
	readGroup.Go(func() error {
		orders := []logistics.PurchaseOrder{}
		query := db.Query(&orders)
		query.Select().CompanyID.Equals(companyID).Period.Equals(periodKey).
			Delta(req.GetDeltaCursor("PurchaseOrders"), operationOrderStatuses...)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener las órdenes de compra del periodo.", queryError)
		}
		response.PurchaseOrders = projectRecords(orders, func(order logistics.PurchaseOrder) PurchaseBookDocument {
			return makeOrderBookDocument(&order)
		})
		return nil
	})
	readGroup.Go(func() error {
		expenses := []finance.Expense{}
		query := db.Query(&expenses)
		query.Select().CompanyID.Equals(companyID).Period.Equals(periodKey).
			Delta(req.GetDeltaCursor("Expenses"), operationExpenseStatuses...)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener los gastos del periodo.", queryError)
		}
		response.Expenses = projectRecords(expenses, func(expense finance.Expense) PurchaseBookDocument {
			return makeExpenseBookDocument(&expense)
		})
		return nil
	})
	readGroup.Go(func() error {
		movements := []logistics.WarehouseProductMovement{}
		query := db.Query(&movements)
		query.Select().CompanyID.Equals(companyID).Period.Equals(periodKey).
			Delta(req.GetDeltaCursor("StockMovements"), operationStockTypes...)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener los movimientos de almacén del periodo.", queryError)
		}
		response.StockMovements = projectRecords(movements, func(movement logistics.WarehouseProductMovement) OperationStockMovement {
			return OperationStockMovement{
				ID: movement.ID, Date: movement.Date, Type: movement.Type, DocumentID: movement.DocumentID,
				ProductID: movement.ProductID, MonetaryValue: movement.MonetaryValue,
				PurchaseAccountCode: movement.PurchaseAccountCode, Version: movement.Version, Metadata: movement.Metadata,
			}
		})
		return nil
	})
	readGroup.Go(func() error {
		movements := []finance.CashBankMovement{}
		query := db.Query(&movements)
		// Pinning SameDayMovement to 0 leaves out every payment the browser infers.
		query.Select().CompanyID.Equals(companyID).Period.Equals(periodKey).SameDayMovement.Equals(0).
			Delta(req.GetDeltaCursor("CashMovements"), operationCashTypes...)
		if queryError := query.Exec(); queryError != nil {
			return core.Err("Error al obtener los movimientos de caja del periodo.", queryError)
		}
		response.CashMovements = projectRecords(movements, func(movement finance.CashBankMovement) OperationCashMovement {
			return OperationCashMovement{
				ID: movement.ID, CashBankID: movement.CashBankID, DocumentID: movement.DocumentID,
				Date: movement.Date, Type: movement.Type, CounterAccountCode: movement.CounterAccountCode,
				Amount: movement.Amount, CashBankRefID: movement.CashBankRefID,
				Version: movement.Version, Metadata: movement.Metadata,
			}
		})
		return nil
	})
	if readError := readGroup.Wait(); readError != nil {
		return req.MakeErr(readError.Error())
	}

	core.Log("GetAccountingOperations period:", period, "documents:", len(response.SalesDocuments),
		"sales:", len(response.Sales), "orders:", len(response.PurchaseOrders), "expenses:", len(response.Expenses),
		"stock:", len(response.StockMovements), "cash:", len(response.CashMovements))
	return req.MakeResponse(response)
}

// projectRecords maps the stored records to the projection a key sends.
func projectRecords[RecordT, ProjectionT any](records []RecordT, project func(RecordT) ProjectionT) []ProjectionT {
	projections := make([]ProjectionT, len(records))
	for index, record := range records {
		projections[index] = project(record)
	}
	return projections
}

func makeOperationSale(order *sales.SaleOrder) OperationSale {
	operationSale := OperationSale{
		ID: order.ID, Date: order.Date, Status: order.Status, CurrencyType: order.CurrencyType,
		ExchangeRate: order.ExchangeRate, TotalAmount: order.TotalAmount, TaxAmount: order.TaxAmount,
		SameDayCashBankID: order.SameDayCashBankID, Version: order.Version, Metadata: order.Metadata,
	}
	for _, saleLine := range order.Lines {
		if saleLine.ProductID <= 0 || saleLine.Quantity <= 0 {
			continue
		}
		lineAmount := core.QuantityAmount(core.UnpackQuantityLine(saleLine.Quantity), saleLine.Price, saleLine.SubPrice)
		operationSale.Lines = append(operationSale.Lines, OperationSaleLine{ProductID: saleLine.ProductID, Amount: lineAmount})
	}
	return operationSale
}
