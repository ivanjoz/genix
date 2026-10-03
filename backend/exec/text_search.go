package exec

import (
	"app/core"

	"github.com/ivanjoz/genix-orm/scylla"
)

// ReindexTextSearch (fn-reindex-text) rebuilds the text index of every table that declares
// TextSearch from its stored records, in every partition that has some, or only in company p1 when
// it is set. Searches keep working while it runs and writes that land meanwhile are kept.
//
// Run it after pointing search.url at another backend (the new one starts empty), or after a
// deploy changes what a table indexes, like the BrandName column products gained.
func ReindexTextSearch(args *core.ExecArgs) core.FuncResponse {
	scylla.MakeScyllaConnection(makeConnParams())

	for _, controller := range MakeScyllaControllers() {
		var err error
		if args.Param1 > 0 {
			err = controller.ReindexTextSearch(int32(args.Param1))
		} else {
			err = controller.ReindexTextSearchAll()
		}
		if err != nil {
			panic("Error al reindexar la búsqueda de texto de " + controller.GetTableName() + ": " + err.Error())
		}
	}
	return core.FuncResponse{}
}
