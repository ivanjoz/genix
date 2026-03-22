package core

import (
	"encoding/json"
	"strconv"
)

type ExecArgs struct {
	LambdaName    string `json:"-" cbor:"-"`
	FuncToExec    string `json:"fn,omitempty" cbor:"-"`
	InvokeType    string `json:"invokeType,omitempty" cbor:"-"`
	Param1        int64  `json:"p1,omitempty" cbor:"1,kayasint,omitempty"`
	Param2        int64  `json:"p2,omitempty" cbor:"2,kayasint,omitempty"`
	Param3        int64  `json:"p3,omitempty" cbor:"3,kayasint,omitempty"`
	Param4        int64  `json:"p4,omitempty" cbor:"4,kayasint,omitempty"`
	Param5        string `json:"p5,omitempty" cbor:"5,kayasint,omitempty"`
	Param6        string `json:"p6,omitempty" cbor:"6,kayasint,omitempty"`
	Message       string `json:"ms,omitempty"  cbor:"-"`
	InvokeAsEvent bool   `json:"-" cbor:"-"`
	ParseResponse bool   `json:"-" cbor:"-"`
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
