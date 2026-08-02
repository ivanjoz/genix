package serialize

import (
	"reflect"
	"sync"
)

type Encoder struct {
	lastType reflect.Type
	registry *FieldRegistry
	seen     map[uintptr]bool
	// typeUsage holds this payload's per-type field usage. Encoder-local, so two concurrent
	// Marshal calls cannot corrupt each other's keys header.
	typeUsage map[int]*encoderTypeUsage
}

func NewEncoder() *Encoder {
	return &Encoder{
		registry:  globalRegistry,
		seen:      make(map[uintptr]bool),
		typeUsage: make(map[int]*encoderTypeUsage),
	}
}

// contentBufferPool recycles the scratch buffer the content is rendered into. Responses in this
// codebase run to hundreds of kilobytes, so without pooling every call re-grows a buffer from
// nothing and hands the whole staircase of intermediate slices to the GC.
var contentBufferPool = sync.Pool{
	New: func() any {
		buffer := make([]byte, 0, 64*1024)
		return &buffer
	},
}

// Marshal converts an object to the compact array format.
// Returns a single array with two elements: [keys, content]
//   - keys: type definitions mapping emit position -> field name (JSON tags)
//   - content: the serialized data, fields in emit order, zero values skipped
//
// Single pass, written straight to bytes. The field emit order comes from the registry, frozen
// the first time a type is encountered and reused for every payload after that; the order
// travels with each payload in the keys header, so the decoder never needs to share the
// encoder's history. Field usage is tracked per call on the Encoder, and the keys header is
// rendered afterwards from it — which works because keys are prepended to the output.
func Marshal(v any) ([]byte, error) {
	e := NewEncoder()

	contentBuffer := contentBufferPool.Get().(*[]byte)
	defer func() {
		*contentBuffer = (*contentBuffer)[:0]
		contentBufferPool.Put(contentBuffer)
	}()

	content, err := e.appendValue((*contentBuffer)[:0], reflect.ValueOf(v))
	if err != nil {
		return nil, err
	}
	// Keep the grown buffer for the next caller rather than the stub we started with.
	*contentBuffer = content

	// Learn the emit order for any type seen here for the first time, so the next payload of
	// this shape can put its most-used fields first.
	defer e.learnFieldOrders()

	keys := e.appendKeysList(make([]byte, 0, 256))

	// Assemble [keys, content]. Sized exactly, so this is the only allocation that escapes.
	output := make([]byte, 0, len(keys)+len(content)+3)
	output = append(output, '[')
	output = append(output, keys...)
	output = append(output, ',')
	output = append(output, content...)
	return append(output, ']'), nil
}
