package colbin

import "sync"

// Per-column scratch buffers are reused across columns via pools: the encoder
// gathers one column's values into a scratch slice, packs it, then returns the
// scratch. This replaces one N-sized allocation per column with a reused buffer.

var (
	i64Pool  = sync.Pool{New: func() any { s := make([]int64, 0, 256); return &s }}
	f64Pool  = sync.Pool{New: func() any { s := make([]float64, 0, 256); return &s }}
	blobPool = sync.Pool{New: func() any { s := make([][]byte, 0, 256); return &s }}
)

func getI64(n int) *[]int64 {
	p := i64Pool.Get().(*[]int64)
	if cap(*p) < n {
		*p = make([]int64, n)
	} else {
		*p = (*p)[:n]
	}
	return p
}
func putI64(p *[]int64) { i64Pool.Put(p) }

func getF64(n int) *[]float64 {
	p := f64Pool.Get().(*[]float64)
	if cap(*p) < n {
		*p = make([]float64, n)
	} else {
		*p = (*p)[:n]
	}
	return p
}
func putF64(p *[]float64) { f64Pool.Put(p) }

func getBlobs(n int) *[][]byte {
	p := blobPool.Get().(*[][]byte)
	if cap(*p) < n {
		*p = make([][]byte, n)
	} else {
		*p = (*p)[:n]
	}
	return p
}
func putBlobs(p *[][]byte) { blobPool.Put(p) }
