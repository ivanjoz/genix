package types

import "app/db"

// One row per finished request, written by server_utils and never by this backend: the ORM owns
// the schema, the daemon owns the writes.
//
// What is here is what answers "which requests failed, on what route, for which company, when".
// What is deliberately not here is the message and the stack — those stay in CloudWatch, findable
// by RequestID, and the only thing this table keeps of them is the ID of the code line that
// failed. That split is the whole point: this table stays small enough to scan a fifteen-minute
// window, and the expensive detail lives where it is already being paid for.
type UserLog struct {
	db.TableStruct[UserLogTable, UserLog]
	Date                 int16   `json:",omitempty"`
	RequestID            int64   `json:",omitempty"`
	CompanyID            int32   `json:",omitempty"`
	UserID               int32   `json:",omitempty"`
	RouteID              int16   `json:",omitempty"`
	FrameRouteCompanyAgg int64   `json:",omitempty"`
	ElapsedMs            int16   `json:",omitempty"`
	ErrorCount           int8    `json:",omitempty"`
	ErrorIDs             []int32 `json:",omitempty" db:",list"`
}

type UserLogTable struct {
	db.TableStruct[UserLogTable, UserLog]
	Date                 db.Col[UserLogTable, int16]
	RequestID            db.Col[UserLogTable, int64]
	CompanyID            db.Col[UserLogTable, int32]
	UserID               db.Col[UserLogTable, int32]
	RouteID              db.Col[UserLogTable, int16]
	FrameRouteCompanyAgg db.Col[UserLogTable, int64]
	ElapsedMs            db.Col[UserLogTable, int16]
	ErrorCount           db.Col[UserLogTable, int8]
	ErrorIDs             db.Col[UserLogTable, []int32]
}

func (e UserLogTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// Written externally in whole rows; no ORM-managed created/updated columns and no update
		// counter, the same arrangement credit_usage uses.
		ID:                    44,
		Name:                  "user_logs",
		Partition:             e.Date,
		Keys:                  db.Cols(e.RequestID),
		DisableDefaultColumns: true,
		Indexes: []db.Index{
			// The grouped index behind the errors dashboard. The frame leads the packed key, so
			// one fifteen-minute slice of a day is a single contiguous clustering range and the
			// dashboard polls forward instead of rereading the day. It carries only the error
			// count and the company, so a chart never reads a payload column it will not show.
			{
				Type: db.TypeView,
				Keys: db.Cols(e.FrameRouteCompanyAgg),
				Cols: db.Cols(e.ErrorCount, e.CompanyID),
			},
		},
	}
}

// Widths of the three fields packed into FrameRouteCompanyAgg. Six bytes in total, so the packed
// value always fits an int64 with room to spare and never goes negative.
const (
	agLoggedFrameShift   = 40 // frame     1 byte
	agLoggedRouteShift   = 24 // routeID   2 bytes
	agLoggedCompanyMask  = 0xFF_FFFF
	FramesPerDay         = 96 // four per hour
	frameSecondsDuration = 15 * 60
)

// MakeFrameRouteCompanyAgg packs the three dimensions the dashboard groups by into one sortable
// integer, so a query over them is a clustering-range read instead of a scan.
//
// The frame leads deliberately. It makes one fifteen-minute slice of the day a single contiguous
// range, which is what lets the dashboard poll forward — read frames 41..42 rather than reread
// everything since midnight. Route and company then order inside the slice, so rows arrive already
// grouped and a client-side sum walks them in one pass.
//
//	bits 47..40  frame     (0..95, four per hour)
//	bits 39..24  routeID
//	bits 23..0   companyID
//
// Mirrored byte for byte in server_utils/src/reqlog/protocol.rs, which is the side that actually
// writes it; the vectors in both test files pin the two implementations together.
func MakeFrameRouteCompanyAgg(frame uint8, routeID int16, companyID int32) int64 {
	return int64(frame)<<agLoggedFrameShift |
		int64(uint16(routeID))<<agLoggedRouteShift |
		int64(companyID)&agLoggedCompanyMask
}

// FrameOfDay is the fifteen-minute slot a UTC timestamp falls in, 0..95. UTC and not local time on
// purpose: the packed key has to mean the same thing to the Rust writer and to every reader,
// whatever timezone either happens to run in.
func FrameOfDay(unixSeconds int64) uint8 {
	secondsIntoDay := unixSeconds % 86_400
	if secondsIntoDay < 0 {
		secondsIntoDay += 86_400
	}
	return uint8(secondsIntoDay / frameSecondsDuration)
}

// FrameRange returns the inclusive bounds that select every row of one frame, all routes and all
// companies included. This is the delta read: give it the frame you last saw and the one you are
// at now, and take the lowest bound of the first with the highest of the second.
func FrameRange(frame uint8) (lowest int64, highest int64) {
	lowest = int64(frame) << agLoggedFrameShift
	return lowest, lowest | (1<<agLoggedFrameShift - 1)
}
