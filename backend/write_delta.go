package main

import (
	"app/core"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// writeDeltaParam is the query param a POST/PUT carries when the client wants the delta of one of
// its cached routes in the same response, instead of a second GET after the write. Its value is the
// GET route and its query as "route~key~value~key~value" (genix-ui GetHandler.postAndSync builds it
// with the route's delta cursors, exactly what its delta GET would send).
const writeDeltaParam = "dl"

// parseWriteDeltaRequest reads "route~key~value~key~value". "~" never appears in a route, a query
// key or a delta cursor, which are digits and dots, so it needs no escaping.
func parseWriteDeltaRequest(deltaRequest string) (route string, query map[string]string, err error) {
	parts := strings.Split(deltaRequest, "~")
	if parts[0] == "" || len(parts)%2 == 0 {
		return "", nil, fmt.Errorf("malformed delta request %q: expected route~key~value pairs", deltaRequest)
	}
	query = make(map[string]string, len(parts)/2)
	for partIndex := 1; partIndex+1 < len(parts); partIndex += 2 {
		query[parts[partIndex]] = parts[partIndex+1]
	}
	return parts[0], query, nil
}

// attachWriteDelta runs the GET handler the write's `dl` names, after the write succeeded, and
// answers both in one body: {"Response": <the write's own answer>, "Delta": <the GET's answer>}.
// The GET runs after the write, so its delta always holds the write; it reads from the client's own
// cursor, so it also holds every other change the client had not seen.
//
// The GET is checked for access like a direct one but charged nothing: it replaces a request the
// client no longer sends. Any failure of the GET leaves "Delta" null and the write still succeeds;
// the client then marks the route for a normal refresh.
func attachWriteDelta(args *core.HandlerArgs, writeResponse core.HandlerResponse) core.HandlerResponse {
	deltaBody := []byte("null")
	if readBody, err := runWriteDeltaRead(args); err != nil {
		core.Log("write delta skipped::", args.Query[writeDeltaParam], "::", err)
	} else {
		deltaBody = readBody
	}

	writeBody := []byte("null")
	if writeResponse.Body != nil && len(*writeResponse.Body) > 0 {
		writeBody = *writeResponse.Body
		// A handler answering plain text (MakeResponse of a string) is wrapped as a JSON string.
		if !json.Valid(writeBody) {
			writeBody, _ = json.Marshal(string(writeBody))
		}
	}

	combinedBody := make([]byte, 0, len(writeBody)+len(deltaBody)+24)
	combinedBody = append(combinedBody, `{"Response":`...)
	combinedBody = append(combinedBody, writeBody...)
	combinedBody = append(combinedBody, `,"Delta":`...)
	combinedBody = append(combinedBody, deltaBody...)
	combinedBody = append(combinedBody, '}')

	writeResponse.Body = &combinedBody
	writeResponse.IsBinary = false
	if writeResponse.Headers == nil {
		writeResponse.Headers = map[string]string{}
	}
	writeResponse.Headers["Content-Type"] = "application/json; charset=utf-8"
	return writeResponse
}

// runWriteDeltaRead authorizes and runs the GET the write's `dl` names, and returns its JSON body.
func runWriteDeltaRead(args *core.HandlerArgs) ([]byte, error) {
	route, query, err := parseWriteDeltaRequest(args.Query[writeDeltaParam])
	if err != nil {
		return nil, err
	}
	readFuncPath := "GET." + route
	readHandler, found := appHandlers[readFuncPath]
	if !found {
		return nil, fmt.Errorf("no handler for %s", readFuncPath)
	}

	readUser, err := authorizeWriteDeltaRead(args, readFuncPath)
	if err != nil {
		return nil, err
	}

	// A copy of the write's request, turned into the GET: same caller, the delta's own query, and
	// JSON so its body can be spliced into the write's.
	readArgs := *args
	readArgs.Method = "GET"
	readArgs.Route = route
	readArgs.Query = query
	readArgs.QueryString = ""
	readArgs.Body = nil
	readArgs.Accept = "application/json"
	readArgs.RouteID = core.APIRouteID(readFuncPath)
	readArgs.User = readUser

	core.Log("Ejecutando write delta::", readFuncPath+formatDevQueryParams(query))
	readResponse := readHandler(&readArgs)
	if readResponse.Error != "" {
		return nil, errors.New(readResponse.Error)
	}
	if readResponse.StreamHandled || readResponse.IsBinary || readResponse.Body == nil {
		return nil, fmt.Errorf("%s did not answer a JSON body", readFuncPath)
	}
	readBody := bytes.TrimSpace(*readResponse.Body)
	if len(readBody) == 0 || (readBody[0] != '[' && readBody[0] != '{') || !json.Valid(readBody) {
		return nil, fmt.Errorf("%s did not answer a JSON body", readFuncPath)
	}
	return readBody, nil
}

// authorizeWriteDeltaRead applies the access rule of a direct GET to the delta read and returns the
// user the GET handler runs as: a copy, since the sub-accesses it holds are those of the GET route,
// not of the write. Only the access is asked of fareward: the read is not charged.
func authorizeWriteDeltaRead(args *core.HandlerArgs, readFuncPath string) (*core.UsuarioToken, error) {
	accessInfos, _ := accessHelper.GetAccesosByRoute(readFuncPath)
	decision := resolveRouteAccess("GET", readFuncPath, args.User.ID, accessInfos)
	if decision.denyMessage != "" {
		return nil, errors.New(decision.denyMessage)
	}

	readUser := *args.User
	readArgs := &core.HandlerArgs{User: &readUser}
	if core.Env.FAREWARD_BYPASS {
		grantEveryRouteAccess(readArgs, accessInfos)
		return &readUser, nil
	}
	if len(decision.requiredAccess) == 0 {
		applyGrantedSubAccesos(readArgs, nil, nil, accessInfos)
		return &readUser, nil
	}

	if !core.Env.IS_SERVERLESS {
		if cachedGrant, found := core.LookupAccessGrant(
			args.User.CompanyID, args.User.ID, decision.requiredAccess); found {
			applyGrantedSubAccesos(readArgs, cachedGrant, decision.accessInfos, accessInfos)
			return &readUser, nil
		}
	}

	requestContext := context.Background()
	if args.ReqContext != nil {
		requestContext = args.ReqContext.Context()
	}
	accessGrant, err := core.ChargeAPIAccessOnly(requestContext, args.User.CompanyID, args.User.ID,
		core.APIRouteID(readFuncPath), decision.requiredAccess)
	core.RecordAccessCheckOutcome(args.User.CompanyID, args.User.ID, decision.requiredAccess, accessGrant, err)
	if err != nil {
		return nil, err
	}
	applyGrantedSubAccesos(readArgs, accessGrant, decision.accessInfos, accessInfos)
	return &readUser, nil
}
