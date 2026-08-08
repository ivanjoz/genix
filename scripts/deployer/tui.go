package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Las tres secciones de la navegación. Environment elige el archivo de credenciales, Actions
// son los despliegues y Scripts los utilitarios que hoy despacha app.sh.
type navTab int

const (
	tabEnvironment navTab = iota
	tabActions
	tabScripts
)

var navTabLabels = []string{"ENVIROMENT", "ACTIONS", "SCRIPTS"}

// Ancho supuesto hasta que llega el primer WindowSizeMsg, para que el primer render no
// amontone todos los botones en una sola columna.
const defaultTerminalWidth = 110

// Sin foco: es el estado inicial y el que queda al cambiar de pestaña. El indicador de foco
// sólo aparece cuando se navega con el teclado; usando el mouse no se resalta nada.
const noFocus = -1

// deployTUI es el modelo Bubble Tea. El estado marcado de Actions y Scripts vive en los
// punteros a Button, así que sobrevive a los cambios de pestaña.
type deployTUI struct {
	activeTab     navTab
	terminalWidth int
	focusedIndex  int
	confirmed     bool

	environmentPaths    []string
	environmentButtons  []*Button
	selectedEnvironment int

	actions actionScreen
	scripts scriptScreen

	// Se vuelve a invocar cada vez que cambia el entorno: el botón de backend depende del
	// providers.backend del archivo elegido.
	backendProviderFor func(configPath string) string
}

func (m deployTUI) Init() tea.Cmd { return nil }

func (m deployTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		// Las cajas se recalculan con la misma función que dibuja, así que coinciden con lo
		// que hay en pantalla.
		_, tabBoxes, buttonBoxes := m.render()
		if index := buttonAt(tabBoxes, msg.X, msg.Y); index >= 0 {
			return m.switchTab(navTab(index)), nil
		}
		if index := buttonAt(buttonBoxes, msg.X, msg.Y); index >= 0 {
			return m.activate(index)
		}
	}

	return m, nil
}

func (m deployTUI) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	buttonCount := len(flattenButtons(m.currentSections()))

	switch msg.String() {
	case "ctrl+c", "esc", "q":
		// Sale sin confirmar: main lo interpreta como cancelación.
		return m, tea.Quit

	case "tab":
		return m.switchTab((m.activeTab + 1) % navTab(len(navTabLabels))), nil

	case "shift+tab":
		return m.switchTab((m.activeTab + navTab(len(navTabLabels)) - 1) % navTab(len(navTabLabels))), nil

	case "left", "up", "h", "k":
		if buttonCount > 0 {
			m.focusedIndex = m.moveFocus(-1, buttonCount)
		}

	case "right", "down", "l", "j":
		if buttonCount > 0 {
			m.focusedIndex = m.moveFocus(1, buttonCount)
		}

	case " ", "space":
		if m.activeTab != tabEnvironment && m.focusedIndex != noFocus {
			return m.activate(m.focusedIndex)
		}

	case "enter":
		if m.activeTab == tabEnvironment {
			if m.focusedIndex != noFocus {
				return m.activate(m.focusedIndex)
			}
			return m, nil
		}
		m.confirmed = true
		return m, tea.Quit
	}

	return m, nil
}

// moveFocus estrena el foco en el primer botón: hasta la primera tecla de navegación no hay
// nada resaltado.
func (m deployTUI) moveFocus(delta, buttonCount int) int {
	if m.focusedIndex == noFocus {
		return 0
	}
	return (m.focusedIndex + delta + buttonCount) % buttonCount
}

// switchTab reinicia el foco porque los índices de botón no significan lo mismo en otra
// pestaña.
func (m deployTUI) switchTab(tab navTab) deployTUI {
	m.activeTab = tab
	m.focusedIndex = noFocus
	return m
}

// activate es lo que hacen tanto el click como la tecla. No toca el foco a propósito: hacer
// click no debe dejar un botón resaltado.
func (m deployTUI) activate(index int) (tea.Model, tea.Cmd) {
	buttons := flattenButtons(m.currentSections())
	if index >= len(buttons) || buttons[index].Disabled {
		return m, nil
	}

	if m.activeTab == tabEnvironment {
		// La elección de entorno es exclusiva y lleva directo a las acciones, que es lo que
		// hay que hacer después de elegirlo.
		for _, button := range m.environmentButtons {
			button.Selected = false
		}
		buttons[index].Selected = true
		m.selectedEnvironment = index
		m.actions = newActionScreen(m.backendProviderFor(m.environmentPaths[index]))
		return m.switchTab(tabActions), nil
	}

	// Los botones son punteros, así que la marca sobrevive a la copia por valor del modelo.
	buttons[index].Selected = !buttons[index].Selected
	return m, nil
}

func (m deployTUI) currentSections() []buttonSection {
	switch m.activeTab {
	case tabEnvironment:
		return []buttonSection{{title: "Archivo de credenciales " + strings.Repeat("─", 6), buttons: m.environmentButtons}}
	case tabScripts:
		return m.scripts.sections
	default:
		return m.actions.sections
	}
}

func (m deployTUI) helpText() string {
	if m.activeTab == tabEnvironment {
		return "tab cambia de sección · ↑↓←→ mover · enter o click elegir entorno · q salir"
	}
	return "tab cambia de sección · ↑↓←→ mover · espacio o click marcar · enter ejecutar · q salir"
}

// render arma la vista completa y devuelve las cajas de las pestañas y de los botones en
// coordenadas absolutas, que es lo que el hit-test del mouse necesita.
func (m deployTUI) render() (lines []string, tabBoxes, buttonBoxes []buttonBox) {
	terminalWidth := m.terminalWidth
	if terminalWidth <= 0 {
		terminalWidth = defaultTerminalWidth
	}

	lines = append(lines, headerStyle.Render("=== GENIX DEPLOYMENT & UTILS ===")+
		helpStyle.Render("   Entorno: "+filepath.Base(m.environmentPaths[m.selectedEnvironment])))

	navLines, tabBoxes := layoutTabs(navTabLabels, int(m.activeTab), terminalWidth)
	offsetBoxes(tabBoxes, len(lines))
	lines = append(lines, navLines...)

	// Sin línea en blanco entre las pestañas y el cuerpo: la divisoria que cierra la barra ya
	// hace de separación, y una fila vacía encima del primer título se lee como un hueco.
	bodyLines, buttonBoxes := layoutSections(m.currentSections(), terminalWidth, m.focusedIndex)
	offsetBoxes(buttonBoxes, len(lines))
	lines = append(lines, bodyLines...)

	lines = append(lines, helpStyle.Render(m.helpText()))
	return lines, tabBoxes, buttonBoxes
}

func (m deployTUI) View() tea.View {
	lines, _, _ := m.render()

	view := tea.NewView(strings.Join(lines, "\n"))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	return view
}

// runDeployTUI muestra la interfaz y devuelve el entorno elegido junto con las acciones y los
// scripts marcados. ok=false significa que el usuario abortó.
func runDeployTUI(environmentPaths []string, backendProviderFor func(string) string) (configFile string, selectedIDs []int, selectedScripts []string, ok bool) {
	model := deployTUI{
		terminalWidth:      defaultTerminalWidth,
		focusedIndex:       noFocus,
		environmentPaths:   environmentPaths,
		backendProviderFor: backendProviderFor,
		scripts:            newScriptScreen(),
		actions:            newActionScreen(backendProviderFor(environmentPaths[0])),
	}
	for index, path := range environmentPaths {
		// El primero viene marcado porque config.toml es el entorno por defecto.
		model.environmentButtons = append(model.environmentButtons,
			&Button{Label: filepath.Base(path), Selected: index == 0})
	}

	// Con un solo archivo de configuración no hay nada que elegir: se abre en las acciones.
	if len(environmentPaths) == 1 {
		model.activeTab = tabActions
	}

	result, err := tea.NewProgram(model).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error en la interfaz: %v\n", err)
		os.Exit(1)
	}

	final := result.(deployTUI)
	if !final.confirmed {
		return "", nil, nil, false
	}

	for index, button := range flattenButtons(final.actions.sections) {
		if button.Selected {
			selectedIDs = append(selectedIDs, final.actions.actionIDs[index])
		}
	}
	for index, button := range flattenButtons(final.scripts.sections) {
		if button.Selected {
			selectedScripts = append(selectedScripts, final.scripts.scriptKeys[index])
		}
	}
	return final.environmentPaths[final.selectedEnvironment], selectedIDs, selectedScripts, true
}
