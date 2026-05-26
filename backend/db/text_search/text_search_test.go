package text_search

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Unit tests for the pure helpers ----------------------------------

func TestCollectionAndBucket(t *testing.T) {
	cases := []struct {
		table         string
		partition     int32
		group         int8
		wantCol, want string
	}{
		{"productos", 1, 0, "productos", "p1_s0"},
		{"productos", 1, 1, "productos", "p1_s1"},
		{"productos", 57, 1, "productos", "p57_s1"},
		{"clientes", 0, 0, "clientes", "p0_s0"},
		// Negative partitions defensively absolute-valued.
		{"clientes", -3, 1, "clientes", "p3_s1"},
	}
	for _, tc := range cases {
		gotCol, gotBucket := CollectionAndBucket(tc.table, tc.partition, tc.group)
		if gotCol != tc.wantCol || gotBucket != tc.want {
			t.Errorf("CollectionAndBucket(%q,%d,%d) = (%q,%q), want (%q,%q)",
				tc.table, tc.partition, tc.group, gotCol, gotBucket, tc.wantCol, tc.want)
		}
	}
}

func TestPickStatusGroup(t *testing.T) {
	if PickStatusGroup(0) != 0 {
		t.Errorf("PickStatusGroup(0) != 0")
	}
	for _, s := range []int8{1, 2, -1, 127, -128} {
		if PickStatusGroup(s) != 1 {
			t.Errorf("PickStatusGroup(%d) != 1", s)
		}
	}
}

func TestNormalizeSearchText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"Hello World", "hello world"},
		{"  Foo--Bar  baz!!  ", "foo bar baz"},
		{"123-abc.DEF", "123 abc def"},
		{"   ", ""},
		{"only_letters", "only letters"},
	}
	for _, tc := range cases {
		if got := NormalizeSearchText(tc.in); got != tc.want {
			t.Errorf("NormalizeSearchText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidateIdentifier(t *testing.T) {
	good := []string{"productos", "p1_s0", "P57", "_meta"}
	for _, s := range good {
		if err := validateIdentifier(s); err != nil {
			t.Errorf("validateIdentifier(%q) = %v, want nil", s, err)
		}
	}
	bad := []string{"", "1abc", "with space", "weird!", "p-1"}
	for _, s := range bad {
		if err := validateIdentifier(s); err == nil {
			t.Errorf("validateIdentifier(%q) returned nil, want error", s)
		}
	}
}

func TestQuoteHandlesQuotesAndBackslashes(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", `""`},
		{"hello", `"hello"`},
		{`a"b`, `"a\"b"`},
		{`a\b`, `"a\\b"`},
		{"a\nb", `"a\nb"`},
		{"a\tb", `"a\tb"`},
	}
	for _, tc := range cases {
		if got := quote(tc.in); got != tc.want {
			t.Errorf("quote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseResultRecognizesAllFrameTypes(t *testing.T) {
	res, err := parseResult("OK")
	if err != nil || res.kind != kindOK {
		t.Fatalf("OK frame: %v %+v", err, res)
	}
	res, err = parseResult("RESULT 5")
	if err != nil || res.kind != kindResult || res.count != 5 {
		t.Fatalf("RESULT frame: %v %+v", err, res)
	}
	res, err = parseResult("PENDING abc123")
	if err != nil || res.kind != kindPending || res.marker != "abc123" {
		t.Fatalf("PENDING frame: %v %+v", err, res)
	}
	res, err = parseResult("EVENT QUERY abc123 7|50 99|30 1234|10")
	if err != nil || res.kind != kindEvent || res.eventKind != "QUERY" ||
		res.marker != "abc123" || len(res.payload) != 3 {
		t.Fatalf("EVENT frame: %v %+v", err, res)
	}
	res, err = parseResult("PONG")
	if err != nil || res.kind != kindPong {
		t.Fatalf("PONG frame: %v %+v", err, res)
	}
	res, err = parseResult("ENDED quit")
	if err != nil || res.kind != kindEnded {
		t.Fatalf("ENDED frame: %v %+v", err, res)
	}
	res, err = parseResult("CONNECTED <genixsearch v0.1.0>")
	if err != nil || res.kind != kindConnected {
		t.Fatalf("CONNECTED frame: %v %+v", err, res)
	}
	res, err = parseResult("STARTED ingest protocol(1) buffer(20000)")
	if err != nil || res.kind != kindStarted || res.bufferSize != 20000 {
		t.Fatalf("STARTED frame: %v %+v", err, res)
	}
	res, err = parseResult("ERR unknown_command")
	var pErr *ProtocolError
	if !errors.As(err, &pErr) || res.kind != kindErr || pErr.Reason != "unknown_command" {
		t.Fatalf("ERR frame: %v %+v", err, res)
	}
}

func TestDecodeIDsParsesKeyScoreTokens(t *testing.T) {
	ids, err := decodeIDs([]string{"1|50", "42|30", "9999|10"})
	if err != nil {
		t.Fatal(err)
	}
	want := []int32{1, 42, 9999}
	if len(ids) != len(want) {
		t.Fatalf("len=%d, want %d", len(ids), len(want))
	}
	for i, v := range want {
		if ids[i] != v {
			t.Errorf("ids[%d] = %d, want %d", i, ids[i], v)
		}
	}
	// Bare-key token (no '|') still accepted for forward-compat.
	if ids, err := decodeIDs([]string{"7"}); err != nil || len(ids) != 1 || ids[0] != 7 {
		t.Errorf("bare key decode: ids=%v err=%v", ids, err)
	}
	if _, err := decodeIDs([]string{"abc|10"}); err == nil {
		t.Errorf("expected error for non-numeric key")
	}
}

func TestBuildQueryLine(t *testing.T) {
	got := buildQueryLine("productos", 7, 1, "fresa", 20, 10)
	want := `QUERY productos p7_s1 "fresa" LIMIT(20) OFFSET(10)`
	if got != want {
		t.Errorf("buildQueryLine = %q, want %q", got, want)
	}
	// No LIMIT / OFFSET when zero.
	got = buildQueryLine("productos", 7, 1, "fresa", 0, 0)
	want = `QUERY productos p7_s1 "fresa"`
	if got != want {
		t.Errorf("buildQueryLine (no limit/offset) = %q, want %q", got, want)
	}
}

// --- Integration tests with a net.Pipe-based fake daemon -------------

// fakeServer is a deterministic in-process replacement for the search
// daemon, reachable over a single net.Pipe. The test installs the
// server side as a dial target by swapping out the global pool. We
// don't go through dialAndHandshake (which would expect a real TCP
// listener); instead the test wires a conn directly into a private
// pool and exercises the ingest functions.

type fakeServer struct {
	t       *testing.T
	r       *bufio.Reader
	w       *bufio.Writer
	conn    net.Conn
	mu      sync.Mutex
	scripts []string // sequence of canned replies in order
}

func newFakeServer(t *testing.T) (*fakeServer, net.Conn) {
	client, server := net.Pipe()
	f := &fakeServer{
		t:    t,
		r:    bufio.NewReader(server),
		w:    bufio.NewWriter(server),
		conn: server,
	}
	return f, client
}

// queueReply appends a canned line that the fake will send after the
// next inbound command is consumed.
func (f *fakeServer) queueReply(line string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts = append(f.scripts, line)
}

// run loops reading commands and replying with canned scripts. Returns
// the list of commands it saw (without trailing CR/LF) when the client
// closes the connection or the test signals done.
func (f *fakeServer) run(stop <-chan struct{}) []string {
	var got []string
	for {
		line, err := f.r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return got
			}
			return got
		}
		got = append(got, strings.TrimRight(line, "\r\n"))
		f.mu.Lock()
		var reply string
		if len(f.scripts) > 0 {
			reply = f.scripts[0]
			f.scripts = f.scripts[1:]
		}
		f.mu.Unlock()
		if reply != "" {
			// Real server writes \r\n; mirror that here so the client's
			// readLine exercises its CR-stripping path.
			_, _ = f.w.WriteString(reply + "\r\n")
			_ = f.w.Flush()
		}
		select {
		case <-stop:
			return got
		default:
		}
	}
}

// makePooledConn returns a searchConn backed by a client net.Conn,
// bypassing dialAndHandshake. We pre-set bufferSize as if STARTED
// already happened.
func makePooledConn(client net.Conn, bufSize int) *searchConn {
	return &searchConn{
		mode:       "ingest",
		netConn:    client,
		reader:     bufio.NewReader(client),
		writer:     bufio.NewWriter(client),
		bufferSize: bufSize,
		usedAt:     time.Now(),
	}
}

// installFakeIngestPool replaces the global ingest pool with one whose
// only connection is the supplied searchConn. initPools() preserves
// non-nil pools, so subsequent UpsertBatch calls reuse this one.
func installFakeIngestPool(c *searchConn) (restore func()) {
	prev := ingestMgr
	p := &connPool{
		mode:      "ingest",
		min:       1,
		max:       1,
		idle:      []*searchConn{c},
		open:      1,
		pendingCh: make(chan struct{}, 1),
	}
	ingestMgr = p
	return func() {
		ingestMgr = prev
	}
}

func TestUpsertBatchSendsPopIThenPushI(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	// PING at pool checkout, then POPI×2 on the other bucket (s0),
	// then one PUSHI multi-key on the current bucket (s1).
	fake.queueReply("PONG")
	fake.queueReply("RESULT 0") // POPI fruits p7_s0 1
	fake.queueReply("RESULT 0") // POPI fruits p7_s0 2
	fake.queueReply("RESULT 2") // PUSHI fruits p7_s1 1 "apple" 2 "banana"

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	conn := makePooledConn(client, 20000)
	restore := installFakeIngestPool(conn)
	defer restore()

	records := []Record{
		{ID: 1, SearchText: "apple"},
		{ID: 2, SearchText: "banana"},
	}
	if err := UpsertBatch(context.Background(), "fruits", 7, 1, records); err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}
	_ = client.Close()
	<-done

	want := []string{
		"PING",
		"POPI fruits p7_s0 1",
		"POPI fruits p7_s0 2",
		`PUSHI fruits p7_s1 1 "apple" 2 "banana"`,
	}
	if len(got) < len(want) {
		t.Fatalf("got %d frames, want at least %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestUpsertBatchEmptyTextOnlyPopsBothBuckets(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	fake.queueReply("RESULT 0") // POPI x p1_s1 9 (other)
	fake.queueReply("RESULT 1") // POPI x p1_s0 9 (current)

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	conn := makePooledConn(client, 20000)
	restore := installFakeIngestPool(conn)
	defer restore()

	if err := UpsertBatch(context.Background(), "x", 1, 0, []Record{{ID: 9, SearchText: ""}}); err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}
	_ = client.Close()
	<-done

	want := []string{
		"PING",
		"POPI x p1_s1 9",
		"POPI x p1_s0 9",
	}
	if len(got) < len(want) {
		t.Fatalf("got %d frames, want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q", i, got[i], w)
		}
	}
	for _, frame := range got {
		if strings.HasPrefix(frame, "PUSHI ") {
			t.Errorf("unexpected PUSHI for empty SearchText: %q", frame)
		}
	}
}

func TestUpsertBatchMixedEmptyAndText(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	// POPI p1_s0 sweep (other bucket) for all three IDs.
	fake.queueReply("RESULT 0")
	fake.queueReply("RESULT 0")
	fake.queueReply("RESULT 0")
	// POPI p1_s1 sweep (current bucket) for the empty-text ID only.
	fake.queueReply("RESULT 1")
	// PUSHI for the two non-empty.
	fake.queueReply("RESULT 2")

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	conn := makePooledConn(client, 20000)
	restore := installFakeIngestPool(conn)
	defer restore()

	records := []Record{
		{ID: 10, SearchText: "alpha"},
		{ID: 11, SearchText: ""},
		{ID: 12, SearchText: "beta"},
	}
	if err := UpsertBatch(context.Background(), "x", 1, 1, records); err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}
	_ = client.Close()
	<-done

	want := []string{
		"PING",
		"POPI x p1_s0 10",
		"POPI x p1_s0 11",
		"POPI x p1_s0 12",
		"POPI x p1_s1 11",
		`PUSHI x p1_s1 10 "alpha" 12 "beta"`,
	}
	if len(got) < len(want) {
		t.Fatalf("got %d frames, want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestDeleteRecordUsesPopI(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	fake.queueReply("RESULT 1") // POPI items p4_s0 42
	fake.queueReply("RESULT 0") // POPI items p4_s1 42

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	conn := makePooledConn(client, 20000)
	restore := installFakeIngestPool(conn)
	defer restore()

	if err := DeleteRecord(context.Background(), "items", 4, 0, 42); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	_ = client.Close()
	<-done

	want := []string{
		"PING",
		"POPI items p4_s0 42",
		"POPI items p4_s1 42",
	}
	if len(got) < len(want) {
		t.Fatalf("got %d frames, want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestDeleteBatchSweepsBothBuckets(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	fake.queueReply("RESULT 1") // s0 / 1
	fake.queueReply("RESULT 1") // s0 / 2
	fake.queueReply("RESULT 0") // s1 / 1
	fake.queueReply("RESULT 0") // s1 / 2

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	conn := makePooledConn(client, 20000)
	restore := installFakeIngestPool(conn)
	defer restore()

	if err := DeleteBatch(context.Background(), "t", 1, []int32{1, 2}); err != nil {
		t.Fatalf("DeleteBatch: %v", err)
	}
	_ = client.Close()
	<-done

	want := []string{
		"PING",
		"POPI t p1_s0 1",
		"POPI t p1_s0 2",
		"POPI t p1_s1 1",
		"POPI t p1_s1 2",
	}
	if len(got) < len(want) {
		t.Fatalf("got %d frames, want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestPushISplitsWhenLineWouldOverflow(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	fake.queueReply("RESULT 0") // POPI t p1_s1 1 (other bucket)
	fake.queueReply("RESULT 0") // POPI t p1_s1 2
	fake.queueReply("RESULT 1") // PUSHI chunk 1
	fake.queueReply("RESULT 1") // PUSHI chunk 2

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	// Buffer sized so two records cannot fit on one PUSHI line.
	// "PUSHI t p1_s0" header = 13 chars; entry ` 1 "ab"` = 7 chars.
	// bufferSize=22 -> budget=21. 13+7=20 fits; +7 more = 27 doesn't.
	conn := makePooledConn(client, 22)
	restore := installFakeIngestPool(conn)
	defer restore()

	records := []Record{
		{ID: 1, SearchText: "ab"},
		{ID: 2, SearchText: "cd"},
	}
	if err := UpsertBatch(context.Background(), "t", 1, 0, records); err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}
	_ = client.Close()
	<-done

	want := []string{
		"PING",
		"POPI t p1_s1 1",
		"POPI t p1_s1 2",
		`PUSHI t p1_s0 1 "ab"`,
		`PUSHI t p1_s0 2 "cd"`,
	}
	if len(got) < len(want) {
		t.Fatalf("got %d frames, want %d: %q", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestPushITruncatesOversizeText(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	fake.queueReply("RESULT 0") // POPI other bucket
	fake.queueReply("RESULT 1") // PUSHI (trimmed)

	stop := make(chan struct{})
	defer close(stop)
	var got []string
	done := make(chan struct{})
	go func() {
		got = fake.run(stop)
		close(done)
	}()

	// bufferSize 34 -> budget 33. Header `PUSHI t p1_s0` = 13.
	// Entry prefix ` 1 "` = 4, trailing `"` = 1, so per-entry overhead = 5.
	// Remaining text budget = 33 - 13 - 5 = 15.
	conn := makePooledConn(client, 34)
	restore := installFakeIngestPool(conn)
	defer restore()

	text := "one two three four five six"
	if err := UpsertBatch(context.Background(), "t", 1, 0, []Record{{ID: 1, SearchText: text}}); err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}
	_ = client.Close()
	<-done

	if len(got) < 3 {
		t.Fatalf("expected >=3 frames, got %d: %q", len(got), got)
	}
	pushLine := got[2]
	if !strings.HasPrefix(pushLine, `PUSHI t p1_s0 1 "`) || !strings.HasSuffix(pushLine, `"`) {
		t.Fatalf("unexpected PUSHI frame: %q", pushLine)
	}
	if len(pushLine) > 33 {
		t.Errorf("PUSHI frame exceeds budget: len=%d frame=%q", len(pushLine), pushLine)
	}
	// The trimmed text must be a word-boundary prefix of the original.
	quoted := strings.TrimSuffix(strings.TrimPrefix(pushLine, `PUSHI t p1_s0 1 "`), `"`)
	if !strings.HasPrefix(text, quoted) || strings.HasSuffix(quoted, " ") {
		t.Errorf("trimmed text not a clean prefix: %q (orig %q)", quoted, text)
	}
}

func TestUpsertBatchRecoversFromProtocolError(t *testing.T) {
	fake, client := newFakeServer(t)
	defer client.Close()
	defer fake.conn.Close()

	fake.queueReply("PONG")
	fake.queueReply("ERR something_broken")

	stop := make(chan struct{})
	defer close(stop)
	done := make(chan struct{})
	go func() {
		_ = fake.run(stop)
		close(done)
	}()

	conn := makePooledConn(client, 20000)
	restore := installFakeIngestPool(conn)
	defer restore()

	err := UpsertBatch(context.Background(), "x", 1, 1, []Record{{ID: 1, SearchText: "foo"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pErr *ProtocolError
	if !errors.As(err, &pErr) {
		t.Fatalf("expected ProtocolError, got %T: %v", err, err)
	}
	if pErr.Reason != "something_broken" {
		t.Errorf("unexpected reason: %q", pErr.Reason)
	}
	_ = client.Close()
	<-done
}

func TestPoolDiscardsBrokenConnection(t *testing.T) {
	// A connection that fails the PING at acquire-time is discarded.
	p := newConnPool("ingest", 0, 1)
	broken := &searchConn{
		mode:    "ingest",
		netConn: brokenConn{},
		reader:  bufio.NewReader(brokenConn{}),
		writer:  bufio.NewWriter(brokenConn{}),
		usedAt:  time.Now(),
	}
	p.idle = append(p.idle, broken)
	p.open = 1
	// acquire will try PING, fail, drop the conn, then attempt to dial
	// — which will fail because Configure was not called in this test.
	// We expect the dial error, NOT a "pool closed" / panic.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := p.acquire(ctx)
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
}

type brokenConn struct{}

func (brokenConn) Read(_ []byte) (int, error)         { return 0, io.ErrClosedPipe }
func (brokenConn) Write(_ []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (brokenConn) Close() error                       { return nil }
func (brokenConn) LocalAddr() net.Addr                { return nil }
func (brokenConn) RemoteAddr() net.Addr               { return nil }
func (brokenConn) SetDeadline(_ time.Time) error      { return nil }
func (brokenConn) SetReadDeadline(_ time.Time) error  { return nil }
func (brokenConn) SetWriteDeadline(_ time.Time) error { return nil }
