// How a sale order id is built, and why it is built here rather than by the ORM.
//
// The id carries two things: the correlativo of the invoicing series and the two
// digits of the series itself.
//
//	[correlativo][series:2]
//
// The correlativo is not a second sequence beside the sale's own — it *is* the
// sale's counter. There is one autoincrement per company and series, which is
// exactly the scope SUNAT requires the document numbering to be unique in, so the
// sale 13001 is the comprobante F001-130 and neither number has to be looked up
// to know the other.
//
// There are no random digits: the counter is reserved under the database's row
// lock, so two tills can never be handed the same correlativo and the id needs
// nothing else to be unique.
//
// The boleta or factura of a sale uses the sale's own id. A credit note is issued
// under its own series and takes that series' own correlativo, which is what the
// Correlativo column is for. Its id must not be the sale's correlativo with the
// note's series swapped in: two sales of different series can share a correlativo,
// so their notes in one note series would collide. How a note is keyed is still
// open — notes are not emitted yet.
//
// The ORM cannot express this: its autoincrement takes the whole key, with no way
// to reserve the low two digits for a column. Minting the id here costs nothing
// and puts the layout somewhere it can be read.

package types

import (
	"app/db"
	"errors"
	"fmt"
)

const (
	// saleSeriesFactor is the two decimal digits the series takes at the tail of the id.
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

	// A comprobante series must be gapless (SUNAT), so it reserves exactly. Series 0 is never
	// declared: its numbers come from the process's block of the counter, one write per block
	// instead of one per sale, and a restart leaves a harmless gap.
	reserveCorrelativo := db.GetAutoincrementID
	if issueSeriesID == 0 {
		reserveCorrelativo = db.GetBlockAutoincrementID
	}
	correlativo, err := reserveCorrelativo(IssueSeriesCounterName(companyID, issueSeriesID), 1)
	if err != nil {
		return 0, err
	}
	if correlativo <= 0 {
		return 0, errors.New("el contador de ventas devolvió un valor inválido")
	}

	return correlativo*saleSeriesFactor + int64(issueSeriesID), nil
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
	return e.ID / saleSeriesFactor
}
