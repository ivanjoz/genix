package types

import "app/db"

// Concept is a company-defined earning or deduction (a productivity bonus, movilidad, a judicial
// withholding…) with the same treatment flags as the legal concepts (legal_concepts.go). It is paid
// through recurring employee concepts or a period input; the payslip line stamps its flags and
// accounts, so editing a concept never rewrites a closed month.
type Concept struct {
	db.TableStruct[ConceptTable, Concept]
	CompanyID             int32  `cb:"1"`
	ID                    int32  `cb:"2"`
	Name                  string `cb:"3"`
	Kind                  int8   `cb:"4"` // ConceptKindEarning or ConceptKindDeduction
	PlameCode             string `cb:"5"`
	IsRemunerative        bool   `cb:"6"`
	AffectsEssalud        bool   `cb:"7"`
	AffectsPension        bool   `cb:"8"`
	IncomeTax             int8   `cb:"9"`
	ComputableForBenefits bool   `cb:"10"`
	ExpenseAccountCode    int32  `cb:"11"`
	PayableAccountCode    int32  `cb:"12"`
	Status                int8   `json:"ss" cb:"ss,13"`
	Updated               int32  `json:"upd" cb:"upd,14"`
	Version               int16  `json:"ver" cb:"ver,15"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,16"`
	UpdatedBy int32   `cb:"17"`
	Created   int32   `cb:"18"`
	CreatedBy int32   `cb:"19"`
}

type ConceptTable struct {
	db.TableStruct[ConceptTable, Concept]
	CompanyID             db.Col[*ConceptTable, int32]
	ID                    db.Col[*ConceptTable, int32]
	Name                  db.Col[*ConceptTable, string]
	Kind                  db.Col[*ConceptTable, int8]
	PlameCode             db.Col[*ConceptTable, string]
	IsRemunerative        db.Col[*ConceptTable, bool]
	AffectsEssalud        db.Col[*ConceptTable, bool]
	AffectsPension        db.Col[*ConceptTable, bool]
	IncomeTax             db.Col[*ConceptTable, int8]
	ComputableForBenefits db.Col[*ConceptTable, bool]
	ExpenseAccountCode    db.Col[*ConceptTable, int32]
	PayableAccountCode    db.Col[*ConceptTable, int32]
	Status                db.Col[*ConceptTable, int8]
	Updated               db.Col[*ConceptTable, int32]
	Version               db.Col[*ConceptTable, int16]
	UpdatedBy             db.Col[*ConceptTable, int32]
	Created               db.Col[*ConceptTable, int32]
	CreatedBy             db.Col[*ConceptTable, int32]
}

func (e ConceptTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          74,
		Name:        "hr_concepts",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}

func (concept Concept) Treatment() ConceptTreatment {
	return ConceptTreatment{Kind: concept.Kind, PlameCode: concept.PlameCode, IsRemunerative: concept.IsRemunerative,
		AffectsEssalud: concept.AffectsEssalud, AffectsPension: concept.AffectsPension, IncomeTax: concept.IncomeTax,
		ComputableForBenefits: concept.ComputableForBenefits, ExpenseAccountCode: concept.ExpenseAccountCode,
		PayableAccountCode: concept.PayableAccountCode}
}
