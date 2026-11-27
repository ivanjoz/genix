package genixbot

import (
	"fmt"
	"regexp"
	"strings"
)

// Action is one `act` line: `[! ]<id> <método> [argumento]`. The argument is the rest of the line.
// "!" marks a save/send/delete the user asked for or confirmed in their last message.
type Action struct {
	Confirmed bool
	ID        string // plain ("79") or composite ("75:300"), copied from the outline
	Method    string
	Argument  string
}

var actionLinePattern = regexp.MustCompile(`^(! )?([0-9]+(?::[0-9]+)?) (setValue|select|search|selectRow|remove|click|open|close)(?: (.+))?$`)

// ParseActions reads the `actions` argument of `act`, one action per line.
func ParseActions(actionLines string) ([]Action, error) {
	actions := []Action{}
	for _, line := range strings.Split(strings.TrimSpace(actionLines), "\n") {
		match := actionLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			return nil, fmt.Errorf("línea %q no sigue `id método argumento`", line)
		}
		actions = append(actions, Action{Confirmed: match[1] != "", ID: match[2], Method: match[3], Argument: match[4]})
	}
	return actions, nil
}

// String renders the action back as its `act` line, for the turn's action log.
func (action Action) String() string {
	line := action.ID + " " + action.Method
	if action.Confirmed {
		line = "! " + line
	}
	if action.Argument != "" {
		line += " " + action.Argument
	}
	return line
}

// Action errors as the model reads them. Genix's browser adapter (agent/genixbot_turn.go) and the
// training simulator (genix-local-ai simpage) both use these, so training shows the texts Genix sends.
const ErrorUnknownID = "el id no está en la página"

// ErrorNoMethod: the component exists but has no such method (setValue on a Checkbox).
func ErrorNoMethod(componentType, method string) string {
	return componentType + " no tiene el método " + method
}

// SearchShown caps the matches a component's search method shows: a short list keeps the prompt small, and "+N más"
// tells the model to narrow the search.
const SearchShown = 10

// SearchMatches renders a search's matches ("[id] label") the way `act` shows them; an empty search
// says so instead of showing nothing.
func SearchMatches(options []string) string {
	if len(options) == 0 {
		return "sin resultados"
	}
	if len(options) <= SearchShown {
		return strings.Join(options, ", ")
	}
	return strings.Join(options[:SearchShown], ", ") + fmt.Sprintf(" (+%d más)", len(options)-SearchShown)
}
