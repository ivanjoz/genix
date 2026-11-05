package hr

import (
	"encoding/json"
	"fmt"
	"strings"

	accounting "app/accounting/types"
	"app/core"
	"app/db"
	"app/hr/types"

	"golang.org/x/sync/errgroup"
)

// GetPayrollConfig syncs the company concepts and the payroll runs in one response, one cursor each.
func GetPayrollConfig(req *core.HandlerArgs) core.HandlerResponse {
	concepts, runs := []types.Concept{}, []types.PayrollRun{}
	errGroup := errgroup.Group{}
	errGroup.Go(func() error {
		query := db.Query(&concepts)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Concepts"), types.StatusActive)
		return query.Exec()
	})
	errGroup.Go(func() error {
		query := db.Query(&runs)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("Runs"),
			types.PayrollStatusOpen, types.PayrollStatusClosed, types.PayrollStatusPaid)
		return query.Exec()
	})
	if err := errGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener la configuración de planilla:", err)
	}
	return core.MakeResponse(req, &map[string]any{"Concepts": concepts, "Runs": runs})
}

// PostConcept creates or edits a company concept; Status 0 deletes it, refused while an employee
// still has it as a recurring concept. Closed payslips keep the treatment they stamped.
func PostConcept(req *core.HandlerArgs) core.HandlerResponse {
	concept := types.Concept{}
	if err := json.Unmarshal([]byte(*req.Body), &concept); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	companyID := req.User.CompanyID
	concept.Name = strings.TrimSpace(concept.Name)

	if concept.Status == types.StatusDeleted && concept.ID > 0 {
		recurring, err := loadCompanyEmployeeConcepts(companyID)
		if err != nil {
			return req.MakeErr(err)
		}
		for _, employeeConcept := range recurring {
			if employeeConcept.ConceptID == concept.ID && employeeConcept.Status == types.StatusActive {
				return req.MakeErr("El concepto está asignado a trabajadores; quíteselo antes de eliminarlo.")
			}
		}
	} else {
		if err := validateConcept(&concept); err != nil {
			return req.MakeErr(err)
		}
		concept.Status = types.StatusActive
	}

	concept.CompanyID, concept.UpdatedBy = companyID, req.User.ID
	records := []types.Concept{concept}
	if concept.ID == 0 {
		records[0].Created, records[0].CreatedBy = core.SUnixTime(), req.User.ID
		if err := db.Insert(&records); err != nil {
			return req.MakeErr("Error al guardar el concepto:", err)
		}
		return req.MakeResponse(records[0])
	}
	if _, err := loadConcept(companyID, concept.ID); err != nil {
		return req.MakeErr(err)
	}
	table := db.TableOf[types.Concept]()
	if err := db.Update(&records, table.Name, table.Kind, table.PlameCode, table.IsRemunerative, table.AffectsEssalud,
		table.AffectsPension, table.IncomeTax, table.ComputableForBenefits, table.ExpenseAccountCode,
		table.PayableAccountCode, table.Status, table.UpdatedBy); err != nil {
		return req.MakeErr("Error al guardar el concepto:", err)
	}
	return req.MakeResponse(records[0])
}

// validateConcept enforces the treatment rules a company concept can't break: a non-remunerative
// concept pays no contributions, an earning posts to a 62x expense, a deduction to the payable owed.
func validateConcept(concept *types.Concept) error {
	if len(concept.Name) < 2 {
		return fmt.Errorf("El nombre del concepto debe tener al menos 2 caracteres.")
	}
	// The PLAME file refuses codes outside SUNAT table 22, and a code from another chapter
	// (an earning filed under a deduction code) would report the line in the wrong column.
	plameCode, isKnownPlameCode := types.FindPlameCode(concept.PlameCode)
	if !isKnownPlameCode {
		return fmt.Errorf("El código PLAME %q no existe en la tabla 22 de SUNAT.", concept.PlameCode)
	}
	if plameCode.Kind != concept.Kind {
		return fmt.Errorf("El código PLAME %s (%s) no corresponde al tipo del concepto.", plameCode.Code, plameCode.Name)
	}
	if !types.HasOption(types.IncomeTaxOptions, concept.IncomeTax) {
		return fmt.Errorf("Indique cómo afecta el concepto a la renta de quinta categoría.")
	}
	switch concept.Kind {
	case types.ConceptKindEarning:
		if !concept.IsRemunerative && (concept.AffectsEssalud || concept.AffectsPension || concept.ComputableForBenefits) {
			return fmt.Errorf("Un ingreso no remunerativo no aporta a EsSalud ni a pensiones, ni es computable para beneficios.")
		}
		if concept.ExpenseAccountCode == 0 || !strings.HasPrefix(fmt.Sprint(concept.ExpenseAccountCode), "62") {
			return fmt.Errorf("Un ingreso debe registrarse en una cuenta de gastos de personal (62).")
		}
		if err := accounting.ValidatePCGEAccount(accounting.PCGEKindExpense, concept.ExpenseAccountCode); err != nil {
			return err
		}
		concept.PayableAccountCode = 0
	case types.ConceptKindDeduction:
		if concept.PayableAccountCode == 0 {
			return fmt.Errorf("Indique la cuenta por pagar del descuento (p.ej. 419 para retenciones judiciales).")
		}
		account, exists := accounting.FindPCGEAccount(concept.PayableAccountCode)
		if !exists || !account.Postable {
			return fmt.Errorf("La cuenta %d no es una cuenta de detalle del plan contable.", concept.PayableAccountCode)
		}
		// A deduction carries no treatment of its own: it lowers the net pay, not the bases.
		concept.IsRemunerative, concept.AffectsEssalud, concept.AffectsPension = false, false, false
		concept.ComputableForBenefits, concept.IncomeTax, concept.ExpenseAccountCode = false, types.IncomeTaxNone, 0
	default:
		return fmt.Errorf("El concepto debe ser un ingreso o un descuento.")
	}
	return nil
}

func loadConcept(companyID, conceptID int32) (types.Concept, error) {
	concepts := []types.Concept{}
	query := db.Query(&concepts)
	query.CompanyID.Equals(companyID).ID.Equals(conceptID)
	if err := query.Exec(); err != nil {
		return types.Concept{}, fmt.Errorf("error al leer el concepto: %v", err)
	}
	if len(concepts) == 0 || concepts[0].Status == types.StatusDeleted {
		return types.Concept{}, fmt.Errorf("el concepto %d no existe", conceptID)
	}
	return concepts[0], nil
}

// loadCompanyConcepts reads every concept of the company, keyed by ID, deleted ones included: an
// old recurring item may still point at one.
func loadCompanyConcepts(companyID int32) (map[int32]types.Concept, error) {
	concepts := []types.Concept{}
	query := db.Query(&concepts)
	query.CompanyID.Equals(companyID)
	if err := query.Exec(); err != nil {
		return nil, fmt.Errorf("error al leer los conceptos: %v", err)
	}
	conceptsByID := map[int32]types.Concept{}
	for _, concept := range concepts {
		conceptsByID[concept.ID] = concept
	}
	return conceptsByID, nil
}
