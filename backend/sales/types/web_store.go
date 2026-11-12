// The web store's customers and its checkout vocabulary.
//
// The tables live in sales/types because the checkout that owns them is a sales flow, and
// because `cloud` (which publishes the public company file) reads the flag ids below: a leaf
// package is the only place both can reach. See docs/WEB_CHECKOUT_PLAN.md.

package types

import (
	"app/db"
	"strings"
)

// The [ecommerce] company flags (backend/company_flags.toml).
const (
	CompanyFlagConfirmEmailOnOnlinePayment   = int16(10)
	CompanyFlagRequireContactNumber          = int16(11)
	CompanyFlagWebReceiptAutomatic           = int16(12)
	CompanyFlagWebCashOnDelivery             = int16(13)
	CompanyFlagConfirmEmailOnDeferredPayment = int16(14)
)

// EcommerceCompanyFlags are the flags the storefront reads from the public company file.
var EcommerceCompanyFlags = []int16{
	CompanyFlagConfirmEmailOnOnlinePayment, CompanyFlagRequireContactNumber,
	CompanyFlagWebReceiptAutomatic, CompanyFlagWebCashOnDelivery,
	CompanyFlagConfirmEmailOnDeferredPayment,
}

// OnlineSaleOrder.DeliveryType values.
const (
	DeliveryTypeHome   = int8(1)
	DeliveryTypePickup = int8(2)
)

// The payment methods the web checkout accepts.
const (
	WebPaymentCard     = int8(1)
	WebPaymentYape     = int8(2)
	WebPaymentDeferred = int8(3) // cash on delivery, or paid at the pickup store
)

// IsOnlineWebPayment is whether the method is charged through Culqi at checkout.
func IsOnlineWebPayment(paymentMethod int8) bool {
	return paymentMethod == WebPaymentCard || paymentMethod == WebPaymentYape
}

// NormalizeCustomerEmail is the form a customer email is stored and matched in.
func NormalizeCustomerEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// WebCustomer is a shopper of one company's web store, unique per email within the company. It is
// not a crm.ClientProvider: most shoppers never need one. ClientID links the CRM client created
// when an order needed an identified receipt (a DNI on a boleta over S/ 700).
type WebCustomer struct {
	db.TableStruct[WebCustomerTable, WebCustomer]
	CompanyID      int32  `cb:"1"`
	ID             int32  `cb:"2"`
	Email          string `cb:"3"` // NormalizeCustomerEmail
	FirstName      string `cb:"4"`
	LastName       string `cb:"5"`
	Phone          string `cb:"6"`
	DocumentNumber string `cb:"7"`
	// EmailConfirmed is set the first time the customer types a code sent to the email.
	EmailConfirmed bool `cb:"8"`
	// PasswordHash is bcrypt, empty until the customer sets a password; until then they log in
	// with an emailed code. Never sent to the browser: PublicWebCustomer blanks it.
	PasswordHash string `cb:"9"`
	ClientID     int32  `cb:"10"`
	// CreatedIP is the client address that created the customer (core.ClientIPKey), so the number
	// of customers one connection can create per window is bounded.
	CreatedIP int64 `cb:"11"`
	Status    int8  `json:"ss" cb:"ss,12"`
	Updated   int32 `json:"upd" cb:"upd,13"`
	Created   int32 `cb:"14"`
}

// PublicWebCustomer is the record as the customer may see it: no password hash, no address.
func (e WebCustomer) PublicWebCustomer() WebCustomer {
	e.PasswordHash = ""
	e.CreatedIP = 0
	return e
}

// HasPassword tells the storefront whether "Cambiar Contraseña" sets or changes one.
func (e WebCustomer) HasPassword() bool { return e.PasswordHash != "" }

type WebCustomerTable struct {
	db.TableStruct[WebCustomerTable, WebCustomer]
	CompanyID      db.Col[*WebCustomerTable, int32]
	ID             db.Col[*WebCustomerTable, int32]
	Email          db.Col[*WebCustomerTable, string]
	FirstName      db.Col[*WebCustomerTable, string]
	LastName       db.Col[*WebCustomerTable, string]
	Phone          db.Col[*WebCustomerTable, string]
	DocumentNumber db.Col[*WebCustomerTable, string]
	EmailConfirmed db.Col[*WebCustomerTable, bool]
	PasswordHash   db.Col[*WebCustomerTable, string]
	ClientID       db.Col[*WebCustomerTable, int32]
	CreatedIP      db.Col[*WebCustomerTable, int64]
	Status         db.Col[*WebCustomerTable, int8]
	Updated        db.Col[*WebCustomerTable, int32]
	Created        db.Col[*WebCustomerTable, int32]
}

func (e WebCustomerTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        92,
		Name:      "web_customers",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			// One customer per email within the store.
			{Type: db.TypeUniqueIndex, Keys: db.Cols(e.Email)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.CreatedIP)},
		},
	}
}

// WebCustomerAddress is a delivery address a customer saved. Orders copy it into
// OnlineSaleOrder.Shipping, so editing or deleting it never changes a past order.
type WebCustomerAddress struct {
	db.TableStruct[WebCustomerAddressTable, WebCustomerAddress]
	CompanyID  int32  `cb:"1"`
	ID         int32  `cb:"2"`
	CustomerID int32  `cb:"3"`
	CityID     int32  `cb:"4"` // district
	Address    string `cb:"5"`
	Reference  string `cb:"6"`
	Status     int8   `json:"ss" cb:"ss,7"`
	Updated    int32  `json:"upd" cb:"upd,8"`
}

type WebCustomerAddressTable struct {
	db.TableStruct[WebCustomerAddressTable, WebCustomerAddress]
	CompanyID  db.Col[*WebCustomerAddressTable, int32]
	ID         db.Col[*WebCustomerAddressTable, int32]
	CustomerID db.Col[*WebCustomerAddressTable, int32]
	CityID     db.Col[*WebCustomerAddressTable, int32]
	Address    db.Col[*WebCustomerAddressTable, string]
	Reference  db.Col[*WebCustomerAddressTable, string]
	Status     db.Col[*WebCustomerAddressTable, int8]
	Updated    db.Col[*WebCustomerAddressTable, int32]
}

func (e WebCustomerAddressTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        93,
		Name:      "web_customer_addresses",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.CustomerID)},
		},
	}
}

// WebCustomerCode is the 6-digit code mailed to confirm an email: one live code per email and
// store, serving registration, login without a password, and checkout's email confirmation.
type WebCustomerCode struct {
	db.TableStruct[WebCustomerCodeTable, WebCustomerCode]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Email     string `cb:"3"`
	Code      string `cb:"4"`
	Attempts  int8   `cb:"5"`
	// Created starts the code's life; LastSent paces the resends.
	Created  int32 `cb:"6"`
	LastSent int32 `cb:"7"`
	// IP is who asked for the code, to bound how many emails one connection can mail per window.
	IP      int64 `cb:"8"`
	Status  int8  `json:"ss" cb:"ss,9"` // WebCustomerCodeSent / WebCustomerCodeUsed
	Updated int32 `json:"upd" cb:"upd,10"`
}

const (
	WebCustomerCodeUsed = int8(0)
	WebCustomerCodeSent = int8(1)
)

type WebCustomerCodeTable struct {
	db.TableStruct[WebCustomerCodeTable, WebCustomerCode]
	CompanyID db.Col[*WebCustomerCodeTable, int32]
	ID        db.Col[*WebCustomerCodeTable, int32]
	Email     db.Col[*WebCustomerCodeTable, string]
	Code      db.Col[*WebCustomerCodeTable, string]
	Attempts  db.Col[*WebCustomerCodeTable, int8]
	Created   db.Col[*WebCustomerCodeTable, int32]
	LastSent  db.Col[*WebCustomerCodeTable, int32]
	IP        db.Col[*WebCustomerCodeTable, int64]
	Status    db.Col[*WebCustomerCodeTable, int8]
	Updated   db.Col[*WebCustomerCodeTable, int32]
}

func (e WebCustomerCodeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        94,
		Name:      "web_customer_codes",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		Indexes: []db.Index{
			// The code is reused, never duplicated: one row per email, rewritten on each new code.
			{Type: db.TypeUniqueIndex, Keys: db.Cols(e.Email)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.IP)},
		},
	}
}
