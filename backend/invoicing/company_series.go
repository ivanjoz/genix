package invoicing

import (
	"app/cloud"
	config "app/config/types"
	"app/core"
	"app/db"
	"app/invoicing/types"
	"errors"
	"fmt"
)

// The series a company issues under travel inline on its record, so reading them
// is one read of one row — it replaced a full scan of a series table on every
// emission.
//
// The mirror branch below is not optional: companies live in Scylla only when the
// cloud mirror is off. It is duplicated from `config` rather than shared because a
// module body may not import another module body, and a lookup that reaches for
// `cloud` cannot live in `config/types` either — the cloud package's own tests
// read Company, so that import would close a cycle.

// loadCompanyRecord reads the company row, from whichever store holds it.
func loadCompanyRecord(companyID int32) (*config.Company, error) {
	var company *config.Company
	var err error

	if cloud.IsDataMirrorEnabled() {
		company, err = cloud.GetByID(config.Company{ID: companyID})
	} else {
		companies := []config.Company{}
		query := db.Query(&companies)
		query.ID.Equals(companyID).Limit(1)
		if err = query.Exec(); err == nil && len(companies) > 0 {
			company = &companies[0]
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error al leer la empresa: %w", err)
	}
	if company == nil {
		return nil, errors.New("no se encontró la empresa")
	}
	return company, nil
}

// LoadCompanySeries reads the series a company may issue under.
func LoadCompanySeries(companyID int32) ([]types.InvoiceSeries, error) {
	company, err := loadCompanyRecord(companyID)
	if err != nil {
		return nil, err
	}
	return company.InvoiceSeries, nil
}

// SaveCompanySeries replaces the whole set of series on a company.
//
// The set is validated as a whole because that is where the rules live: ids and
// codes have to be unique across it, and only one series per document type can be
// the default.
func SaveCompanySeries(companyID int32, allSeries []types.InvoiceSeries) error {
	company, err := loadCompanyRecord(companyID)
	if err != nil {
		return err
	}
	if err := types.ValidateSeries(allSeries); err != nil {
		return err
	}

	company.InvoiceSeries = allSeries
	company.Updated = core.SUnixTime()
	companies := &[]config.Company{*company}
	if err := db.Insert(companies); err != nil {
		return fmt.Errorf("error al guardar las series: %w", err)
	}
	if cloud.IsDataMirrorEnabled() {
		if err := cloud.Insert([]config.Company{(*companies)[0]}); err != nil {
			return fmt.Errorf("error al guardar las series en cloud: %w", err)
		}
	}
	return nil
}
