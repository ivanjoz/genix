package cloud

import (
	config "app/config/types"
	"app/core"
	sales "app/sales/types"
	"encoding/json"
	"fmt"
	"slices"
)

// PublishCompanyPub writes the public company file, empresas/e-{companyID}.json, which the
// storefront reads without a session: the name, the Culqi public key of the current mode, the
// pickup stores and the [ecommerce] flags. It lives in cloud because both the company save
// (config) and the Sede save (business) rewrite it, and neither module body may import the other.
func PublishCompanyPub(companyID int32) error {
	company, err := readCompanyForConfig(companyID)
	if err != nil {
		return err
	}
	sites, err := readSitesForConfig(companyID)
	if err != nil {
		return err
	}

	companyPub := config.CompanyPub{
		ID:            company.ID,
		Name:          company.Name,
		CulqiLlave:    company.CulqiConfig.PublicKey(),
		CulqiRsaKey:   company.CulqiConfig.RsaKey,
		CulqiRsaKeyID: company.CulqiConfig.RsaKeyID,
		YapeEnabled:   company.CulqiConfig.YapeEnabled,
		PickupSites:   []config.CompanyPubSite{},
		Flags:         []int16{},
	}
	for _, site := range sites {
		if site.IsPickupPoint && site.Status > 0 {
			companyPub.PickupSites = append(companyPub.PickupSites, config.CompanyPubSite{
				ID: site.ID, Name: site.Name, Address: site.Address, CityID: site.CityID,
			})
		}
	}
	for _, flagID := range company.Flags {
		if slices.Contains(sales.EcommerceCompanyFlags, flagID) {
			companyPub.Flags = append(companyPub.Flags, flagID)
		}
	}

	companyPubBytes, err := json.Marshal(companyPub)
	if err != nil {
		return fmt.Errorf("no se pudo serializar el archivo público de la empresa: %w", err)
	}
	core.Log("PublishCompanyPub::", string(companyPubBytes))
	return SaveFile(SaveFileArgs{
		Bucket:      core.Env.S3_BUCKET,
		Path:        "empresas",
		FileContent: companyPubBytes,
		Name:        fmt.Sprintf("e-%v.json", companyID),
	})
}
