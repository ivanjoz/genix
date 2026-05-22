package text_search

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ErrProtocol is returned when Sonic replies with an unexpected frame.
// Callers treat it as fatal for the connection (discard, don't recycle).
var ErrProtocol = errors.New("text_search: sonic protocol error")

// SonicError wraps an ERR <reason> reply from the daemon.
type SonicError struct{ Reason string }

func (e *SonicError) Error() string { return "text_search: sonic ERR: " + e.Reason }

// resultKind tags the shape of a parsed reply.
type resultKind int

const (
	kindOK resultKind = iota + 1
	kindResult
	kindPending
	kindEvent
	kindPong
	kindEnded
	kindStarted
	kindConnected
	kindErr
)

// sonicResult holds whatever parseResult extracted from one line.
type sonicResult struct {
	kind  resultKind
	count int
	// For PENDING / EVENT: the marker that ties them together.
	marker string
	// For EVENT: the kind (QUERY / SUGGEST) and payload tokens.
	eventKind string
	payload   []string
	// For STARTED: the buffer size advertised by the server.
	bufferSize int
	// For ERR: the reason text.
	reason string
}

// parseResult decodes a single response line from a Sonic connection.
// It does not consume from the reader — callers pass an already-trimmed
// line. parseResult never reads multiple lines (PENDING + EVENT is
// two parseResult calls).
func parseResult(line string) (sonicResult, error) {
	if line == "" {
		return sonicResult{}, fmt.Errorf("%w: empty line", ErrProtocol)
	}
	head, rest, _ := strings.Cut(line, " ")
	switch head {
	case "OK":
		return sonicResult{kind: kindOK}, nil
	case "PONG":
		return sonicResult{kind: kindPong}, nil
	case "RESULT":
		n, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return sonicResult{}, fmt.Errorf("%w: RESULT not an int: %q", ErrProtocol, rest)
		}
		return sonicResult{kind: kindResult, count: n}, nil
	case "PENDING":
		return sonicResult{kind: kindPending, marker: strings.TrimSpace(rest)}, nil
	case "EVENT":
		// EVENT <kind> <marker> <token1> <token2> ...
		eventKind, after, ok := strings.Cut(rest, " ")
		if !ok {
			return sonicResult{}, fmt.Errorf("%w: EVENT missing kind", ErrProtocol)
		}
		marker, payload, _ := strings.Cut(after, " ")
		var tokens []string
		if payload != "" {
			tokens = strings.Fields(payload)
		}
		return sonicResult{kind: kindEvent, eventKind: eventKind, marker: marker, payload: tokens}, nil
	case "CONNECTED":
		return sonicResult{kind: kindConnected}, nil
	case "STARTED":
		// STARTED <mode> protocol(1) buffer(20000)
		bufSize := parseBufferSize(rest)
		return sonicResult{kind: kindStarted, bufferSize: bufSize}, nil
	case "ENDED":
		return sonicResult{kind: kindEnded, reason: strings.TrimSpace(rest)}, nil
	case "ERR":
		return sonicResult{kind: kindErr, reason: strings.TrimSpace(rest)}, &SonicError{Reason: strings.TrimSpace(rest)}
	default:
		return sonicResult{}, fmt.Errorf("%w: unknown frame head %q", ErrProtocol, head)
	}
}

// parseBufferSize extracts N from a STARTED frame's "buffer(N)" segment.
// Defaults to 20000 if absent or malformed (the documented Sonic default).
func parseBufferSize(rest string) int {
	const defaultBuf = 20000
	for _, tok := range strings.Fields(rest) {
		if !strings.HasPrefix(tok, "buffer(") || !strings.HasSuffix(tok, ")") {
			continue
		}
		inner := tok[len("buffer(") : len(tok)-1]
		if n, err := strconv.Atoi(inner); err == nil && n > 0 {
			return n
		}
	}
	return defaultBuf
}

// readLine reads a single CRLF-terminated frame. Returns the line
// without the trailing CRLF. Returns io.EOF if the server closed.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF && line == "" {
			return "", io.EOF
		}
		if err != io.EOF {
			return "", err
		}
	}
	line = strings.TrimRight(line, "\r\n")
	return line, nil
}

// writeLine writes a CRLF-terminated command and flushes the writer.
func writeLine(w *bufio.Writer, line string) error {
	if _, err := w.WriteString(line); err != nil {
		return err
	}
	if _, err := w.WriteString("\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

// quote wraps a payload in double quotes and escapes the four bytes
// Sonic treats specially: backslash, double quote, CR, LF. The
// normalizer already strips everything outside [a-z0-9 ], so this is a
// defensive belt for callers that bypass NormalizeSearchText.
func quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
