// How a sale order id is built, and why it is built here rather than by the ORM.
//
// The id carries three things: the counter, two random digits, and the two digits
// of the invoicing series the sale is meant to be issued under.
//
//	[counter][rand:2][series:2]
//
// The series tail is what lets an electronic document derive its own id from the
// sale's — same counter, same random digits, its own series — so a boleta and the
// credit note that corrects it differ in the tail alone and never collide on a key.
//
// The ORM cannot express this: its autoincrement appends random digits and then
// takes the rest of the key, with no way to reserve the low two digits for a
// column. Since the reservation itself is already serialized by the fareward
// allocator, minting the id here costs nothing and puts the layout somewhere it
// can be read.

package types

import (
	"app/db"
	"errors"
	"fmt"
	"math/rand/v2"
)

const (
	// saleRandomDigits and saleSeriesDigits are each two decimal digits wide.
	saleRandomFactor = int64(100)
	saleSeriesFactor = int64(100)
	// MaxIssueSeriesID mirrors the invoicing module's ceiling. Duplicated as a
	// number rather than imported: sales must not depend on invoicing, and the
	// constraint is really about the width of this id.
	MaxIssueSeriesID = int8(99)
)

// saleOrderCounterName is the sequence sale ids are minted from.
//
// It deliberately keeps the name the ORM's autoincrement used, because the counter
// is live: starting a fresh one at 1 would mint ids in the range of rows already
// written. Nothing else writes this row now that the table declares no
// autoincrement, so owning it here is safe.
func saleOrderCounterName(companyID int32) string {
	return fmt.Sprintf("x%v_sale_order_0", companyID)
}

// MakeSaleOrderID reserves the next sale id for a company.
//
// issueSeriesID may be zero: a till that does not say which series it will invoice
// under leaves the tail at 00, and the document takes its own series when it is
// issued.
func MakeSaleOrderID(companyID int32, issueSeriesID int8) (int64, error) {
	if issueSeriesID < 0 || issueSeriesID > MaxIssueSeriesID {
		return 0, errors.New("la serie de la venta debe estar entre 0 y 99")
	}

	counter, err := db.GetAutoincrementID(saleOrderCounterName(companyID), 1)
	if err != nil {
		return 0, err
	}
	if counter <= 0 {
		return 0, errors.New("el contador de ventas devolvió un valor inválido")
	}

	random := rand.Int64N(saleRandomFactor)
	return counter*saleRandomFactor*saleSeriesFactor + random*saleSeriesFactor + int64(issueSeriesID), nil
}

// SeriesID is the invoicing series the sale was created for, read off its id.
//
// Zero means the till named none. Read it from here rather than from
// IssueSeriesID, which only carries a value on the request that created the sale.
func (e *SaleOrder) SeriesID() int8 {
	return int8(e.ID % saleSeriesFactor)
}
