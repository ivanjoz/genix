package types

import "app/db"

// OnlineSaleOrder is an order the web store received (docs/ONLINE_SALE_ORDER_PLAN.md). It is not a
// sale: no comprobante number is spent on it and it never enters the sales summaries. A SaleOrder is
// created from it the moment it is paid or its goods are delivered, whichever comes first; from then
// on the SaleOrder rules and every write to it copies Status and DebtAmount back here.
type OnlineSaleOrder struct {
	db.TableStruct[OnlineSaleOrderTable, OnlineSaleOrder]
	CompanyID int32 `cb:"1"`
	// ID is the order number the shopper sees: 1, 2, 3… per company.
	ID   int32 `cb:"2"`
	Date int16 `cb:"3"`
	// Lines are priced by the backend at checkout and kept: the SaleOrder bills what the shopper
	// agreed to, even if the catalog price changed before the payment.
	Lines []SaleOrderLine `cb:"4"`
	// Amounts in cents, PEN. TotalAmount includes ShippingAmount.
	TotalAmount    int32 `cb:"5"`
	DebtAmount     int32 `cb:"6"`
	ShippingAmount int32 `cb:"7"`
	// ExchangeRate is the rate a USD-priced product was converted with at checkout (×1000), 0 when
	// none was; the SaleOrder inherits it.
	ExchangeRate int32 `cb:"8"`
	// DeliveryType is DeliveryTypeHome or DeliveryTypePickup.
	DeliveryType int8 `cb:"9"`
	// PickupSiteID is the Sede the order is collected at. Set only on pickup.
	PickupSiteID int32 `cb:"10"`
	// Shipping is the contact and address of the order, copied so editing a saved address never
	// rewrites it.
	Shipping      SaleOrderShipping `cb:"11"`
	WebCustomerID int32             `cb:"12"`
	// ClientID is the CRM client the boleta names, set when the total required the DNI.
	ClientID int32 `cb:"13"`
	// PaymentMethod is WebPaymentCard, WebPaymentYape or WebPaymentDeferred.
	PaymentMethod int8 `cb:"14"`
	// CulqiChargeID is written as soon as Culqi approves the charge, before the SaleOrder exists:
	// with it set and SaleOrderID 0 the money moved and the sale still has to be registered.
	CulqiChargeID string `cb:"15"`
	// SaleOrderID is the sale created from this order; 0 until it is paid or delivered.
	SaleOrderID int64  `cb:"16"`
	AnnulReason string `cb:"17"`
	// Status mirrors the SaleOrder's: 0 annulled, 1 pending, 2 paid, 3 delivered, 4 both.
	Status    int8    `json:"ss" cb:"ss,18"`
	Created   int32   `cb:"19"`
	Updated   int32   `json:"upd" cb:"upd,20"`
	Version   int16   `json:"ver" cb:"ver,21"`
	Metadata  *string `json:"mdt" cb:"mdt,22"`
	UpdatedBy int32   `cb:"23"`
	// PendingWarehouseID is the warehouse whose ProductStock.PendingDeliveryOnline holds this order's
	// units until it is delivered or annulled: the company's first active warehouse open to online
	// sales at checkout. Kept so the release hits the row the reservation went to, even if the
	// warehouses change meanwhile. The SaleOrder created on payment inherits it as its WarehouseID.
	PendingWarehouseID int32 `cb:"24"`
	// FulfillmentStatus is the back office's progress on the shipment before the goods are handed
	// over: 0 none, FulfillmentPreparing, FulfillmentInTransit (on the way, or ready for pickup). It
	// moves no stock and records no delivery; the delivery itself is Status 3/4, as on any sale.
	FulfillmentStatus int8 `cb:"25"`
}

// The OnlineSaleOrder.FulfillmentStatus steps; "delivered" is read from Status, never stored here.
const (
	FulfillmentPreparing = int8(1)
	FulfillmentInTransit = int8(2)
)

// SaleOrderShipping is the contact and destination a web order was placed with.
type SaleOrderShipping struct {
	FirstName string `cb:"1"`
	LastName  string `cb:"2"`
	Email     string `cb:"3"`
	CityID    int32  `cb:"4"` // district; 0 on pickup
	Address   string `cb:"5"` // empty on pickup
	Reference string `cb:"6"`
	// Phone is required when flag 11 "Requerir número de contacto" is on.
	Phone string `cb:"7"`
	// DocumentNumber is the DNI, required for a boleta over S/ 700.
	DocumentNumber string `cb:"8"`
}

// ProductIDs is the product of every line, in line order.
func (e *OnlineSaleOrder) ProductIDs() []int32 {
	productIDs := make([]int32, 0, len(e.Lines))
	for _, line := range e.Lines {
		productIDs = append(productIDs, line.ProductID)
	}
	return productIDs
}

// NeedsCulqiReconciliation: Culqi charged the card but the sale was never registered.
func (e *OnlineSaleOrder) NeedsCulqiReconciliation() bool {
	return e.Status == OrderStatusPending && e.CulqiChargeID != "" && e.SaleOrderID == 0
}

type OnlineSaleOrderTable struct {
	db.TableStruct[OnlineSaleOrderTable, OnlineSaleOrder]
	CompanyID          db.Col[*OnlineSaleOrderTable, int32]
	ID                 db.Col[*OnlineSaleOrderTable, int32]
	Date               db.Col[*OnlineSaleOrderTable, int16]
	Lines              db.Col[*OnlineSaleOrderTable, []SaleOrderLine]
	TotalAmount        db.Col[*OnlineSaleOrderTable, int32]
	DebtAmount         db.Col[*OnlineSaleOrderTable, int32]
	ShippingAmount     db.Col[*OnlineSaleOrderTable, int32]
	ExchangeRate       db.Col[*OnlineSaleOrderTable, int32]
	DeliveryType       db.Col[*OnlineSaleOrderTable, int8]
	PickupSiteID       db.Col[*OnlineSaleOrderTable, int32]
	Shipping           db.Col[*OnlineSaleOrderTable, SaleOrderShipping]
	WebCustomerID      db.Col[*OnlineSaleOrderTable, int32]
	ClientID           db.Col[*OnlineSaleOrderTable, int32]
	PaymentMethod      db.Col[*OnlineSaleOrderTable, int8]
	CulqiChargeID      db.Col[*OnlineSaleOrderTable, string]
	SaleOrderID        db.Col[*OnlineSaleOrderTable, int64]
	AnnulReason        db.Col[*OnlineSaleOrderTable, string]
	Status             db.Col[*OnlineSaleOrderTable, int8]
	Created            db.Col[*OnlineSaleOrderTable, int32]
	Updated            db.Col[*OnlineSaleOrderTable, int32]
	Version            db.Col[*OnlineSaleOrderTable, int16]
	UpdatedBy          db.Col[*OnlineSaleOrderTable, int32]
	PendingWarehouseID db.Col[*OnlineSaleOrderTable, int32]
	FulfillmentStatus  db.Col[*OnlineSaleOrderTable, int8]
}

func (e OnlineSaleOrderTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          95,
		Name:        "online_sale_order",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 4}},
		Indexes: []db.Index{
			// Pedidos Web: one delta-synced list per status group.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
			// Mis Pedidos and the per-customer cap on pending orders.
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.WebCustomerID)},
		},
	}
}

// The ids of the "Pedidos Web" entry in backend/access.toml.
const (
	AccesoIDPedidosWeb         = int32(59)
	SubAccesoIDAnularPedidoWeb = int32(2)
)
