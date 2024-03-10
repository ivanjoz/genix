package types

func ConcatInt64(num1, num2 int64) int64 {
	if num1 == 0 {
		return 0
	}
	return num1*10_000_000_000 + num2
}
