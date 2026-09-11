package types

import "testing"

func TestValidateCustomerIdentityFactura(t *testing.T) {
	cases := []struct {
		name           string
		customerName   string
		registryNumber string
		wantError      bool
	}{
		{"anonymous", "", "", true},
		{"named without RUC", "JUAN PEREZ", "", true},
		{"DNI instead of RUC", "JUAN PEREZ", "12345678", true},
		{"RUC without name", "", "20123456789", true},
		{"eleven non-digits", "EMPRESA SAC", "2012345678A", true},
		{"RUC and name", "EMPRESA SAC", "20123456789", false},
		{"padded RUC and name", "  EMPRESA SAC ", " 20123456789 ", false},
	}

	for _, testCase := range cases {
		err := ValidateCustomerIdentity(DocTypeFactura, 1000, testCase.customerName, testCase.registryNumber)
		if (err != nil) != testCase.wantError {
			t.Errorf("%v: got %v, wantError %v", testCase.name, err, testCase.wantError)
		}
	}
}

// Below the threshold a boleta takes anybody, including nobody. From it, SUNAT
// wants the buyer named.
func TestValidateCustomerIdentityBoletaThreshold(t *testing.T) {
	if err := ValidateCustomerIdentity(DocTypeBoleta, BoletaIdentifiedFrom-1, "", ""); err != nil {
		t.Errorf("a small anonymous boleta was rejected: %v", err)
	}
	if err := ValidateCustomerIdentity(DocTypeBoleta, BoletaIdentifiedFrom, "", ""); err == nil {
		t.Error("an anonymous boleta of S/ 700 was accepted")
	}
	if err := ValidateCustomerIdentity(DocTypeBoleta, BoletaIdentifiedFrom, "", "12345678"); err == nil {
		t.Error("a boleta of S/ 700 was accepted without a name")
	}
	if err := ValidateCustomerIdentity(DocTypeBoleta, BoletaIdentifiedFrom, "JUAN PEREZ", "1234567"); err == nil {
		t.Error("a boleta of S/ 700 was accepted with a 7-character document")
	}
	if err := ValidateCustomerIdentity(DocTypeBoleta, BoletaIdentifiedFrom, "JUAN PEREZ", "12345678"); err != nil {
		t.Errorf("a boleta of S/ 700 with a DNI was rejected: %v", err)
	}
	// A foreign document has no fixed length, so anything from eight characters up
	// identifies the buyer well enough for a boleta.
	if err := ValidateCustomerIdentity(DocTypeBoleta, BoletaIdentifiedFrom, "JOHN SMITH", "X1234567AB"); err != nil {
		t.Errorf("a boleta of S/ 700 with a foreign document was rejected: %v", err)
	}
}

// A note inherits the customer of the document it corrects, so it is not checked
// against a buyer it never carried on its own.
func TestValidateCustomerIdentitySkipsNotes(t *testing.T) {
	for _, docType := range []int8{DocTypeCreditNote, DocTypeDebitNote} {
		if err := ValidateCustomerIdentity(docType, 500_000, "", ""); err != nil {
			t.Errorf("doc type %v was validated as a sale document: %v", docType, err)
		}
	}
}
