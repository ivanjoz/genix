package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/xuri/excelize/v2"
)

func main() {
	psvPath := "../../libs/index_builder/productos.psv"
	xlsxPath := "../productos.xlsx"

	// Open PSV file
	f, err := os.Open(psvPath)
	if err != nil {
		log.Fatalf("Unable to read input file %s: %v", psvPath, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = '|'
	r.LazyQuotes = true

	// Create new Excel file
	ef := excelize.NewFile()
	// Rename default sheet
	sheetName := "Productos"
	index, err := ef.NewSheet(sheetName)
    if err != nil {
        log.Fatalf("Unable to create sheet %s: %v", sheetName, err)
    }
    // Delete the default "Sheet1" if it exists and is different from "Productos"
    // Usually NewFile creates "Sheet1". If we rename it, we don't need to delete.
    // Let's just set the active sheet.
    ef.SetSheetName("Sheet1", sheetName)
	ef.SetActiveSheet(index)

	// Set Title in Row 1
	ef.SetCellValue(sheetName, "A1", "Productos")

	// Set Headers in Row 2
	headers := []string{
		"ID", "Producto", "Categorías", "Precio", "Descuento", "Precio Final",
		"Sub Unidades", "Marca", "Unidad", "Volumen", "Peso", "Moneda",
	}
	
    // Write headers
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		ef.SetCellValue(sheetName, cell, h)
	}

	// Read PSV and write to Excel
	// PSV Headers: product|brand|categories|price
    // We skip the first line of PSV if it contains headers. 
    // The snippet shows: product|brand|categories|price as first line.
    
    _, err = r.Read()
    if err != nil {
        log.Fatal(err)
    }
    // Verify headers roughly (optional, but good practice)
    // Assuming file structure matches description.

    rowIdx := 3
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading record: %v", err)
			continue
		}

        // record[0] -> product -> Column B (Producto)
        // record[1] -> brand -> Column H (Marca)
        // record[2] -> categories -> Column C (Categorías)
        // record[3] -> price -> Column D (Precio)

        productName := record[0]
        brand := record[1]
        categories := record[2]
        priceStr := record[3]

        // Parse price
        price, _ := strconv.ParseFloat(priceStr, 64)

        // Write to Excel
        // ID (A)
        // Producto (B)
        ef.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), productName)
        // Categorías (C)
        ef.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), categories)
        // Precio (D)
        ef.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), price)
        // Descuento (E)
        // Precio Final (F)
        // Sub Unidades (G)
        // Marca (H)
        ef.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIdx), brand)
        // Unidad (I)
        // Volumen (J)
        // Peso (K)
        // Moneda (L)

		rowIdx++
	}

	// Save Excel file
	if err := ef.SaveAs(xlsxPath); err != nil {
		log.Fatalf("Unable to save file %s: %v", xlsxPath, err)
	}

	fmt.Printf("Successfully converted %s to %s\n", psvPath, xlsxPath)
}
