package sales

import (
	business "app/business/types"
	"app/core"
	"app/db"
	production "app/production/types"
	"app/sales/types"
	"fmt"
	"strings"
	"time"

	"github.com/gpdf-dev/gpdf/document"
	"github.com/gpdf-dev/gpdf/pdf"
	"github.com/gpdf-dev/gpdf/template"
)

// The "Orden de Compra" a web order paid later (cash on delivery, at the store) is confirmed with:
// the customer shows it to collect the goods. Built on demand from the order, never stored, so it
// always says what the order says now.

// webOrderNumber is the order number the customer sees and the store searches by: the
// OnlineSaleOrder ID, never the sale's (which is the boleta's correlativo). Padded to 4 digits
// ("0001"), as the storefront's formatWebOrderNumber shows it.
func webOrderNumber(onlineOrderID int32) string {
	return fmt.Sprintf("%04d", onlineOrderID)
}

func formatCents(amount int32) string {
	return fmt.Sprintf("S/ %.2f", float64(amount)/100)
}

// webOrderPlaceNames resolves a district to "District, Province, Department" from the ubigeo
// catalog: the code holds its parents' (150101 under 1501 under 15).
func webOrderPlaceNames(districtID int32) string {
	if districtID <= 0 {
		return ""
	}
	cities := []business.CityLocation{}
	query := db.Query(&cities)
	query.Select(query.ID, query.Name).CountryID.Equals(604).
		ID.In(districtID, districtID/100, districtID/10000)
	if err := query.Exec(); err != nil {
		core.Log("webOrderPlaceNames:: no se pudo leer el ubigeo", districtID, err)
		return ""
	}
	nameByID := map[int32]string{}
	for _, city := range cities {
		nameByID[city.ID] = city.Name
	}
	names := []string{}
	for _, cityID := range types.ShippingRateCityIDs(districtID) {
		if name := nameByID[cityID]; name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

// webOrderPurchaseDocument is what the PDF prints, resolved before drawing.
type webOrderPurchaseDocument struct {
	StoreName    string
	Order        types.OnlineSaleOrder
	ProductNames map[int32]string
	// PickupSite is the store the order is collected at, when DeliveryType is pickup.
	PickupSiteName    string
	PickupSiteAddress string
}

// loadWebOrderPurchaseDocument reads what the PDF needs about an order: the store, the product
// names and the pickup site.
func loadWebOrderPurchaseDocument(companyID int32, order types.OnlineSaleOrder) (*webOrderPurchaseDocument, error) {
	companyConfig, err := loadWebStoreConfig(companyID)
	if err != nil {
		return nil, err
	}
	purchaseDocument := webOrderPurchaseDocument{
		StoreName: companyConfig.Company.TradeName, Order: order, ProductNames: map[int32]string{},
	}
	for _, site := range companyConfig.Sites {
		if site.ID == order.PickupSiteID {
			purchaseDocument.PickupSiteName, purchaseDocument.PickupSiteAddress = site.Name, site.Address
		}
	}

	products := []production.Product{}
	query := db.Query(&products)
	query.Select(query.ID, query.Name).CompanyID.Equals(companyID).ID.In(order.ProductIDs()...)
	if err := query.Exec(); err != nil {
		return nil, core.Err("Error al obtener los productos del pedido:", err)
	}
	for _, product := range products {
		purchaseDocument.ProductNames[product.ID] = product.Name
	}
	return &purchaseDocument, nil
}

// buildWebOrderPurchasePDF draws the Orden de Compra.
func buildWebOrderPurchasePDF(purchaseDocument *webOrderPurchaseDocument) ([]byte, error) {
	order := purchaseDocument.Order
	accentColor := pdf.RGBHex(0x4F46E5)

	doc := template.New(
		template.WithPageSize(document.A4),
		template.WithMargins(document.UniformEdges(document.Mm(18))),
		template.WithMetadata(document.DocumentMetadata{
			Title:  "Orden de Compra " + webOrderNumber(order.ID),
			Author: purchaseDocument.StoreName,
		}),
	)
	page := doc.AddPage()

	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(7, func(col *template.ColBuilder) {
			col.Text(purchaseDocument.StoreName, template.FontSize(20), template.Bold(), template.TextColor(accentColor))
		})
		row.Col(5, func(col *template.ColBuilder) {
			col.Text("ORDEN DE COMPRA", template.FontSize(16), template.Bold(), template.AlignRight())
			col.Text("N° "+webOrderNumber(order.ID), template.AlignRight())
			col.Text("Fecha: "+time.Unix(int64(order.Date)*86400, 0).UTC().Format("02/01/2006"), template.AlignRight())
		})
	})
	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(12, func(col *template.ColBuilder) {
			col.Spacer(document.Mm(4))
			col.Line(template.LineColor(accentColor), template.LineThickness(document.Pt(1.5)))
			col.Spacer(document.Mm(4))
		})
	})

	shipping := order.Shipping
	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(6, func(col *template.ColBuilder) {
			col.Text("Cliente", template.Bold(), template.TextColor(pdf.Gray(0.4)))
			col.Text(shipping.FirstName + " " + shipping.LastName)
			col.Text(shipping.Email)
			if shipping.Phone != "" {
				col.Text("Celular: " + shipping.Phone)
			}
			if shipping.DocumentNumber != "" {
				col.Text("DNI: " + shipping.DocumentNumber)
			}
		})
		row.Col(6, func(col *template.ColBuilder) {
			if order.DeliveryType == types.DeliveryTypePickup {
				col.Text("Recojo en tienda", template.Bold(), template.TextColor(pdf.Gray(0.4)))
				col.Text(purchaseDocument.PickupSiteName)
				col.Text(purchaseDocument.PickupSiteAddress)
				col.Text("Pago en tienda al recoger el pedido.")
			} else {
				col.Text("Entrega a domicilio", template.Bold(), template.TextColor(pdf.Gray(0.4)))
				col.Text(shipping.Address)
				if shipping.Reference != "" {
					col.Text("Ref.: " + shipping.Reference)
				}
				col.Text(webOrderPlaceNames(shipping.CityID))
				col.Text("Pago contra entrega.")
			}
		})
	})
	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(12, func(col *template.ColBuilder) { col.Spacer(document.Mm(8)) })
	})

	tableRows := [][]string{}
	for _, line := range order.Lines {
		quantity := core.UnpackQuantityLine(line.Quantity)
		productName := purchaseDocument.ProductNames[line.ProductID]
		if productName == "" {
			productName = fmt.Sprintf("Producto %v", line.ProductID)
		}
		tableRows = append(tableRows, []string{
			productName, fmt.Sprintf("%v", quantity.Units), formatCents(line.Price),
			formatCents(core.QuantityAmount(quantity, line.Price, line.SubPrice)),
		})
	}
	if order.ShippingAmount > 0 {
		tableRows = append(tableRows, []string{"Envío", "1", formatCents(order.ShippingAmount), formatCents(order.ShippingAmount)})
	}
	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(12, func(col *template.ColBuilder) {
			col.Table(
				[]string{"Producto", "Cant.", "P. Unit.", "Importe"},
				tableRows,
				template.ColumnWidths(55, 10, 17, 18),
				template.TableHeaderStyle(template.TextColor(pdf.White), template.BgColor(accentColor)),
				template.TableStripe(pdf.RGBHex(0xF5F5F5)),
			)
		})
	})
	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(8, func(col *template.ColBuilder) {})
		row.Col(4, func(col *template.ColBuilder) {
			col.Spacer(document.Mm(4))
			col.Text("Total: "+formatCents(order.TotalAmount), template.AlignRight(), template.Bold(), template.FontSize(14))
		})
	})
	page.AutoRow(func(row *template.RowBuilder) {
		row.Col(12, func(col *template.ColBuilder) {
			col.Spacer(document.Mm(12))
			col.Line(template.LineColor(pdf.Gray(0.8)))
			col.Spacer(document.Mm(3))
			col.Text("Presente esta orden al recibir o recoger su pedido. Este documento no es un comprobante de pago.",
				template.AlignCenter(), template.Italic(), template.TextColor(pdf.Gray(0.5)))
		})
	})

	return doc.Generate()
}
