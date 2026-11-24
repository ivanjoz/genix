package finance

import (
	accounting "app/accounting/types"
	"app/core"
	"app/db"
	"app/finance/types"
	logistics "app/logistics/types"
	sales "app/sales/types"
	"fmt"
	"slices"
)

// GET.cash-documents-day?kind=&dates=&site= : the open documents behind one cell of the cash flow
// (plan/CASH_DOCUMENTS_FRAME_PLAN.md §7). The frame holds amounts, not documents: the browser sends
// the open Kind and the document dates of the frame rows that fed the cell, and dates each document
// it gets back with the same rule it dated those rows. site is left out to take every site.

// cashDocumentsDayMaxDates bounds the document dates of one drill-down: the frame window the browser reads.
const cashDocumentsDayMaxDates = 400

func GetCashDocumentsDay(req *core.HandlerArgs) core.HandlerResponse {
	selection := types.CashDocumentSelection{Kind: int8(req.GetQueryInt("kind")), Dates: map[int16]bool{}}
	for _, date := range req.GetQueryIntSlice("dates") {
		selection.Dates[int16(date)] = true
	}
	if len(selection.Dates) == 0 || len(selection.Dates) > cashDocumentsDayMaxDates {
		return req.MakeErr("Se necesitan entre 1 y", cashDocumentsDayMaxDates, "fechas de documento.")
	}
	_, hasSite := req.Query["site"]
	selection.AllSites, selection.SiteID = !hasSite, req.GetQueryInt16("site")

	var details []types.CashDocumentDetail
	var err error
	switch selection.Kind {
	case types.CashDocumentKindSaleOpen:
		details, err = openSaleDocuments(req.User.CompanyID, selection)
	case types.CashDocumentKindExpenseOpen, types.CashDocumentKindScheduledExpenseOpen:
		details, err = openExpenseDocuments(req.User.CompanyID, selection)
	case types.CashDocumentKindPurchaseOrderOpen:
		details, err = openPurchaseOrderDocuments(req.User.CompanyID, selection)
	case types.CashDocumentKindAssetOpen:
		details, err = openAssetDocuments(req.User.CompanyID, selection)
	default:
		return req.MakeErr("El tipo de documento no es un saldo abierto:", selection.Kind)
	}
	if err != nil {
		return req.MakeErr(err)
	}
	slices.SortFunc(details, func(left, right types.CashDocumentDetail) int { return int(left.Date) - int(right.Date) })
	return req.MakeResponse(details)
}

// Each reader takes the open records through the status the old per-document scans used (few: what
// is still owed) and keeps those the selection names.

func openSaleDocuments(companyID int32, selection types.CashDocumentSelection) ([]types.CashDocumentDetail, error) {
	saleOrders := []sales.SaleOrder{}
	query := db.Query(&saleOrders)
	query.Select(query.ID, query.Date, query.SiteID, query.Status, query.TotalAmount, query.DebtAmount, query.PaymentDueDate,
		query.CurrencyType).CompanyID.Equals(companyID).Status.In(sales.OrderStatusPending, sales.OrderStatusDelivered)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las ventas por cobrar:", err)
	}
	details := []types.CashDocumentDetail{}
	for _, saleOrder := range saleOrders {
		if amount := selection.OpenAmountOf(saleOrder.ToCashDocumentFrames(nil)); amount != 0 {
			details = append(details, types.CashDocumentDetail{ID: saleOrder.ID, Label: fmt.Sprintf("Venta %d", saleOrder.ID),
				Date: saleOrder.Date, DueDate: saleOrder.PaymentDueDate, Amount: amount, CurrencyType: saleOrder.CurrencyType})
		}
	}
	return details, nil
}

func openExpenseDocuments(companyID int32, selection types.CashDocumentSelection) ([]types.CashDocumentDetail, error) {
	expenses := []types.Expense{}
	query := db.Query(&expenses)
	query.Select(query.ID, query.Name, query.Date, query.DueDate, query.SiteID, query.Type, query.Status, query.AccountCode,
		query.ExpenseScheduledID, query.Amount, query.PaidAmount, query.CurrencyType).
		CompanyID.Equals(companyID).Status.Equals(1) // 1 pending: a paid expense owes nothing
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los gastos pendientes:", err)
	}
	details := []types.CashDocumentDetail{}
	for _, expense := range expenses {
		if amount := selection.OpenAmountOf(expense.ToCashDocumentFrames(nil)); amount != 0 {
			details = append(details, types.CashDocumentDetail{ID: int64(expense.ID), Label: expense.Name, Date: expense.Date,
				DueDate: expense.DueDate, Amount: amount, CurrencyType: expense.CurrencyType})
		}
	}
	return details, nil
}

func openPurchaseOrderDocuments(companyID int32, selection types.CashDocumentSelection) ([]types.CashDocumentDetail, error) {
	orders := []logistics.PurchaseOrder{}
	query := db.Query(&orders)
	query.Select(query.ID, query.Date, query.SiteID, query.Status, query.TotalAmount, query.DebtAmount, query.PaymentDate,
		query.CurrencyType).CompanyID.Equals(companyID).
		Status.In(logistics.PurchaseOrderStatusConfirmed, logistics.PurchaseOrderStatusFulfilled)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener las órdenes de compra por pagar:", err)
	}
	details := []types.CashDocumentDetail{}
	for _, order := range orders {
		if amount := selection.OpenAmountOf(order.ToCashDocumentFrames(nil)); amount != 0 {
			details = append(details, types.CashDocumentDetail{ID: int64(order.ID), Label: fmt.Sprintf("OC %d", order.ID),
				Date: order.Date, DueDate: order.PaymentDate, Amount: amount, CurrencyType: order.CurrencyType})
		}
	}
	return details, nil
}

func openAssetDocuments(companyID int32, selection types.CashDocumentSelection) ([]types.CashDocumentDetail, error) {
	assets := []accounting.Asset{}
	query := db.Query(&assets)
	query.Select(query.ID, query.Name, query.AcquisitionDate, query.SiteID, query.Status, query.PaymentStatus, query.DueDate,
		query.PurchaseAmount, query.PaidAmount, query.CurrencyType).CompanyID.Equals(companyID).
		Status.In(accounting.AssetStatusActive, accounting.AssetStatusFullyDepreciated)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los activos por pagar:", err)
	}
	details := []types.CashDocumentDetail{}
	for _, asset := range assets {
		if amount := selection.OpenAmountOf(asset.ToCashDocumentFrames(nil)); amount != 0 {
			details = append(details, types.CashDocumentDetail{ID: int64(asset.ID), Label: asset.Name, Date: asset.AcquisitionDate,
				DueDate: asset.DueDate, Amount: amount, CurrencyType: asset.CurrencyType})
		}
	}
	return details, nil
}
