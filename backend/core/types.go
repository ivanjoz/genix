package core

import (
	"encoding/json"
	"strconv"
)

type ExecArgs struct {
	Params  map[string]string
	Param2  int
	Param3  string
	Message string
}

type FuncResponse struct {
	Message string
	Error   string
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
