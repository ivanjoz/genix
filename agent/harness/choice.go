package harness

import "github.com/cloudwego/eino/schema"

// ChoiceQuestion is what a tool interrupts with when it needs the user to pick (an ambiguous
// product or client name). The chat shows Choices as buttons; the picked Choice.ID comes back as
// the resume answer, and the tool checks it against its own saved state — the browser's answer is
// never trusted as is.
type ChoiceQuestion struct {
	Question string
	Choices  []Choice
}

type Choice struct {
	ID    string
	Label string
}

// Interrupt info is gob-encoded into the checkpoint, so its types must be registered by name.
func init() {
	schema.RegisterName[ChoiceQuestion]("genix_choice_question")
	schema.RegisterName[Choice]("genix_choice")
}
