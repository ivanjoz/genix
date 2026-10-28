package exec

import (
	config "app/config/types"
	"app/core"
	"app/db"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// RebuildSaleFrames (fn-rebuild-sale-frames [companyID]) builds the sale_order DataFrames again from
// the sales, for every company or the one given. Run it after a frame's Version changes, so no user
// read pays the rebuild. A company that fails does not stop the others.
func RebuildSaleFrames(args *core.ExecArgs) core.FuncResponse {
	onlyCompanyID, _ := strconv.ParseInt(strings.TrimSpace(args.Message), 10, 32)
	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(0).AllowFilter()
	if err := companyQuery.Exec(); err != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", err)
	}
	var saleOrderController db.Controller
	for _, controller := range MakeScyllaControllers() {
		if controller.GetTableName() == "sale_order" {
			saleOrderController = controller
		}
	}
	var companyErrors []error
	for _, company := range companies {
		if onlyCompanyID > 0 && int64(company.ID) != onlyCompanyID {
			continue
		}
		if err := saleOrderController.RebuildDataFramesPartition(company.ID); err != nil {
			companyErrors = append(companyErrors, fmt.Errorf("company %d: %w", company.ID, err))
			continue
		}
		core.Log("sale frames rebuilt, company::", company.ID)
	}
	if err := errors.Join(companyErrors...); err != nil {
		return args.MakeErr("Error al reconstruir los DataFrames de ventas:", err)
	}
	return core.FuncResponse{}
}
