package hr

import (
	"encoding/json"
	"fmt"

	"app/core"
	"app/db"
	"app/hr/types"

	"golang.org/x/sync/errgroup"
)

// GetLegalParameters syncs the country's legal values and AFP rates, one keyless delta cursor each.
// Every company reads them (the payroll page shows the values in force); only the SaaS company writes.
func GetLegalParameters(req *core.HandlerArgs) core.HandlerResponse {
	parameters, afpRates := []types.LegalParameter{}, []types.AfpRate{}
	errGroup := errgroup.Group{}
	errGroup.Go(func() error {
		query := db.Query(&parameters)
		query.Select().CountryID.Equals(types.CountryIDPeru).Delta(req.GetDeltaCursor("Parameters"))
		return query.Exec()
	})
	errGroup.Go(func() error {
		query := db.Query(&afpRates)
		query.Select().CountryID.Equals(types.CountryIDPeru).Delta(req.GetDeltaCursor("AfpRates"))
		return query.Exec()
	})
	if err := errGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los parámetros laborales:", err)
	}
	return core.MakeResponse(req, &map[string]any{"Parameters": parameters, "AfpRates": afpRates})
}

// PostLegalParameter creates or edits one dated legal value (SaaS only); Status 0 deletes it. A key
// holds one value per ValidFrom, so a change of law is a new row, never an edit of the old one.
func PostLegalParameter(req *core.HandlerArgs) core.HandlerResponse {
	parameter := types.LegalParameter{}
	if err := json.Unmarshal([]byte(*req.Body), &parameter); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	stored, _, err := loadLegalRecords(types.CountryIDPeru)
	if err != nil {
		return req.MakeErr(err)
	}

	if parameter.Status != types.StatusDeleted || parameter.ID == 0 {
		if err := validateLegalParameter(parameter, stored); err != nil {
			return req.MakeErr(err)
		}
		parameter.Status = types.StatusActive
	}
	if parameter.ID > 0 && !legalParameterExists(stored, parameter.ID) {
		return req.MakeErr(fmt.Sprintf("El parámetro %d no existe.", parameter.ID))
	}

	parameter.CountryID, parameter.UpdatedBy = types.CountryIDPeru, req.User.ID
	records := []types.LegalParameter{parameter}
	if parameter.ID == 0 {
		err = db.Insert(&records)
	} else {
		table := db.TableOf[types.LegalParameter]()
		err = db.Update(&records, table.Key, table.ValidFrom, table.Value, table.Status, table.UpdatedBy)
	}
	if err != nil {
		return req.MakeErr("Error al guardar el parámetro laboral:", err)
	}
	return req.MakeResponse(records[0])
}

func validateLegalParameter(parameter types.LegalParameter, stored []types.LegalParameter) error {
	option, found := legalParameterOption(parameter.Key)
	if !found {
		return fmt.Errorf("El parámetro %d no es un parámetro laboral conocido.", parameter.Key)
	}
	if parameter.ValidFrom <= 0 {
		return fmt.Errorf("Indique la fecha de vigencia de %q.", option.Name)
	}
	if parameter.Value <= 0 {
		return fmt.Errorf("El valor de %q debe ser mayor a cero.", option.Name)
	}
	if option.Unit == "bp" && parameter.Value > types.RateScale {
		return fmt.Errorf("La tasa de %q no puede superar el 100%%.", option.Name)
	}
	for _, other := range stored {
		if other.ID != parameter.ID && other.Status == types.StatusActive &&
			other.Key == parameter.Key && other.ValidFrom == parameter.ValidFrom {
			return fmt.Errorf("Ya existe un valor de %q vigente desde esa fecha; edítelo en lugar de crear otro.", option.Name)
		}
	}
	return nil
}

// PostAfpRates saves the SBS table of one or more AFPs for a period (SaaS only). A row for an AFP
// and period that already exists is overwritten, so re-entering a month corrects it.
func PostAfpRates(req *core.HandlerArgs) core.HandlerResponse {
	afpRates := []types.AfpRate{}
	if err := json.Unmarshal([]byte(*req.Body), &afpRates); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if len(afpRates) == 0 {
		return req.MakeErr("No se enviaron tasas de AFP.")
	}
	_, stored, err := loadLegalRecords(types.CountryIDPeru)
	if err != nil {
		return req.MakeErr(err)
	}

	newRates, changedRates := []types.AfpRate{}, []types.AfpRate{}
	for _, afpRate := range afpRates {
		if afpRate.Status != types.StatusDeleted {
			if err := validateAfpRate(afpRate); err != nil {
				return req.MakeErr(err)
			}
			afpRate.Status = types.StatusActive
		}
		afpRate.ID = 0
		for _, storedRate := range stored {
			if storedRate.PensionSystem == afpRate.PensionSystem && storedRate.Period == afpRate.Period {
				afpRate.ID = storedRate.ID
			}
		}
		afpRate.CountryID, afpRate.UpdatedBy = types.CountryIDPeru, req.User.ID
		if afpRate.ID == 0 {
			if afpRate.Status == types.StatusDeleted {
				continue
			}
			newRates = append(newRates, afpRate)
		} else {
			changedRates = append(changedRates, afpRate)
		}
	}

	if len(newRates) > 0 {
		if err := db.Insert(&newRates); err != nil {
			return req.MakeErr("Error al guardar las tasas de AFP:", err)
		}
	}
	if len(changedRates) > 0 {
		table := db.TableOf[types.AfpRate]()
		if err := db.Update(&changedRates, table.FundRate, table.PremiumRate, table.FlowCommission,
			table.MixedCommission, table.Status, table.UpdatedBy); err != nil {
			return req.MakeErr("Error al guardar las tasas de AFP:", err)
		}
	}
	return req.MakeResponse(append(newRates, changedRates...))
}

func validateAfpRate(afpRate types.AfpRate) error {
	if !types.IsAfp(afpRate.PensionSystem) {
		return fmt.Errorf("El sistema de pensiones %d no es una AFP.", afpRate.PensionSystem)
	}
	month := afpRate.Period % 100
	if afpRate.Period <= 0 || month < 1 || month > 12 {
		return fmt.Errorf("El periodo %d no es válido (formato AAMM).", afpRate.Period)
	}
	for _, rate := range []int32{afpRate.FundRate, afpRate.PremiumRate, afpRate.FlowCommission, afpRate.MixedCommission} {
		if rate < 0 || rate > types.RateScale {
			return fmt.Errorf("Las tasas de AFP deben estar entre 0%% y 100%%.")
		}
	}
	if afpRate.FundRate == 0 || afpRate.PremiumRate == 0 {
		return fmt.Errorf("Indique el aporte al fondo y la prima de seguro de la AFP.")
	}
	return nil
}

// loadLegalRecords reads every legal value and AFP rate of a country; both lists are small (a few
// rows per year), so the payroll and the validations read them whole.
func loadLegalRecords(countryID int32) ([]types.LegalParameter, []types.AfpRate, error) {
	parameters, afpRates := []types.LegalParameter{}, []types.AfpRate{}
	parameterQuery := db.Query(&parameters)
	parameterQuery.CountryID.Equals(countryID)
	if err := parameterQuery.Exec(); err != nil {
		return nil, nil, fmt.Errorf("error al leer los parámetros laborales: %v", err)
	}
	afpQuery := db.Query(&afpRates)
	afpQuery.CountryID.Equals(countryID)
	if err := afpQuery.Exec(); err != nil {
		return nil, nil, fmt.Errorf("error al leer las tasas de AFP: %v", err)
	}
	return parameters, afpRates, nil
}

// loadLegalValues builds the dated lookup the payroll engine reads.
func loadLegalValues(countryID int32) (types.LegalValues, error) {
	parameters, afpRates, err := loadLegalRecords(countryID)
	if err != nil {
		return types.LegalValues{}, err
	}
	return types.NewLegalValues(parameters, afpRates), nil
}

func legalParameterOption(key int8) (types.LegalParameterOption, bool) {
	for _, option := range types.LegalParameterOptions {
		if option.ID == key {
			return option, true
		}
	}
	return types.LegalParameterOption{}, false
}

func legalParameterExists(stored []types.LegalParameter, id int32) bool {
	for _, parameter := range stored {
		if parameter.ID == id {
			return true
		}
	}
	return false
}
