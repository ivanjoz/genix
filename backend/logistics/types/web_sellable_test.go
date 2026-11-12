package types

import (
	"maps"
	"testing"
)

func webStockRow(presentationID int16, quantity, detailQuantity, pendingStore, pendingOnline int32) ProductStock {
	return ProductStock{PresentationID: presentationID, Quantity: quantity, DetailQuantity: detailQuantity,
		PendingDeliveryStore: pendingStore, PendingDeliveryOnline: pendingOnline}
}

func TestWebSellableStock(t *testing.T) {
	cases := []struct {
		name            string
		stockRows       []ProductStock
		presentationIDs []int16
		webExtraStock   int32
		want            map[int16]int32
	}{
		{
			name:            "two warehouses add up, lots count, reservations come off",
			stockRows:       []ProductStock{webStockRow(0, 5, 2, 1, 0), webStockRow(0, 4, 0, 0, 3)},
			presentationIDs: []int16{0},
			want:            map[int16]int32{0: 7},
		},
		{
			name:            "no stock at all: the margin alone is sellable",
			presentationIDs: []int16{0},
			webExtraStock:   100,
			want:            map[int16]int32{0: 100},
		},
		{
			name: "the margin is shared: an oversold presentation uses it up for the others",
			stockRows: []ProductStock{
				webStockRow(1, 5, 0, 0, 0), // net 5
				webStockRow(2, 0, 0, 0, 4), // oversold by 4
			},
			presentationIDs: []int16{1, 2, 3},
			webExtraStock:   10,
			want:            map[int16]int32{1: 11, 2: 6, 3: 6},
		},
		{
			name:            "margin exhausted: only real stock is sellable",
			stockRows:       []ProductStock{webStockRow(1, 3, 0, 0, 0), webStockRow(2, 0, 0, 12, 0)},
			presentationIDs: []int16{1, 2},
			webExtraStock:   10,
			want:            map[int16]int32{1: 3, 2: 0},
		},
		{
			name:            "a deleted presentation's stock is not sellable and uses no margin",
			stockRows:       []ProductStock{webStockRow(9, 0, 0, 0, 50)},
			presentationIDs: []int16{1},
			webExtraStock:   10,
			want:            map[int16]int32{1: 10},
		},
	}
	for _, testCase := range cases {
		got := WebSellableStock(testCase.stockRows, testCase.presentationIDs, testCase.webExtraStock)
		if !maps.Equal(got, testCase.want) {
			t.Errorf("%s: got %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestWebOrderFits(t *testing.T) {
	stockRows := []ProductStock{webStockRow(1, 5, 0, 0, 0), webStockRow(2, 0, 0, 0, 4)}
	presentationIDs := []int16{1, 2}
	cases := []struct {
		name          string
		webExtraStock int32
		ordered       map[int16]int32
		want          bool
	}{
		{"within real stock", 0, map[int16]int32{1: 5}, true},
		{"one past real stock, no margin", 4, map[int16]int32{1: 6}, false},
		{"uses the rest of the shared margin", 10, map[int16]int32{1: 5, 2: 6}, true},
		{"one past the shared margin", 10, map[int16]int32{1: 6, 2: 6}, false},
		{"margin lowered below what is oversold: in-stock units still sell", 2, map[int16]int32{1: 5}, true},
		{"margin lowered below what is oversold: no more overselling", 2, map[int16]int32{2: 1}, false},
		{"a presentation the product does not sell", 100, map[int16]int32{7: 1}, false},
	}
	for _, testCase := range cases {
		if got := WebOrderFits(stockRows, presentationIDs, testCase.webExtraStock, testCase.ordered); got != testCase.want {
			t.Errorf("%s: got %v, want %v", testCase.name, got, testCase.want)
		}
	}
}
