package types

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"

	"app/db"
	production "app/production/types"

	"github.com/ivanjoz/genix-orm/dataframe"
)

// The FrameSQL columns follow the frame structs field by field: a sum added to the struct and not
// here would shift every column after it.
func TestSaleFrameSQLFramesMatchTheFrameStructs(t *testing.T) {
	countByTag := func(frameStruct any) map[string]int {
		counts := map[string]int{}
		structType := reflect.TypeOf(frameStruct)
		for i := range structType.NumField() {
			counts[structType.Field(i).Tag.Get("frame")]++
		}
		return counts
	}
	frames := SaleFrameSQLFrames(1)
	for i, frameStruct := range []any{SaleSummaryFrame{}, SaleClientProductFrame{}} {
		counts := countByTag(frameStruct)
		frame := frames[i]
		if len(frame.Keys) != counts["key"] || len(frame.Rows) != counts["row"] || len(frame.Sums) != counts["sum"] {
			t.Errorf("%s declares %d/%d/%d keys/rows/sums, its struct has %v", frame.Name, len(frame.Keys), len(frame.Rows), len(frame.Sums), counts)
		}
	}
	// The category views name the dictionary's rows "category_id", "product_id": ProductCategoryFrame's
	// row fields, in that order.
	categoryRowFields := []string{}
	categoryFrameType := reflect.TypeOf(production.ProductCategoryFrame{})
	for i := range categoryFrameType.NumField() {
		if categoryFrameType.Field(i).Tag.Get("frame") == "row" {
			categoryRowFields = append(categoryRowFields, categoryFrameType.Field(i).Name)
		}
	}
	if !slices.Equal(categoryRowFields, []string{"CategoryID", "ProductID"}) {
		t.Errorf("ProductCategoryFrame rows are %v: update the productCategories dictionary's Rows", categoryRowFields)
	}
}

// The category views fan a sale out under each category of its product: grouped by category each total
// is exact, and a product without a category is category 0.
func TestSaleCategoryViewSumsEachCategory(t *testing.T) {
	frames := SaleFrameSQLFrames(1)
	summaryRows := []dataframe.Row{
		{Keys: []int64{20000}, RowValues: []int64{77, 0}, Sums: []int64{1000, 0, 0, 0, 0, 0, 0, 0}},
		{Keys: []int64{20000}, RowValues: []int64{78, 0}, Sums: []int64{300, 0, 0, 0, 0, 0, 0, 0}},
	}
	categoryRows := []dataframe.Row{{Keys: []int64{0}, RowValues: []int64{3, 77}}, {Keys: []int64{0}, RowValues: []int64{5, 77}}}
	frames[0].Source = func(dataframe.Selection) ([]dataframe.Row, error) { return summaryRows, nil }
	categoryView := db.SQLView("sale-category", frames[0], db.SQLDictionary{
		Rows: []string{"category_id", "product_id"}, On: "product_id", Column: frames[2].Rows[1],
		Source: func(dataframe.Selection) ([]dataframe.Row, error) { return categoryRows, nil },
	})
	result, err := db.PrepareFrameSQL("sale_category", "SELECT category_id, SUM(amount) WHERE date = 20000 GROUP BY category_id",
		[]db.SQLFrame{categoryView}, db.SQLOptions{})
	if err != nil {
		t.Fatal(err)
	}
	answer, err := result.Run(db.SQLResolvers{})
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(answer.Columns[0].Values, answer.Columns[1].Values); got != "[0 3 5] [300 1000 1000]" {
		t.Errorf("sale_category by category = %s", got)
	}
}

// A box of six and a box of twelve, sold loose in one day: the agent's quantity adds the sixth and
// the twelfth as whole units, and never sees the divisors.
func TestSaleFrameSQLQuantityAddsSubUnitsOfAnyDivisor(t *testing.T) {
	sales := []SaleOrder{
		{Date: 20000, ClientID: 7, Status: OrderStatusCompleted, Lines: []SaleOrderLine{
			{ProductID: 77, Quantity: 1001, Price: 600, SubPrice: 100, SubDivisor: 6},
		}},
		{Date: 20000, ClientID: 8, Status: OrderStatusPending, Lines: []SaleOrderLine{
			{ProductID: 77, Quantity: 1001, Price: 1200, SubPrice: 100, SubDivisor: 12},
			{ProductID: 78, Quantity: 2000, Price: 300},
		}},
	}
	var rows []dataframe.Row
	for _, sale := range sales {
		for _, line := range sale.ToSaleClientProductFrames(nil) {
			scaled := func(value float64) int64 { return int64(math.Round(value * dataframe.FloatScale)) }
			rows = append(rows, dataframe.Row{
				Keys:      []int64{int64(line.Date)},
				RowValues: []int64{int64(line.ClientID), int64(line.ProductID), int64(line.SubDivisor)},
				Sums: []int64{int64(line.AmountPEN), int64(line.DebtAmountPEN), int64(line.SoldUnits), int64(line.SoldSub),
					int64(line.PendingUnits), int64(line.PendingSub), scaled(line.SoldQuantity), scaled(line.PendingQuantity)},
			})
		}
	}
	frames := SaleFrameSQLFrames(1)
	frames[1].Source = func(dataframe.Selection) ([]dataframe.Row, error) { return rows, nil }

	cases := []struct{ statement, want string }{
		{"SELECT product_id, SUM(quantity), SUM(amount) WHERE date = 20000 GROUP BY product_id",
			"product_id=[77 78] quantity=[2.25 2] amount=[2000 600]"},
		{"SELECT client_id, SUM(debt_amount), SUM(pending_quantity) WHERE date = 20000 GROUP BY client_id",
			"client_id=[7 8] debt_amount=[0 1900] pending_quantity=[0 3.083333]"},
	}
	for _, testCase := range cases {
		statement, err := db.PrepareFrameSQL("sale_client_product", testCase.statement, frames, db.SQLOptions{})
		if err != nil {
			t.Fatalf("%s: %v", testCase.statement, err)
		}
		result, err := statement.Run(db.SQLResolvers{})
		if err != nil {
			t.Fatalf("%s: %v", testCase.statement, err)
		}
		got := ""
		for i, column := range result.Columns {
			if i > 0 {
				got += " "
			}
			got += fmt.Sprintf("%s=%v", column.Name, column.Values)
		}
		if got != testCase.want {
			t.Errorf("%s\n got  %s\n want %s", testCase.statement, got, testCase.want)
		}
	}
	if _, err := db.PrepareFrameSQL("sale_summary", "SELECT SUM(sold_sub) WHERE date = 20000", frames, db.SQLOptions{}); err == nil {
		t.Error("the split sub-unit sums must stay hidden from statements")
	}
}
