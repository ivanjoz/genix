package core

import (
	"encoding/json"
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
}

func (e *ExecArgs) MakeErr(msgs ...any) FuncResponse {
	return FuncResponse{Error: Concat(" ", msgs...)}
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
