package dining

import (
	"slices"

	"app/core"
	"app/db"
	diningModule "app/dining"
	diningTypes "app/dining/types"
	"app/seeds/runner"
)

// The two dining rooms. Coordinates are grid cells; y grows downward.
const (
	mainRoomName    = "Salón principal"
	terraceRoomName = "Terraza"
	// Station IDs inside each layout: hot dishes are the default, entradas and postres go cold.
	stationHotID  = int16(1)
	stationColdID = int16(2)
)

// floorRect is the four corners of a cell rectangle, clockwise from its top-left (LayoutShape.Points).
func floorRect(x1, y1, x2, y2 int16) []int16 {
	return []int16{x1, y1, x2, y1, x2, y2, x1, y2}
}

func floorTable(label string, seats int8, outline int8, x1, y1, x2, y2 int16) diningTypes.LayoutShape {
	return diningTypes.LayoutShape{Kind: diningTypes.ShapeTable, Label: label, Seats: seats, Outline: outline, Points: floorRect(x1, y1, x2, y2)}
}

func floorWall(x1, y1, x2, y2 int16) diningTypes.LayoutShape {
	return diningTypes.LayoutShape{Kind: diningTypes.ShapeLine, Points: []int16{x1, y1, x2, y2}}
}

func floorBlock(label string, x1, y1, x2, y2 int16) diningTypes.LayoutShape {
	return diningTypes.LayoutShape{Kind: diningTypes.ShapeRect, Label: label, Points: floorRect(x1, y1, x2, y2)}
}

func floorMark(kind int8, label string, stationID int16, x1, y1, x2, y2 int16) diningTypes.LayoutShape {
	return diningTypes.LayoutShape{Kind: kind, Label: label, StationID: stationID, Points: floorRect(x1, y1, x2, y2)}
}

// diningStations routes the platos de fondo to the hot kitchen (the default) and the entradas and
// postres to the cold one.
func diningStations(menu diningMenu) []diningTypes.DiningStation {
	return []diningTypes.DiningStation{
		{ID: stationHotID, Name: "Cocina caliente", IsDefault: true},
		{ID: stationColdID, Name: "Cocina fría", CategoryIDs: []int32{menu.categoryIDByName[categoryEntradas], menu.categoryIDByName[categoryPostres]}},
	}
}

// mainRoomShapes: a 48×32 room, the kitchen in the top-right corner with its two pickup points at
// the pass, 17 tables in four rows, the till and the restrooms on the right and the street door at
// the bottom.
func mainRoomShapes() []diningTypes.LayoutShape {
	const rect, round = diningTypes.TableOutlineRect, diningTypes.TableOutlineRound
	return []diningTypes.LayoutShape{
		floorWall(0, 0, 48, 0), floorWall(0, 0, 0, 32), floorWall(48, 0, 48, 32),
		floorWall(0, 32, 20, 32), floorWall(26, 32, 48, 32), floorWall(34, 12, 48, 12),
		floorBlock("Cocina", 38, 0, 48, 6),
		floorMark(diningTypes.ShapeStation, "Caliente", stationHotID, 38, 6, 43, 8),
		floorMark(diningTypes.ShapeStation, "Fría", stationColdID, 43, 6, 48, 8),
		floorMark(diningTypes.ShapePickup, "Recojo caliente", stationHotID, 32, 1, 37, 4),
		floorMark(diningTypes.ShapePickup, "Recojo frío", stationColdID, 32, 5, 37, 8),
		floorTable("M1", 4, rect, 2, 2, 5, 5), floorTable("M2", 4, rect, 8, 2, 11, 5),
		floorTable("M3", 4, rect, 14, 2, 17, 5), floorTable("M4", 4, rect, 20, 2, 23, 5),
		floorTable("M5", 4, rect, 25, 2, 28, 5),
		floorTable("M6", 2, round, 3, 9, 5, 11), floorTable("M7", 2, round, 9, 9, 11, 11),
		floorTable("M8", 2, round, 15, 9, 17, 11), floorTable("M9", 2, round, 21, 9, 23, 11),
		floorTable("M10", 4, rect, 2, 15, 5, 18), floorTable("M11", 4, rect, 8, 15, 11, 18),
		floorTable("M12", 4, rect, 14, 15, 17, 18), floorTable("M13", 4, rect, 20, 15, 23, 18),
		floorTable("M14", 4, round, 26, 15, 29, 18),
		floorTable("M15", 8, rect, 2, 22, 10, 25), floorTable("M16", 6, rect, 13, 22, 19, 25),
		floorTable("M17", 4, round, 22, 22, 25, 25),
		floorBlock("SS.HH.", 40, 14, 48, 20),
		floorBlock("Caja", 34, 25, 44, 27),
		{Kind: diningTypes.ShapeLabel, Label: "Entrada", Points: floorRect(20, 29, 26, 31)},
	}
}

// terraceRoomShapes: a 36×24 open terrace behind a railing, its door to the main room at the top,
// ten tables and the waiters' pickup counter by the door.
func terraceRoomShapes() []diningTypes.LayoutShape {
	const rect, round = diningTypes.TableOutlineRect, diningTypes.TableOutlineRound
	return []diningTypes.LayoutShape{
		floorWall(0, 0, 14, 0), floorWall(22, 0, 36, 0),
		floorWall(0, 0, 0, 24), floorWall(36, 0, 36, 24), floorWall(0, 24, 36, 24),
		{Kind: diningTypes.ShapeLabel, Label: "Puerta al salón", Points: floorRect(14, 1, 22, 2)},
		floorMark(diningTypes.ShapePickup, "Recojo caliente", stationHotID, 12, 18, 17, 20),
		floorMark(diningTypes.ShapePickup, "Recojo frío", stationColdID, 19, 18, 24, 20),
		floorTable("T1", 4, round, 3, 4, 6, 7), floorTable("T2", 4, round, 10, 4, 13, 7),
		floorTable("T3", 2, round, 17, 4, 19, 6), floorTable("T4", 4, round, 24, 4, 27, 7),
		floorTable("T5", 2, round, 31, 4, 33, 6),
		floorTable("T6", 4, rect, 3, 12, 6, 15), floorTable("T7", 4, rect, 10, 12, 13, 15),
		floorTable("T8", 2, round, 17, 12, 19, 14), floorTable("T9", 4, rect, 24, 12, 27, 15),
		floorTable("T10", 6, rect, 29, 12, 35, 15),
		floorBlock("Jardinera", 2, 21, 9, 23), floorBlock("Jardinera", 27, 21, 34, 23),
	}
}

// ensureDiningLayouts saves the two rooms on the first warehouse. The company's first room becomes
// the main room (a test plan is replaced, its open orders annulled first so its tables can go);
// isNew is false when the terrace already exists, which means the rooms were seeded before.
func ensureDiningLayouts(run *runner.Run, menu diningMenu) ([]diningTypes.DiningLayout, bool, error) {
	storedLayouts := []diningTypes.DiningLayout{}
	query := db.Query(&storedLayouts)
	query.Select().CompanyID.Equals(runner.CompanyID).Status.Equals(1)
	if err := query.Exec(); err != nil {
		return nil, false, core.Err("no se pudieron leer los salones:", err)
	}
	slices.SortFunc(storedLayouts, func(left, right diningTypes.DiningLayout) int { return int(left.ID - right.ID) })
	if slices.ContainsFunc(storedLayouts, func(layout diningTypes.DiningLayout) bool { return layout.Name == terraceRoomName }) {
		return storedLayouts, false, nil
	}

	cartaID, err := ensureDiningCarta(menu)
	if err != nil {
		return nil, false, core.Err("carta:", err)
	}
	warehouseID := run.Config.WarehouseIDs[0]
	mainRoom := diningTypes.DiningLayout{Name: mainRoomName, WarehouseID: warehouseID, GridWidth: 48, GridHeight: 32,
		Shapes: mainRoomShapes(), Stations: diningStations(menu), WaitWarnMinutes: 15, WaitLateMinutes: 25, MenuID: cartaID, Status: 1}
	if len(storedLayouts) > 0 {
		mainRoom.ID = storedLayouts[0].ID
		if err := annulOpenTableOrders(mainRoom.ID); err != nil {
			return nil, false, err
		}
	}
	terraceRoom := diningTypes.DiningLayout{Name: terraceRoomName, WarehouseID: warehouseID, GridWidth: 36, GridHeight: 24,
		Shapes: terraceRoomShapes(), Stations: diningStations(menu), WaitWarnMinutes: 15, WaitLateMinutes: 25, MenuID: cartaID, Status: 1}

	savedLayouts := []diningTypes.DiningLayout{}
	for _, layout := range []diningTypes.DiningLayout{mainRoom, terraceRoom} {
		savedLayout := diningTypes.DiningLayout{}
		if err := callAsDiningUser("POST.dining-layout", diningModule.PostDiningLayout, layout, &savedLayout); err != nil {
			return nil, false, err
		}
		savedLayouts = append(savedLayouts, savedLayout)
		core.Log("SeedDining:: salón guardado", savedLayout.ID, savedLayout.Name, "elementos", len(savedLayout.Shapes))
	}
	return savedLayouts, true, nil
}

// annulOpenTableOrders annuls a room's open orders, which would keep its old tables from being replaced.
func annulOpenTableOrders(layoutID int32) error {
	openOrders := []diningTypes.TableOrder{}
	query := db.Query(&openOrders)
	query.Select(query.ID).CompanyID.Equals(runner.CompanyID).LayoutID.Equals(layoutID).Status.Equals(diningTypes.TableOrderOpen)
	if err := query.Exec(); err != nil {
		return core.Err("no se pudieron leer los pedidos abiertos:", err)
	}
	for _, order := range openOrders {
		payload := map[string]any{"OrderID": order.ID, "Reason": "Reemplazo del plano por los datos de demostración"}
		if err := callAsDiningUser("POST.dining-table-order-annul", diningModule.PostDiningTableOrderAnnul, payload, nil); err != nil {
			return err
		}
	}
	return nil
}
