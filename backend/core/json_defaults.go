package core

import "github.com/bytedance/sonic"

// Every JSON the backend writes omits empty fields, so no struct tag needs `omitempty`.
// This relies on the ivanjoz/sonic fork (see the replace in go.mod). It runs here, in the package
// every other package imports, because sonic caches the compiled encoder of each type on first use.
// The one exception is a pointer to a number (*int32, *float64...): nil is sent as null and a pointer
// to 0 is sent as 0, so a field that must keep its zero value has to be a *number.
func init() {
	sonic.SetOmitEmptyDefault(true)
}
