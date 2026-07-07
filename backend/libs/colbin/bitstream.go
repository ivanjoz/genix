package colbin

// bitstream provides LSB-first bit-level packing. Values are written low bit
// first; the reader mirrors that order. This is the primitive that lets integer
// columns use non-byte-aligned widths (12/24/48 bits).
//
// Writes/reads are split into <=32-bit chunks so the uint64 accumulator (which
// holds up to 7 leftover bits between calls) can never overflow: 7 + 32 < 64.

// bitWriter accumulates bits LSB-first into buf.
type bitWriter struct {
	buf     []byte
	current uint64 // pending bits not yet flushed to buf (always < 8 between calls)
	nbits   uint8  // number of valid bits currently held in current
}

// writeBits appends the low `width` bits of v to the stream (width 0..64).
func (w *bitWriter) writeBits(v uint64, width uint8) {
	for width > 32 {
		w.putChunk(v&0xFFFFFFFF, 32)
		v >>= 32
		width -= 32
	}
	w.putChunk(v, width)
}

// putChunk writes width<=32 bits; keeps every shift within uint64 range.
func (w *bitWriter) putChunk(v uint64, width uint8) {
	if width == 0 {
		return
	}
	v &= (uint64(1) << width) - 1
	w.current |= v << w.nbits
	w.nbits += width
	for w.nbits >= 8 {
		w.buf = append(w.buf, byte(w.current))
		w.current >>= 8
		w.nbits -= 8
	}
}

// flush writes any remaining partial byte (zero-padded high bits) and returns buf.
func (w *bitWriter) flush() []byte {
	if w.nbits > 0 {
		w.buf = append(w.buf, byte(w.current))
		w.current = 0
		w.nbits = 0
	}
	return w.buf
}

// bitReader reads bits LSB-first from buf, mirroring bitWriter.
type bitReader struct {
	buf     []byte
	pos     int    // next byte index to consume from buf
	current uint64 // bits loaded but not yet returned (always < 8 between calls)
	nbits   uint8  // valid bits in current
}

// readBits returns the next `width` bits (width 0..64) as the low bits of the result.
func (r *bitReader) readBits(width uint8) uint64 {
	var out uint64
	var shift uint8
	for width > 32 {
		out |= r.getChunk(32) << shift
		shift += 32
		width -= 32
	}
	out |= r.getChunk(width) << shift
	return out
}

// getChunk reads width<=32 bits; keeps every shift within uint64 range.
func (r *bitReader) getChunk(width uint8) uint64 {
	if width == 0 {
		return 0
	}
	for r.nbits < width && r.pos < len(r.buf) {
		r.current |= uint64(r.buf[r.pos]) << r.nbits
		r.pos++
		r.nbits += 8
	}
	out := r.current & ((uint64(1) << width) - 1)
	r.current >>= width
	r.nbits -= width
	return out
}
