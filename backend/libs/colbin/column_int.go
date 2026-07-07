package colbin

// Integer column: frame-of-reference (FOR) + bit-packing.
//
// Two modes, chosen automatically per column and flagged by is_signed:
//
//   unsigned (is_signed=0): the column has no negative values. 0 is a reserved
//     sentinel meaning "empty/absent" and decodes back to 0. base = min_nonzero-1
//     so every real value maps to >=1 and never collides with the sentinel.
//     enc(v) = v==0 ? 0 : v-base ; decode: enc==0 ? 0 : enc+base.
//
//   signed (is_signed=1): the column contains at least one negative value, so 0
//     can no longer be a sentinel. base = true minimum (may be negative) and
//     enc(v) = v-base (still unsigned, but the wider span typically costs ~1 more
//     bit). decode: v = enc+base. base is stored two's-complement, sign-extended.
//
// base is stored in the field's native bit width (known from the Go type on both
// sides); the deltas are stored at the selected precision width.

// appendIntColumn encodes values as an int column (flags byte + packed payload)
// straight onto out, and returns out. values holds one int64 per record (0 == Go
// zero value == absent). A single pass gathers min/minNonZero/max so base and the
// max delta are known without a second scan; the bit packer writes directly into
// out (no intermediate buffer/copy).
func appendIntColumn(out []byte, values []int64, nativeWidth uint8) []byte {
	allZero, hasNeg := true, false
	var minVal, minNonZero, maxVal int64
	for _, v := range values {
		if v != 0 {
			allZero = false
		}
		if v < 0 {
			hasNeg = true
		}
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
		if v > 0 && (minNonZero == 0 || v < minNonZero) {
			minNonZero = v
		}
	}
	if allZero {
		return append(out, ftInt|boolBit(true, 7)) // empty column: flags only
	}

	// signed mode: base = true min (may be negative); unsigned: min_nonzero-1 with
	// 0 kept as sentinel. maxEnc is the largest delta, derived from the scan.
	var base int64
	var maxEnc uint64
	if hasNeg {
		base = minVal
		maxEnc = uint64(maxVal - base)
	} else {
		base = minNonZero - 1
		maxEnc = uint64(maxVal - base)
	}
	precCode, width := selectIntPrecision(maxEnc)

	out = append(out, ftInt|boolBit(hasNeg, 3)|precCode<<4)
	bw := bitWriter{buf: out}
	bw.writeBits(uint64(base), nativeWidth) // base at native width (two's complement)
	for _, v := range values {
		e := uint64(v - base)
		if !hasNeg && v == 0 { // preserve sentinel in unsigned mode
			e = 0
		}
		bw.writeBits(e, width)
	}
	return bw.flush()
}

// decodeIntColumn reads n values from br into out, reversing encodeIntColumn.
func decodeIntColumn(br *bitReader, n int, nativeWidth uint8, isSigned, empty bool, precCode uint8, out []int64) {
	if empty {
		for i := 0; i < n; i++ {
			out[i] = 0
		}
		return
	}
	width := intWidths[precCode]
	rawBase := br.readBits(nativeWidth)
	base := int64(rawBase)
	if isSigned {
		base = signExtend(rawBase, nativeWidth) // base may be negative
	}
	for i := 0; i < n; i++ {
		e := br.readBits(width)
		if !isSigned && e == 0 { // sentinel -> zero value
			out[i] = 0
			continue
		}
		out[i] = base + int64(e)
	}
}
