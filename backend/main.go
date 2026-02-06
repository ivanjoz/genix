package main

import (
	"app/core"
	"app/db"
	"app/exec"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/cors"
)

func LambdaHandler(_ context.Context, request *events.APIGatewayV2HTTPRequest) (resp *events.APIGatewayV2HTTPResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic in LambdaHandler): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))
			resp = core.MakeErrRespFinal(500, errStr)
			err = nil // return nil error to Lambda runtime so it sends our response
		}
	}()
	clearEnvVariables()

	core.Env.REQ_IP = request.RequestContext.HTTP.SourceIP
	if len(request.Body) > 0 {
		core.Log("*body enviado: ", core.StrCut(request.Body, 400))
	}

	// Revisa si lo que se está pidiendo es ejecutar una funcion
	if len(request.Body) > 11 && request.Body[0:11] == `{"fn_exec":` {
		funcResponse := ExecFuncHandler(request.Body)
		bodyBytes := []byte(core.ToJsonNoErr(funcResponse))
		core.Log("*Body response::" + core.StrCut(string(bodyBytes), 400))
		response := core.HandlerResponse{Body: &bodyBytes, Headers: map[string]string{}}
		return core.MakeResponseFinal(&response), nil
	}

	route := request.RequestContext.HTTP.Path
	if len(route) == 0 {
		route = request.RawPath
	}
	if len(route) == 0 {
		core.Log("No custom path given, but AWS routed this request to this Lambda anyways.")
		route = "MISSING"
	}

	args := core.HandlerArgs{
		Body:    &request.Body,
		Query:   request.QueryStringParameters,
		Headers: request.Headers,
		Route:   route,
		Method:  request.RequestContext.HTTP.Method,
	}
	response := mainHandler(&args)
	return response.LambdaResponse, nil
}

func LocalHandler(w http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("Internal Server Error (Panic in LocalHandler): %v", r)
			core.Logx(5, errStr)
			core.Log(string(debug.Stack()))

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			errorMap := map[string]string{
				"error": errStr,
			}
			errorJson := core.ToJsonNoErr(errorMap)
			w.Write([]byte(errorJson))
		}
	}()

	const maxBodyBytes = int64(10 << 20) // 10 MiB
	bodyReader := http.MaxBytesReader(w, request.Body, maxBodyBytes)
	bodyBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	body := string(bodyBytes)

	args := core.HandlerArgs{
		Body:           &body,
		Method:         strings.ToUpper(request.Method),
		Route:          request.URL.Path,
		ResponseWriter: &w,
		ReqContext:     request,
		StartTime:      time.Now().UnixMilli(),
	}

	// Revisa si lo que se está pidiendo es ejecutar una funcion
	if len(body) > 11 && body[0:11] == `{"fn_exec":` {
		core.Log("Ejecutando funcion...")
		funcResponse := ExecFuncHandler(body)
		lambdaResponse := map[string]any{
			"statusCode": 200,
			"body":       core.ToJsonNoErr(funcResponse),
			"headers": map[string]string{
				"Content-Type": "application/json; charset=utf-8",
			},
		}
		body := core.ToJsonNoErr(lambdaResponse)
		bodyBytes := []byte(body)
		response := core.HandlerResponse{Body: &bodyBytes}
		core.SendLocalResponse(args, response)
		return
	}

	// Convierte los query params en un map[string]: stirng
	queryString := request.URL.Query()
	args.Query = make(map[string]string)

	for key, values := range queryString {
		value := strings.Join(values[:], ",")
		args.Query[key] = value
	}

	// Convierte los headers en un map[string]: string
	args.Headers = make(map[string]string)

	for key, values := range request.Header {
		value := strings.Join(values[:], ",")
		args.Headers[key] = value
	}

	mainHandler(&args)
}

func OnPanic(panicMessage interface{}) {
	core.Logx(5, "Error 500 (Panic): ", panicMessage)
	core.Log(string(debug.Stack()))
}

func main() {
	serverPort := ":3589"
	core.PopulateVariables()
	makeAppHandlers()
	fmt.Println("Setting full logs...")
	// os.Setenv("LOGS_FULL", "1")

	if !core.Env.IS_LOCAL { // Controla los panic error
		defer func() {
			if r := recover(); r != nil {
				OnPanic(r)
			}
		}()
	}

	db.SetScyllaConnection(db.ConnParams{
		Host:     core.Env.DB_HOST,
		Port:     int(core.Env.DB_PORT),
		User:     core.Env.DB_USER,
		Password: core.Env.DB_PASSWORD,
		Keyspace: core.Env.DB_NAME,
	})

	invokeFun := ""
	for _, value := range os.Args {
		if len(value) >= 2 && value[0:2] == "fn" {
			core.Env.LOGS_FULL = true
			core.Env.APP_CODE = "smbr-qas"
			invokeFun = value
		}
		if value == "prod" {
			core.Env.APP_CODE = "smbr-prod"
		}
	}

	// Revisa si lo que se requiere es ejecutar una función
	if len(invokeFun) != 0 {
		fmt.Println("Invocando función...")
		funcToInvoke, ok := exec.ExecHandlers[invokeFun]
		if !ok {
			funcToInvoke, ok = exec.ExecHandlersTesting[invokeFun]
		}
		if !ok {
			core.Log("No se encontró la función a ejecutar:: ", invokeFun)
			return
		}
		args := core.ExecArgs{Message: ""}
		funcToInvoke(&args)
		return
	}

	// Si se está desarrollando en local
	if core.Env.IS_LOCAL {
		core.Log("Ejecutando en local. http://localhost" + serverPort)

		corsMiddleware := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{http.MethodPost, http.MethodPut, http.MethodGet},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: false,
		})
		// Inicia el servidor con timeouts (previene slowloris y mejora resiliencia).
		srv := &http.Server{
			Addr:              serverPort,
			Handler:           corsMiddleware.Handler(http.HandlerFunc(LocalHandler)),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1 MiB
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			core.Log("HTTP server error:", err)
		}

	} else {
		// Si se está en Lamnda
		logger := log.New(os.Stdout, "", log.LstdFlags|log.Llongfile)
		logger.Println("Lambda has started.")
		// The main goroutine in a Lambda might never run its deferred statements.
		// This is because of how the Lambda is shutdown.
		defer logger.Println("Lambda has stopped.")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lambda.StartWithOptions(LambdaHandler, lambda.WithContext(ctx))
	}
}
