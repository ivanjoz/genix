// The Código de Anotación de Registro.
//
// SUNAT assigns the CAR itself, from the first calendar day of the period until the RVIE is
// generated, and the replacement file leaves its field empty (Anexo 3, field 4). But the
// construction is published — tabla 7 and Anexo C of RS 112-2021 — and it is entirely
// derived from the comprobante, so the CAR of a document this company issued can be computed
// here without asking SUNAT for anything.
//
// That is what the Libro Diario needs: field 20 of formats 5.1 and 5.2 is the CAR, for a
// taxpayer whose RVIE lives in the SIRE module, and it is what ties an accounting entry to
// the sale it came from.

package types

import (
	"strconv"
	"strings"
)

// The four parts, and their widths. They add up to 27 — an27, as the annex puts it.
const (
	carIssuerWidth = 11
	carTypeWidth   = 2
	carSeriesWidth = 4
	carNumberWidth = 10
	// CARWidth is what a well-formed CAR always measures.
	CARWidth = carIssuerWidth + carTypeWidth + carSeriesWidth + carNumberWidth
)

// BuildCAR assembles the 27-character code for one comprobante.
//
// Every part is padded with leading zeros when short and cut from the right when long, which
// is the annex's rule verbatim: a value longer than its slot keeps its rightmost digits,
// because those are the ones that distinguish two documents. An empty series becomes zeros.
func BuildCAR(issuerDocNumber string, docTypeCode string, seriesCode string, number int32) string {
	return fitRight(issuerDocNumber, carIssuerWidth) +
		fitRight(docTypeCode, carTypeWidth) +
		fitRight(seriesCode, carSeriesWidth) +
		fitRight(strconv.FormatInt(int64(number), 10), carNumberWidth)
}

// fitRight forces a value into exactly width characters, keeping the rightmost ones.
func fitRight(value string, width int) string {
	value = strings.TrimSpace(value)
	if len(value) > width {
		return value[len(value)-width:]
	}
	return strings.Repeat("0", width-len(value)) + value
}
