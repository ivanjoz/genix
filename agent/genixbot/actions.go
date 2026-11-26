package genixbot

import (
	"fmt"
	"regexp"
	"strings"
)

// Action is one `hacer` line: `[! ]<id> <método> [argumento]`. The argument is the rest of the line.
// "!" marks a save/send/delete the user confirmed in their last message.
type Action struct {
	Confirmed bool
	ID        string // plain ("79") or composite ("75:300"), copied from the outline
	Method    string
	Argument  string
}

var actionLinePattern = regexp.MustCompile(`^(! )?([0-9]+(?::[0-9]+)?) (setValue|select|search|selectRow|remove|click|open|close)(?: (.+))?$`)

// ParseActions reads the `acciones` argument of `hacer`, one action per line.
func ParseActions(acciones string) ([]Action, error) {
	actions := []Action{}
	for _, line := range strings.Split(strings.TrimSpace(acciones), "\n") {
		match := actionLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			return nil, fmt.Errorf("línea %q no sigue `id método argumento`", line)
		}
		actions = append(actions, Action{Confirmed: match[1] != "", ID: match[2], Method: match[3], Argument: match[4]})
	}
	return actions, nil
}

// String renders the action back as its `hacer` line, for the turn's action log.
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
