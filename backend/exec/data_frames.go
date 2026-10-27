package exec

import (
	"app/core"
	"errors"

	"github.com/ivanjoz/genix-orm/scylla"
)

// MaterializeDataFrames (fn-materialize-frames) brings the DataFrames of every table up to date, every
// company at once: a frame not built yet is built from the table, the others take the records
// written since their last run. Schedule it every 10 minutes. A frame another run holds is skipped,
// and a table that fails does not stop the others.
func MaterializeDataFrames(args *core.ExecArgs) core.FuncResponse {
	scylla.MakeScyllaConnection(makeConnParams())

	var tableErrors []error
	for _, controller := range MakeScyllaControllers() {
		if err := controller.MaterializeDataFrames(); err != nil {
			tableErrors = append(tableErrors, errors.New(controller.GetTableName()+": "+err.Error()))
		}
	}
	if err := errors.Join(tableErrors...); err != nil {
		panic("Error al materializar los DataFrames: " + err.Error())
	}
	return core.FuncResponse{}
}
