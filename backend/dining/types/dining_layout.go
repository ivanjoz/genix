package types

import "app/db"

// DiningLayout is one physical space a restaurant serves in (docs/DINING_PLAN.md §1): its floor
// plan, its prep stations and how its QR orders behave. The plan is edited and saved as one
// document, so its shapes and stations are blobs, not rows.
type DiningLayout struct {
	db.TableStruct[DiningLayoutTable, DiningLayout]
	CompanyID int32  `cb:"1"`
	ID        int32  `cb:"2"`
	Name      string `cb:"3"`
	// WarehouseID is where the space's goods come from: the menu's stock, the table orders'
	// reservations and the charged sales' exits.
	WarehouseID int32 `cb:"4"`
	// The canvas in grid cells. The cell's size in pixels is only the editor's zoom.
	GridWidth  int16           `cb:"5"`
	GridHeight int16           `cb:"6"`
	Shapes     []LayoutShape   `cb:"7"`
	Stations   []DiningStation `cb:"8"`
	// QRRequiresConfirmation makes a round ordered from a table's QR wait for a waiter (D6).
	QRRequiresConfirmation bool `cb:"9"`
	// QRVersion is signed into every QR of the layout: bumping it voids the printed ones.
	QRVersion int16 `cb:"10"`
	// The floor map's wait thresholds, in minutes (§9).
	WaitWarnMinutes int16 `cb:"11"`
	WaitLateMinutes int16 `cb:"12"`
	// NextShapeID is the ID the next new shape takes. Shape IDs are never reused, so a deleted
	// table's ID never reaches a new table that old orders or printed QRs would point to.
	NextShapeID int16 `cb:"13"`
	Status      int8  `json:"ss" cb:"ss,14"`
	Updated     int32 `json:"upd" cb:"upd,15"`
	UpdatedBy   int32 `cb:"16"`
	Version     int16 `json:"ver" cb:"ver,17"`
	// Metadata is set only on the delta log records a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,18"`
}

type DiningLayoutTable struct {
	db.TableStruct[DiningLayoutTable, DiningLayout]
	CompanyID              db.Col[*DiningLayoutTable, int32]
	ID                     db.Col[*DiningLayoutTable, int32]
	Name                   db.Col[*DiningLayoutTable, string]
	WarehouseID            db.Col[*DiningLayoutTable, int32]
	GridWidth              db.Col[*DiningLayoutTable, int16]
	GridHeight             db.Col[*DiningLayoutTable, int16]
	Shapes                 db.Col[*DiningLayoutTable, []LayoutShape]
	Stations               db.Col[*DiningLayoutTable, []DiningStation]
	QRRequiresConfirmation db.Col[*DiningLayoutTable, bool]
	QRVersion              db.Col[*DiningLayoutTable, int16]
	WaitWarnMinutes        db.Col[*DiningLayoutTable, int16]
	WaitLateMinutes        db.Col[*DiningLayoutTable, int16]
	NextShapeID            db.Col[*DiningLayoutTable, int16]
	Status                 db.Col[*DiningLayoutTable, int8]
	Updated                db.Col[*DiningLayoutTable, int32]
	UpdatedBy              db.Col[*DiningLayoutTable, int32]
	Version                db.Col[*DiningLayoutTable, int16]
}

func (e DiningLayoutTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          97,
		Name:        "dining_layout",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.Autoincrement(0)),
		FixedValues: []db.FixedValues{{Col: e.Status, Min: 0, Max: 1}},
		Indexes: []db.Index{
			// The company's layouts, delta-synced by every dining page.
			{Type: db.TypeDelta, Keys: db.Cols(e.Status)},
		},
	}
}

// The kinds of LayoutShape.
const (
	ShapeTable   = int8(1) // a table orders are taken at
	ShapeRect    = int8(2) // a block: bar counter, column, planter
	ShapeLine    = int8(3) // a wall or divider: two grid points
	ShapeLabel   = int8(4) // free text
	ShapeStation = int8(5) // where a prep station sits
	ShapePickup  = int8(6) // where waiters collect a station's dishes
)

// The outline of a table shape.
const (
	TableOutlineRect  = int8(1)
	TableOutlineRound = int8(2)
)

// LayoutShape is one element of the plan.
type LayoutShape struct {
	ShapeID int16 `cb:"1"`
	Kind    int8  `cb:"2"`
	// Points are grid points, x then y. A line holds its two ends {x1,y1,x2,y2}. Every other shape
	// holds the four corners of the cells it covers {x1,y1,…,x4,y4}, clockwise from the corner that
	// is its content's top-left: the corner order is its orientation, so a 90° turn starts the list
	// one corner later. One encoding, no separate rotation to disagree with the corners.
	Points []int16 `cb:"3"`
	// Label is a table's name ("M4"), a station's name or free text.
	Label string `cb:"4"`
	Seats int8   `cb:"5"` // tables
	// Outline is a table's TableOutline*.
	Outline int8 `cb:"6"`
	// StationID is the station a station or pickup mark stands for.
	StationID int16 `cb:"7"`
}

// DiningStation is a prep station (kitchen, bar, grill). A product goes to the first station that
// names it in ProductIDs, else the first that names one of its categories, else the default.
type DiningStation struct {
	ID          int16   `cb:"1"`
	Name        string  `cb:"2"`
	ProductIDs  []int32 `cb:"3"`
	CategoryIDs []int32 `cb:"4"`
	IsDefault   bool    `cb:"5"`
}
