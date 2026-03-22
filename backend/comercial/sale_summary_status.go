package comercial

import (
	"app/comercial/types"
	"app/core"
	"app/db"
)

// GetSaleSummary returns sale summary rows for a company with delta support.
// It always flattens 16-bit rows into *_32 slices in the response payload.
func GetSaleSummary(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	fechaInicio := req.GetQueryInt16("fecha-inicio")
	fechaFin := req.GetQueryInt16("fecha-fin")

	// Debug trace to diagnose sync ranges and filters from clients.
	core.Log("GetSaleSummary params:", "empresaID", req.Usuario.EmpresaID, "updated", updated, "fechaInicio", fechaInicio, "fechaFin", fechaFin)

	summaries := []types.SaleSummary{}
	query := db.Query(&summaries)
	query.EmpresaID.Equals(req.Usuario.EmpresaID)

	// Delta mode fetches only changed records after the known update token.
	if updated > 0 {
		query.Updated.GreaterThan(updated)
	}

	// Optional fecha range filter for backfills/reports.
	if fechaInicio > 0 && fechaFin > 0 {
		if fechaInicio > fechaFin {
			return req.MakeErr("La fecha de inicio no puede ser mayor que la fecha final.")
		}
		query.Fecha.Between(fechaInicio, fechaFin)
	}

	if err := query.AllowFilter().Exec(); err != nil {
		core.Log("GetSaleSummary query error:", err)
		return req.MakeErr("Error al obtener el resumen de ventas.", err)
	}

	core.Log("GetSaleSummary response count:", len(summaries))
	return req.MakeResponse(summaries)
}
