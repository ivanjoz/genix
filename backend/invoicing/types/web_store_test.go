package types

import (
	sales "app/sales/types"
	"testing"
)

// A web order's delivery is billed as one more taxed item, and the sale's IGV is split the same
// way, so the sale and its boleta agree to the cent.
func TestBuildLinesAddsTheShippingItem(t *testing.T) {
	order := &sales.SaleOrder{
		Lines:          []sales.SaleOrderLine{{ProductID: 101, Quantity: 2000, Price: 1180}},
		ShippingAmount: 590,
	}
	lines, _, err := buildLines(order, candyBoxProducts())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 || lines[1].Description != ShippingLineDescription {
		t.Fatalf("expected the product line plus the shipping item, got %+v", lines)
	}
	lineTax := int64(0)
	for _, line := range lines {
		lineTax += int64(line.IGV)
	}
	if saleTax := SaleOrderTaxAmount(order); int64(saleTax) != lineTax {
		t.Fatalf("sale IGV %v differs from the document's %v", saleTax, lineTax)
	}
}

func TestBuildLinesOmitsShippingOnTillSales(t *testing.T) {
	order := &sales.SaleOrder{Lines: []sales.SaleOrderLine{{ProductID: 101, Quantity: 1000, Price: 500}}}
	lines, _, err := buildLines(order, candyBoxProducts())
	if err != nil || len(lines) != 1 {
		t.Fatalf("expected one line and no shipping item, got %v lines (err %v)", len(lines), err)
	}
}

func TestWebStoreSeriesIsOneBoletaSeries(t *testing.T) {
	allSeries := DefaultInvoiceSeries()
	var boletaID, facturaID int8
	for _, series := range allSeries {
		if series.DocType == DocTypeBoleta && boletaID == 0 {
			boletaID = series.SeriesID
		}
		if series.DocType == DocTypeFactura && facturaID == 0 {
			facturaID = series.SeriesID
		}
	}

	FindSeries(allSeries, facturaID).UseInWebStore = 1
	if err := ValidateSeries(allSeries); err == nil {
		t.Fatal("a factura series must not be usable in the web store")
	}
	FindSeries(allSeries, facturaID).UseInWebStore = 0

	FindSeries(allSeries, boletaID).UseInWebStore = 1
	if err := ValidateSeries(allSeries); err != nil {
		t.Fatalf("a boleta series must be accepted: %v", err)
	}
	if found := FindWebStoreSeries(allSeries); found == nil || found.SeriesID != boletaID {
		t.Fatalf("expected series %v as the web store series, got %+v", boletaID, found)
	}
}

func TestApplyWebStoreSeriesMovesTheMark(t *testing.T) {
	allSeries := []InvoiceSeries{
		{SeriesID: 1, DocType: DocTypeBoleta, SeriesCode: "B001", Status: 1, UseInWebStore: 1},
		{SeriesID: 2, DocType: DocTypeBoleta, SeriesCode: "B002", Status: 1, UseInWebStore: 1},
	}
	ApplyWebStoreSeries(allSeries, 2)
	if allSeries[0].UseInWebStore != 0 || allSeries[1].UseInWebStore != 1 {
		t.Fatalf("the series that claims the mark must be the only one with it: %+v", allSeries)
	}
}
