package invoicing

import (
	config "app/config/types"
	"testing"
)

func fiscalCompany() config.CompanyConfigCompany {
	return config.CompanyConfigCompany{
		RUC: "20000000001", LegalName: "FRUTAS DEL SUR S.A.C.",
		Address: "Av. Siempre Viva 742", CityID: 150101,
		District: "LIMA", Province: "LIMA", Department: "LIMA",
	}
}

// The four observations SUNAT returned on the first documents this system emitted
// were all about this block: an invalid ubigeo and three empty names.
func TestFiscalAddressCarriesEverythingSunatValidates(t *testing.T) {
	address, err := fiscalAddress(fiscalCompany())
	if err != nil {
		t.Fatalf("fiscalAddress: %v", err)
	}

	if address.Ubigeo != "150101" {
		t.Errorf("ubigeo = %q, want 150101", address.Ubigeo)
	}
	if address.District != "LIMA" || address.Province != "LIMA" || address.Department != "LIMA" {
		t.Errorf("the names did not travel: %+v", address)
	}
	if address.Line != "Av. Siempre Viva 742" {
		t.Errorf("line = %q", address.Line)
	}
	// 0000 is the main establishment, which is what makes this the company's own
	// address and not a branch's.
	if address.AnnexCode != defaultAnnexCode {
		t.Errorf("annex code = %q, want %q", address.AnnexCode, defaultAnnexCode)
	}
}

func TestFiscalAddressRefusesACompanyWithoutACity(t *testing.T) {
	company := fiscalCompany()
	company.CityID = 0

	if _, err := fiscalAddress(company); err == nil {
		t.Error("a company with no district was allowed to issue")
	}
}

// A ubigeo whose names did not resolve means the blob was built against a catalog
// that no longer has that district. Emitting it would earn the same observations.
func TestFiscalAddressRefusesAnUnresolvedUbigeo(t *testing.T) {
	company := fiscalCompany()
	company.Province = ""

	if _, err := fiscalAddress(company); err == nil {
		t.Error("an address with no province was accepted")
	}
}
