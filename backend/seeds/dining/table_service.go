package dining

import (
	"slices"
	"time"

	"app/core"
	diningModule "app/dining"
	diningTypes "app/dining/types"
	productionTypes "app/production/types"
	"app/seeds/runner"
)

// The table service of yesterday and today. Every order is planned first as a timeline (round,
// kitchen ready, served, dessert round, charge); then the events of every table are merged and run
// in time order with the clock frozen on each, so the floor reads like a real shift. Today stops at
// the real now: what had not happened yet stays open, in whatever state the timeline left it.

// arrivalsByHour is how many tables sit down in each hour of a service day ({hour, tables}): lunch
// peaks at 13h. A slice, not a map: the draws must come in the same order for the same --seed.
var arrivalsByHour = [][2]int{{11, 2}, {12, 7}, {13, 9}, {14, 6}, {15, 3}, {19, 3}, {20, 5}, {21, 4}, {22, 1}}

// lateArrivalsToday are tables that sat down in the last 50 minutes before the real now: most of
// them are still open, in the kitchen, ready or eating, so today's floor is live.
const lateArrivalsToday = 8

// annulledArrivalsYesterday are the indexes of yesterday's arrivals that left before being served.
var annulledArrivalsYesterday = []int{9, 27}

var lineNotes = []string{"sin cebolla", "sin ají", "poco picante", "término medio", "sin culantro", "arroz aparte"}

// plannedLine is one line of a round as the waiter sends it.
type plannedLine struct {
	ProductID      int32
	PresentationID int16
	Quantity       int32
	Note           string
	stationID      int16
}

// plannedOrder is one table's visit.
type plannedOrder struct {
	layout       diningTypes.DiningLayout
	tableShapeID int16
	guests       int8
	rounds       [][]plannedLine
	// Filled as it runs: the order and, per round, the LineIDs its lines got.
	orderID        int32
	lineIDsByRound [][]int16
	// cancelledLine is the round-1 line the table changed its mind about, -1 when none.
	cancelledLine int
}

type serviceEvent struct {
	at     time.Time
	action func() error
}

// tableSlot is one table of a room, with when it is free again.
type tableSlot struct {
	layout    diningTypes.DiningLayout
	shape     diningTypes.LayoutShape
	freeAfter time.Time
}

func runTableService(run *runner.Run, menu diningMenu, layouts []diningTypes.DiningLayout) error {
	tables := []*tableSlot{}
	for _, layout := range layouts {
		for _, shape := range layout.Shapes {
			if shape.Kind == diningTypes.ShapeTable {
				tables = append(tables, &tableSlot{layout: layout, shape: shape})
			}
		}
	}
	// By seats: a couple is not sat at the eight-seat table while a two-seat one is free.
	slices.SortStableFunc(tables, func(left, right *tableSlot) int { return int(left.shape.Seats - right.shape.Seats) })

	today := core.TimeToFechaUnix(run.RealNow)
	events := []serviceEvent{}
	for _, serviceDay := range []int16{today - 1, today} {
		dayStart := run.DayStartTime(serviceDay)
		arrivals := []time.Time{}
		for _, hourArrivals := range arrivalsByHour {
			for range hourArrivals[1] {
				arrivals = append(arrivals, dayStart.Add(time.Duration(hourArrivals[0])*time.Hour+time.Duration(run.Random.Intn(60))*time.Minute))
			}
		}
		isToday := serviceDay == today
		if isToday {
			arrivals = slices.DeleteFunc(arrivals, func(arrival time.Time) bool { return arrival.After(run.RealNow.Add(-50 * time.Minute)) })
			for range lateArrivalsToday {
				arrivals = append(arrivals, run.RealNow.Add(-time.Duration(3+run.Random.Intn(47))*time.Minute))
			}
		}
		slices.SortFunc(arrivals, func(left, right time.Time) int { return left.Compare(right) })

		for arrivalIndex, arrival := range arrivals {
			guests := drawGuests(run)
			table := pickFreeTable(run, tables, guests, arrival)
			if table == nil {
				continue
			}
			order := planOrder(run, menu, table, guests)
			isAnnulled := !isToday && slices.Contains(annulledArrivalsYesterday, arrivalIndex)
			orderEvents, freeAt := scheduleOrder(run, order, arrival, isAnnulled)
			table.freeAfter = freeAt.Add(5 * time.Minute)
			events = append(events, orderEvents...)
		}
	}

	// Run what already happened, in time order; the rest of today's timeline never comes.
	slices.SortStableFunc(events, func(left, right serviceEvent) int { return left.at.Compare(right.at) })
	executedCount := 0
	for _, event := range events {
		if event.at.After(run.RealNow) {
			continue
		}
		core.SetHistoricalUnix(event.at.Unix())
		if err := event.action(); err != nil {
			return core.Err("evento de", event.at.Format("2006-01-02 15:04"), ":", err)
		}
		executedCount++
	}
	core.Log("SeedDining:: servicio de mesas", "eventos", executedCount, "de", len(events))
	return nil
}

// drawGuests: mostly couples and foursomes, now and then a big group.
func drawGuests(run *runner.Run) int8 {
	draw := run.Random.Intn(100)
	switch {
	case draw < 20:
		return 1
	case draw < 55:
		return 2
	case draw < 70:
		return 3
	case draw < 90:
		return 4
	case draw < 97:
		return int8(5 + run.Random.Intn(2))
	default:
		return int8(7 + run.Random.Intn(2))
	}
}

// pickFreeTable is one of the smallest tables that seat the group and are free at arrival, drawn
// at random so both rooms fill; nil when the restaurant is full (the group goes elsewhere).
func pickFreeTable(run *runner.Run, tables []*tableSlot, guests int8, arrival time.Time) *tableSlot {
	candidates := []*tableSlot{}
	for _, table := range tables {
		if table.shape.Seats < guests || table.freeAfter.After(arrival) {
			continue
		}
		// tables is sorted by seats: a larger table than the first candidates is not one of them.
		if len(candidates) > 0 && table.shape.Seats > candidates[0].shape.Seats {
			break
		}
		candidates = append(candidates, table)
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[run.Random.Intn(len(candidates))]
}

// planOrder draws what each guest eats: most take a menu (entrada + fondo, often a postre), some a
// single fondo, some one of the dish combos. Entradas and fondos go in the first round, postres in a
// second one after the fondos.
func planOrder(run *runner.Run, menu diningMenu, table *tableSlot, guests int8) *plannedOrder {
	randomDish := func(category string) productionTypes.Product {
		dishes := menu.dishesByCategory[category]
		return dishes[run.Random.Intn(len(dishes))]
	}
	firstRound, dessertRound := []productionTypes.Product{}, []productionTypes.Product{}
	for range guests {
		draw := run.Random.Intn(100)
		switch {
		case draw < 40:
			firstRound = append(firstRound, randomDish(categoryEntradas), randomDish(categoryFondos))
			dessertRound = append(dessertRound, randomDish(categoryPostres))
		case draw < 65:
			firstRound = append(firstRound, randomDish(categoryEntradas), randomDish(categoryFondos))
		case draw < 80:
			firstRound = append(firstRound, randomDish(categoryFondos))
		case draw < 88:
			firstRound = append(firstRound, randomDish(categoryFondos))
			dessertRound = append(dessertRound, randomDish(categoryPostres))
		default:
			// A combo (menuPacks after the two category menus): its dishes by name, a category
			// slot drawn from the category.
			combo := menuPacks[2+run.Random.Intn(len(menuPacks)-2)]
			for _, slot := range combo.Slots {
				dish := menu.dishByName[slot.DishName]
				if slot.Category != "" {
					dish = randomDish(slot.Category)
				}
				if slices.Contains(dish.CategoryIDs, menu.categoryIDByName[categoryPostres]) {
					dessertRound = append(dessertRound, dish)
				} else {
					firstRound = append(firstRound, dish)
				}
			}
		}
	}
	order := &plannedOrder{layout: table.layout, tableShapeID: table.shape.ShapeID, guests: guests, cancelledLine: -1}
	order.rounds = append(order.rounds, plannedRoundLines(run, table.layout, firstRound))
	if len(dessertRound) > 0 {
		order.rounds = append(order.rounds, plannedRoundLines(run, table.layout, dessertRound))
	}
	// One table in fifteen sends back a dish before the kitchen has it ready.
	if len(order.rounds[0]) > 1 && run.Random.Intn(15) == 0 {
		order.cancelledLine = run.Random.Intn(len(order.rounds[0]))
	}
	return order
}

// plannedRoundLines folds the dishes into one line per product ("2× Lomo saltado"), in order of first
// appearance, now and then with a note.
func plannedRoundLines(run *runner.Run, layout diningTypes.DiningLayout, dishes []productionTypes.Product) []plannedLine {
	lines := []plannedLine{}
	for _, dish := range dishes {
		lineIndex := slices.IndexFunc(lines, func(line plannedLine) bool { return line.ProductID == dish.ID })
		if lineIndex >= 0 {
			lines[lineIndex].Quantity += core.QuantityLineScale
			continue
		}
		line := plannedLine{ProductID: dish.ID, Quantity: core.QuantityLineScale,
			stationID: diningTypes.StationForProduct(layout.Stations, dish.ID, dish.CategoryIDs)}
		if run.Random.Intn(10) == 0 {
			line.Note = lineNotes[run.Random.Intn(len(lineNotes))]
		}
		lines = append(lines, line)
	}
	return lines
}

// scheduleOrder turns a visit into its timed events and returns when the table is left: the cold
// station plates entradas fast, the hot one takes longer, waiters pick up a minute or two after.
// An annulled visit is a table that left a few minutes after ordering.
func scheduleOrder(run *runner.Run, order *plannedOrder, arrival time.Time, isAnnulled bool) ([]serviceEvent, time.Time) {
	minutes := func(fromMinutes, toMinutes int) time.Duration {
		return time.Duration(run.RandomInRange(fromMinutes, toMinutes)) * time.Minute
	}
	events := []serviceEvent{{at: arrival, action: func() error { return postRound(order, 0) }}}
	if isAnnulled {
		annulAt := arrival.Add(minutes(4, 9))
		events = append(events, serviceEvent{at: annulAt, action: func() error { return annulOrder(order) }})
		return events, annulAt
	}
	if order.cancelledLine >= 0 {
		events = append(events, serviceEvent{at: arrival.Add(minutes(1, 3)), action: func() error {
			return moveLines(order, 0, -1, diningTypes.LineStatusCancelled)
		}})
	}

	lastServedAt := arrival
	scheduleRoundService := func(roundIndex int, startAt time.Time, readyAfter map[int16][2]int) {
		for _, stationID := range []int16{stationColdID, stationHotID} {
			hasLines := slices.ContainsFunc(order.rounds[roundIndex], func(line plannedLine) bool { return line.stationID == stationID })
			if !hasLines {
				continue
			}
			readyAt := startAt.Add(minutes(readyAfter[stationID][0], readyAfter[stationID][1]))
			servedAt := readyAt.Add(minutes(1, 3))
			events = append(events,
				serviceEvent{at: readyAt, action: func() error { return moveLines(order, roundIndex, stationID, diningTypes.LineStatusReady) }},
				serviceEvent{at: servedAt, action: func() error { return moveLines(order, roundIndex, stationID, diningTypes.LineStatusServed) }})
			lastServedAt = core.If(servedAt.After(lastServedAt), servedAt, lastServedAt)
		}
	}
	scheduleRoundService(0, arrival, map[int16][2]int{stationColdID: {5, 9}, stationHotID: {13, 24}})
	if len(order.rounds) > 1 {
		dessertAt := lastServedAt.Add(minutes(14, 24))
		events = append(events, serviceEvent{at: dessertAt, action: func() error { return postRound(order, 1) }})
		scheduleRoundService(1, dessertAt, map[int16][2]int{stationColdID: {2, 5}})
	}
	chargeAt := lastServedAt.Add(minutes(8, 15))
	events = append(events, serviceEvent{at: chargeAt, action: func() error { return chargeOrder(run, order) }})
	return events, chargeAt
}

// postRound sends one round of the order; the first one opens it.
func postRound(order *plannedOrder, roundIndex int) error {
	payload := map[string]any{"LayoutID": order.layout.ID, "TableShapeID": order.tableShapeID, "Guests": order.guests,
		"Lines": order.rounds[roundIndex]}
	savedOrder := diningTypes.TableOrder{}
	if err := callAsDiningUser("POST.dining-table-order-round", diningModule.PostDiningTableOrderRound, payload, &savedOrder); err != nil {
		return err
	}
	order.orderID = savedOrder.ID
	// The round's lines are the order's last ones, in the order they were sent.
	roundLines := savedOrder.Lines[len(savedOrder.Lines)-len(order.rounds[roundIndex]):]
	lineIDs := []int16{}
	for _, line := range roundLines {
		lineIDs = append(lineIDs, line.LineID)
	}
	order.lineIDsByRound = append(order.lineIDsByRound, lineIDs)
	return nil
}

// moveLines moves a round's lines of one station (stationID -1: the cancelled line alone) to
// targetStatus. The cancelled line never reaches the kitchen's pass.
func moveLines(order *plannedOrder, roundIndex int, stationID int16, targetStatus int8) error {
	lineIDs := []int16{}
	for lineIndex, line := range order.rounds[roundIndex] {
		isCancelledLine := roundIndex == 0 && lineIndex == order.cancelledLine
		if (stationID < 0 && isCancelledLine) || (stationID >= 0 && line.stationID == stationID && !isCancelledLine) {
			lineIDs = append(lineIDs, order.lineIDsByRound[roundIndex][lineIndex])
		}
	}
	if len(lineIDs) == 0 {
		return nil
	}
	payload := map[string]any{"OrderID": order.orderID, "LineIDs": lineIDs, "Status": targetStatus}
	return callAsDiningUser("POST.dining-table-order-lines", diningModule.PostDiningTableOrderLines, payload, nil)
}

// chargeOrder closes the bill into the till: one table in eight asks for a factura with its RUC,
// the rest take a boleta, half of them naming a known client.
func chargeOrder(run *runner.Run, order *plannedOrder) error {
	payload := map[string]any{"OrderID": order.orderID, "CashBankID": run.TillByWarehouse[order.layout.WarehouseID],
		"IssueSeriesID": run.BoletaSeriesID}
	if run.FacturaSeriesID > 0 && len(run.BusinessClientIDs) > 0 && run.Random.Intn(8) == 0 {
		payload["IssueSeriesID"] = run.FacturaSeriesID
		payload["ClientID"] = run.BusinessClientIDs[run.Random.Intn(len(run.BusinessClientIDs))]
	} else if run.Random.Intn(2) == 0 && len(run.ClientIDs) > 0 {
		payload["ClientID"] = run.ClientIDs[run.Random.Intn(len(run.ClientIDs))]
	}
	return callAsDiningUser("POST.dining-table-order-charge", diningModule.PostDiningTableOrderCharge, payload, nil)
}

func annulOrder(order *plannedOrder) error {
	payload := map[string]any{"OrderID": order.orderID, "Reason": "Los clientes se retiraron antes de ser atendidos"}
	return callAsDiningUser("POST.dining-table-order-annul", diningModule.PostDiningTableOrderAnnul, payload, nil)
}
