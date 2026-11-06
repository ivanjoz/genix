package bench

import (
	"app/core"
	"fmt"
	"net/url"

	"github.com/ivanjoz/colbin"
	"github.com/ivanjoz/genix-orm/utils"
)

// Endpoint is one benchmarked GET route and the query its page sends on a first sync. CursorParam
// is the query parameter its delta cursor travels in.
type Endpoint struct {
	Name        string
	Route       string
	BaseQuery   url.Values
	CursorParam string
}

// Endpoints are the three delta routes a sales/logistics page loads on open.
var Endpoints = []Endpoint{
	{Name: "products", Route: "products", CursorParam: "records"},
	// pending-status=2 is the "por pagar" tab: Status 1 (pending) and 3, the seeded orders.
	{Name: "sale-orders", Route: "sale-orders", BaseQuery: url.Values{"pending-status": {"2"}}, CursorParam: "records"},
	{Name: "products-stock", Route: "products-stock", CursorParam: core.DeltaCursorParamOfDefaultKey},
}

// SyncKind is the two shapes a delta route is called with. A first sync lists the whole view; an
// incremental sync with nothing new is what every later page load sends, so it is the bulk of the
// traffic.
type SyncKind string

const (
	FirstSync       SyncKind = "first"
	IncrementalSync SyncKind = "incremental"
)

// RequestURL builds the URL of one call. The incremental cursor is "C.V" with V ten ticks after C
// and C now-ish: the unsettled window [V-8, C] is empty and no seeded record is newer than C, so the
// server answers the steady state — two empty reads (changed rows, moves) and a metadata record.
func RequestURL(endpoint Endpoint, kind SyncKind) string {
	query := url.Values{}
	for key, values := range endpoint.BaseQuery {
		query[key] = values
	}
	if kind == IncrementalSync {
		readTime := core.SUnixTime()
		query.Set(endpoint.CursorParam, fmt.Sprintf("%d.%d", readTime-10, readTime))
	}
	encodedQuery := query.Encode()
	if encodedQuery == "" {
		return "/api/" + endpoint.Route
	}
	return "/api/" + endpoint.Route + "?" + encodedQuery
}

// MakeSessionToken builds the Authorization value login.go hands the browser, without the login
// round trip: CheckUser validates the keyed hash, not that a login happened.
func MakeSessionToken(companyID, userID int32) string {
	Setup()
	sessionToken := core.UsuarioToken{
		CompanyID: companyID,
		ID:        userID,
		Created:   core.SUnixTime(),
		User:      "bench",
	}
	sessionToken.Hash = core.ComputeUsuarioTokenHash(sessionToken)
	tokenBytes, err := colbin.Marshal(sessionToken)
	if err != nil {
		panic("bench: encoding the session token: " + err.Error())
	}
	return "Bearer " + utils.Base64.EncodeToString(tokenBytes)
}
