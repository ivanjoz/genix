package types

import (
	"strings"
	"testing"
)

// rect is the corners of w × h cells from (x, y), clockwise from the top-left: an unturned shape.
func rect(x, y, w, h int16) []int16 {
	return []int16{x, y, x + w, y, x + w, y + h, x, y + h}
}

func TestTurnedRectangleIsValid(t *testing.T) {
	layout := layoutFixture()
	// M1 turned 90°: its list starts at the top-right corner.
	layout.Shapes[0].Points = []int16{4, 2, 4, 4, 2, 4, 2, 2}
	if err := ValidateDiningLayout(layout); err != nil {
		t.Fatal(err)
	}
}

func layoutFixture() DiningLayout {
	return DiningLayout{
		Name: "Salón", WarehouseID: 1, GridWidth: 40, GridHeight: 30, WaitWarnMinutes: 10, WaitLateMinutes: 20,
		NextShapeID: 5,
		Stations: []DiningStation{
			{ID: 1, Name: "Cocina", IsDefault: true},
			{ID: 2, Name: "Bar", ProductIDs: []int32{7}, CategoryIDs: []int32{3}},
		},
		Shapes: []LayoutShape{
			{ShapeID: 1, Kind: ShapeTable, Points: rect(2, 2, 2, 2), Label: "M1", Seats: 4, Outline: TableOutlineRect},
			{ShapeID: 2, Kind: ShapeTable, Points: rect(6, 2, 2, 2), Label: "M2", Seats: 2, Outline: TableOutlineRound},
			{ShapeID: 3, Kind: ShapeLine, Points: []int16{0, 10, 40, 10}},
			{ShapeID: 4, Kind: ShapePickup, Points: rect(30, 20, 2, 1), StationID: 2},
		},
	}
}

func TestStationForProduct(t *testing.T) {
	stations := layoutFixture().Stations
	if station := StationForProduct(stations, 7, nil); station != 2 {
		t.Errorf("by product: got %v", station)
	}
	if station := StationForProduct(stations, 8, []int32{9, 3}); station != 2 {
		t.Errorf("by category: got %v", station)
	}
	if station := StationForProduct(stations, 8, []int32{9}); station != 1 {
		t.Errorf("default: got %v", station)
	}
}

func TestAssignLayoutShapeIDsNeverReusesAnID(t *testing.T) {
	stored := layoutFixture()
	layout := layoutFixture()
	// M2 (ID 2) was deleted; two new shapes come with temporary negative IDs.
	layout.Shapes = append(layout.Shapes[:1], layout.Shapes[2:]...)
	layout.Shapes = append(layout.Shapes,
		LayoutShape{ShapeID: -1, Kind: ShapeTable, Points: rect(10, 2, 2, 2), Label: "M3", Outline: TableOutlineRect},
		LayoutShape{ShapeID: -2, Kind: ShapeLabel, Points: rect(1, 25, 6, 1), Label: "Terraza"})
	if err := AssignLayoutShapeIDs(&layout, &stored); err != nil {
		t.Fatal(err)
	}
	if layout.Shapes[3].ShapeID != 5 || layout.Shapes[4].ShapeID != 6 || layout.NextShapeID != 7 {
		t.Fatalf("ids: %v %v next %v", layout.Shapes[3].ShapeID, layout.Shapes[4].ShapeID, layout.NextShapeID)
	}
	if err := ValidateDiningLayout(layout); err != nil {
		t.Fatal(err)
	}

	// Sending the deleted ID 2 back is refused.
	revived := layoutFixture()
	stored.Shapes = stored.Shapes[:1]
	if err := AssignLayoutShapeIDs(&revived, &stored); err == nil {
		t.Fatal("a shape ID missing from the stored layout must be refused")
	}
}

func TestValidateDiningLayout(t *testing.T) {
	if err := ValidateDiningLayout(layoutFixture()); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	cases := []struct {
		name       string
		edit       func(layout *DiningLayout)
		wantReason string
	}{
		{"outside the grid", func(layout *DiningLayout) { layout.Shapes[0].Points = rect(39, 2, 2, 2) }, "fuera del salón"},
		{"line outside", func(layout *DiningLayout) { layout.Shapes[2].Points[2] = 41 }, "fuera del salón"},
		{"line of no length", func(layout *DiningLayout) { layout.Shapes[2].Points = []int16{0, 10, 0, 10} }, "no tiene tamaño"},
		{"line with three numbers", func(layout *DiningLayout) { layout.Shapes[2].Points = []int16{0, 10, 4} }, "fuera del salón"},
		{"counter-clockwise corners", func(layout *DiningLayout) { layout.Shapes[0].Points = []int16{2, 2, 2, 4, 4, 4, 4, 2} }, "rectángulo"},
		{"corners of no rectangle", func(layout *DiningLayout) { layout.Shapes[0].Points = []int16{2, 2, 4, 2, 5, 4, 2, 4} }, "rectángulo"},
		{"flat rectangle", func(layout *DiningLayout) { layout.Shapes[0].Points = []int16{2, 2, 4, 2, 4, 2, 2, 2} }, "rectángulo"},
		{"repeated table label", func(layout *DiningLayout) { layout.Shapes[1].Label = "m1" }, "repetido"},
		{"repeated shape id", func(layout *DiningLayout) { layout.Shapes[1].ShapeID = 1 }, "repetido"},
		{"id beyond next", func(layout *DiningLayout) { layout.Shapes[1].ShapeID = 9 }, "inválido"},
		{"two defaults", func(layout *DiningLayout) { layout.Stations[1].IsDefault = true }, "exactamente una"},
		{"pickup of a missing station", func(layout *DiningLayout) { layout.Shapes[3].StationID = 5 }, "no existe"},
		{"thresholds", func(layout *DiningLayout) { layout.WaitWarnMinutes = 20 }, "espera"},
		{"table without a name", func(layout *DiningLayout) { layout.Shapes[0].Label = "" }, "nombre"},
	}
	for _, testCase := range cases {
		layout := layoutFixture()
		testCase.edit(&layout)
		err := ValidateDiningLayout(layout)
		if err == nil || !strings.Contains(err.Error(), testCase.wantReason) {
			t.Errorf("%s: want an error containing %q, got %v", testCase.name, testCase.wantReason, err)
		}
	}
}
