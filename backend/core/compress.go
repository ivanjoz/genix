package core

// Pooled response compression: zstd as the primary codec, gzip for compatibility.
//
// This runs on the Lambda side — the backend is fronted by a Function URL, not an API Gateway
// stage that could compress at the edge — so every response body is compressed in-process.
// Once the serializer stopped building a []any tree, the compressors became the dominant
// allocation left in the response path, because a fresh encoder (with its window and hash
// tables) was being built and discarded for every single response.
//
// Both the encoders and their destination buffers are pooled. Callers receive the compressed
// bytes through a callback so the buffer can be recycled the moment they are done with it,
// instead of escaping into the response.

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// Each pooled encoder is built with concurrency 1: a pooled encoder is only ever used by one
// goroutine at a time, so the default GOMAXPROCS worker fan-out would allocate per-worker state
// for nothing. Pooling rather than sharing one encoder also means concurrent requests on the
// VPS never block each other, which they would with a single concurrency-1 encoder.
var zstdEncoderPool = sync.Pool{
	New: func() any {
		encoder, err := zstd.NewWriter(nil,
			zstd.WithEncoderConcurrency(1),
			zstd.WithEncoderLevel(zstd.SpeedDefault),
		)
		if err != nil {
			// Only reachable via a bad option constant, which is a programming error.
			panic("no se pudo crear el encoder zstd: " + err.Error())
		}
		return encoder
	},
}

var gzipWriterPool = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// zstdBufferPool holds []byte scratch for EncodeAll, which appends rather than writing to an
// io.Writer. gzip needs a bytes.Buffer instead, hence the two pools.
var zstdBufferPool = sync.Pool{
	New: func() any {
		buffer := make([]byte, 0, 64*1024)
		return &buffer
	},
}

var gzipBufferPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(make([]byte, 0, 64*1024)) },
}

// CompressZstdPooled compresses body and hands the bytes to consume. The slice is only valid
// for the duration of the call — it points into a pooled buffer.
func CompressZstdPooled(body []byte, consume func(compressed []byte)) {
	encoder := zstdEncoderPool.Get().(*zstd.Encoder)
	buffer := zstdBufferPool.Get().(*[]byte)

	compressed := encoder.EncodeAll(body, (*buffer)[:0])
	consume(compressed)

	// Hand back the grown buffer, not the stub we borrowed.
	*buffer = compressed
	zstdBufferPool.Put(buffer)
	zstdEncoderPool.Put(encoder)
}

// pooledBufferReadCloser streams compressed bytes out of a pooled buffer and returns that
// buffer to its pool on Close. Lambda's streaming response closes the body once it has been
// written to the client, so the buffer is recycled exactly when the caller is finished with it
// — the borrow-inside-a-callback trick the other helpers use cannot work here, because the
// bytes have to outlive the compress call.
type pooledBufferReadCloser struct {
	reader *bytes.Reader
	buffer *[]byte
	pool   *sync.Pool
}

func (p *pooledBufferReadCloser) Read(destination []byte) (int, error) {
	return p.reader.Read(destination)
}

func (p *pooledBufferReadCloser) Close() error {
	if p.buffer == nil {
		return nil // Already closed; the runtime is allowed to call Close more than once.
	}
	p.pool.Put(p.buffer)
	p.buffer = nil
	p.reader = nil
	return nil
}

// CompressZstdReader compresses body and returns a reader over the result. The caller must
// Close it to recycle the underlying buffer.
func CompressZstdReader(body []byte) io.ReadCloser {
	encoder := zstdEncoderPool.Get().(*zstd.Encoder)
	buffer := zstdBufferPool.Get().(*[]byte)

	compressed := encoder.EncodeAll(body, (*buffer)[:0])
	zstdEncoderPool.Put(encoder)
	*buffer = compressed

	return &pooledBufferReadCloser{
		reader: bytes.NewReader(compressed),
		buffer: buffer,
		pool:   &zstdBufferPool,
	}
}

// CompressGzipReader is the gzip counterpart of CompressZstdReader.
func CompressGzipReader(body []byte) (io.ReadCloser, error) {
	buffer := gzipBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	writer := gzipWriterPool.Get().(*gzip.Writer)
	writer.Reset(buffer)
	defer gzipWriterPool.Put(writer)

	if _, err := writer.Write(body); err != nil {
		gzipBufferPool.Put(buffer)
		return nil, err
	}
	if err := writer.Close(); err != nil {
		gzipBufferPool.Put(buffer)
		return nil, err
	}

	// gzip writes into a bytes.Buffer rather than an appended slice, so this pool holds a
	// different type and needs its own read-closer wiring.
	return &pooledGzipBufferReadCloser{
		reader: bytes.NewReader(buffer.Bytes()),
		buffer: buffer,
	}, nil
}

type pooledGzipBufferReadCloser struct {
	reader *bytes.Reader
	buffer *bytes.Buffer
}

func (p *pooledGzipBufferReadCloser) Read(destination []byte) (int, error) {
	return p.reader.Read(destination)
}

func (p *pooledGzipBufferReadCloser) Close() error {
	if p.buffer == nil {
		return nil
	}
	gzipBufferPool.Put(p.buffer)
	p.buffer = nil
	p.reader = nil
	return nil
}

// CompressGzipPooled compresses body and hands the bytes to consume, under the same borrowing
// rule. Kept for clients that do not advertise zstd.
func CompressGzipPooled(body []byte, consume func(compressed []byte)) error {
	buffer := gzipBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	writer := gzipWriterPool.Get().(*gzip.Writer)
	writer.Reset(buffer)

	defer func() {
		gzipWriterPool.Put(writer)
		gzipBufferPool.Put(buffer)
	}()

	if _, err := writer.Write(body); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	consume(buffer.Bytes())
	return nil
}
