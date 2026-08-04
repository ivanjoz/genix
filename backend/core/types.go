package core

import (
	"encoding/json"
	"slices"
	"strconv"
)

type ExecArgs struct {
	LambdaName    string `json:"-" cb:"-"`
	FuncToExec    string `json:"fn,omitempty" cb:"-"`
	InvokeType    string `json:"invokeType,omitempty" cb:"-"`
	Param1        int64  `json:"p1,omitempty"`
	Param2        int64  `json:"p2,omitempty"`
	Param3        int64  `json:"p3,omitempty"`
	Param4        int64  `json:"p4,omitempty"`
	Param5        string `json:"p5,omitempty"`
	Param6        string `json:"p6,omitempty"`
	Message       string `json:"ms,omitempty" cb:"-"`
	InvokeAsEvent bool   `json:"-" cb:"-"`
	ParseResponse bool   `json:"-" cb:"-"`
	// Output, not input: whatever the running process reported through AddMessage. Skipped by
	// colbin because the cron executor persists these in the row's own Messages column, and keeping
	// them out of the Params blob prevents the two copies from ever disagreeing. The json tag stays
	// so the messages still survive a Lambda invoke round trip.
	ProcessMessages []string `json:"msg,omitempty" cb:"-"`
}

func (e *ExecArgs) MakeErr(messages ...any) FuncResponse {
	return FuncResponse{Error: Concat(" ", messages...)}
}

func (e *ExecArgs) AddMessage(message string) {
	if !slices.Contains(e.ProcessMessages, message) {
		e.ProcessMessages = append(e.ProcessMessages, message)
	}
}

type FuncResponse struct {
	ElapsedTime int    `json:",omitempty"`
	Message     string `json:",omitempty"`
	Error       string `json:",omitempty"`
	Content     map[string]any
	ContentJson string `json:",omitempty"`
}

type AppRouterType map[string]func(args *HandlerArgs) HandlerResponse

type Int int

func (fi *Int) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		return json.Unmarshal(b, (*int)(fi))
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*fi = Int(i)
	return nil
}
