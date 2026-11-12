package sales

import (
	"app/sales/types"
	"bytes"
	"testing"
)

func TestBuildWebOrderPurchasePDF(t *testing.T) {
	purchaseDocument := &webOrderPurchaseDocument{
		StoreName: "Bodega Ñuñoa",
		Order: types.OnlineSaleOrder{
			ID: 125, Date: 20400, DeliveryType: types.DeliveryTypePickup, TotalAmount: 2360,
			Lines:    []types.SaleOrderLine{{ProductID: 5, Quantity: 2000, Price: 1180}},
			Shipping: types.SaleOrderShipping{FirstName: "José", LastName: "Peña", Email: "jose@example.pe"},
		},
		ProductNames:      map[int32]string{5: "Aceite de girasol 1 l."},
		PickupSiteName:    "Sede Miraflores",
		PickupSiteAddress: "Av. Larco 123",
	}
	pdfContent, err := buildWebOrderPurchasePDF(purchaseDocument)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdfContent, []byte("%PDF")) {
		t.Fatalf("not a PDF: %q", pdfContent[:min(8, len(pdfContent))])
	}
}
