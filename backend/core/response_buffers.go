package core

import "runtime"

// Reused colbin body buffers for the VPS server.
//
// A first sync encodes megabytes of colbin only to compress it to a tenth of that and write it out;
// the uncompressed body is garbage the moment the response is written. Encoding into a buffer that
// SendLocalResponse hands back once the socket write returns lets the next response reuse it
// instead of allocating its own, which lowers the allocation rate and so how often the GC runs.
//
// Only the VPS path reuses them: on Lambda the body outlives the handler (it is copied into the
// runtime's response), so there is no point at which it is known to be free.
//
// Like the zstd encoders, a channel and not a sync.Pool, so a GC does not empty it. It holds one
// buffer per P, and a buffer past maxPooledBodyBytes is dropped rather than kept: a 50k-row
// catalog response would otherwise pin 12 MB per slot for a request that is rare.
var colbinBodyBuffers = make(chan []byte, runtime.GOMAXPROCS(0))

const maxPooledBodyBytes = 4 << 20

func takeColbinBodyBuffer() []byte {
	select {
	case buffer := <-colbinBodyBuffers:
		return buffer[:0]
	default:
		return nil // AppendSelfDescribing reserves the message's size itself
	}
}

func returnColbinBodyBuffer(buffer []byte) {
	if cap(buffer) > maxPooledBodyBytes {
		return
	}
	select {
	case colbinBodyBuffers <- buffer:
	default: // every slot is taken: let this one go
	}
}
