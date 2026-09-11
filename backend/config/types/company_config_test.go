package types

import (
	business "app/business/types"
	"app/core"
	invoicing "app/invoicing/types"
	"bytes"
	"testing"

	"github.com/ivanjoz/colbin"
)

func testCompany() *Company {
	return &Company{
		ID:                42,
		Name:              "Frutas del Sur",
		LegalName:         "FRUTAS DEL SUR S.A.C.",
		RUC:               "20000000001",
		Email:             "ventas@frutas.pe",
		NotificationEmail: "avisos@frutas.pe",
		Phone:             "987654321",
		Address:           "Av. Siempre Viva 742",
		CityID:            150101,
		Updated:           1000,
		CulqiConfig: CulqiConfig{
			RsaKey: "rsa", RsaKeyID: "rsa-id", KeyLive: "sk_live", KeyDev: "sk_test",
			PubKeyLive: "pk_live", PubKeyDev: "pk_test",
		},
		InvoiceSeries: []invoicing.InvoiceSeries{
			{SeriesID: 1, DocType: 1, SeriesCode: "F001", IsDefault: 1, Status: 1},
		},
	}
}

// The ubigeo catalog is keyed by the code itself, so a district contains its
// parents: 150101 sits under 1501 under 15.
func peruCities() []business.CityLocation {
	return []business.CityLocation{
		{ID: 15, Name: "LIMA", Hierarchy: 1},
		{ID: 1501, Name: "LIMA", ParentID: 15, Hierarchy: 2},
		{ID: 150101, Name: "LIMA", ParentID: 1501, Hierarchy: 3},
		{ID: 150102, Name: "ANCON", ParentID: 1501, Hierarchy: 3},
	}
}

func TestAssembleCompanyConfigMapsTheCompany(t *testing.T) {
	config, err := AssembleCompanyConfig(testCompany(), nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}

	if config.Version != CompanyConfigVersion || config.CompanyID != 42 {
		t.Fatalf("unexpected header: %+v", config)
	}
	if config.Company.RUC != "20000000001" || config.Company.TradeName != "Frutas del Sur" {
		t.Fatalf("unexpected company: %+v", config.Company)
	}
	if len(config.Sunat.Series) != 1 || config.Sunat.Series[0].SeriesCode != "F001" {
		t.Fatalf("the series did not travel: %+v", config.Sunat.Series)
	}
	if config.GeneratedAt == 0 {
		t.Error("GeneratedAt was not stamped")
	}
}

// The public Culqi keys are the browser's half. A secret store holding what is
// already public is a liability with no upside, so they must not be copied.
func TestAssembleCompanyConfigDropsThePublicCulqiKeys(t *testing.T) {
	config, err := AssembleCompanyConfig(testCompany(), nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}

	if config.Culqi.KeyLive != "sk_live" || config.Culqi.RsaKeyID != "rsa-id" {
		t.Fatalf("the backend keys did not travel: %+v", config.Culqi)
	}
	marshalled, err := colbin.Marshal(config)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, public := range []string{"pk_live", "pk_test"} {
		if bytes.Contains(marshalled, []byte(public)) {
			t.Errorf("the public key %q reached the blob", public)
		}
	}
}

// SUNAT observes a document whose issuer address is missing the ubigeo or any of
// the three names, so the blob has to carry them resolved.
func TestAssembleCompanyConfigResolvesTheFiscalAddress(t *testing.T) {
	config, err := AssembleCompanyConfig(testCompany(), nil, nil, nil, peruCities())
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}

	if config.Company.Ubigeo() != "150101" {
		t.Errorf("ubigeo = %q, want 150101", config.Company.Ubigeo())
	}
	if config.Company.District != "LIMA" || config.Company.Province != "LIMA" ||
		config.Company.Department != "LIMA" {
		t.Errorf("the address names did not resolve: %+v", config.Company)
	}
}

// A department's code is two digits, so the ubigeo has to be padded back to six
// rather than printed as the integer the catalog stores.
func TestCompanyUbigeoIsPaddedAndEmptyWithoutACity(t *testing.T) {
	company := CompanyConfigCompany{CityID: 10101}
	if company.Ubigeo() != "010101" {
		t.Errorf("ubigeo = %q, want 010101", company.Ubigeo())
	}
	if (CompanyConfigCompany{}).Ubigeo() != "" {
		t.Error("a company with no city produced a ubigeo")
	}
}

// A company that has not picked its district yet still has a config worth caching:
// it just cannot issue, which is what the issuer says when it reads this.
func TestAssembleCompanyConfigToleratesACompanyWithoutACity(t *testing.T) {
	company := testCompany()
	company.CityID = 0

	config, err := AssembleCompanyConfig(company, nil, nil, nil, peruCities())
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}
	if config.Company.District != "" || config.Company.Ubigeo() != "" {
		t.Errorf("an address appeared out of nowhere: %+v", config.Company)
	}
}

func TestAssembleCompanyConfigFallsBackToTheTradeName(t *testing.T) {
	company := testCompany()
	company.LegalName = ""

	config, err := AssembleCompanyConfig(company, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}
	if config.Company.LegalName != company.Name {
		t.Fatalf("expected the trade name as a fallback, got %q", config.Company.LegalName)
	}
}

func TestAssembleCompanyConfigKeepsOnlyActiveParameters(t *testing.T) {
	parameters := []Parameters{
		{Group: 1, Key: "moneda", Value: "PEN", Status: 1, Updated: 1200},
		{Group: 1, Key: "retirado", Value: "x", Status: 0, Updated: 9999},
		{Group: 2, Key: "igv", ValueInt: 18, Values: []int32{18}, Status: 1},
	}

	config, err := AssembleCompanyConfig(testCompany(), nil, parameters, nil, nil)
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}

	if len(config.Parameters) != 2 {
		t.Fatalf("expected 2 active parameters, got %v", len(config.Parameters))
	}
	// The retired row must not raise the watermark either, or a deletion would look
	// like the freshest thing in the blob.
	if config.SourceUpdated != 1200 {
		t.Fatalf("expected SourceUpdated 1200, got %v", config.SourceUpdated)
	}
}

func TestAssembleCompanyConfigRefusesAnEmptyCompany(t *testing.T) {
	if _, err := AssembleCompanyConfig(nil, nil, nil, nil, nil); err == nil {
		t.Fatal("a nil company was accepted")
	}
}

// A company with no SUNAT credentials still has a configuration worth caching.
func TestAssembleCompanyConfigToleratesMissingSecrets(t *testing.T) {
	config, err := AssembleCompanyConfig(testCompany(), nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}
	if config.Sunat.SolUser != "" || len(config.Sunat.CertificateDER) != 0 {
		t.Fatalf("credentials appeared out of nowhere: %+v", config.Sunat)
	}
}

func TestAssembleCompanyConfigCarriesTheSolCredentials(t *testing.T) {
	if core.Env == nil {
		core.Env = &core.EnvStruct{}
	}
	previous := core.Env.SECRET_PHRASE
	core.Env.SECRET_PHRASE = "una-frase-secreta-de-prueba-larga"
	defer func() { core.Env.SECRET_PHRASE = previous }()

	solPassword, err := core.Encrypt([]byte("clave-sol"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	secrets := &invoicing.CompanySecrets{
		SolUser: "USUARIO01", SolPasswordEnc: solPassword,
		Environment: invoicing.SunatEnvProduction, Status: 1, Updated: 5000,
	}

	config, err := AssembleCompanyConfig(testCompany(), secrets, nil, nil, nil)
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}
	if config.Sunat.SolUser != "USUARIO01" || config.Sunat.SolPassword != "clave-sol" {
		t.Fatalf("the SOL credentials did not travel: %+v", config.Sunat)
	}
	if config.Sunat.Environment != invoicing.SunatEnvProduction {
		t.Errorf("the environment did not travel: %v", config.Sunat.Environment)
	}
	if config.SourceUpdated != 5000 {
		t.Errorf("the secrets did not raise the watermark: %v", config.SourceUpdated)
	}
}

// The blob is decoded by whatever build reads it, so the numbered ids have to
// survive a round trip through colbin unchanged.
func TestCompanyConfigSurvivesAColbinRoundTrip(t *testing.T) {
	original, err := AssembleCompanyConfig(testCompany(), nil, []Parameters{
		{Group: 2, Key: "igv", ValueInt: 18, Values: []int32{18, 10}, Status: 1},
	}, []business.Site{
		{ID: 3, Name: "Tienda Centro", Address: "Jr. Union 100", CityID: 150101, Status: 1},
	}, peruCities())
	if err != nil {
		t.Fatalf("AssembleCompanyConfig: %v", err)
	}
	original.Sunat.PrivateKeyDER = []byte{1, 2, 3, 4}
	original.Sunat.CertificateDER = []byte{5, 6, 7, 8}

	payload, err := colbin.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := CompanyConfig{}
	if err := colbin.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.CompanyID != original.CompanyID || decoded.Version != original.Version {
		t.Fatalf("header changed: %+v", decoded)
	}
	if decoded.Company != original.Company {
		t.Fatalf("company changed: %+v vs %+v", decoded.Company, original.Company)
	}
	if decoded.Culqi != original.Culqi {
		t.Fatalf("culqi changed: %+v vs %+v", decoded.Culqi, original.Culqi)
	}
	if len(decoded.Parameters) != 1 || decoded.Parameters[0].Key != "igv" ||
		len(decoded.Parameters[0].Values) != 2 {
		t.Fatalf("parameters changed: %+v", decoded.Parameters)
	}
	if len(decoded.Sunat.Series) != 1 || decoded.Sunat.Series[0].SeriesCode != "F001" {
		t.Fatalf("series changed: %+v", decoded.Sunat.Series)
	}
	// The ubigeo is what a document declares as the place it was issued from, so a
	// site that does not survive the round trip is a document that cannot be built.
	if len(decoded.Sites) != 1 || decoded.Sites[0] != original.Sites[0] {
		t.Fatalf("sites changed: %+v vs %+v", decoded.Sites, original.Sites)
	}
	if string(decoded.Sunat.PrivateKeyDER) != string(original.Sunat.PrivateKeyDER) ||
		string(decoded.Sunat.CertificateDER) != string(original.Sunat.CertificateDER) {
		t.Fatalf("the signing material changed: %+v", decoded.Sunat)
	}
}
