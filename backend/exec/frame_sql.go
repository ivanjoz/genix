package exec

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"app/core"
	"app/db"
	sales "app/sales/types"
)

// FrameSQLConsole (fn-frame-sql) runs one FrameSQL statement on a company's sale frames and prints the
// result, to check a statement the agent wrote against real data. Message:
//
//	<companyID> <frame> <statement>
//	fn-frame-sql 1 sale_summary SELECT product_id, SUM(amount) WHERE date IN PERIOD("D-29..D") GROUP BY product_id SORT BY amount DESC LIMIT 10
//
// PERIOD() takes "D-a..D-b", days back from today; record names in quotes are not resolved here.
func FrameSQLConsole(args *core.ExecArgs) core.FuncResponse {
	parts := strings.SplitN(strings.TrimSpace(args.Message), " ", 3)
	if len(parts) < 3 {
		return core.FuncResponse{Error: "usage: fn-frame-sql <companyID> <frame> <statement>"}
	}
	companyID, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return core.FuncResponse{Error: "the company ID is not a number: " + parts[0]}
	}
	result, err := db.PrepareFrameSQL(parts[1], parts[2], saleFrameSQLFrames(int32(companyID)), db.SQLOptions{})
	if err == nil {
		var statementResult db.SQLResult
		statementResult, err = result.Run(db.SQLResolvers{Period: daysBackPeriod})
		if err == nil {
			printFrameSQLResult(statementResult)
		}
	}
	if err != nil {
		return core.FuncResponse{Error: err.Error()}
	}
	return core.FuncResponse{}
}

// saleFrameSQLFrames are the sale frames as FrameSQL reads them (version 1 of their sums).
func saleFrameSQLFrames(companyID int32) []db.SQLFrame {
	dateColumn := db.SQLColumn{Name: "date", Label: "Date", Kind: db.SQLDay}
	productColumn := db.SQLColumn{Name: "product_id", Label: "Product", Kind: db.SQLRef, Collection: "products"}
	clientColumn := db.SQLColumn{Name: "client_id", Label: "Client", Kind: db.SQLRef, Collection: "clients"}
	divisorColumn := db.SQLColumn{Name: "sub_divisor", Hidden: true}
	sums := []db.SQLColumn{
		{Name: "amount", Label: "Amount", Kind: db.SQLCents},
		{Name: "debt_amount", Label: "Debt", Kind: db.SQLCents},
		{Name: "sold_units", Label: "Sold units"},
		{Name: "sold_sub", Label: "Sold sub-units"},
		{Name: "pending_units", Label: "Pending units"},
		{Name: "pending_sub", Label: "Pending sub-units"},
	}
	return []db.SQLFrame{{
		Name: "sale-summary", Keys: []db.SQLColumn{dateColumn}, Rows: []db.SQLColumn{productColumn, divisorColumn}, Sums: sums,
		Source: db.FrameSource[sales.SaleOrder](companyID, "sale-summary"),
	}, {
		Name: "sale-client-product", Keys: []db.SQLColumn{dateColumn}, Rows: []db.SQLColumn{clientColumn, productColumn, divisorColumn}, Sums: sums,
		Source: db.FrameSource[sales.SaleOrder](companyID, "sale-client-product"),
	}}
}

// daysBackPeriod reads "D-a..D-b" (or "D") as the days a and b back from today.
func daysBackPeriod(text string) (int64, int64, error) {
	today := int64(core.FechaUnix())
	daysBack := func(bound string) (int64, error) {
		if bound == "D" {
			return 0, nil
		}
		value, err := strconv.ParseInt(strings.TrimPrefix(bound, "D-"), 10, 64)
		if err != nil || !strings.HasPrefix(bound, "D-") {
			return 0, fmt.Errorf("%q is not D or D-n", bound)
		}
		return value, nil
	}
	fromText, toText, isRange := strings.Cut(text, "..")
	if !isRange {
		toText = fromText
	}
	fromBack, err := daysBack(fromText)
	if err != nil {
		return 0, 0, err
	}
	toBack, err := daysBack(toText)
	if err != nil {
		return 0, 0, err
	}
	return today - fromBack, today - toBack, nil
}

func printFrameSQLResult(result db.SQLResult) {
	names := make([]string, len(result.Columns))
	for i, column := range result.Columns {
		names[i] = column.Name
	}
	fmt.Println(strings.Join(names, "\t"))
	for row := range result.RowCount {
		values := make([]string, len(result.Columns))
		for i, column := range result.Columns {
			value := column.Values[row]
			switch {
			case math.IsNaN(value):
				values[i] = "-"
			case column.Kind == db.SQLDay:
				values[i] = time.Unix(int64(value)*86400, 0).UTC().Format(time.DateOnly)
			default:
				values[i] = strconv.FormatFloat(value, 'f', -1, 64)
			}
		}
		fmt.Println(strings.Join(values, "\t"))
	}
	fmt.Printf("%d rows (truncated %v), %d frame rows read, days %d..%d\n",
		result.RowCount, result.Truncated, result.RowsRead, result.FromKey, result.ToKey)
}
