package types

// ExpenseCategoryOption is one expense category. DefaultAccountCode is the PCGE account an
// expense of the category is classified under when the user does not pick one. It is 0 where no
// single account is right: "Taxes" can be an IGV payment (40111) or a cost (64x), and "Other"
// says nothing. Those expenses stay unclassified until the user picks an account.
//
// The codes are repeated here because finance cannot import the chart (accounting/types imports
// finance/types). TestExpenseCategoryDefaultsAreAllowed keeps each one a valid expense account.
type ExpenseCategoryOption struct {
	ID                 int8   `cb:"1"`
	Label              string `cb:"2"`
	DefaultAccountCode int32  `cb:"3"`
}

// ExpenseCategoryOptions is what the backend validates CategoryID against and — via the
// //CATALOG: marker in frontend/routes/finance/expenses/expenses.svelte.ts — what the forms offer.
// The ids are persisted, so they must not be renumbered.
var ExpenseCategoryOptions = []ExpenseCategoryOption{
	{ID: 1, Label: "Rent|Alquiler", DefaultAccountCode: 6352},                                 // Alquileres — edificaciones
	{ID: 2, Label: "Utilities|Servicios", DefaultAccountCode: 6361},                           // Energía eléctrica
	{ID: 3, Label: "Payroll|Planilla", DefaultAccountCode: 6211},                              // Sueldos y salarios
	{ID: 4, Label: "Supplies|Insumos", DefaultAccountCode: 656},                               // Suministros
	{ID: 5, Label: "Taxes|Impuestos"},                                                         // 40111 or 64x: the user decides
	{ID: 6, Label: "Maintenance|Mantenimiento", DefaultAccountCode: 6343},                     // Mantenimiento — propiedad, planta y equipo
	{ID: 7, Label: "Transport|Transporte", DefaultAccountCode: 6311},                          // Transporte
	{ID: 8, Label: "Marketing|Marketing", DefaultAccountCode: 6371},                           // Publicidad
	{ID: 9, Label: "Professional services|Servicios profesionales", DefaultAccountCode: 6329}, // Asesoría — otros
	{ID: 10, Label: "Other|Otros"},                                                            // nothing to infer
}

// FindExpenseCategory returns the category with this id, and whether it exists.
func FindExpenseCategory(categoryID int8) (ExpenseCategoryOption, bool) {
	for _, category := range ExpenseCategoryOptions {
		if category.ID == categoryID {
			return category, true
		}
	}
	return ExpenseCategoryOption{}, false
}
