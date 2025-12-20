package serialize

import (
	"fmt"
)

// Demo structures for testing
type DemoStruct2 struct {
	Property1 string `json:"property1"`
	Count     int32  `json:"count"`
}

type DemoStruct struct {
	Name    string      `json:"name"`
	Age     int32       `json:",omitempty"`
	Decimal float64     `json:"decimal"`
	AnArray []int32     `json:"anArray"`
	Demo    DemoStruct2 `json:"demo"`
	Demo2   *DemoStruct2
}

func Test() {
	// Demo data to show field optimization:
	// - Name and Age are used in all 5 records (most used)
	// - AnArray is used in 3 records
	// - Demo is used in 2 records
	// - Decimal is used in only 1 record (least used)
	//
	// After optimization, field order will be: Name, Age, AnArray, Demo, Decimal
	// This minimizes skip indices since rarely-used fields come last
	demo := []DemoStruct{
		{Name: "John", Age: 27, AnArray: []int32{1, 2, 3}},
		{Name: "Jane", Age: 25, AnArray: []int32{4, 5}},
		{Name: "Bob", Age: 30, Demo: DemoStruct2{Property1: "X", Count: 1}},
		{Name: "Alice", Age: 28, AnArray: []int32{6}, Demo: DemoStruct2{Property1: "Y", Count: 2}},
		{Name: "Charlie", Age: 35, Decimal: 3.14, Demo2: &DemoStruct2{
			Property1: "N", Count: 99,
		}}, // Only record with Decimal
	}

	fmt.Println("=== Two-Pass Optimization ===")
	fmt.Println("Original field order: Name(0), Age(1), Decimal(2), AnArray(3), Demo(4)")
	fmt.Println("Usage counts: Name=5, Age=5, Decimal=1, AnArray=3, Demo=2")
	fmt.Println("Optimized order: Name, Age, AnArray, Demo, Decimal")
	fmt.Println()

	// Marshal produces: [keys, content]
	output, err := Marshal(&demo)
	if err != nil {
		fmt.Printf("Marshal Error: %v\n", err)
		return
	}
	fmt.Printf("Output [keys, content]:\n%s\n\n", string(output))

	var decoded []DemoStruct
	err = Unmarshal(output, &decoded)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v\n", err)
		return
	}

	fmt.Println("Decoded records:")
	for i, d := range decoded {
		fmt.Printf("  %d: %+v\n", i, d)
	}
}

type DemoStruct3 struct {
	Records []DemoStruct
	Hello   string
}

func Test2() {
	// Demo data to show field optimization:
	// - Name and Age are used in all 5 records (most used)
	// - AnArray is used in 3 records
	// - Demo is used in 2 records
	// - Decimal is used in only 1 record (least used)
	//
	// After optimization, field order will be: Name, Age, AnArray, Demo, Decimal
	// This minimizes skip indices since rarely-used fields come last
	demo := []DemoStruct{
		{Name: "John", Age: 27, AnArray: []int32{1, 2, 3}},
		{Name: "Jane", Age: 25, AnArray: []int32{4, 5}},
		{Name: "Bob", Age: 30, Demo: DemoStruct2{Property1: "X", Count: 1}},
		{Name: "Alice", Age: 28, AnArray: []int32{6}, Demo: DemoStruct2{Property1: "Y", Count: 2}},
		{Name: "Charlie", Age: 35, Decimal: 3.14, Demo2: &DemoStruct2{
			Property1: "N", Count: 99,
		}}, // Only record with Decimal
	}

	demo2 := DemoStruct3{
		Records: demo,
		Hello:   "world",
	}

	fmt.Println("=== Two-Pass Optimization ===")
	fmt.Println("Original field order: Name(0), Age(1), Decimal(2), AnArray(3), Demo(4)")
	fmt.Println("Usage counts: Name=5, Age=5, Decimal=1, AnArray=3, Demo=2")
	fmt.Println("Optimized order: Name, Age, AnArray, Demo, Decimal")
	fmt.Println()

	// Marshal produces: [keys, content]
	output, err := Marshal(&demo2)
	if err != nil {
		fmt.Printf("Marshal Error: %v\n", err)
		return
	}
	fmt.Printf("Output [keys, content]:\n%s\n\n", string(output))

	var decoded DemoStruct3
	err = Unmarshal(output, &decoded)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v\n", err)
		return
	}

	fmt.Println("Decoded records:")
	fmt.Printf("  Hello: %s\n", decoded.Hello)
	for i, d := range decoded.Records {
		fmt.Printf("  Record %d: %+v\n", i, d)
	}
}

func Test3() {
	fmt.Println("=== Map Optimization ===")

	data := map[string]any{
		"Sedes": []DemoStruct2{
			{Property1: "Sede 1", Count: 10},
			{Property1: "Sede 2", Count: 20},
		},
		"Almacenes": []DemoStruct2{
			{Property1: "Alm 1", Count: 5},
		},
		"Status": "OK",
	}

	output, err := Marshal(data)
	if err != nil {
		fmt.Printf("Marshal Error: %v\n", err)
		return
	}
	fmt.Printf("Output [keys, content]:\n%s\n\n", string(output))

	var decoded map[string]any
	err = Unmarshal(output, &decoded)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v\n", err)
		return
	}

	fmt.Println("Decoded map:")
	for k, v := range decoded {
		fmt.Printf("  %s: %v (%T)\n", k, v, v)
	}
}

func Test4() {
	fmt.Println("=== Empty Array Optimization ===")

	data := []DemoStruct2{}

	output, err := Marshal(data)
	if err != nil {
		fmt.Printf("Marshal Error: %v\n", err)
		return
	}
	fmt.Printf("Output [keys, content]:\n%s\n\n", string(output))

	var decoded []DemoStruct2
	err = Unmarshal(output, &decoded)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v\n", err)
		return
	}

	fmt.Printf("Decoded slice: %v (len: %d)\n", decoded, len(decoded))
}
