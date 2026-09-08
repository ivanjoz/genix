package types

import "testing"

func activeSeries() []InvoiceSeries {
	return []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, IsDefault: 1, Status: 1},
		{SeriesID: 2, DocType: DocTypeFactura, SeriesCode: "F001", SiteID: 1, IsDefault: 1, Status: 1},
		{SeriesID: 3, DocType: DocTypeBoleta, SeriesCode: "B002", SiteID: 2, Status: 1},
	}
}

// The id and the code are independent: series 1 is a boleta and series 2 a
// factura, which is the case the packed-key design used to make impossible.
func TestResolveSeriesByID(t *testing.T) {
	found, err := ResolveSeries(activeSeries(), 0, 2)
	if err != nil {
		t.Fatalf("ResolveSeries: %v", err)
	}
	if found.SeriesCode != "F001" || found.DocType != DocTypeFactura {
		t.Errorf("got %v/%v, want F001/factura", found.SeriesCode, found.DocType)
	}
}

func TestResolveSeriesRejectsTypeMismatch(t *testing.T) {
	if _, err := ResolveSeries(activeSeries(), DocTypeBoleta, 2); err == nil {
		t.Error("a factura series accepted a boleta")
	}
}

func TestResolveSeriesPrefersDefault(t *testing.T) {
	// B002 comes after B001 but is not the default, so B001 has to win.
	found, err := ResolveSeries(activeSeries(), DocTypeBoleta, 0)
	if err != nil {
		t.Fatalf("ResolveSeries: %v", err)
	}
	if found.SeriesCode != "B001" {
		t.Errorf("got %v, want the default B001", found.SeriesCode)
	}
}

// No document type named means a boleta, which is what a till sells.
func TestResolveSeriesDefaultsToBoleta(t *testing.T) {
	found, err := ResolveSeries(activeSeries(), 0, 0)
	if err != nil {
		t.Fatalf("ResolveSeries: %v", err)
	}
	if found.DocType != DocTypeBoleta {
		t.Errorf("got doc type %v, want boleta", found.DocType)
	}
}

func TestResolveSeriesSkipsInactive(t *testing.T) {
	allSeries := []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeFactura, SeriesCode: "F001", SiteID: 1, IsDefault: 1, Status: 0},
	}
	if _, err := ResolveSeries(allSeries, DocTypeFactura, 0); err == nil {
		t.Error("an inactive series was chosen to issue under")
	}
}

// No site means the company's own fiscal address, which is the ordinary case for
// a single-site company and what the seeded series carry.
func TestValidateSeriesAcceptsASeriesWithNoSite(t *testing.T) {
	allSeries := []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", Status: 1},
	}
	if err := ValidateSeries(allSeries); err != nil {
		t.Errorf("a series without a site was rejected: %v", err)
	}
}

// The set every company starts with has to pass its own validation, and it has to
// cover notes over boletas as well as over facturas.
func TestDefaultSeriesAreValidAndCoverBothFamilies(t *testing.T) {
	defaults := DefaultInvoiceSeries()
	if err := ValidateSeries(defaults); err != nil {
		t.Fatalf("the seeded series do not validate: %v", err)
	}
	if len(defaults) != 6 {
		t.Fatalf("got %v series, want 6", len(defaults))
	}

	// A note must carry the prefix of the document it corrects, so each note type
	// needs both an F and a B series.
	for _, docType := range []int8{DocTypeCreditNote, DocTypeDebitNote} {
		prefixes := map[byte]bool{}
		for _, series := range defaults {
			if series.DocType == docType {
				prefixes[series.SeriesCode[0]] = true
			}
		}
		if !prefixes['F'] || !prefixes['B'] {
			t.Errorf("document type %v has no series for one of the two families", docType)
		}
	}
}

// A note's series depends on the document it corrects, which only the caller
// knows — so resolving one by default would be a wrong answer waiting to be used.
func TestResolveSeriesRefusesToGuessANoteSeries(t *testing.T) {
	defaults := DefaultInvoiceSeries()
	for _, docType := range []int8{DocTypeCreditNote, DocTypeDebitNote} {
		if _, err := ResolveSeries(defaults, docType, 0); err == nil {
			t.Errorf("a series was guessed for document type %v", docType)
		}
	}
	// Named explicitly, it resolves.
	if _, err := ResolveSeries(defaults, DocTypeCreditNote, 5); err != nil {
		t.Errorf("an explicitly named note series was refused: %v", err)
	}
}

func TestValidateSeriesAcceptsAValidSet(t *testing.T) {
	if err := ValidateSeries(activeSeries()); err != nil {
		t.Errorf("valid set rejected: %v", err)
	}
}

func TestValidateSeriesRejects(t *testing.T) {
	cases := []struct {
		name      string
		allSeries []InvoiceSeries
	}{
		{"duplicate id", []InvoiceSeries{
			{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, Status: 1},
			{SeriesID: 1, DocType: DocTypeFactura, SeriesCode: "F001", SiteID: 1, Status: 1},
		}},
		{"duplicate active code", []InvoiceSeries{
			{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, Status: 1},
			{SeriesID: 2, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, Status: 1},
		}},
		{"two defaults for one type", []InvoiceSeries{
			{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, IsDefault: 1, Status: 1},
			{SeriesID: 2, DocType: DocTypeBoleta, SeriesCode: "B002", SiteID: 1, IsDefault: 1, Status: 1},
		}},
		{"id above the two digits a document carries", []InvoiceSeries{
			{SeriesID: 100, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, Status: 1},
		}},
		{"factura on a B series", []InvoiceSeries{
			{SeriesID: 1, DocType: DocTypeFactura, SeriesCode: "B001", SiteID: 1, Status: 1},
		}},
		{"code that is not four characters", []InvoiceSeries{
			{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B1", SiteID: 1, Status: 1},
		}},
	}
	for _, testCase := range cases {
		if err := ValidateSeries(testCase.allSeries); err == nil {
			t.Errorf("%v was accepted", testCase.name)
		}
	}
}

// A retired series keeps its code, so the same code may reappear once the old one
// is inactive — otherwise a company could never re-register a series with SUNAT.
func TestValidateSeriesAllowsARetiredCodeToRepeat(t *testing.T) {
	allSeries := []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, Status: 0},
		{SeriesID: 2, DocType: DocTypeBoleta, SeriesCode: "B001", SiteID: 1, Status: 1},
	}
	if err := ValidateSeries(allSeries); err != nil {
		t.Errorf("retired code blocked its replacement: %v", err)
	}
}

func TestValidateSeriesUppercasesTheCode(t *testing.T) {
	allSeries := []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: " b001 ", SiteID: 1, Status: 1},
	}
	if err := ValidateSeries(allSeries); err != nil {
		t.Fatalf("ValidateSeries: %v", err)
	}
	if allSeries[0].SeriesCode != "B001" {
		t.Errorf("got %q, want B001", allSeries[0].SeriesCode)
	}
}

// Ids are never reused: a document issued under series 3 points at it forever, so
// deleting 3 must not let a new series take that id.
func TestNextSeriesIDNeverReuses(t *testing.T) {
	allSeries := activeSeries()
	if next := NextSeriesID(allSeries); next != 4 {
		t.Errorf("got %v, want 4", next)
	}

	remaining := []InvoiceSeries{allSeries[0], allSeries[2]} // dropped id 2
	if next := NextSeriesID(remaining); next != 4 {
		t.Errorf("got %v after a deletion, want 4", next)
	}
}

func TestNextSeriesIDStartsAtOne(t *testing.T) {
	if next := NextSeriesID(nil); next != 1 {
		t.Errorf("got %v, want 1", next)
	}
}
