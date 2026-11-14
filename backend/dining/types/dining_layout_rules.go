package types

import (
	"app/core"
	"fmt"
	"slices"
	"strings"
)

// The limits a saved layout must respect. The grid is in cells; a cell is drawn at 8–16 px.
const (
	MinGridSize         = int16(8)
	MaxGridSize         = int16(200)
	MaxLayoutShapes     = 500
	MaxShapeLabelLength = 40
	MaxTableSeats       = int8(40)
	MaxDiningStations   = 20
	DefaultWaitWarn     = int16(10)
	DefaultWaitLate     = int16(20)
)

// StationForProduct is the station a product's lines go to: the first station naming the product,
// else the first naming one of its categories, else the default one (§1.1).
func StationForProduct(stations []DiningStation, productID int32, categoryIDs []int32) int16 {
	for _, station := range stations {
		if slices.Contains(station.ProductIDs, productID) {
			return station.ID
		}
	}
	for _, station := range stations {
		for _, categoryID := range categoryIDs {
			if slices.Contains(station.CategoryIDs, categoryID) {
				return station.ID
			}
		}
	}
	for _, station := range stations {
		if station.IsDefault {
			return station.ID
		}
	}
	return 0
}

// AssignLayoutShapeIDs gives each new shape (ShapeID <= 0) the layout's next ID, and refuses an
// existing ID the stored layout does not hold: IDs are never reused, so a deleted table's ID can
// never come back on another table (§1.2). stored is nil for a new layout.
func AssignLayoutShapeIDs(layout *DiningLayout, stored *DiningLayout) error {
	storedShapeIDs := map[int16]bool{}
	layout.NextShapeID = 1
	if stored != nil {
		layout.NextShapeID = max(stored.NextShapeID, 1)
		for _, shape := range stored.Shapes {
			storedShapeIDs[shape.ShapeID] = true
		}
	}
	for index := range layout.Shapes {
		shape := &layout.Shapes[index]
		if shape.ShapeID <= 0 {
			shape.ShapeID = layout.NextShapeID
			layout.NextShapeID++
		} else if !storedShapeIDs[shape.ShapeID] {
			return core.Err(fmt.Sprintf("El elemento %v no existe en el salón guardado.", shape.ShapeID))
		}
	}
	return nil
}

// NormalizeDiningLayout trims the free texts and fills the thresholds left at 0 with the defaults.
func NormalizeDiningLayout(layout *DiningLayout) {
	layout.Name = strings.TrimSpace(layout.Name)
	for index := range layout.Shapes {
		layout.Shapes[index].Label = strings.TrimSpace(layout.Shapes[index].Label)
	}
	for index := range layout.Stations {
		layout.Stations[index].Name = strings.TrimSpace(layout.Stations[index].Name)
	}
	if layout.WaitWarnMinutes == 0 {
		layout.WaitWarnMinutes = DefaultWaitWarn
	}
	if layout.WaitLateMinutes == 0 {
		layout.WaitLateMinutes = DefaultWaitLate
	}
}

// ValidateDiningLayout checks what a layout must be once its shape IDs are assigned (§1.2): the
// warehouse and the open tables are checked by the handler, against the database.
func ValidateDiningLayout(layout DiningLayout) error {
	if layout.Name == "" {
		return core.Err("El salón necesita un nombre.")
	}
	if layout.WarehouseID <= 0 {
		return core.Err("El salón necesita el almacén del que salen sus productos.")
	}
	if layout.GridWidth < MinGridSize || layout.GridWidth > MaxGridSize ||
		layout.GridHeight < MinGridSize || layout.GridHeight > MaxGridSize {
		return core.Err(fmt.Sprintf("El tamaño del salón debe estar entre %v y %v celdas por lado.", MinGridSize, MaxGridSize))
	}
	if layout.WaitWarnMinutes <= 0 || layout.WaitWarnMinutes >= layout.WaitLateMinutes {
		return core.Err("La espera de alerta debe ser mayor a 0 y menor a la espera de retraso.")
	}
	if len(layout.Shapes) > MaxLayoutShapes {
		return core.Err(fmt.Sprintf("El salón admite hasta %v elementos.", MaxLayoutShapes))
	}
	if err := validateDiningStations(layout.Stations); err != nil {
		return err
	}
	stationIDs := map[int16]bool{}
	for _, station := range layout.Stations {
		stationIDs[station.ID] = true
	}

	shapeIDs := map[int16]bool{}
	tableLabels := map[string]bool{}
	for _, shape := range layout.Shapes {
		if shape.ShapeID <= 0 || shape.ShapeID >= layout.NextShapeID || shapeIDs[shape.ShapeID] {
			return core.Err(fmt.Sprintf("El identificador del elemento %v es inválido o está repetido.", shape.ShapeID))
		}
		shapeIDs[shape.ShapeID] = true
		if len([]rune(shape.Label)) > MaxShapeLabelLength {
			return core.Err(fmt.Sprintf(`El texto "%s" supera los %v caracteres.`, shape.Label, MaxShapeLabelLength))
		}
		if err := validateShapeGeometry(shape, layout.GridWidth, layout.GridHeight); err != nil {
			return err
		}
		switch shape.Kind {
		case ShapeTable:
			if shape.Label == "" {
				return core.Err("Cada mesa necesita un nombre.")
			}
			// Case-insensitive: "m4" and "M4" are the same table to a waiter.
			labelKey := strings.ToLower(shape.Label)
			if tableLabels[labelKey] {
				return core.Err(fmt.Sprintf(`El nombre de mesa "%s" está repetido.`, shape.Label))
			}
			tableLabels[labelKey] = true
			if shape.Seats < 0 || shape.Seats > MaxTableSeats {
				return core.Err(fmt.Sprintf(`La mesa "%s" debe tener entre 0 y %v asientos.`, shape.Label, MaxTableSeats))
			}
			if shape.Outline != TableOutlineRect && shape.Outline != TableOutlineRound {
				return core.Err(fmt.Sprintf(`La forma de la mesa "%s" es inválida.`, shape.Label))
			}
		case ShapeStation, ShapePickup:
			if !stationIDs[shape.StationID] {
				return core.Err("Una marca de estación o de recojo apunta a una estación que no existe.")
			}
		case ShapeRect, ShapeLine, ShapeLabel:
		default:
			return core.Err(fmt.Sprintf("Tipo de elemento desconocido: %v.", shape.Kind))
		}
	}
	return nil
}

func validateDiningStations(stations []DiningStation) error {
	if len(stations) == 0 || len(stations) > MaxDiningStations {
		return core.Err(fmt.Sprintf("El salón necesita entre 1 y %v estaciones de preparación.", MaxDiningStations))
	}
	stationIDs := map[int16]bool{}
	defaultCount := 0
	for _, station := range stations {
		if station.ID <= 0 || stationIDs[station.ID] {
			return core.Err(fmt.Sprintf("El identificador de la estación %v es inválido o está repetido.", station.ID))
		}
		stationIDs[station.ID] = true
		if station.Name == "" || len([]rune(station.Name)) > MaxShapeLabelLength {
			return core.Err(fmt.Sprintf("Cada estación necesita un nombre de hasta %v caracteres.", MaxShapeLabelLength))
		}
		if station.IsDefault {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		return core.Err("Debe haber exactamente una estación por defecto.")
	}
	return nil
}

// validateShapeGeometry checks a shape's Points (see LayoutShape): every point on the grid (0..width,
// 0..height); a line's two ends apart; any other shape's four corners a grid-aligned rectangle of at
// least one cell, listed clockwise from any of its corners.
func validateShapeGeometry(shape LayoutShape, gridWidth int16, gridHeight int16) error {
	points := shape.Points
	isValid := len(points) == core.If(shape.Kind == ShapeLine, 4, 8)
	for index := 0; isValid && index < len(points); index += 2 {
		isValid = points[index] >= 0 && points[index] <= gridWidth && points[index+1] >= 0 && points[index+1] <= gridHeight
	}
	if isValid && shape.Kind == ShapeLine {
		isValid = points[0] != points[2] || points[1] != points[3]
	} else if isValid {
		isValid = isClockwiseGridRectangle(points)
	}
	if !isValid {
		return core.Err(fmt.Sprintf("El elemento %v (%s) queda fuera del salón, no tiene tamaño o sus esquinas no forman un rectángulo.",
			shape.ShapeID, shape.Label))
	}
	return nil
}

// isClockwiseGridRectangle: the 8 values are the corners of their bounding box, of positive size, in
// the order top-left, top-right, bottom-right, bottom-left (y grows downward) starting at any corner.
func isClockwiseGridRectangle(points []int16) bool {
	minX, maxX := min(points[0], points[2], points[4], points[6]), max(points[0], points[2], points[4], points[6])
	minY, maxY := min(points[1], points[3], points[5], points[7]), max(points[1], points[3], points[5], points[7])
	if minX == maxX || minY == maxY {
		return false
	}
	clockwiseCorners := []int16{minX, minY, maxX, minY, maxX, maxY, minX, maxY}
	for firstCorner := 0; firstCorner < 4; firstCorner++ {
		rotatedCorners := append(slices.Clone(clockwiseCorners[firstCorner*2:]), clockwiseCorners[:firstCorner*2]...)
		if slices.Equal(points, rotatedCorners) {
			return true
		}
	}
	return false
}
