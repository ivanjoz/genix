package colbin

import "math/bits"

// Wire format constants shared by encoder and decoder.

// magic/version byte prefixing every message. Bump on any wire-format change.
const formatVersion byte = 0x01

// reserved field-id: 255 is never assigned, so a struct may have at most 254
// fields and the id space always has a free "terminator" slot.
const reservedFieldID uint8 = 255

// field_type codes (3 bits). is_signed distinguishes signed/unsigned integers,
// so a single ftInt covers both. Code 7 is reserved (no 8th slot in use).
const (
	ftInt    uint8 = 0 // integers (int8..int64, uint8..uint32); bool encoded here too
	ftFloat  uint8 = 1 // IEEE-754, width from precision (float16/32/64)
	ftString uint8 = 2 // length column + concatenated UTF-8 bytes
	ftBytes  uint8 = 3 // length column + concatenated raw bytes
	ftArray  uint8 = 4 // length column + flattened element sub-column
	ftStruct uint8 = 5 // nested sub-table of columns
	ftMap    uint8 = 6 // length column + flattened keys column + flattened values column
)

// intWidths maps a 3-bit precision code -> packed bit width for integer deltas.
// True bit-level widths (12/24/48 straddle byte boundaries). No 8th slot.
var intWidths = [7]uint8{8, 12, 16, 24, 32, 48, 64}

// selectIntPrecision returns the smallest precision code whose width can hold
// maxEnc, plus that width. maxEnc==0 still uses the minimum 8-bit slot.
func selectIntPrecision(maxEnc uint64) (code uint8, width uint8) {
	need := uint8(bits.Len64(maxEnc)) // bits required to represent maxEnc (0 => 0)
	for i, w := range intWidths {
		if need <= w {
			return uint8(i), w
		}
	}
	return 6, 64 // maxEnc needs full 64 bits
}

// signExtend interprets the low `width` bits of v as a two's-complement signed
// integer and widens it to int64.
func signExtend(v uint64, width uint8) int64 {
	if width < 64 && v&(uint64(1)<<(width-1)) != 0 {
		v |= ^((uint64(1) << width) - 1) // set all high bits
	}
	return int64(v)
}
