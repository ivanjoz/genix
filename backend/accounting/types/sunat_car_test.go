package types

import "testing"

func TestBuildCARPadsEveryPartToItsWidth(t *testing.T) {
	car := BuildCAR("20123456789", "01", "F001", 15)

	if len(car) != CARWidth {
		t.Fatalf("el CAR mide %d caracteres, deben ser %d: %q", len(car), CARWidth, car)
	}
	// RUC(11) + tipo(2) + serie(4) + número(10), el número con ceros a la izquierda.
	if car != "20123456789"+"01"+"F001"+"0000000015" {
		t.Errorf("CAR inesperado: %q", car)
	}
}

// Anexo C: a part longer than its slot keeps its rightmost characters, because those are the
// ones that tell two documents apart.
func TestBuildCARKeepsTheRightmostCharactersWhenTooLong(t *testing.T) {
	// The número never overflows its slot: a correlativo is an int32, whose largest value is
	// exactly ten digits. Only the two free-text parts can be too long.
	car := BuildCAR("9920123456789", "01", "FACT01", 2147483647)

	if car[:11] != "20123456789" {
		t.Errorf("esperaba los 11 dígitos de la derecha del emisor, obtuve %q", car[:11])
	}
	if car[13:17] != "CT01" {
		t.Errorf("esperaba los 4 caracteres de la derecha de la serie, obtuve %q", car[13:17])
	}
	if len(car) != CARWidth {
		t.Errorf("el CAR mide %d caracteres, deben ser %d", len(car), CARWidth)
	}
}

func TestBuildCARFillsAnEmptySeriesWithZeros(t *testing.T) {
	car := BuildCAR("20123456789", "01", "", 15)

	if car[13:17] != "0000" {
		t.Errorf("una serie vacía se completa con ceros, obtuve %q", car[13:17])
	}
}
