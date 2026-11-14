package types

import (
	"app/core"
	logistics "app/logistics/types"
	"cmp"
	"fmt"
	"math"
	"slices"
)

// How a production takes a supply out of its stock row, shared by the daily close and a production
// order's receipt (docs/MANUFACTURING_PLAN.md §6.2): the row's buckets at the supply's divisor, the
// part not reserved by released orders, and the walk that draws each product's share bucket by bucket.

// SupplyStockBucket is one bucket of a supply's stock row in the supplies warehouse: the free bucket
// (no lot, no serial) or one detail row.
type SupplyStockBucket struct {
	LotID        int32
	SerialNumber string
	Quantity     core.Quantity // at the row's SubDivisor
	// The lot's, for a lot bucket (docs/MANUFACTURING_PLAN.md §10). A blocked lot is neither
	// available nor walked; ExpirationDate (UnixDay, 0 = none) orders the walk.
	QualityStatus  int8
	ExpirationDate int16
}

func (bucket SupplyStockBucket) isBlocked() bool {
	return logistics.IsLotQualityBlocked(bucket.QualityStatus)
}

// SupplyStock is a supply's stock row in the supplies warehouse.
type SupplyStock struct {
	SubDivisor int16
	StockValue int64
	// Buckets in the order a consumption walks them (OrderSupplyStockBuckets): the free bucket first,
	// then the details.
	Buckets []SupplyStockBucket
	// ReservedProduction is what released production orders hold of the row (at SubDivisor)
	// (docs/MANUFACTURING_PLAN.md §5).
	ReservedProduction core.Quantity
}

// SupplyConsumption is one consumption exit: what a product took of a supply from one bucket.
type SupplyConsumption struct {
	ProductID    int32
	SupplyID     int32
	LotID        int32
	SerialNumber string
	SubUnits     int32
	SubDivisor   int16
	// EstimatedValue is the bucket's share of the row value, as the stock engine will take it.
	EstimatedValue int32
}

// OrderSupplyStockBuckets sorts the buckets in the order a consumption takes them (§6.2, §10, M23):
//   - the free bucket first: it has no lot to account for;
//   - then the lots that have not expired, the earliest expiry first (one with no expiry last), then
//     the oldest lot (its ID is autoincremented), then the serial;
//   - then the expired ones (expiry before today): still consumed, warned in red by the stock screen,
//     but only once nothing fresher is left;
//   - the blocked lots last, though the walk never takes them.
func OrderSupplyStockBuckets(buckets []SupplyStockBucket, today int16) {
	walkRank := func(bucket SupplyStockBucket) int {
		switch {
		case bucket.LotID == 0 && bucket.SerialNumber == "":
			return 0
		case bucket.isBlocked():
			return 3
		case bucket.ExpirationDate > 0 && bucket.ExpirationDate < today:
			return 2
		}
		return 1
	}
	// No expiry sorts after every date.
	expirySortKey := func(bucket SupplyStockBucket) int32 {
		return core.If(bucket.ExpirationDate > 0, int32(bucket.ExpirationDate), math.MaxInt32)
	}
	slices.SortStableFunc(buckets, func(left, right SupplyStockBucket) int {
		return cmp.Or(
			cmp.Compare(walkRank(left), walkRank(right)),
			cmp.Compare(expirySortKey(left), expirySortKey(right)),
			cmp.Compare(left.LotID, right.LotID),
			cmp.Compare(left.SerialNumber, right.SerialNumber),
		)
	})
}

// SupplyStockAtDivisor is the row read at the supply's divisor: each bucket's sub-units, the balance,
// and what is available to a production that holds no reservation of it (the positive buckets that
// are not blocked lots, minus what released orders reserved). A row still at a coarser divisor
// converts exactly.
type SupplyStockAtDivisor struct {
	BucketSubUnits    []int64
	BalanceSubUnits   int64
	AvailableSubUnits int64
	ReservedSubUnits  int64
}

func ReadSupplyStockAtDivisor(stock SupplyStock, divisor int16) (SupplyStockAtDivisor, error) {
	rowDivisor := max(stock.SubDivisor, core.QuantityDivisorNone)
	atDivisor := SupplyStockAtDivisor{BucketSubUnits: make([]int64, len(stock.Buckets))}
	for bucketIndex, bucket := range stock.Buckets {
		converted, err := bucket.Quantity.ConvertToDivisor(rowDivisor, divisor)
		if err != nil {
			return atDivisor, err
		}
		atDivisor.BucketSubUnits[bucketIndex] = converted.TotalSubUnits(divisor)
		// The balance is the whole row's, blocked lots included: it is what the engine values an exit against.
		atDivisor.BalanceSubUnits += atDivisor.BucketSubUnits[bucketIndex]
		if !bucket.isBlocked() {
			atDivisor.AvailableSubUnits += max(atDivisor.BucketSubUnits[bucketIndex], 0)
		}
	}
	reserved, err := stock.ReservedProduction.ConvertToDivisor(rowDivisor, divisor)
	if err != nil {
		return atDivisor, err
	}
	atDivisor.ReservedSubUnits = reserved.TotalSubUnits(divisor)
	atDivisor.AvailableSubUnits = max(atDivisor.AvailableSubUnits-atDivisor.ReservedSubUnits, 0)
	return atDivisor, nil
}

// ShareByWeight splits total in proportion to weights: each takes the floor of its share and the
// largest weight (the first on a tie) takes the remainder, so the shares always add up to total.
// With no positive weight nothing is shared.
func ShareByWeight(total int64, weights []float64) []int64 {
	shares := make([]int64, len(weights))
	weightSum := 0.0
	largestIndex := -1
	for index, weight := range weights {
		weightSum += weight
		if weight > 0 && (largestIndex < 0 || weight > weights[largestIndex]) {
			largestIndex = index
		}
	}
	if largestIndex < 0 {
		return shares
	}
	distributed := int64(0)
	for index, weight := range weights {
		shares[index] = int64(float64(total) * weight / weightSum)
		distributed += shares[index]
	}
	shares[largestIndex] += total - distributed
	return shares
}

// ProductSupplyTake is what one product takes of a supply, in sub-units of the supply's divisor.
type ProductSupplyTake struct {
	ProductID int32
	SubUnits  int64
}

// WalkSupplyBuckets draws each take from the row's buckets in order, the free bucket first, and values
// each piece as the engine will: its share of what the row is worth at that point. A blocked lot is
// skipped: the engine refuses it as a consumption's source. It refuses a take the buckets cannot cover.
func WalkSupplyBuckets(supplyID int32, divisor int16, stock SupplyStock, atDivisor SupplyStockAtDivisor,
	takes []ProductSupplyTake) ([]SupplyConsumption, error) {
	bucketSubUnits := append([]int64(nil), atDivisor.BucketSubUnits...)
	for bucketIndex, bucket := range stock.Buckets {
		if bucket.isBlocked() {
			bucketSubUnits[bucketIndex] = 0
		}
	}
	remainingValue, remainingBalance := stock.StockValue, atDivisor.BalanceSubUnits
	bucketIndex := 0
	consumptions := []SupplyConsumption{}
	for _, take := range takes {
		toConsume := take.SubUnits
		for toConsume > 0 && bucketIndex < len(stock.Buckets) {
			taken := min(toConsume, max(bucketSubUnits[bucketIndex], 0))
			if taken > 0 {
				value := logistics.StockOutflowValue(remainingValue, remainingBalance, taken)
				remainingValue -= int64(value)
				remainingBalance -= taken
				bucketSubUnits[bucketIndex] -= taken
				toConsume -= taken
				bucket := stock.Buckets[bucketIndex]
				consumptions = append(consumptions, SupplyConsumption{
					ProductID: take.ProductID, SupplyID: supplyID, LotID: bucket.LotID, SerialNumber: bucket.SerialNumber,
					SubUnits: int32(taken), SubDivisor: divisor, EstimatedValue: value,
				})
			}
			if bucketSubUnits[bucketIndex] <= 0 {
				bucketIndex++
			}
		}
		if toConsume > 0 {
			return consumptions, core.Err(fmt.Sprintf("No hay stock suficiente del insumo %v.", supplyID))
		}
	}
	return consumptions, nil
}
