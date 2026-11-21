package types

import (
	sales "app/sales/types"
	"strings"
	"testing"

	"github.com/ivanjoz/facturago"
	"github.com/ivanjoz/facturago/model"
)

// Only a factura still owing money with a due date after the issue date is Crédito; anything
// else stays Contado, since SUNAT rejects a cuota due on or before the issue date.
func TestCreditTermsForSale(t *testing.T) {
	const issueDate = int16(20778)
	owingSale := &sales.SaleOrder{DebtAmount: 23_600, PaymentDueDate: issueDate + 30}

	cases := []struct {
		name    string
		order   *sales.SaleOrder
		docType int8
		want    DocumentCreditTerms
	}{
		{"factura owing, due later", owingSale, DocTypeFactura, DocumentCreditTerms{23_600, issueDate + 30}},
		{"boleta owing", owingSale, DocTypeBoleta, DocumentCreditTerms{}},
		{"paid", &sales.SaleOrder{PaymentDueDate: issueDate + 30}, DocTypeFactura, DocumentCreditTerms{}},
		{"due the issue day", &sales.SaleOrder{DebtAmount: 100, PaymentDueDate: issueDate}, DocTypeFactura, DocumentCreditTerms{}},
		{"no due date", &sales.SaleOrder{DebtAmount: 100}, DocTypeFactura, DocumentCreditTerms{}},
	}
	for _, testCase := range cases {
		if got := CreditTermsForSale(testCase.order, testCase.docType, issueDate); got != testCase.want {
			t.Errorf("%s: got %+v, want %+v", testCase.name, got, testCase.want)
		}
	}
}

// A credit factura validates and serializes with its one cuota on the sale's due date, printed as
// that same calendar day in Peru.
func TestCreditFacturaSerializesItsCuota(t *testing.T) {
	document := testDocument()
	// 2026-12-21 is UnixDay 20808.
	applyCreditTerms(document, DocumentCreditTerms{PendingAmount: 23_600, PaymentDueDate: 20808})
	if err := model.CompleteTotals(document); err != nil {
		t.Fatalf("CompleteTotals: %v", err)
	}
	document.Correlativo = 123
	if problems := model.ValidateDocument(document); len(problems) > 0 {
		t.Fatalf("the credit document is not valid: %v", problems)
	}

	issuer := model.Issuer{
		RUC: "20000000001", LegalName: "EMPRESA DEMO S.A.C.",
		Address: model.Address{Ubigeo: "150101", Line: "AV. LIMA 100"},
	}
	xmlBytes, err := facturago.BuildXML(issuer, document)
	if err != nil {
		t.Fatalf("the credit document does not serialize: %v", err)
	}
	xmlText := string(xmlBytes)
	for _, expected := range []string{"Credito", "Cuota001", "<cbc:PaymentDueDate>2026-12-21</cbc:PaymentDueDate>"} {
		if !strings.Contains(xmlText, expected) {
			t.Errorf("the XML does not contain %q", expected)
		}
	}
}
