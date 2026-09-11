package types

import (
	"strconv"
	"strings"
)

// MaxSeriesID is the ceiling on a series id, and it is not a preference: the id
// is the two-digit suffix a document id carries, so ninety-nine is all the room
// there is.
const MaxSeriesID = int8(99)

// InvoiceSeries is a series a company is allowed to issue under.
//
// It lives inline on the company record rather than in a table of its own. A
// company has a handful of series, every emission needs one, and a table meant
// a full scan per document to find it.
//
// SeriesID and SeriesCode carry no relation to each other. SeriesID is a small
// local number that partitions the correlativo counter; SeriesCode is the
// alphanumeric string SUNAT sees. Series 1 may well be B001 and series 2 F001 —
// the code is a label hanging off the id, nothing more.
type InvoiceSeries struct {
	// SeriesID is 1..99, and is what a document id carries in its last two digits.
	SeriesID int8 `json:",omitempty"`
	// DocType is SUNAT catalog 01: 1 factura, 3 boleta, 7 credit note, 8 debit note.
	DocType int8 `json:",omitempty"`
	// SeriesCode is what SUNAT sees, e.g. F001.
	SeriesCode string `json:",omitempty"`
	// SiteID is the establishment issuing under this series. The code SUNAT
	// assigned to the site travels in the XML, so a company with several branches
	// keeps one series per branch.
	SiteID int32 `json:",omitempty"`
	// IsDefault marks the series used when the caller names none for a document
	// type. At most one per type.
	IsDefault int8 `json:",omitempty"`
	Status    int8 `json:"ss,omitempty"`
}

// DefaultInvoiceSeries is the set every new company starts with.
//
// Six rather than four: a note has to carry the prefix of the document it
// corrects — F for a factura, B for a boleta — so covering notes over boletas,
// which are the majority of sales, needs its own pair. The last three characters
// of a note series do not have to match the document it affects, which is why
// FC01 can correct any F-series factura.
//
// Only the two selling series are marked default. A note's series is decided by
// what it corrects, not by a per-type preference, so a default there would be a
// wrong answer waiting to be used.
// They are seeded with no site, because a company has none when it is created.
// SiteID 0 means "the company's only establishment": BuildIssuer resolves it to the
// single active site, which is the right answer for a single-site company and the
// only unambiguous one. A company with several branches has to name the site on the
// series, because the address travels to SUNAT and the wrong branch is a
// misdeclaration.
func DefaultInvoiceSeries() []InvoiceSeries {
	return []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeFactura, SeriesCode: "F001", IsDefault: 1, Status: 1},
		{SeriesID: 2, DocType: DocTypeBoleta, SeriesCode: "B001", IsDefault: 1, Status: 1},
		{SeriesID: 3, DocType: DocTypeCreditNote, SeriesCode: "FC01", Status: 1},
		{SeriesID: 4, DocType: DocTypeDebitNote, SeriesCode: "FD01", Status: 1},
		{SeriesID: 5, DocType: DocTypeCreditNote, SeriesCode: "BC01", Status: 1},
		{SeriesID: 6, DocType: DocTypeDebitNote, SeriesCode: "BD01", Status: 1},
	}
}

// IsNote is true for the two document types that correct another document.
func IsNote(docType int8) bool {
	return docType == DocTypeCreditNote || docType == DocTypeDebitNote
}

// FindSeries returns the series with that id, or nil. Inactive series are
// returned: a document issued last year still has to name the series it used.
func FindSeries(allSeries []InvoiceSeries, seriesID int8) *InvoiceSeries {
	for index := range allSeries {
		if allSeries[index].SeriesID == seriesID {
			return &allSeries[index]
		}
	}
	return nil
}

// ResolveSeries picks the series a document is numbered in.
//
// Naming one explicitly wins. Otherwise the default for that document type is
// used, which is what a till does: it knows it is selling, not which series the
// accountant set up.
func ResolveSeries(allSeries []InvoiceSeries, docType int8, seriesID int8) (*InvoiceSeries, error) {
	if seriesID != 0 {
		found := FindSeries(allSeries, seriesID)
		if found == nil || found.Status != 1 {
			return nil, errSeries("La serie indicada no existe o está inactiva.")
		}
		if docType != 0 && found.DocType != docType {
			return nil, errSeries("La serie indicada no corresponde a ese tipo de comprobante.")
		}
		return found, nil
	}

	if docType == 0 {
		docType = DocTypeBoleta
	}
	// A note has to be issued in a series of the same family as the document it
	// corrects, and only the caller knows which that is. Picking one here would
	// mean issuing an F-series note against a boleta about a third of the time.
	if IsNote(docType) {
		return nil, errSeries("Una nota de crédito o débito debe indicar la serie con la que se emite.")
	}

	var fallback *InvoiceSeries
	for index := range allSeries {
		candidate := &allSeries[index]
		if candidate.Status != 1 || candidate.DocType != docType {
			continue
		}
		if candidate.IsDefault == 1 {
			return candidate, nil
		}
		if fallback == nil {
			fallback = candidate
		}
	}
	if fallback == nil {
		return nil, errSeries("La empresa no tiene una serie configurada para ese tipo de comprobante.")
	}
	return fallback, nil
}

// ValidateSeries checks the whole set, because the rules that matter are about
// the set: ids and codes have to be unique, and a document type can only have
// one default. It also normalizes the codes to upper case in place.
//
// The per-series rules are SUNAT's, not preferences — the wrong first letter
// means every document numbered under it is rejected.
func ValidateSeries(allSeries []InvoiceSeries) error {
	seenID := map[int8]bool{}
	seenCode := map[string]bool{}
	defaultOfType := map[int8]bool{}

	for index := range allSeries {
		series := &allSeries[index]
		series.SeriesCode = strings.ToUpper(strings.TrimSpace(series.SeriesCode))

		switch series.DocType {
		case DocTypeFactura, DocTypeBoleta, DocTypeCreditNote, DocTypeDebitNote:
		default:
			return errSeries("El tipo de comprobante de la serie " + series.SeriesCode + " no es válido.")
		}

		if series.SeriesID <= 0 || series.SeriesID > MaxSeriesID {
			return errSeries("El número interno de la serie debe estar entre 1 y 99.")
		}
		if seenID[series.SeriesID] {
			return errSeries("Hay dos series con el número interno " + itoa(int64(series.SeriesID)) + ".")
		}
		seenID[series.SeriesID] = true

		if len(series.SeriesCode) != 4 {
			return errSeries("El código de serie debe tener 4 caracteres, por ejemplo F001.")
		}
		prefix := series.SeriesCode[0]
		if prefix != 'F' && prefix != 'B' {
			return errSeries("El código de serie " + series.SeriesCode + " debe empezar con F o con B.")
		}
		if series.DocType == DocTypeFactura && prefix != 'F' {
			return errSeries("Una factura necesita una serie que empiece con F.")
		}
		if series.DocType == DocTypeBoleta && prefix != 'B' {
			return errSeries("Una boleta necesita una serie que empiece con B.")
		}

		// Only active codes have to be unique: a retired series keeps its code so
		// the documents issued under it still read correctly.
		if series.Status == 1 {
			if seenCode[series.SeriesCode] {
				return errSeries("Ya existe una serie activa con el código " + series.SeriesCode + ".")
			}
			seenCode[series.SeriesCode] = true
		}

		// SiteID 0 is allowed and means the company's only establishment, resolved
		// at emission. Requiring a site here would make the seeded series unsaveable
		// until somebody created one, and a company with a single site never has to
		// think about it.

		if series.IsDefault == 1 && series.Status == 1 {
			if defaultOfType[series.DocType] {
				return errSeries("Solo una serie puede ser la predeterminada por tipo de comprobante.")
			}
			defaultOfType[series.DocType] = true
		}
	}
	return nil
}

// ApplyDefaultSeries settles the defaults across the whole set, after the series
// identified by changedSeriesID was added, edited or deactivated.
//
// Being the default is a property of the set, not of a row, so it cannot be saved
// one series at a time: a caller marking a new default without clearing the old one
// produces two, which is what ValidateSeries refuses. It is resolved here so every
// writer gets the same answer.
//
// Two rules, in order. A series that is active and claims the default takes it from
// whoever held it. Then any document type that has an active series but no active
// default adopts one — which is what keeps deactivating a default from leaving its
// type with none, and what makes the first series of a type its default.
func ApplyDefaultSeries(allSeries []InvoiceSeries, changedSeriesID int8) {
	changed := FindSeries(allSeries, changedSeriesID)
	if changed != nil && changed.Status == 1 && changed.IsDefault == 1 {
		for index := range allSeries {
			if allSeries[index].DocType == changed.DocType && allSeries[index].SeriesID != changedSeriesID {
				allSeries[index].IsDefault = 0
			}
		}
	}

	// An inactive series is never the default: it cannot be issued under.
	for index := range allSeries {
		if allSeries[index].Status != 1 {
			allSeries[index].IsDefault = 0
		}
	}

	for docType := range activeDocTypes(allSeries) {
		if hasActiveDefault(allSeries, docType) {
			continue
		}
		for index := range allSeries {
			if allSeries[index].Status == 1 && allSeries[index].DocType == docType {
				allSeries[index].IsDefault = 1
				break
			}
		}
	}
}

func activeDocTypes(allSeries []InvoiceSeries) map[int8]bool {
	docTypes := map[int8]bool{}
	for index := range allSeries {
		if allSeries[index].Status == 1 {
			docTypes[allSeries[index].DocType] = true
		}
	}
	return docTypes
}

func hasActiveDefault(allSeries []InvoiceSeries, docType int8) bool {
	for index := range allSeries {
		if allSeries[index].Status == 1 && allSeries[index].DocType == docType &&
			allSeries[index].IsDefault == 1 {
			return true
		}
	}
	return false
}

// NextSeriesID is the id a newly added series takes. Ids are never reused:
// documents already issued point at theirs forever.
func NextSeriesID(allSeries []InvoiceSeries) int8 {
	highest := int8(0)
	for index := range allSeries {
		if allSeries[index].SeriesID > highest {
			highest = allSeries[index].SeriesID
		}
	}
	return highest + 1
}

// seriesError keeps this package free of an app/core import: it is a leaf that
// the config module reads, and core would drag a module body across the line.
type seriesError string

func (e seriesError) Error() string { return string(e) }

func errSeries(message string) error { return seriesError(message) }

// itoa is the one number-to-string this package needs, kept here so the
// document type does not import strconv for a single call.
func itoa(value int64) string { return strconv.FormatInt(value, 10) }
