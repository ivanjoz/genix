//go:build local

package core

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/kr/pretty"
)

func Print(Struct any) {
	pretty.Println(Struct)
}

func Logx(style int8, messageInColor string, params ...any) {
	var c *color.Color

	if style == 1 {
		c = color.New(color.FgCyan, color.Bold)
	} else if style == 2 {
		c = color.New(color.FgGreen, color.Bold)
	} else if style == 3 {
		c = color.New(color.FgYellow, color.Bold)
	} else if style == 4 {
		c = color.New(color.FgBlue, color.Bold)
	} else if style == 5 {
		c = color.New(color.FgRed, color.Bold)
	} else if style == 6 {
		c = color.New(color.FgMagenta, color.Bold)
	}

	c.Print(messageInColor)
	if len(params) > 0 {
		fmt.Print(" | ")
		for _, e := range params {
			fmt.Print(e)
			fmt.Print(" ")
		}
		Log("")
	}
}
