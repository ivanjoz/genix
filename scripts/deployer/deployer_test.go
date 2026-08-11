package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// El hit-test del mouse depende de que las cajas de layoutSections caigan exactamente sobre
// el borde izquierdo dibujado. Se verifica en varias anchuras porque son las que cambian el
// reparto de botones por fila.
func TestBoxesMatchRenderedBorders(t *testing.T) {
	for _, backendProvider := range []string{"aws", "none", "cloudflare"} {
		for _, terminalWidth := range []int{60, 80, 110, 200} {
			screen := newActionScreen(backendProvider)
			lines, boxes := layoutSections(screen.sections, terminalWidth, 3)

			if len(boxes) != len(screen.actionIDs) {
				t.Fatalf("%s/%d: %d cajas para %d acciones", backendProvider, terminalWidth, len(boxes), len(screen.actionIDs))
			}

			for index, box := range boxes {
				plainLine := []rune(ansiPattern.ReplaceAllString(lines[box.y], ""))
				if box.x >= len(plainLine) || plainLine[box.x] != '╭' {
					t.Errorf("%s/%d: el botón %d (x=%d, y=%d) no cae sobre su borde",
						backendProvider, terminalWidth, index, box.x, box.y)
				}
			}
		}
	}
}

// La grilla debe ser uniforme (mismo ancho y alto en todos los botones) y no debe recortar
// ninguna etiqueta en las anchuras de terminal usables.
func TestGridIsUniformAndUntruncated(t *testing.T) {
	for _, backendProvider := range []string{"aws", "none", "cloudflare"} {
		for terminalWidth := 40; terminalWidth <= 200; terminalWidth += 5 {
			lines, boxes := layoutSections(newActionScreen(backendProvider).sections, terminalWidth, noFocus)

			for _, box := range boxes {
				if box.width != boxes[0].width || box.height != boxes[0].height {
					t.Fatalf("ancho %d: botón %v no mide igual que %v", terminalWidth, box, boxes[0])
				}
			}
			for _, line := range lines {
				if strings.Contains(line, "…") {
					t.Errorf("%s/%d: etiqueta recortada: %s", backendProvider, terminalWidth, line)
				}
			}
		}
	}
}

// Sin foco no debe dibujarse el indicador: usando el mouse no se resalta ningún botón.
func TestNoFocusDrawsNoIndicator(t *testing.T) {
	sections := newActionScreen("none").sections

	withoutFocus, _ := layoutSections(sections, 145, noFocus)
	if strings.Contains(strings.Join(withoutFocus, "\n"), "━") {
		t.Error("se dibujó la línea de foco sin haber foco")
	}

	withFocus, _ := layoutSections(sections, 145, 0)
	if !strings.Contains(strings.Join(withFocus, "\n"), "━") {
		t.Error("no se dibujó la línea de foco del botón enfocado")
	}
}

func TestParseArguments(t *testing.T) {
	cases := []struct {
		arguments         []string
		expectedActions   string
		expectedScripts   string
		expectedArguments string
		expectedCompany   string
	}{
		{[]string{"6", "9"}, "[6 9]", "[]", "[]", ""},
		{[]string{"11", "42"}, "[11]", "[]", "[]", "42"},
		{[]string{"1,2,3"}, "[1 2 3]", "[]", "[]", ""},
		{[]string{"99"}, "[]", "[]", "[]", "99"},
		{[]string{"check_tables"}, "[]", "[check_tables]", "[]", ""},
		{[]string{"1", "check_tables"}, "[1]", "[check_tables]", "[]", ""},
		// Los scripts que piden argumentos se invocan igual que en app.sh.
		{[]string{"edit", "product_inventory", "category:string"}, "[]", "[edit]", "[product_inventory category:string]", ""},
		{[]string{"search_documentation", "-question", "arqueo", "-limit", "3"}, "[]", "[search_documentation]", "[-question arqueo -limit 3]", ""},
	}

	for _, testCase := range cases {
		selection := parseArguments(testCase.arguments)
		got := fmt.Sprintf("%v|%v|%v|%s", selection.actionIDs, selection.scriptKeys,
			selection.scriptArguments, selection.companyID)
		want := fmt.Sprintf("%s|%s|%s|%s", testCase.expectedActions, testCase.expectedScripts,
			testCase.expectedArguments, testCase.expectedCompany)
		if got != want {
			t.Errorf("%v =>\n  got  %s\n  want %s", testCase.arguments, got, want)
		}
	}
}

// Cada script del menú debe ser ejecutable por su clave, y las claves deben coincidir con las
// que despachaba app.sh.
func TestScriptsAreResolvable(t *testing.T) {
	screen := newScriptScreen()
	if len(screen.scriptKeys) != len(deployScripts) {
		t.Fatalf("%d botones para %d scripts", len(screen.scriptKeys), len(deployScripts))
	}
	for _, scriptKey := range screen.scriptKeys {
		if findScript(scriptKey) == nil {
			t.Errorf("el script %q no se puede resolver", scriptKey)
		}
	}
}

// El backend debe quedar SIEMPRE en un solo botón, elegido por el BACKEND_PROVIDER.
func TestBackendButtonDependsOnProvider(t *testing.T) {
	expectedByProvider := map[string]string{
		"aws":        "Backend (AWS Cloud)",
		"none":       "Backend (VPS)",
		"":           "Backend (VPS)",
		"cloudflare": "Backend (cloudflare no soportado)",
	}

	for backendProvider, expectedLabel := range expectedByProvider {
		screen := newActionScreen(backendProvider)
		buttons := flattenButtons(screen.sections)

		var backendButtons []*Button
		for index, actionID := range screen.actionIDs {
			if actionID == 2 || actionID == 3 {
				backendButtons = append(backendButtons, buttons[index])
			}
		}

		if len(backendButtons) != 1 {
			t.Fatalf("provider %q: %d botones de backend, se esperaba 1", backendProvider, len(backendButtons))
		}
		if backendButtons[0].Label != expectedLabel {
			t.Errorf("provider %q => %q, se esperaba %q", backendProvider, backendButtons[0].Label, expectedLabel)
		}
		if shouldBeDisabled := backendProvider == "cloudflare"; backendButtons[0].Disabled != shouldBeDisabled {
			t.Errorf("provider %q: Disabled=%v, se esperaba %v", backendProvider, backendButtons[0].Disabled, shouldBeDisabled)
		}
	}
}

// Cada acción del menú debe estar en executionOrder, o quedaría inalcanzable.
func TestEveryActionIsExecutable(t *testing.T) {
	for _, action := range deployActions {
		found := false
		for _, actionID := range executionOrder {
			found = found || actionID == action.id
		}
		if !found {
			t.Errorf("la acción %d (%s) no está en executionOrder", action.id, action.label)
		}
	}
}
