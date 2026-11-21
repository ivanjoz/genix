package types

import "testing"

func TestValidateCompanyTaxSetup(t *testing.T) {
	general := Company{TaxRegime: TaxRegimeGeneral, IncomeTaxPrepaymentRate: 150, NrusCategory: NrusCategory2}
	if err := ValidateCompanyTaxSetup(&general); err != nil {
		t.Fatalf("Régimen General at 1.5 %% should be valid: %v", err)
	}
	if general.NrusCategory != 0 {
		t.Error("a non-NRUS company kept an NRUS category")
	}

	nrus := Company{TaxRegime: TaxRegimeNrus, IncomeTaxPrepaymentRate: 150, NrusCategory: NrusCategory1}
	if err := ValidateCompanyTaxSetup(&nrus); err != nil {
		t.Fatalf("NRUS category 1 should be valid: %v", err)
	}
	if nrus.IncomeTaxPrepaymentRate != 0 {
		t.Error("an NRUS company kept a renta prepayment rate")
	}

	for _, invalid := range []Company{
		{TaxRegime: TaxRegimeNrus}, // no category
		{TaxRegime: 5},             // unknown regime
		{TaxRegime: TaxRegimeMype, IncomeTaxPrepaymentRate: 10001}, // over 100 %
	} {
		if ValidateCompanyTaxSetup(&invalid) == nil {
			t.Errorf("%+v was accepted", invalid)
		}
	}

	// Not set up yet is allowed: the forecast warns about it instead.
	if err := ValidateCompanyTaxSetup(&Company{}); err != nil {
		t.Errorf("a company without a regime was rejected: %v", err)
	}
}
