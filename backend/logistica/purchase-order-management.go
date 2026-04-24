package logistica

import (
	"app/core"
	"app/db"
	logisticaTypes "app/logistica/types"
	"encoding/json"
	"time"
)

func GetPurchaseOrders(req *core.HandlerArgs) core.HandlerResponse {
	updated := core.Coalesce(req.GetQueryInt("upd"), req.GetQueryInt("updated"))
	statusParam := int8(req.GetQueryInt("status"))
	if statusParam == 0 {
		statusParam = logisticaTypes.PurchaseOrderStatusPending
	}

	records := []logisticaTypes.PurchaseOrder{}
	query := db.Query(&records)
	query.CompanyID.Equals(req.Usuario.EmpresaID).
		Status.Equals(statusParam).
		Updated.GreaterThan(updated)

	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener órdenes de compra:", err)
	}
	return req.MakeResponse(records)
}

func PostPurchaseOrder(req *core.HandlerArgs) core.HandlerResponse {
	record := logisticaTypes.PurchaseOrder{}
	if err := json.Unmarshal([]byte(*req.Body), &record); err != nil {
		return req.MakeErr("Error al deserializar el body.", err)
	}

	if record.ProviderID <= 0 {
		return req.MakeErr("Debe seleccionar un proveedor.")
	}
	if len(record.DetailProductIDs) == 0 {
		return req.MakeErr("Debe agregar al menos un producto.")
	}
	n := len(record.DetailProductIDs)
	if len(record.DetailQuantities) != n || len(record.DetailPrices) != n {
		return req.MakeErr("Los detalles de la orden son inconsistentes.")
	}

	now := core.SUnixTime()
	todayFecha := core.TimeToFechaUnix(time.Now())
	currentSemana := core.MakeSemanaFromFechaUnix(todayFecha, false)

	record.CompanyID = req.Usuario.EmpresaID
	record.Status = logisticaTypes.PurchaseOrderStatusPending
	record.Updated = now
	record.UpdatedBy = req.Usuario.ID
	if record.ID == 0 {
		record.Created = now
		record.CreatedBy = req.Usuario.ID
		record.Date = todayFecha
		record.Week = currentSemana.Code
	}

	records := []logisticaTypes.PurchaseOrder{record}
	if err := db.Merge(&records, nil,
		func(prev, curr *logisticaTypes.PurchaseOrder) bool {
			curr.CompanyID = req.Usuario.EmpresaID
			curr.Created = prev.Created
			curr.CreatedBy = prev.CreatedBy
			curr.Date = prev.Date
			curr.Week = prev.Week
			curr.Status = logisticaTypes.PurchaseOrderStatusPending
			curr.Updated = now
			curr.UpdatedBy = req.Usuario.ID
			return true
		},
		func(curr *logisticaTypes.PurchaseOrder) {
			curr.CompanyID = req.Usuario.EmpresaID
			curr.Status = logisticaTypes.PurchaseOrderStatusPending
			curr.Updated = now
			curr.UpdatedBy = req.Usuario.ID
			curr.Date = todayFecha
			curr.Week = currentSemana.Code
		},
	); err != nil {
		return req.MakeErr("Error al guardar la orden de compra.", err)
	}

	return req.MakeResponse(records[0])
}
