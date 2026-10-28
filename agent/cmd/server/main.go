// The agent's own binary: the chat turn, the model list and the browser stream, apart from the
// backend so the backend never links the LLM stack. It runs as a Lambda (BUFFERED Function URL,
// the stream then lives on the SSE bridge) or as a plain HTTP server locally and on a VPS.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"app/core"
	"genix-agent"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	fareward "github.com/ivanjoz/fareward/go"
	"github.com/ivanjoz/genix-orm/scylla"
	"github.com/rs/cors"
)

// serveAgentRoute is the agent's whole router. The backend's gate (access catalog, base credits)
// does not run here: p-agent-turn validates the session and charges its own turn, and agent-models
// only needs a valid session.
func serveAgentRoute(args *core.HandlerArgs) (handlerResponse core.HandlerResponse) {
	defer func() {
		if recovered := recover(); recovered != nil {
			core.Log(string(debug.Stack()))
			handlerResponse = core.HandlerResponse{Error: fmt.Sprintf("Internal Server Error (Panic): %v", recovered), StatusCode: 500}
		}
	}()
	args.Authorization = core.MapGetKeys(args.Headers, "Authorization", "authorization")
	args.Encoding = core.MapGetKeys(args.Headers, "Accept-Encoding", "accept-encoding")
	args.Accept = core.MapGetKeys(args.Headers, "Accept", "accept")
	args.RequestID = core.MakeRequestID()

	// The frontend calls <agent host>/api/<route>, the same shape as the backend's routes.
	route := strings.TrimPrefix(strings.TrimPrefix(args.Route, "/"), "api/")
	args.Route = route
	funcPath := args.Method + "." + route
	args.RouteID = core.APIRouteID(funcPath)
	core.Log("agent route::", funcPath, "clientIP:", args.ClientIP)

	handlerFunc, exists := agent.ModuleHandlers[funcPath]
	if !exists {
		return core.HandlerResponse{Error: "the agent has no route " + funcPath, StatusCode: 404}
	}
	if strings.HasPrefix(route, "p-") {
		args.User = &core.UsuarioToken{}
	} else if args.User = core.CheckUser(args, 0); len(args.User.Error) > 0 {
		return core.HandlerResponse{Error: args.User.Error, StatusCode: 401}
	}
	return handlerFunc(args)
}

// lambdaHandler serves the BUFFERED Function URL: a turn answers {Ok:true} and everything else
// travels on the bridge, so no response is large enough to need streaming.
func lambdaHandler(_ context.Context, request *events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	args := core.HandlerArgs{
		Body:     &request.Body,
		Query:    request.QueryStringParameters,
		Headers:  request.Headers,
		Route:    request.RequestContext.HTTP.Path,
		Method:   request.RequestContext.HTTP.Method,
		ClientIP: request.RequestContext.HTTP.SourceIP,
	}
	handlerResponse := serveAgentRoute(&args)
	if len(handlerResponse.Error) > 0 {
		return core.MakeErrRespFinal(int32(core.If(handlerResponse.StatusCode != 0, handlerResponse.StatusCode, 400)), handlerResponse.Error), nil
	}
	return core.MakeResponseFinal(&handlerResponse), nil
}

func localHandler(writer http.ResponseWriter, request *http.Request) {
	bodyBytes, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, 10<<20))
	if err != nil {
		http.Error(writer, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	body := string(bodyBytes)
	args := core.HandlerArgs{
		Body:           &body,
		Method:         strings.ToUpper(request.Method),
		Route:          request.URL.Path,
		ClientIP:       core.ClientIPFromRequest(request),
		ResponseWriter: &writer,
		ReqContext:     request,
		StartTime:      time.Now().UnixMilli(),
		Query:          map[string]string{},
		Headers:        map[string]string{},
	}
	for key, values := range request.URL.Query() {
		args.Query[key] = strings.Join(values, ",")
	}
	for key, values := range request.Header {
		args.Headers[key] = strings.Join(values, ",")
	}
	core.SendLocalResponse(args, serveAgentRoute(&args))
}

func main() {
	core.PopulateVariables()
	core.LogHistoricalClockIfActive()
	fareward.SetLogger(core.Log)
	if err := core.ConfigureFareward(core.Env.FAREWARD_ADDRESS, core.Env.INTERNAL_APIKEY); err != nil {
		panic("invalid fareward configuration: " + err.Error())
	}
	core.ConfigureTextSearch()
	core.ConfigureFileStore()
	scylla.SetScyllaConnection(scylla.ConnParams{
		Host:             core.Env.DB_HOST,
		Port:             int(core.Env.DB_PORT),
		User:             core.Env.DB_USER,
		Password:         core.Env.DB_PASSWORD,
		Keyspace:         core.Env.DB_NAME,
		MaxClusteringKey: int(core.Env.MAX_CLUSTERING_KEY),
	})
	agent.InitPromptLog()

	if core.Env.IS_SERVERLESS {
		lambda.Start(lambdaHandler)
		return
	}

	serverPort := fmt.Sprintf(":%d", core.If(core.Env.AGENT_PORT > 0, core.Env.AGENT_PORT, 3590))
	// A dev run holds the production database credentials on a port with nothing in front of it,
	// so it answers only on loopback (the backend also binds its tailnet address; the agent does not).
	if core.Env.IS_DEV_ARG {
		serverPort = "127.0.0.1" + serverPort
	}
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodPost, http.MethodGet},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"X-Metadata", "X-Rate-Limit-Code"},
		AllowCredentials: false,
	})
	// /agent/stream is the tab's permanent event stream (chat events AND page commands); /agent/in
	// carries every browser→agent message. The turn itself is the plain route POST p-agent-turn, so
	// the same client code works against Lambda, where the stream lives on the SSE bridge instead.
	// No method prefix on the CORS patterns: the middleware answers the JSON POST's preflight too.
	mux := http.NewServeMux()
	mux.Handle("/agent/stream", corsMiddleware.Handler(http.HandlerFunc(agent.HandleStream)))
	mux.Handle("/agent/in", corsMiddleware.Handler(http.HandlerFunc(agent.HandleIn)))
	// HTTP entrypoint for external LLM agents (Claude Code / Gemini): batch actions in, post-action
	// page snapshot out. GET serves read-only side-channel queries (`?get=menu`).
	mux.HandleFunc("POST /agent", agent.HandleAgentHTTP)
	mux.HandleFunc("GET /agent", agent.HandleAgentGet)
	mux.Handle("/", corsMiddleware.Handler(http.HandlerFunc(localHandler)))

	server := &http.Server{
		Addr:              serverPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// The stream and a turn are long-lived by design.
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
	core.Log("Agent server listening on " + serverPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		core.Log("agent HTTP server error:", err)
		os.Exit(1)
	}
}
