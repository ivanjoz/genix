package sales

import (
	"app/core"
	"app/db"
	"app/sales/types"
	"encoding/json"

	"golang.org/x/sync/errgroup"
)

// superAdminUserID is the bootstrap user, the only one who writes the global calendar (see
// grantEveryRouteAccess in main-handlers.go).
const superAdminUserID = int32(1)

// GetCalendarEvents answers the global events (partition 0) and the company's own, each list
// delta-synced with its own cursor.
func GetCalendarEvents(req *core.HandlerArgs) core.HandlerResponse {
	globalEvents, companyEvents := []types.CalendarEvent{}, []types.CalendarEvent{}
	errGroup := errgroup.Group{}
	errGroup.Go(func() error {
		query := db.Query(&globalEvents)
		query.Select().CompanyID.Equals(types.CalendarGlobalCompanyID).Delta(req.GetDeltaCursor("GlobalEvents"), int8(1))
		return query.Exec()
	})
	errGroup.Go(func() error {
		query := db.Query(&companyEvents)
		query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor("CompanyEvents"), int8(1))
		return query.Exec()
	})
	if err := errGroup.Wait(); err != nil {
		return req.MakeErr("Error al obtener los eventos del calendario:", err)
	}
	return core.MakeResponse(req, &map[string]any{"GlobalEvents": globalEvents, "CompanyEvents": companyEvents})
}

// PostCalendarEvents creates, updates or (ss = 0) deletes one event. IsGlobal targets the shared
// partition 0, writable only by the super-admin; anyone else writes their own company's partition.
func PostCalendarEvents(req *core.HandlerArgs) core.HandlerResponse {
	event := types.CalendarEvent{}
	if err := json.Unmarshal([]byte(*req.Body), &event); err != nil {
		return req.MakeErr("Error al deserializar el evento:", err)
	}

	partitionID := req.User.CompanyID
	if event.IsGlobal {
		if req.User.ID != superAdminUserID {
			return req.MakeErr("Solo el super-administrador puede editar los feriados globales.")
		}
		partitionID = types.CalendarGlobalCompanyID
	}
	event.CompanyID = partitionID
	event.UpdatedBy = req.User.ID
	event.Updated = core.SUnixTime()

	if event.Status == 0 {
		if event.ID <= 0 {
			return req.MakeErr("No se puede eliminar un evento que no se ha guardado.")
		}
	} else {
		event.Status = 1
		if err := types.ValidateCalendarEvent(&event); err != nil {
			return req.MakeErr(err)
		}
	}

	events := []types.CalendarEvent{event}
	if event.ID <= 0 {
		if err := db.Insert(&events); err != nil {
			return req.MakeErr("Error al registrar el evento:", err)
		}
		return req.MakeResponse(events[0])
	}

	// An update must hit an event of the same partition: the client's ID is not trusted.
	storedEvents := []types.CalendarEvent{}
	storedQuery := db.Query(&storedEvents)
	storedQuery.Select(storedQuery.ID).CompanyID.Equals(partitionID).ID.Equals(event.ID)
	if err := storedQuery.Exec(); err != nil {
		return req.MakeErr("Error al obtener el evento:", err)
	}
	if len(storedEvents) == 0 {
		return req.MakeErr("El evento no existe.")
	}

	eventTable := db.TableOf[types.CalendarEvent]()
	columnsToUpdate := db.Cols(eventTable.Status, eventTable.Updated, eventTable.UpdatedBy)
	// A delete only changes the status: the event keeps its date, type and name.
	if event.Status != 0 {
		columnsToUpdate = append(columnsToUpdate, eventTable.Date, eventTable.EventType, eventTable.Name)
	}
	if err := db.Update(&events, columnsToUpdate...); err != nil {
		return req.MakeErr("Error al guardar el evento:", err)
	}
	return req.MakeResponse(events[0])
}
