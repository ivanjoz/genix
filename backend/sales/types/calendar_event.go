package types

import (
	"app/core"
	"app/db"
	"strings"
	"unicode/utf8"
)

// CalendarEvent is a date that moves sales: a holiday, a payday, a gratificación or a commercial
// date (plan/SALES_FORECAST_PLAN.md §10.4). Partition CompanyID = 0 holds the global Peru holidays
// shared by every company; every other partition holds that company's own events.
type CalendarEvent struct {
	db.TableStruct[CalendarEventTable, CalendarEvent]
	CompanyID int32 `cb:"1"`
	ID        int32 `cb:"2"`
	// Date is a UnixDay.
	Date      int16  `cb:"3"`
	EventType int8   `cb:"4"`
	Name      string `cb:"5"`
	Status    int8   `json:"ss" cb:"ss,6"`
	Updated   int32  `json:"upd" cb:"upd,7"`
	Version   int16  `json:"ver" cb:"ver,8"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata  *string `json:"mdt" cb:"mdt,9"`
	UpdatedBy int32   `cb:"10"`
	// IsGlobal is request-only: the event is written to partition 0, which only the super-admin may.
	IsGlobal bool `db:"-" cb:"11"`
}

type CalendarEventTable struct {
	db.TableStruct[CalendarEventTable, CalendarEvent]
	CompanyID db.Col[*CalendarEventTable, int32]
	ID        db.Col[*CalendarEventTable, int32]
	Date      db.Col[*CalendarEventTable, int16]
	EventType db.Col[*CalendarEventTable, int8]
	Name      db.Col[*CalendarEventTable, string]
	Status    db.Col[*CalendarEventTable, int8]
	Updated   db.Col[*CalendarEventTable, int32]
	Version   db.Col[*CalendarEventTable, int16]
	UpdatedBy db.Col[*CalendarEventTable, int32]
}

func (e CalendarEventTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          128,
		Name:        "calendar_event",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// Active events, delta-synced per partition: the global one and the company's own.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// The kinds of CalendarEvent.
const (
	CalendarEventHoliday        = int8(1)
	CalendarEventPayday         = int8(2)
	CalendarEventGratificacion  = int8(3)
	CalendarEventCommercialDate = int8(4)
)

// CalendarGlobalCompanyID is the partition of the events every company shares.
const CalendarGlobalCompanyID = int32(0)

const calendarEventNameMaxLength = 120

// ValidateCalendarEvent checks an event about to be saved active; it trims the name in place.
func ValidateCalendarEvent(event *CalendarEvent) error {
	event.Name = strings.TrimSpace(event.Name)
	if event.Date <= 0 {
		return core.Err("El evento debe tener una fecha válida.")
	}
	if event.EventType < CalendarEventHoliday || event.EventType > CalendarEventCommercialDate {
		return core.Err("El tipo de evento no es válido:", event.EventType)
	}
	if event.Name == "" {
		return core.Err("El evento debe tener un nombre.")
	}
	if utf8.RuneCountInString(event.Name) > calendarEventNameMaxLength {
		return core.Err("El nombre del evento no puede superar", calendarEventNameMaxLength, "caracteres.")
	}
	return nil
}
