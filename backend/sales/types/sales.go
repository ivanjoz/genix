package types

import (
	business "app/business/types"
	"app/core"
	crm "app/crm/types"
	"app/db"
	finance "app/finance/types"
	"fmt"
	"maps"
	"math"
	"slices"
)

// SaleOrderClientInfo is a walk-in client typed at the till. It is request-only — SaleOrderTable has
// no column for it — and is turned into a ClientProvider row, which (with its snapshot) is the only
// place these fields are stored.
type SaleOrderClientInfo struct {
	Name           string `cb:"1"`
	RegistryNumber string `cb:"2"`
	// IdentityDocType is the SUNAT document type the till picked (crm.IdentityDoc*). 0 leaves
	// SaveClientProviders to derive it from the number's shape.
	IdentityDocType int8 `cb:"3"`
	OnlyInsert      bool `cb:"4"`
}

// SaleOrderLine is one product line of a sale.
type SaleOrderLine struct {
	ProductID int32 `cb:"1"`
	// Price is the price of one whole unit; SubPrice the price of one sub-unit. Both are needed
	// per line and neither is derivable from the other: selling loose is deliberately not
	// proportional (four candies from a 5000-cent box of six is not 3333). A summary reprocess
	// has to reconstruct what was charged, not what the product costs now.
	Price    int32 `cb:"2"`
	SubPrice int32 `cb:"3"`
	// Quantity is packed: Units*core.QuantityLineScale + Sub, so 1001 at divisor 6 is one box
	// plus one candy. A sale order is a document — it is never accumulated and never SUM()-ed —
	// so it packs both halves into one value, where the stock and ledger tables keep them
	// separate. core.UnpackQuantityLine splits it.
	Quantity int32 `cb:"4"`
	// SubDivisor is the divisor the packed sub-unit part is expressed in, recorded per line so a
	// later refinement of the product never restates a past sale.
	SubDivisor     int16  `cb:"5"`
	Sku            string `cb:"6"`
	LotID          int32  `cb:"7"`
	PresentationID int16  `cb:"8"`
	// StockExitPending is set when the sale was delivered without this line's warehouse exit: a made
	// product leaves its stock at the daily production close, which settles the line and clears it
	// (docs/PRODUCTION_PLAN.md §3). Until then the line keeps its unit reserved in PendingDelivery*.
	StockExitPending bool `cb:"9"`
	// PackID is the pack (production.ProductPack) that priced this line, 0 when none: the line's
	// Price is its prorated share of the pack price, not the catalog price (ApplySalePacks).
	PackID int32 `cb:"10"`
	// TaxGroup is the line's IGV treatment (finance.TaxGroup*, 1–5), pinned from the product when the
	// line is priced (PriceSaleOrderLine) and never taken from the client: every send rebuilds the CPE
	// lines from it, so a later change to the product never restates a past sale.
	TaxGroup int8 `cb:"11"`
}

// SaleImposedTax is a tax treatment of the sale besides the IGV of its lines (plan/CASH_DOCUMENTS_FRAME_PLAN.md
// §9.1). Type is a finance.TaxGroup* number. Pinned when the sale is created, in the sale's currency.
//
// TaxGroupFree marks line Line as given away: the line is charged 0 (Price = SubPrice = 0) and Amount
// holds the gross value it would have been sold at, which the CPE declares and whose IGV the company
// still owes. TaxGroupDetraction and TaxGroupRetention are whole-sale entries (Line 0) the backend
// computes (ApplySaleWithholdings), never the client.
type SaleImposedTax struct {
	Type   int8  `cb:"1"`
	Amount int32 `cb:"2"`
	// Line is the line's 1-based position in Lines; 0 = the whole sale.
	Line int16 `cb:"3"`
	// Code is the tax's own code: the SPOT catalog 54 code of a detracción.
	Code int16 `cb:"4"`
	// Rate is the rate pinned when the sale was created, in hundredths of a percent (1200 = 12 %).
	Rate int16 `cb:"5"`
	// SettledAmount is what of a detracción the customer already deposited into the detracciones
	// account: the sale's open detracción is Amount − SettledAmount.
	SettledAmount int32 `cb:"6"`
}

// ApplyFreeLines checks the imposed taxes the client sent and prices the free lines, after the lines
// were priced from the catalog: each named line is charged 0 and its entry's Amount becomes the
// catalog value it gives away. Only free lines may come from the client; the amount it sent is
// ignored. A free line sells whole units of a product that is not an export: the CPE declares one
// free item per line.
func (e *SaleOrder) ApplyFreeLines() error {
	freeLines := map[int16]bool{}
	for index := range e.ImposedTaxes {
		imposedTax := &e.ImposedTaxes[index]
		if imposedTax.Type != finance.TaxGroupFree {
			return core.Err("La venta solo admite marcar líneas como gratuitas.")
		}
		if imposedTax.Line < 1 || int(imposedTax.Line) > len(e.Lines) || freeLines[imposedTax.Line] {
			return core.Err(fmt.Sprintf("La línea gratuita %v no es válida.", imposedTax.Line))
		}
		freeLines[imposedTax.Line] = true
		line := &e.Lines[imposedTax.Line-1]
		quantity := core.UnpackQuantityLine(line.Quantity)
		if quantity.Sub > 0 {
			return core.Err(fmt.Sprintf("La línea gratuita %v no puede llevar sub-unidades.", imposedTax.Line))
		}
		// SUNAT has a free code for a taxed (11), exonerado (21) and inafecto (37) line, none for an export.
		if line.TaxGroup == finance.TaxGroupExport {
			return core.Err(fmt.Sprintf("La línea gratuita %v no puede ser una exportación.", imposedTax.Line))
		}
		imposedTax.Amount = core.QuantityAmount(quantity, line.Price, line.SubPrice)
		if imposedTax.Amount <= 0 {
			return core.Err(fmt.Sprintf("La línea gratuita %v no tiene un valor referencial.", imposedTax.Line))
		}
		line.Price, line.SubPrice = 0, 0
	}
	return nil
}

// FreeLineAmountOf is the reference gross value of line lineIndex (0-based) when the sale gives it
// away, and whether it does.
func (e *SaleOrder) FreeLineAmountOf(lineIndex int) (int32, bool) {
	for _, imposedTax := range e.ImposedTaxes {
		if imposedTax.Type == finance.TaxGroupFree && int(imposedTax.Line) == lineIndex+1 {
			return imposedTax.Amount, true
		}
	}
	return 0, false
}

// DetailProductIDs is the product of every line, in line order. It feeds the computed column of
// the same name, which the product index groups hash.
func (e *SaleOrder) DetailProductIDs() []int32 {
	productIDs := make([]int32, 0, len(e.Lines))
	for _, line := range e.Lines {
		productIDs = append(productIDs, line.ProductID)
	}
	return productIDs
}

type SaleOrder struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID   int32 `cb:"1"`
	Date        int16 `cb:"2"`
	WarehouseID int32 `cb:"3"`
	ID          int64 `cb:"4"`

	// Lines is stored as one colbin blob. Its product IDs are indexed through the computed
	// DetailProductIDs column, so they are not stored a second time.
	Lines []SaleOrderLine `cb:"5"`

	// The line prices and the three amounts below are all in the sale's currency: a USD sale
	// stores dollars, converted per unit from the catalog on creation.
	TotalAmount int32 `cb:"13"`
	TaxAmount   int32 `cb:"14"`
	DebtAmount  int32 `cb:"15"`
	// CurrencyType is the sale's currency: 1 = PEN, 2 = USD (the cash-bank codes). A paid
	// sale takes its cash-bank's currency; an unpaid one takes what the till picked.
	CurrencyType int8 `cb:"16"`
	// ExchangeRate is the effective rate × 1000 the lines were converted with, spread
	// included — what a report needs to restate a USD sale in PEN. A USD sale always has one;
	// a PEN sale only when it sold a USD-priced product.
	ExchangeRate int32 `cb:"17"`
	ClientID     int32 `cb:"18"`
	Created      int32 `cb:"19"`
	Updated      int32 `json:"upd" cb:"upd,20"`
	Version      int16 `json:"ver" cb:"ver,22"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,23"`
	UpdatedBy int32   `cb:"24"`
	// 0 = Anulado, 1 = Generado, 2 = Pagado, 3 = Entregado, 4 = Pagado + Entregado
	Status            int8  `json:"ss" cb:"ss,25"`
	LastPaymentCajaID int32 `cb:"26"`
	// If contains 2 = the payment is done
	// If contains 3 = the delivery of the product is done
	ActionsIncluded []int8 `cb:"27"`
	// Audit trail fields for payment and delivery actions.
	LastPaymentTime int32                `cb:"28"`
	LastPaymentUser int32                `cb:"29"`
	DeliveryTime    int32                `cb:"30"`
	DeliveryUser    int32                `cb:"31"`
	ClientInfo      *SaleOrderClientInfo `cb:"32"`
	// IssueSeriesID names the invoicing series this sale will be issued under. It
	// is a request field, not a column: it is folded into the last two digits of
	// the id on creation, and read back with SeriesID(). Zero when the till does
	// not say.
	IssueSeriesID  int8  `cb:"33"`
	PaymentDueDate int16 `cb:"34"`
	// AnnulReason is why the sale was annulled, required by the annul endpoint. There is no
	// AnnulledTime/AnnulledUser beside it: annulment is terminal, so Updated and UpdatedBy are
	// already the "when and by whom" and a second pair could only drift from them.
	AnnulReason string `cb:"35"`
	// Period is the YYMM of Date (core.PeriodOfDay): the pinned key of the period delta index the
	// accounting operations sync a month through. Set by StampPeriod on every write.
	Period int16 `cb:"36"`
	// SameDayCashBankID is the cash bank of the one collection that settled the whole sale on its
	// Date; 0 when no collection did. The accounting operations infer that collection from the
	// sale instead of downloading it — its movement carries SameDayMovement = 1. See
	// finance.SameDaySettlement.
	SameDayCashBankID int32 `cb:"37"`
	// ShippingAmount is the home-delivery charge (cents, in the sale's currency), already included
	// in TotalAmount: the boleta bills it as its own line. 0 on pickup and on till sales.
	ShippingAmount int32 `cb:"41"`
	// OnlineSaleOrderID is the web store order this sale was created from; 0 on till sales. Every
	// write to the sale copies its Status and DebtAmount to that order (syncOnlineSaleOrderIn).
	OnlineSaleOrderID int32 `cb:"43"`
	// PendingStockMovement is 1 while the sale is delivered (3/4) but some line's stock exit waits for
	// the daily production close (SaleOrderLine.StockExitPending); 0 otherwise. An int8 so it can pin
	// a delta index: the "Por regularizar" list and the close read the sales with 1.
	PendingStockMovement int8 `cb:"44"`
	// SiteID is the site (sede) of WarehouseID, 0 for a sale with no warehouse. Set by StampSiteID on
	// every write that sets the warehouse: what sale-summary sums by, since its ToFrames reads only the
	// sale. A delivery from another warehouse moves the sale to that warehouse's site.
	SiteID int16 `cb:"45"`
	// ImposedTaxes are the tax treatments besides the lines' IGV (SaleImposedTax): free lines for now.
	ImposedTaxes []SaleImposedTax `cb:"46"`
}

// RefreshPendingStockMovement sets PendingStockMovement from the lines' StockExitPending flags.
func (e *SaleOrder) RefreshPendingStockMovement() {
	e.PendingStockMovement = 0
	for _, line := range e.Lines {
		if line.StockExitPending {
			e.PendingStockMovement = 1
			return
		}
	}
}

// StampPeriod sets Period from the sale's accounting date, its Date.
func (e *SaleOrder) StampPeriod() {
	e.Period = core.PeriodOfDay(e.Date)
}

// StampSiteID sets SiteID from the sale's warehouse: 0 without one, else its site, refusing a
// warehouse that doesn't exist.
func (e *SaleOrder) StampSiteID() error {
	if e.WarehouseID == 0 {
		e.SiteID = 0
		return nil
	}
	siteIDByWarehouseID, err := business.SiteIDsOfWarehouses(e.CompanyID, []int32{e.WarehouseID})
	if err != nil {
		return err
	}
	e.SiteID = siteIDByWarehouseID[e.WarehouseID]
	return nil
}

// StampPaymentDueDate gives a sale still owing money and saved without a due date the one its
// client's credit days set (the sale's date for a cash client or no client). A due date the user picked
// is kept. The cash flow forecast collects a receivable on this date.
func (e *SaleOrder) StampPaymentDueDate() error {
	if e.DebtAmount <= 0 || e.PaymentDueDate != 0 {
		return nil
	}
	dueDate, err := crm.DueDateByCreditDays(e.CompanyID, e.ClientID, e.Date)
	if err != nil {
		return err
	}
	e.PaymentDueDate = dueDate
	return nil
}

// SaleSummaryFrame is one sale line as the "sale-summary" DataFrame sums it: one file per day and site,
// one row per product. Amounts are in soles. Quantities are whole units with sub-units as their fraction
// (units + sub / divisor), so the sub-units of any divisor add up in one row. The frame keeps a float
// as round(v × 1e6) (dataframe.FloatScale): each line rounds once to 6 decimals (1/3 is 0.333333), and
// the sums of those are exact integers.
type SaleSummaryFrame struct {
	// Date is the frame's Part, spread by week (UseWeekSpread): partition_id = CompanyID << 13 | the
	// epoch week, one aggregate row per week, site and product with a column per day. A client gets a
	// file per day and site, named by the Date then the SiteID.
	Date int16 `frame:"part,13"`
	// SiteID is the sale's (SaleOrder.SiteID): the sales forecast works per site and product, against
	// the per-site stock of daily-product-stock. 0 is the sales with no warehouse.
	SiteID          int16   `frame:"key"`
	ProductID       int32   `frame:"row"`
	AmountPEN       int32   `frame:"sum"`
	DebtAmountPEN   int32   `frame:"sum"`
	SoldQuantity    float64 `frame:"sum"`
	PendingQuantity float64 `frame:"sum"`
}

// SaleClientProductFrame is one sale line as the "sale-client-product" DataFrame sums it: one file
// per day, one row per client and product, for the queries by client. ClientID 0 is the sales without
// a client. The sums are SaleSummaryFrame's.
type SaleClientProductFrame struct {
	// Date is the week-spread Part, as in SaleSummaryFrame. The packed key (ClientID, ProductID) holds the
	// per-company sequences: 20 bits ~1M clients, 17 bits ~131k products. A value past its width fails
	// the write.
	Date int16 `frame:"part,13"`
	// ClientID leads the rows: each client is written once in a file, its products under it.
	ClientID        int32   `frame:"row,20"`
	ProductID       int32   `frame:"row,17"`
	AmountPEN       int32   `frame:"sum"`
	DebtAmountPEN   int32   `frame:"sum"`
	SoldQuantity    float64 `frame:"sum"`
	PendingQuantity float64 `frame:"sum"`
}

// SaleQuantityFrame is the "sale-quantity" DataFrame: derived from "sale-summary" (not stored), the
// units sold per day, site and product in whole units, for the sales forecast
// (plan/SALES_FORECAST_PLAN.md). One int32 sum instead of four, so its files are a fraction of
// sale-summary's to cache and send.
type SaleQuantityFrame struct {
	// Date and SiteID are sale-summary's Part and Key, which a derived frame keeps.
	Date         int16 `frame:"part,13"`
	SiteID       int16 `frame:"key"`
	ProductID    int32 `frame:"row"`
	SoldQuantity int32 `frame:"sum"`
}

// deriveSaleQuantity rounds a day/product row of sale-summary to whole units; a product that rounds
// to 0 (fractions of a unit only) is left out of the day.
func deriveSaleQuantity(summary *SaleSummaryFrame, quantity *SaleQuantityFrame) bool {
	quantity.ProductID = summary.ProductID
	quantity.SoldQuantity = int32(math.Round(summary.SoldQuantity))
	return quantity.SoldQuantity != 0
}

// ToSaleSummaryFrames is what the sale adds to its day summary. It appends to lines, which a rebuild
// reuses from sale to sale.
func (e *SaleOrder) ToSaleSummaryFrames(lines []SaleSummaryFrame) []SaleSummaryFrame {
	e.eachSummaryLine(func(line SaleSummaryFrame) { lines = append(lines, line) })
	return lines
}

// isPaymentPending: the sale still owes money (Generado or Entregado); its DebtAmount is what is left.
func (e *SaleOrder) isPaymentPending() bool {
	return e.Status == OrderStatusPending || e.Status == OrderStatusDelivered
}

// ToCashDocumentFrames is what the sale adds to the cash-documents frame: what it charged per line
// tax group (shipping taxed at 18 %), the open part what it still owes, due on PaymentDueDate (with a
// retención, the total less it was ever owed: the retención is a settled part). Beside the total, as
// memo rows: the taxed free lines' reference value (settled, its IGV is owed), the detracción still to
// deposit (open, part of the debt) and the retención (settled, an IGV credit).
func (e *SaleOrder) ToCashDocumentFrames(lines []finance.CashDocumentFrame) []finance.CashDocumentFrame {
	if e.Status == OrderStatusAnnulled {
		return lines
	}
	openAmount, openDetraction := int32(0), int32(0)
	if e.isPaymentPending() {
		openAmount, openDetraction = e.DebtAmount, e.OpenDetractionAmount()
	}
	memos := []finance.CashDocumentMemo{{TaxGroup: finance.TaxGroupDetraction, Amount: openDetraction, IsOpen: true}}
	for _, imposedTax := range e.ImposedTaxes {
		// A free exonerado or inafecto line owes no IGV, so it adds nothing to the cash flow.
		isTaxedFreeLine := imposedTax.Type == finance.TaxGroupFree && finance.IgvBasisPointsOf(e.Lines[imposedTax.Line-1].TaxGroup) > 0
		if isTaxedFreeLine || imposedTax.Type == finance.TaxGroupRetention {
			memos = append(memos, finance.CashDocumentMemo{TaxGroup: imposedTax.Type, Amount: imposedTax.Amount})
		}
	}
	return finance.AppendCashDocumentFrames(lines, finance.CashDocument{
		Date: e.Date, SiteID: e.SiteID, OpenKind: finance.CashDocumentKindSaleOpen, SettledKind: finance.CashDocumentKindSaleSettled,
		DueDate: e.PaymentDueDate, DocumentDate: e.Date, IsForeignCurrency: e.CurrencyType == finance.CurrencyUSD,
		Groups: e.cashDocumentGroups(), OpenAmount: openAmount, Memos: memos,
	})
}

// cashDocumentGroups sums the sale per tax group, in a fixed group order so a rewrite of the same
// sale yields the same rows.
func (e *SaleOrder) cashDocumentGroups() []finance.CashDocumentAmount {
	amountByGroup := map[int8]int32{finance.TaxGroupTaxed18: e.ShippingAmount}
	for _, line := range e.Lines {
		amountByGroup[line.TaxGroup] += core.QuantityAmount(core.UnpackQuantityLine(line.Quantity), line.Price, line.SubPrice)
	}
	groups := make([]finance.CashDocumentAmount, 0, len(amountByGroup))
	for _, taxGroup := range slices.Sorted(maps.Keys(amountByGroup)) {
		if amountByGroup[taxGroup] != 0 {
			groups = append(groups, finance.CashDocumentAmount{TaxGroup: taxGroup, Amount: amountByGroup[taxGroup]})
		}
	}
	return groups
}

// SaleClientDebtBucketBits groups the clients of sale-client-debt into files of 4 096.
const SaleClientDebtBucketBits = 12

// SaleClientDebtFrame is what a client owes, as the "sale-client-debt" DataFrame sums it: one file per
// bucket of 4 096 clients, one row per client. The cash flow joins it with the clients' credit days
// (client-credit-days-missing); only the buckets that changed travel.
type SaleClientDebtFrame struct {
	ClientBucket int32 `frame:"key"`
	ClientID     int32 `frame:"row"`
	// DebtLocal is the open debt in local currency at the sale's rate: an amount at stake for a warning.
	DebtLocal int32 `frame:"sum"`
}

// ToSaleClientDebtFrames is the sale's open debt under its client; a paid sale, or one without a
// client, adds nothing.
func (e *SaleOrder) ToSaleClientDebtFrames(lines []SaleClientDebtFrame) []SaleClientDebtFrame {
	if !e.isPaymentPending() || e.ClientID <= 0 || e.DebtAmount <= 0 {
		return lines
	}
	return append(lines, SaleClientDebtFrame{ClientBucket: e.ClientID >> SaleClientDebtBucketBits, ClientID: e.ClientID,
		DebtLocal: e.AmountInPEN(e.DebtAmount)})
}

// ToSaleClientProductFrames is ToSaleSummaryFrames with the sale's client in every line.
func (e *SaleOrder) ToSaleClientProductFrames(lines []SaleClientProductFrame) []SaleClientProductFrame {
	e.eachSummaryLine(func(line SaleSummaryFrame) {
		lines = append(lines, SaleClientProductFrame{
			Date: line.Date, ClientID: e.ClientID, ProductID: line.ProductID,
			AmountPEN: line.AmountPEN, DebtAmountPEN: line.DebtAmountPEN,
			SoldQuantity: line.SoldQuantity, PendingQuantity: line.PendingQuantity,
		})
	})
	return lines
}

// eachSummaryLine is what the sale adds to the sale DataFrames, line by line, from its current state
// alone: an annulled sale adds nothing, an undelivered one keeps its quantities pending and an unpaid
// one its amount as debt. A frame subtracts what the sale added before each write, so no write has to
// undo anything by hand. Both frames read it, so they never disagree on a line.
func (e *SaleOrder) eachSummaryLine(onLine func(line SaleSummaryFrame)) {
	if e.Status == OrderStatusAnnulled {
		return
	}
	isDeliveryPending := e.IsDeliveryPending()
	isPaymentPending := e.isPaymentPending()
	for _, saleLine := range e.Lines {
		if saleLine.ProductID <= 0 || saleLine.Quantity <= 0 {
			continue
		}
		quantity := core.UnpackQuantityLine(saleLine.Quantity)
		subDivisor := saleLine.SubDivisor
		if subDivisor <= 0 {
			subDivisor = core.QuantityDivisorNone
		}
		// Sub-units are charged at their own price, so the line total is exact, not prorated.
		line := SaleSummaryFrame{
			Date: e.Date, SiteID: e.SiteID, ProductID: saleLine.ProductID,
			AmountPEN:    e.AmountInPEN(core.QuantityAmount(quantity, saleLine.Price, saleLine.SubPrice)),
			SoldQuantity: float64(quantity.Units) + float64(quantity.Sub)/float64(subDivisor),
		}
		if isDeliveryPending {
			line.PendingQuantity = line.SoldQuantity
		}
		if isPaymentPending {
			line.DebtAmountPEN = line.AmountPEN
		}
		onLine(line)
	}
}

// AmountInPEN restates an amount of this sale in soles, with the rate the sale was converted
// with — a USD sale always carries one. The sale summaries and the S/ 700 boleta rule are
// both in soles whatever the sale was charged in.
func (e *SaleOrder) AmountInPEN(amount int32) int32 {
	if e.CurrencyType != finance.CurrencyUSD {
		return amount
	}
	return finance.ConvertUnitPrice(amount, finance.CurrencyUSD, finance.CurrencyPEN, e.ExchangeRate)
}

// TotalInPEN is the total restated in soles (see AmountInPEN).
func (e *SaleOrder) TotalInPEN() int32 {
	return e.AmountInPEN(e.TotalAmount)
}

func (e *SaleOrder) AddStatus(orderState int8) error {
	if orderState == 2 {
		if e.Status == 1 || e.Status == 3 {
			e.Status += 1
		} else {
			core.Log("Error: No se puede agregar el estado Pagado a ", e.Status)
			return core.Err("Error: No se puede agregar el estado Pagado a ", e.Status)
		}
	} else if orderState == 3 {
		if e.Status == 1 || e.Status == 2 {
			e.Status += 2
		} else {
			core.Log("Error: No se puede agregar el estado Entregado a ", e.Status)
			return core.Err("Error: No se puede agregar el estado Entregado a ", e.Status)
		}
	}
	return nil
}

type SaleOrderTable struct {
	db.TableStruct[SaleOrderTable, SaleOrder]
	CompanyID         db.Col[*SaleOrderTable, int32]
	ID                db.Col[*SaleOrderTable, int64]
	Date              db.Col[*SaleOrderTable, int16]
	WarehouseID       db.Col[*SaleOrderTable, int32]
	LastPaymentCajaID db.Col[*SaleOrderTable, int32]
	Lines             db.Col[*SaleOrderTable, []SaleOrderLine]
	// DetailProductIDs has no field in SaleOrder: GetSchema computes it from Lines.
	DetailProductIDs db.Col[*SaleOrderTable, []int32]
	TotalAmount      db.Col[*SaleOrderTable, int32]
	TaxAmount        db.Col[*SaleOrderTable, int32]
	DebtAmount       db.Col[*SaleOrderTable, int32]
	CurrencyType     db.Col[*SaleOrderTable, int8]
	ExchangeRate     db.Col[*SaleOrderTable, int32]
	Created          db.Col[*SaleOrderTable, int32]
	ClientID         db.Col[*SaleOrderTable, int32]
	Updated          db.Col[*SaleOrderTable, int32]
	Version          db.Col[*SaleOrderTable, int16]
	UpdatedBy        db.Col[*SaleOrderTable, int32]
	Status           db.Col[*SaleOrderTable, int8]
	LastPaymentTime  db.Col[*SaleOrderTable, int32]
	LastPaymentUser  db.Col[*SaleOrderTable, int32]
	DeliveryTime     db.Col[*SaleOrderTable, int32]
	DeliveryUser     db.Col[*SaleOrderTable, int32]
	PaymentDueDate   db.Col[*SaleOrderTable, int16]
	AnnulReason      db.Col[*SaleOrderTable, string]
	Period           db.Col[*SaleOrderTable, int16]
	// SameDayCashBankID is the cash bank of the collection that settled the sale on its Date.
	SameDayCashBankID    db.Col[*SaleOrderTable, int32]
	ShippingAmount       db.Col[*SaleOrderTable, int32]
	OnlineSaleOrderID    db.Col[*SaleOrderTable, int32]
	PendingStockMovement db.Col[*SaleOrderTable, int8]
	SiteID               db.Col[*SaleOrderTable, int16]
	ImposedTaxes         db.Col[*SaleOrderTable, []SaleImposedTax]
}

func (e SaleOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        25,
		Name:      "sale_order",
		Partition: e.CompanyID,
		// A period's credit note reaches an older sale through the by-IDs cache.
		CacheByIDs: true,
		// A plain key. The id is built by MakeSaleOrderID, which packs the series'
		// correlativo, two random digits and the series itself into it — a layout
		// the ORM's autoincrement cannot express.
		Keys: db.Cols(e.ID),
		// Sizes the Status slot of the delta index.
		FixedValues: []db.FixedValues{
			{Col: e.Status, Min: 0, Max: 4},
			{Col: e.Period, Min: 0, Max: 9912},
			{Col: e.PendingStockMovement, Min: 0, Max: 1},
		},
		// Only the product index groups below read it, from the record in memory.
		Computed: []db.ComputedCol{
			{Col: e.DetailProductIDs, From: db.Cols(e.Lines), Value: db.Compute((*SaleOrder).DetailProductIDs)},
		},
		Indexes: []db.Index{
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Date),
			},
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Date, e.Status),
			},
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Date, e.ClientID),
			},
			{
				Type: db.TypeCacheGroup,
				// The group's packed value takes 15 (Date) + 26 + 23 bits: up to ~67M clients, ~8M products.
				Keys: db.Cols(e.Date, e.ClientID.Size(26), e.DetailProductIDs.Size(23)),
			},
			{
				Type: db.TypeCacheGroup,
				Keys: db.Cols(e.Date, e.DetailProductIDs),
			},
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// One log per accounting period: GET.accounting-operations syncs a month through it.
			{Type: db.TypeDelta, Keys: db.Cols(e.Period, e.Status)},
			// The sales delivered with a stock exit still pending: PendingStockMovement pinned to 1 by an
			// equality, Status (3, 4) by the Delta() values. Status is a key so a read by Status alone
			// keeps resolving to the Status index (fewer prefixes) instead of tying with this one.
			{Type: db.TypeDelta, Keys: db.Cols(e.PendingStockMovement, e.Status)},
		},
		DataFrames: []db.DataFrame{{
			// The day/product sales summary of /sales/sale_orders_charts. Most sales are settled and
			// delivered, so the zero sums mask drops the pending/debt columns of a day that has none.
			// The Date is a week-spread Part, which only the SQL drivers keep: they hold the frame exact
			// in the sale's own transaction, with nothing to compact. Version 3 dropped the split
			// units/sub sums and the SubDivisor row; version 4 added the SiteID key. UseFileCache keeps
			// each settled day's file, so a first load of the charts reads the cached days instead of
			// their rows.
			Name:            "sale-summary",
			Version:         4,
			UseZeroSumsMask: true,
			UseWeekSpread:   true,
			// UseFileCache:    true,
			ToFrames:        db.ToFrames((*SaleOrder).ToSaleSummaryFrames),
		}, {
			// The day/client/product detail, for the queries by client: its files are larger than
			// sale-summary's (one row per client and product), so that one stays for the day totals.
			Name:            "sale-client-product",
			Version:         3,
			UseZeroSumsMask: true,
			UseWeekSpread:   true,
			// UseFileCache:    true,
			ToFrames:        db.ToFrames((*SaleOrder).ToSaleClientProductFrames),
		}, {
			// The units sold per day and product, for the sales forecast: read from sale-summary's
			// aggregate rows and file marks, nothing stored. Its files are small and change only while
			// their day takes sales, so the file cache serves the settled days.
			Name:         "sale-quantity",
			DerivedFrom:  "sale-summary",
			Version:      1,
			UseFileCache: true,
			Derive:       db.DeriveFrame(deriveSaleQuantity),
		}, {
			Shared: finance.CashDocumentsFrame, Version: 1, ToFrames: db.ToFrames((*SaleOrder).ToCashDocumentFrames),
			OwnedRows: finance.OwnedCashDocumentKinds(finance.CashDocumentKindSaleOpen, finance.CashDocumentKindSaleSettled),
		}, {
			Name: "sale-client-debt", Version: 1, ToFrames: db.ToFrames((*SaleOrder).ToSaleClientDebtFrames),
		}},
	}
}

const (
	OrderStatusPending   = int8(1)
	OrderStatusPaid      = int8(2)
	OrderStatusDelivered = int8(3)
	OrderStatusCompleted = int8(4)
	OrderStatusAnnulled  = int8(0)
)

// The ids of the "Gestión Ventas" entry in backend/access.toml. Named here rather than written
// as literals in the handler so a catalog edit can be found by grep from the code that depends
// on it — the catalog is data and the compiler cannot check the link.
const (
	AccesoIDGestionVentas  = int32(11)
	SubAccesoIDAnularVenta = int32(2)
)

// MaxAnnulReasonLength bounds the free-text reason so the column cannot be used as storage.
const MaxAnnulReasonLength = 200
