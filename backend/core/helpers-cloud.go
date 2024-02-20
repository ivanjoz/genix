//go:build !local

package core

import (
	"fmt"
)

func Print(Struct any) {
	fmt.Println(Struct)
}

func Logx(style int8, messageInColor string, params ...any) {
	fmt.Println(params...)
}
