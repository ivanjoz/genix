package types

import (
	"app/db"
	"hash/fnv"
)

// One row per distinct code line that has ever failed.
//
// The code line is the identity, not the message: two failures at responses.go:539 are the same
// error however differently they phrase themselves. That is what keeps this table bounded by the
// size of the codebase instead of by traffic — a route failing ten thousand times an hour still
// occupies one row.
//
// Text is a preview only. The real message and its stack are in CloudWatch, under the RequestID of
// any user_logs row whose ErrorIDs contain this ID.
type RequestError struct {
	db.TableStruct[RequestErrorTable, RequestError]
	ID       int32  `json:",omitempty"`
	CodeLine string `json:",omitempty"`
	Text     string `json:",omitempty"`
	Updated  int32  `json:"upd,omitempty"`
}

type RequestErrorTable struct {
	db.TableStruct[RequestErrorTable, RequestError]
	ID       db.Col[RequestErrorTable, int32]
	CodeLine db.Col[RequestErrorTable, string]
	Text     db.Col[RequestErrorTable, string]
	Updated  db.Col[RequestErrorTable, int32]
}

func (e RequestErrorTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// Written externally, like user_logs and credit_usage.
		ID:        45,
		Name:      "request_errors",
		Partition: e.ID,
		// Clustering by the code line rather than letting the hash be the whole key: two distinct
		// lines that collide on the same int32 then become two rows in one partition instead of
		// silently overwriting each other, and a reader can tell them apart.
		Keys:                  db.Cols(e.CodeLine),
		DisableDefaultColumns: true,
	}
}

// MakeRequestErrorID hashes a code line into the ID that user_logs rows reference. FNV-1a 32,
// masked to stay positive because the column is a signed int32 and a negative ID reads as a bug
// every time someone sees one.
//
// Mirrored in server_utils/src/reqlog/protocol.rs — though the daemon only ever receives this
// value, it never recomputes it, so the Go side is the sole authority on what a code line hashes
// to. The Rust test vectors exist to catch the day that stops being true.
func MakeRequestErrorID(codeLine string) int32 {
	hash := fnv.New32a()
	hash.Write([]byte(codeLine))
	return int32(hash.Sum32() & 0x7FFF_FFFF)
}
