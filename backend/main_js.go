//go:build js

package main

// main is the browser build's entry point (GOOS=js GOARCH=wasm): the backend runs inside a Web
// Worker over IndexedDB. For now it only builds the router and keeps the Go runtime alive; exposing
// mainHandler to JavaScript is phase 3 of docs/WASM_LOCAL_MODE_PLAN.md.
func main() {
	makeAppHandlers()
	select {}
}
