package core

import (
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

var trackedHTTPConnections sync.Map
var totalActiveHTTPConnections int64

// UpdateHTTPConnectionState tracks active TCP connections handled by net/http.
// The metric is connection-level (not request-level), which is useful for server load visibility.
func UpdateHTTPConnectionState(connection net.Conn, currentState http.ConnState) {
	if connection == nil {
		return
	}

	switch currentState {
	case http.StateNew:
		// Count each connection once when first seen.
		if _, wasLoaded := trackedHTTPConnections.LoadOrStore(connection, struct{}{}); !wasLoaded {
			atomic.AddInt64(&totalActiveHTTPConnections, 1)
		}
	case http.StateHijacked, http.StateClosed:
		// Remove and decrement only if it was previously tracked.
		if _, wasTracked := trackedHTTPConnections.LoadAndDelete(connection); wasTracked {
			atomic.AddInt64(&totalActiveHTTPConnections, -1)
		}
	}
}

// GetActiveHTTPConnections returns the latest active connection count for operational metrics.
func GetActiveHTTPConnections() int64 {
	return atomic.LoadInt64(&totalActiveHTTPConnections)
}
