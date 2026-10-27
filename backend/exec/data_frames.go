package exec

import (
	"app/core"
	"errors"

	"github.com/ivanjoz/genix-orm/scylla"
)

// CompactDataFrames (fn-compact-frames) brings the DataFrames of every table of one company (p1) up to
// date: the manual compaction. A frame not built yet is built from the table, the others take the
// records written since their last compaction. The reads compact a frame with CompactOn on their own,
// so nothing schedules this. A table that fails does not stop the others.
func CompactDataFrames(args *core.ExecArgs) core.FuncResponse {
	if args.Param1 <= 0 {
		panic("fn-compact-frames necesita el ID de la empresa (p1).")
	}
	scylla.MakeScyllaConnection(makeConnParams())

	var tableErrors []error
	for _, controller := range MakeScyllaControllers() {
		if err := controller.CompactDataFrames(int32(args.Param1)); err != nil {
			tableErrors = append(tableErrors, errors.New(controller.GetTableName()+": "+err.Error()))
		}
	}
	if err := errors.Join(tableErrors...); err != nil {
		panic("Error al compactar los DataFrames: " + err.Error())
	}
	return core.FuncResponse{}
}
