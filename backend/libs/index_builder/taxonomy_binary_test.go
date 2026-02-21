package index_builder

import "testing"

func TestValidateForBinary_OK(t *testing.T) {
	packedCategoryCount, packErr := packTwoBitCategoryCounts([]uint8{0, 1, 0})
	if packErr != nil {
		t.Fatalf("pack category count: %v", packErr)
	}
	packedBrandIndexes, packBrandErr := packIntSliceToUint12([]int{1, 2, 3}, "brand index", maxBrandIndexUint12)
	if packBrandErr != nil {
		t.Fatalf("pack brand indexes: %v", packBrandErr)
	}

	taxonomyBuildResult := &ProductosIndexBuild{
		SortedIDs:                       []int32{10, 11, 12},
		BrandIDs:                        []uint16{20, 21, 22},
		BrandNames:                      []string{"a", "b", "c"},
		CategoryIDs:                     []uint16{30, 31, 32, 33},
		CategoryNames:                   []string{"x", "y", "z", "otros"},
		BrandIndexEncodingFlag:          BrandIndexEncodingUint12,
		ProductBrandIndexesUint12Packed: packedBrandIndexes,
		ProductCategoryCount:            packedCategoryCount,
		ProductCategoryIndexes:          []uint8{0, 1, 2, 3},
	}

	if validateErr := taxonomyBuildResult.ValidateForBinary(); validateErr != nil {
		t.Fatalf("unexpected validate error: %v", validateErr)
	}
}

func TestValidateForBinary_CategoryCountBytesMismatch(t *testing.T) {
	taxonomyBuildResult := &ProductosIndexBuild{
		SortedIDs:                 []int32{1, 2, 3, 4, 5},
		BrandIDs:                  []uint16{1},
		BrandNames:                []string{"a"},
		CategoryIDs:               []uint16{1},
		CategoryNames:             []string{"x"},
		BrandIndexEncodingFlag:    BrandIndexEncodingUint16,
		ProductBrandIndexesUint16: []uint16{0, 0, 0, 0, 0},
		ProductCategoryCount:      []uint8{0},
		ProductCategoryIndexes:    []uint8{0, 0, 0, 0, 0},
	}

	if validateErr := taxonomyBuildResult.ValidateForBinary(); validateErr == nil {
		t.Fatalf("expected validate error for category count bytes mismatch")
	}
}
