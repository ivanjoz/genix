package types

// TaxGroupOption is one IGV treatment a product can carry.
type TaxGroupOption struct {
	ID   int8   `cb:"1"`
	Name string `cb:"2"`
}

// ProductTaxGroupOptions is what the product form's "Afectación IGV" offers, via the //CATALOG:
// marker in frontend/services/production/products.svelte.ts. Gratuito (7) is not a product's: a sale
// gives a line away (SaleImposedTax), whatever the product's treatment.
var ProductTaxGroupOptions = []TaxGroupOption{
	{ID: TaxGroupTaxed18, Name: "Taxed 18 %|Gravado 18 %"},
	{ID: TaxGroupTaxed10, Name: "Taxed 10 % (MYPE restaurants, hotels)|Gravado 10 % (MYPE restaurantes, hoteles)"},
	{ID: TaxGroupExonerated, Name: "Exempt|Exonerado"},
	{ID: TaxGroupUnaffected, Name: "Unaffected|Inafecto"},
	{ID: TaxGroupExport, Name: "Export|Exportación"},
}

// IsProductTaxGroup: the treatment is one ProductTaxGroupOptions offers.
func IsProductTaxGroup(taxGroup int8) bool {
	return taxGroup >= TaxGroupTaxed18 && taxGroup <= TaxGroupExport
}

// How a tax charged besides the IGV moves the sale's cash (plan/CASH_DOCUMENTS_FRAME_PLAN.md §9.1). The
// customer owes the company TotalAmount + Σ OnTop − Σ Withheld; a DepositedApart tax stays owed, but is
// collected into the Banco de la Nación detracciones account.
const (
	ImposedTaxNoCash         int8 = 1 // gratuito: a value given away, never charged
	ImposedTaxInsideTotal    int8 = 2 // ISC, ICBPER: already in the line prices
	ImposedTaxOnTop          int8 = 3 // percepción: charged over the total
	ImposedTaxWithheld       int8 = 4 // retención: the customer pays it to SUNAT, not to the company
	ImposedTaxDepositedApart int8 = 5 // detracción: deposited into the detracciones account
)

// ImposedTaxCashBehaviorOf is the ImposedTax* behavior of a tax group charged besides the IGV; 0 for
// the IGV treatments.
func ImposedTaxCashBehaviorOf(taxGroup int8) int8 {
	switch taxGroup {
	case TaxGroupFree:
		return ImposedTaxNoCash
	case TaxGroupIsc, TaxGroupIcbper:
		return ImposedTaxInsideTotal
	case TaxGroupPerception:
		return ImposedTaxOnTop
	case TaxGroupRetention:
		return ImposedTaxWithheld
	case TaxGroupDetraction:
		return ImposedTaxDepositedApart
	}
	return 0
}

// The SPOT and retention rules a sale is checked against, amounts in PEN cents and rates in
// hundredths of a percent (1200 = 12 %).
const (
	// DetractionMinAmountPEN: a factura is subject to its detracción above S/ 700.
	DetractionMinAmountPEN = 70000
	// RetentionMinAmountPEN: a retention agent withholds on a factura above S/ 700.
	RetentionMinAmountPEN = 70000
	// RetentionRate is the IGV retention regime's 3 % of the total.
	RetentionRate = 300
)

// DetractionCodeOption is a good or service of the SPOT system: ID is its SUNAT catalog 54 code
// (printed with three digits, 37 → "037") and Rate the share of the total deposited, in hundredths.
type DetractionCodeOption struct {
	ID   int16  `cb:"1"`
	Name string `cb:"2"`
	Rate int16  `cb:"3"`
}

// DetractionCodeOptions is what the product form's "Detracción (SPOT)" offers, via the //CATALOG:
// marker in frontend/services/production/products.svelte.ts: the annex 2 goods and annex 3 services a
// company most often sells, with the rates in force (Resolución 183-2004/SUNAT and its amendments). A
// sale pins the code and the rate, so a later change here never restates it.
var DetractionCodeOptions = []DetractionCodeOption{
	{ID: 1, Name: "001 Azúcar y melaza de caña", Rate: 1000},
	{ID: 4, Name: "004 Recursos hidrobiológicos", Rate: 400},
	{ID: 5, Name: "005 Maíz amarillo duro", Rate: 400},
	{ID: 8, Name: "008 Madera", Rate: 400},
	{ID: 9, Name: "009 Arena y piedra", Rate: 1000},
	{ID: 10, Name: "010 Residuos, subproductos, desechos", Rate: 1500},
	{ID: 12, Name: "012 Intermediación laboral y tercerización", Rate: 1200},
	{ID: 14, Name: "014 Carnes y despojos comestibles", Rate: 400},
	{ID: 19, Name: "019 Arrendamiento de bienes muebles", Rate: 1000},
	{ID: 20, Name: "020 Mantenimiento y reparación de bienes muebles", Rate: 1200},
	{ID: 21, Name: "021 Movimiento de carga", Rate: 1000},
	{ID: 22, Name: "022 Otros servicios empresariales", Rate: 1200},
	{ID: 24, Name: "024 Comisión mercantil", Rate: 1000},
	{ID: 25, Name: "025 Fabricación de bienes por encargo", Rate: 1000},
	{ID: 26, Name: "026 Servicio de transporte de personas", Rate: 1000},
	{ID: 27, Name: "027 Servicio de transporte de carga", Rate: 400},
	{ID: 30, Name: "030 Contratos de construcción", Rate: 400},
	{ID: 37, Name: "037 Demás servicios gravados con el IGV", Rate: 1200},
}

// DetractionCodeOf is the catalog entry of a SPOT code, nil when it is not in the catalog.
func DetractionCodeOf(code int16) *DetractionCodeOption {
	for index := range DetractionCodeOptions {
		if DetractionCodeOptions[index].ID == code {
			return &DetractionCodeOptions[index]
		}
	}
	return nil
}

// IgvBasisPointsOf is the IGV rate of a treatment in basis points: 1800, 1000, or 0 for the
// treatments that charge none (exonerado, inafecto, export).
func IgvBasisPointsOf(taxGroup int8) int64 {
	switch taxGroup {
	case TaxGroupTaxed18:
		return 1800
	case TaxGroupTaxed10:
		return 1000
	}
	return 0
}
