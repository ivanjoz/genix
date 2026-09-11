// How a sale order id is built, and why it is built here rather than by the ORM.
//
// The id carries three things: the correlativo of the invoicing series, two random
// digits, and the two digits of the series itself.
//
//	[correlativo][rand:2][series:2]
//
// The correlativo is not a second sequence beside the sale's own — it *is* the
// sale's counter. There is one autoincrement per company and series, which is
// exactly the scope SUNAT requires the document numbering to be unique in, so the
// sale 1301102 is the comprobante F001-130 and neither number has to be looked up
// to know the other.
//
// The series tail is what lets an electronic document derive its own id from the
// sale's — same correlativo, same random digits, its own series — so a boleta and
// the credit note that corrects it differ in the tail alone and never collide on a
// key. A note is the one document that does not read its correlativo off the id:
// it is issued under its own series and takes that series' own number, which is
// what the Correlativo column is for.
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

// IssueSeriesCounterName is the sequence a sale id — and with it the correlativo of
// its comprobante — is minted from.
//
// One counter per company and series. It lives in `sales` and not in `invoicing`
// because this is where the number is now reserved, and because invoicing/types
// already imports this package: the other direction would be a cycle. A note that
// needs its own series counter reaches it from here.
//
// Series 0 is the sale that names no comprobante. It gets a counter of its own so
// those sales still have ids, and nothing is ever declared to SUNAT from it.
func IssueSeriesCounterName(companyID int32, seriesID int8) string {
	return fmt.Sprintf("cpe_%v_%v", companyID, seriesID)
}

// MakeSaleOrderID reserves the next sale id for a company and series, which is the
// same act as reserving the correlativo the comprobante will carry.
//
// issueSeriesID may be zero: a till that does not issue a comprobante draws from
// the series-0 counter and leaves the tail at 00.
func MakeSaleOrderID(companyID int32, issueSeriesID int8) (int64, error) {
	if issueSeriesID < 0 || issueSeriesID > MaxIssueSeriesID {
		return 0, errors.New("la serie de la venta debe estar entre 0 y 99")
	}

	correlativo, err := db.GetAutoincrementID(IssueSeriesCounterName(companyID, issueSeriesID), 1)
	if err != nil {
		return 0, err
	}
	if correlativo <= 0 {
		return 0, errors.New("el contador de ventas devolvió un valor inválido")
	}

	random := rand.Int64N(saleRandomFactor)
	return correlativo*saleRandomFactor*saleSeriesFactor + random*saleSeriesFactor + int64(issueSeriesID), nil
}

// SeriesID is the invoicing series the sale was created for, read off its id.
//
// Zero means the till named none. Read it from here rather than from
// IssueSeriesID, which only carries a value on the request that created the sale.
func (e *SaleOrder) SeriesID() int8 {
	return int8(e.ID % saleSeriesFactor)
}

// Correlativo is the number the sale's comprobante carries in its series, read off
// the id. It is the same number for both because they are minted together.
func (e *SaleOrder) Correlativo() int64 {
	return e.ID / (saleRandomFactor * saleSeriesFactor)
}
