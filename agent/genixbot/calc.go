package genixbot

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Calculate evaluates a `calc` expression: decimal numbers, + - * / ^,
// parentheses and unary minus. The result is rounded half away from zero to 2 decimals.
func Calculate(expression string) (float64, error) {
	parser := &calcParser{text: strings.ReplaceAll(expression, " ", "")}
	if parser.text == "" {
		return 0, fmt.Errorf("empty expression")
	}
	value, err := parser.sum()
	if err == nil && parser.position < len(parser.text) {
		err = fmt.Errorf("unexpected %q", parser.text[parser.position:])
	}
	if err == nil && (math.IsInf(value, 0) || math.IsNaN(value)) {
		err = fmt.Errorf("division by zero")
	}
	if err != nil {
		return 0, err
	}
	return math.Round(value*100) / 100, nil
}

// CalculateResult is the tool response of `calc`.
func CalculateResult(expression string) string {
	value, err := Calculate(expression)
	if err != nil {
		return "Error | expresión no válida: " + err.Error()
	}
	return fmt.Sprintf("Resultado | %s = %.2f", strings.TrimSpace(expression), value)
}

type calcParser struct {
	text     string
	position int
}

func (parser *calcParser) peek() byte {
	if parser.position < len(parser.text) {
		return parser.text[parser.position]
	}
	return 0
}

func (parser *calcParser) sum() (float64, error) {
	left, err := parser.product()
	for err == nil && (parser.peek() == '+' || parser.peek() == '-') {
		operator := parser.peek()
		parser.position++
		var right float64
		if right, err = parser.product(); operator == '+' {
			left += right
		} else {
			left -= right
		}
	}
	return left, err
}

func (parser *calcParser) product() (float64, error) {
	left, err := parser.power()
	for err == nil && (parser.peek() == '*' || parser.peek() == '/') {
		operator := parser.peek()
		parser.position++
		var right float64
		if right, err = parser.power(); operator == '*' {
			left *= right
		} else {
			left /= right
		}
	}
	return left, err
}

func (parser *calcParser) power() (float64, error) {
	base, err := parser.unary()
	if err == nil && parser.peek() == '^' {
		parser.position++
		var exponent float64
		exponent, err = parser.power() // right-associative
		base = math.Pow(base, exponent)
	}
	return base, err
}

func (parser *calcParser) unary() (float64, error) {
	if parser.peek() == '-' {
		parser.position++
		value, err := parser.unary()
		return -value, err
	}
	if parser.peek() == '(' {
		parser.position++
		value, err := parser.sum()
		if err == nil && parser.peek() != ')' {
			err = fmt.Errorf("missing )")
		}
		parser.position++
		return value, err
	}
	start := parser.position
	for parser.position < len(parser.text) && (parser.text[parser.position] >= '0' && parser.text[parser.position] <= '9' || parser.text[parser.position] == '.') {
		parser.position++
	}
	if start == parser.position {
		return 0, fmt.Errorf("expected a number at %q", parser.text[start:])
	}
	return strconv.ParseFloat(parser.text[start:parser.position], 64)
}
