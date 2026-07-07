package colbin

import (
	"math"
	"testing"
)

// TestBitstreamRoundTrip exercises the LSB-first packer across odd widths and
// the 64-bit-at-offset case that the 32-bit chunking must handle.
func TestBitstreamRoundTrip(t *testing.T) {
	type item struct {
		v uint64
		w uint8
	}
	items := []item{
		{5, 3}, {0xFFF, 12}, {1, 1}, {0xABCDEF, 24},
		{math.MaxUint64, 64}, {0, 8}, {0x1234567890, 48}, {7, 5},
	}
	var bw bitWriter
	for _, it := range items {
		bw.writeBits(it.v, it.w)
	}
	br := bitReader{buf: bw.flush()}
	for i, it := range items {
		got := br.readBits(it.w)
		want := it.v
		if it.w < 64 {
			want &= (uint64(1) << it.w) - 1
		}
		if got != want {
			t.Fatalf("item %d width %d: got %x want %x", i, it.w, got, want)
		}
	}
}

// TestIntColumnRoundTrip covers the three shapes: unsigned+sentinel, signed, empty.
func TestIntColumnRoundTrip(t *testing.T) {
	cases := map[string][]int64{
		"unsigned_sentinel": {0, 100, 105, 0, 110}, // 0 must survive as sentinel
		"signed":            {-5, 0, 3, -100, 42},
		"empty":             {0, 0, 0},
		"single_positive":   {7},
		"wide":              {0, math.MaxInt32, 1},
	}
	for name, values := range cases {
		data := appendIntColumn(nil, values, 64)
		flags := data[0]
		isSigned, prec, empty := flags>>3&1 == 1, flags>>4&7, flags>>7&1 == 1
		br := bitReader{buf: data[1:]}
		out := make([]int64, len(values))
		decodeIntColumn(&br, len(values), 64, isSigned, empty, prec, out)
		for i := range values {
			if out[i] != values[i] {
				t.Fatalf("%s[%d]: got %d want %d (signed=%v empty=%v prec=%d)",
					name, i, out[i], values[i], isSigned, empty, prec)
			}
		}
	}
}
