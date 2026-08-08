package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// Ancho mínimo de celda: por debajo de esto se reduce el número de columnas. Con 34 la
	// etiqueta más larga entra en dos líneas en las anchuras de terminal habituales.
	minCellWidth = 34
	// Separación horizontal entre botones, aplicada como margen derecho.
	buttonGap = 1
	// Las etiquetas se parten en varias líneas; lo que pase de aquí se recorta con "…".
	maxButtonContentLines = 2
	// Ancho reservado a cada lado del texto del botón. A la izquierda vive la marca de
	// selección; a la derecha se deja el mismo hueco para que el texto quede centrado
	// respecto del botón y no del espacio que sobra a la derecha de la marca.
	buttonMarkGutter = 2
	// Trazo fino dibujado a un tercio de la altura de su celda (HORIZONTAL SCAN LINE-3): más
	// abajo que ▔, que quedaba pegado a las etiquetas, y más arriba que ─ o ━, que se dibujan
	// a media altura. Cae dentro del medio bloque inferior de la pestaña activa, así que el
	// empalme con ella está garantizado por geometría.
	tabRuleCharacter = "⎻"
	// El borde superior es media celda, para que la pestaña no ocupe una fila entera de más.
	tabTopEdgeCharacter = "▄"
	// Media celda desde arriba: el pie de la pestaña activa baja hasta pasar la divisoria, de
	// modo que la línea desemboca en el bloque blanco en vez de cortarse contra él.
	tabBottomEdgeCharacter = "▀"
	// Espacio a cada lado de la etiqueta dentro de la pestaña.
	tabHorizontalPadding = 2
)

// Blanco fijo de la paleta de 256 colores, no el "blanco brillante" ANSI, que cada tema del
// terminal remapea a su gusto.
var tabWhite = lipgloss.Color("255")

// El prefijo de marcado ocupa siempre 2 celdas, esté o no seleccionado el botón, para que
// marcar uno nunca cambie el ancho del texto disponible.
const (
	selectedMark   = "✓ "
	unselectedMark = "  "
)

var (
	buttonBaseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			MarginRight(buttonGap)

	// El foco NO pinta el borde completo: eso se leía como "seleccionado". Sólo engrosa el
	// borde inferior y lo pinta de azul.
	focusedBorder = heavyBottomBorder()

	normalColor      = lipgloss.Color("240")
	focusedLineColor = lipgloss.Color("39")
	selectedColor    = lipgloss.Color("42")
	disabledColor    = lipgloss.Color("238")

	sectionTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	headerStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// La pestaña activa se marca invirtiendo el color (fondo blanco, texto negro): se lee de
	// un vistazo, que es lo que el subrayado no lograba. La etiqueta ocupa una sola fila; el
	// alto lo dan las medias celdas de arriba y abajo.
	activeTabStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(tabWhite).Bold(true).Padding(0, tabHorizontalPadding)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, tabHorizontalPadding)
	// Los bordes de la pestaña y la divisoria son el mismo blanco, que es lo que hace que se
	// lean como una sola pieza.
	tabRuleStyle = lipgloss.NewStyle().Foreground(tabWhite)
)

func heavyBottomBorder() lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.Bottom, border.BottomLeft, border.BottomRight = "━", "┗", "┛"
	return border
}

// Button es el único componente interactivo del TUI. Todos los botones de una pantalla se
// dibujan con el mismo ancho y alto: la grilla y el hit-test dependen de eso.
type Button struct {
	Label    string
	Selected bool
	Disabled bool
}

// Render dibuja el botón ocupando exactamente cellWidth columnas (margen incluido) y
// contentLines líneas de texto.
func (b Button) Render(cellWidth, contentLines int, focused bool) string {
	mark := unselectedMark
	if b.Selected {
		mark = selectedMark
	}

	style := buttonBaseStyle.Width(cellWidth - buttonGap)
	switch {
	case b.Disabled:
		style = style.BorderForeground(disabledColor).Foreground(disabledColor)
	case b.Selected:
		style = style.BorderForeground(selectedColor).Foreground(selectedColor)
	default:
		style = style.BorderForeground(normalColor)
	}

	// El foco se superpone a cualquier estado y va último, para que su color gane sobre el
	// del borde inferior que hubiera puesto el estado.
	if focused && !b.Disabled {
		style = style.BorderStyle(focusedBorder).BorderBottomForeground(focusedLineColor)
	}

	return style.Render(fitLabel(b.Label, mark, textWidthFor(cellWidth), contentLines))
}

// contentWidthFor descuenta del ancho de celda el margen, los dos bordes y los dos espacios
// de padding, que es lo que queda para el texto.
func contentWidthFor(cellWidth int) int {
	contentWidth := cellWidth - buttonGap - 4
	if contentWidth < 4 {
		return 4
	}
	return contentWidth
}

// textWidthFor descuenta además las dos canaletas: el texto se centra en lo que queda, que
// está centrado respecto del botón entero.
func textWidthFor(cellWidth int) int {
	textWidth := contentWidthFor(cellWidth) - 2*buttonMarkGutter
	if textWidth < 4 {
		return 4
	}
	return textWidth
}

// fitLabel centra la etiqueta en el botón, horizontal y verticalmente, y la deja siempre con
// la misma cantidad de líneas para que todos los botones de la grilla midan igual. La marca de
// selección vive en la canaleta izquierda y a la derecha se reserva el mismo hueco: si la
// marca formara parte del texto centrado, lo correría media canaleta hacia la derecha.
func fitLabel(label, mark string, textWidth, contentLines int) string {
	lines := strings.Split(wrapToWidth(label, textWidth), "\n")
	for index := range lines {
		lines[index] = strings.TrimSpace(lines[index])
	}

	if len(lines) > contentLines {
		lines = lines[:contentLines]
		lines[contentLines-1] = truncateWithEllipsis(lines[contentLines-1], textWidth)
	}

	// Las líneas sobrantes se reparten arriba para centrar en vertical. Con un solo sobrante
	// no hay centro exacto posible: la celda es la unidad mínima.
	topPadding := (contentLines - len(lines)) / 2
	emptyGutter := strings.Repeat(" ", buttonMarkGutter)
	centeredText := lipgloss.NewStyle().Width(textWidth).Align(lipgloss.Center)

	rendered := make([]string, contentLines)
	for index := range contentLines {
		textIndex := index - topPadding
		if textIndex < 0 || textIndex >= len(lines) {
			rendered[index] = strings.Repeat(" ", textWidth+2*buttonMarkGutter)
			continue
		}
		// La marca acompaña a la primera línea de texto, no al tope del botón.
		gutter := emptyGutter
		if textIndex == 0 {
			gutter = mark
		}
		rendered[index] = gutter + centeredText.Render(lines[textIndex]) + emptyGutter
	}

	return strings.Join(rendered, "\n")
}

func wrapToWidth(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

func truncateWithEllipsis(line string, width int) string {
	runes := []rune(strings.TrimRight(line, " "))
	if len(runes) >= width {
		runes = runes[:width-1]
	}
	return string(runes) + "…"
}

// buttonSection es un grupo de botones bajo un encabezado, equivalente a las líneas
// "Publicar Código ----" del menú original.
type buttonSection struct {
	title   string
	buttons []*Button
}

// buttonBox es el rectángulo que ocupa un botón en pantalla, relativo a la primera línea
// devuelta por layoutSections.
type buttonBox struct{ x, y, width, height int }

// layoutSections es la ÚNICA fuente de verdad del layout: devuelve en una sola pasada las
// líneas dibujadas y la caja de cada botón, así que un click no puede quedar desalineado de
// lo que se ve. La grilla es de ancho uniforme: se elige el número de columnas que entra en
// la terminal y todos los botones miden lo mismo, partiendo el texto largo en dos líneas.
func layoutSections(sections []buttonSection, terminalWidth, focusedIndex int) ([]string, []buttonBox) {
	columns := chooseColumns(sections, terminalWidth)
	cellWidth := terminalWidth / columns
	contentLines := min(maxLabelLines(sections, textWidthFor(cellWidth)), maxButtonContentLines)
	buttonRowHeight := contentLines + 2 // + bordes superior e inferior

	var lines []string
	var boxes []buttonBox
	buttonIndex := 0

	for _, section := range sections {
		if section.title != "" {
			lines = append(lines, sectionTitleStyle.Render(section.title))
		}

		var rowBlocks []string
		rowY := len(lines)

		flushRow := func() {
			if len(rowBlocks) == 0 {
				return
			}
			joined := lipgloss.JoinHorizontal(lipgloss.Top, rowBlocks...)
			lines = append(lines, strings.Split(joined, "\n")...)
			rowBlocks = nil
			rowY = len(lines)
		}

		for _, button := range section.buttons {
			if len(rowBlocks) == columns {
				flushRow()
			}
			boxes = append(boxes, buttonBox{
				x: len(rowBlocks) * cellWidth, y: rowY,
				width: cellWidth, height: buttonRowHeight,
			})
			rowBlocks = append(rowBlocks, button.Render(cellWidth, contentLines, buttonIndex == focusedIndex))
			buttonIndex++
		}

		flushRow()
		lines = append(lines, "")
	}

	return lines, boxes
}

// chooseColumns elige la grilla más ancha en la que ninguna etiqueta necesite recortarse. En
// una terminal ancha entrarían más columnas, pero las etiquetas largas perderían texto; es
// preferible una columna menos y que se lean enteras.
func chooseColumns(sections []buttonSection, terminalWidth int) int {
	maxColumns := terminalWidth / minCellWidth
	if maxColumns < 1 {
		return 1
	}
	for columns := maxColumns; columns > 1; columns-- {
		if maxLabelLines(sections, textWidthFor(terminalWidth/columns)) <= maxButtonContentLines {
			return columns
		}
	}
	return 1
}

// maxLabelLines es la cantidad de líneas que necesita la etiqueta más larga a un ancho dado.
// Mide sólo la etiqueta: la marca de selección ya no ocupa lugar en el texto, vive aparte en
// la canaleta, así que incluirla sobreestimaría el alto.
func maxLabelLines(sections []buttonSection, textWidth int) int {
	lineCount := 1
	for _, section := range sections {
		for _, button := range section.buttons {
			needed := lipgloss.Height(wrapToWidth(button.Label, textWidth))
			if needed > lineCount {
				lineCount = needed
			}
		}
	}
	return lineCount
}

// layoutTabs dibuja la barra de navegación con el mismo criterio que layoutSections: las
// cajas salen del mismo recorrido que el render, así que el click cae donde se ve.
func layoutTabs(labels []string, activeIndex, terminalWidth int) ([]string, []buttonBox) {
	var topEdges, labelBlocks []string
	activeBox := buttonBox{}
	var boxes []buttonBox
	x := 0

	// Todas las pestañas miden lo mismo: la más larga manda y el resto se centra en ese ancho.
	uniformWidth := 0
	for _, label := range labels {
		if width := lipgloss.Width(label) + 2*tabHorizontalPadding; width > uniformWidth {
			uniformWidth = width
		}
	}

	for index, label := range labels {
		style := inactiveTabStyle
		topEdge := ""
		if index == activeIndex {
			style = activeTabStyle
		}

		labelBlock := style.Width(uniformWidth).Align(lipgloss.Center).Render(label)
		blockWidth := lipgloss.Width(labelBlock)
		if index == activeIndex {
			topEdge = tabRuleStyle.Render(strings.Repeat(tabTopEdgeCharacter, blockWidth))
		} else {
			topEdge = strings.Repeat(" ", blockWidth)
		}

		// El click vale en el borde superior y en la fila de la etiqueta.
		box := buttonBox{x: x, y: 0, width: blockWidth, height: 2}
		if index == activeIndex {
			activeBox = box
		}

		boxes = append(boxes, box)
		topEdges = append(topEdges, topEdge)
		labelBlocks = append(labelBlocks, labelBlock)
		x += blockWidth
	}

	return []string{
		lipgloss.JoinHorizontal(lipgloss.Top, topEdges...),
		lipgloss.JoinHorizontal(lipgloss.Top, labelBlocks...),
		tabRuleStyle.Render(tabRuleLine(activeBox, terminalWidth)),
	}, boxes
}

// tabRuleLine dibuja la divisoria y, en el tramo de la pestaña activa, el pie de la pestaña:
// medio bloque que baja hasta pasar la altura de la línea, para que ambos se lean como una
// sola pieza sin que la línea engorde de lado a lado.
func tabRuleLine(activeBox buttonBox, terminalWidth int) string {
	var rule strings.Builder
	for column := range terminalWidth {
		if column >= activeBox.x && column < activeBox.x+activeBox.width {
			rule.WriteString(tabBottomEdgeCharacter)
		} else {
			rule.WriteString(tabRuleCharacter)
		}
	}
	return rule.String()
}

// offsetBoxes corre las cajas hacia abajo según lo que ya se haya dibujado encima, para que
// queden en coordenadas absolutas de la vista.
func offsetBoxes(boxes []buttonBox, deltaY int) {
	for index := range boxes {
		boxes[index].y += deltaY
	}
}

// buttonAt resuelve qué botón hay bajo una coordenada del mouse, o -1 si no hay ninguno.
func buttonAt(boxes []buttonBox, x, y int) int {
	for index, box := range boxes {
		if x >= box.x && x < box.x+box.width && y >= box.y && y < box.y+box.height {
			return index
		}
	}
	return -1
}

// flattenButtons recorre las secciones en el mismo orden que layoutSections, de modo que el
// índice de un botón sirve indistintamente para el foco, el hit-test y los IDs de acción.
func flattenButtons(sections []buttonSection) []*Button {
	var buttons []*Button
	for _, section := range sections {
		buttons = append(buttons, section.buttons...)
	}
	return buttons
}
