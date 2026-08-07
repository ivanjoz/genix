// Command sse_bridge relays Server-Sent Events between the Genix backend and
// browser tabs.
//
// Why it exists: AWS Lambda cannot hold a stream open for the duration of an
// agent turn, and it certainly cannot receive the browser's answer inside the
// same invocation. This process runs on a normal server (EC2/VPS), keeps the
// browser's connection, and acts as the rendezvous point in both directions:
// the backend pushes events with POST /publish and issues blocking commands with
// POST /rpc, while the browser reads GET /sse and answers on POST /in.
//
// It holds no business logic and no database connection. Messages are opaque
// JSON and nothing is buffered: a message for a disconnected tab is dropped.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	config, configError := LoadBridgeConfig()
	if configError != nil {
		log.Fatalln("[sse-bridge] configuración inválida ::", configError)
	}

	server := newBridgeServer(config)
	listenAddress := ":" + strconv.Itoa(config.ListenPort)

	httpServer := &http.Server{
		Addr:              listenAddress,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// WriteTimeout and IdleTimeout stay at zero on purpose: every deadline
		// here would kill a healthy long-lived SSE stream. Dead connections are
		// detected by the keepalive write failing instead.
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logInfo("escuchando en", listenAddress, ":: verbose:", config.VerboseLogs)
		if listenError := httpServer.ListenAndServe(); listenError != nil && listenError != http.ErrServerClosed {
			log.Fatalln("[sse-bridge] no se pudo escuchar ::", listenError)
		}
	}()

	<-shutdownSignals
	logInfo("apagando ::", server.registry.ConnectedChannelCount(), "canales conectados")

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if shutdownError := httpServer.Shutdown(shutdownContext); shutdownError != nil {
		logWarn("apagado forzado ::", shutdownError)
	}
}

func logInfo(messageParts ...any) {
	log.Println(append([]any{"[sse-bridge]"}, messageParts...)...)
}

func logWarn(messageParts ...any) {
	log.Println(append([]any{"[sse-bridge][warn]"}, messageParts...)...)
}
