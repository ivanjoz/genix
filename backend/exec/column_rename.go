package exec

import (
	"app/core"
	"app/db"
	"strings"
)

// fn-rename-columns [apply]
//
// The migration itself lives with the driver that needs it: db.RenameColumns.
func RenameColumns(args *core.ExecArgs) core.FuncResponse {
	db.Connect(MakeConnParams())
	if err := db.RenameColumns(strings.Contains(args.Message, "apply")); err != nil {
		return args.MakeErr(err)
	}
	return core.FuncResponse{Message: "ok"}
}
