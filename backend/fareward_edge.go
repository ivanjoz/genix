package main

import (
	"encoding/hex"
	"net"
	"strconv"
	"strings"

	"app/core"

	fareward "github.com/ivanjoz/fareward/go"
)

// The headers HAProxy sets once the fareward SPOE agent admitted a request (fareward plan D5,
// deploy/haproxy/haproxy.cfg in the fareward repository). HAProxy deletes any copy the client sent
// before the agent runs, so on a connection that only HAProxy can open they are the agent's.
const (
	edgeAdmittedHeader = "X-Fareward-Admitted"
	edgeCompanyHeader  = "X-Fareward-Company"
	edgeUserHeader     = "X-Fareward-User"
	edgeRouteHeader    = "X-Fareward-Route"
	edgeGrantHeader    = "X-Fareward-Grant"
)

// edgeAdmission is the gate's verdict when the edge already took it: the agent authenticated the
// same token, asked the limiter the same question with the same slots and charged the same base, so
// asking the daemon again would charge the request twice.
//
// It answers handled=false whenever the headers cannot be believed or do not describe this very
// request, and the caller then runs the normal gate. That fallback is safe in both directions: a
// request the edge did not admit is judged here as before, and the worst a mismatch costs is a
// second charge. Only a grant that cannot be read fails closed, because by then the edge has
// already charged and the slots it granted are unknown.
func edgeAdmission(args *core.HandlerArgs, funcPath string) (handled bool, response *core.HandlerResponse) {
	if !core.Env.FAREWARD_TRUST_EDGE || core.Env.FAREWARD_BYPASS || args.ReqContext == nil {
		return false, nil
	}
	request := args.ReqContext
	if request.Header.Get(edgeAdmittedHeader) != "1" {
		return false, nil
	}
	if !isLoopbackPeer(request.RemoteAddr) {
		core.Log("fareward edge: admitted header from a non-local peer ignored::", request.RemoteAddr)
		return false, nil
	}

	companyID, companyErr := strconv.ParseInt(request.Header.Get(edgeCompanyHeader), 10, 32)
	userID, userErr := strconv.ParseInt(request.Header.Get(edgeUserHeader), 10, 32)
	routeID, routeErr := strconv.ParseInt(request.Header.Get(edgeRouteHeader), 10, 16)
	if companyErr != nil || userErr != nil || routeErr != nil ||
		int32(companyID) != args.User.CompanyID || int32(userID) != args.User.ID ||
		int16(routeID) != args.RouteID {
		// The route table and this binary disagree (a stale file) or the token decoded differently:
		// judge the request here rather than act on a verdict about some other request.
		core.Log("fareward edge: headers do not match the request, running the gate::",
			" company::", request.Header.Get(edgeCompanyHeader), "/", args.User.CompanyID,
			" user::", request.Header.Get(edgeUserHeader), "/", args.User.ID,
			" route::", request.Header.Get(edgeRouteHeader), "/", args.RouteID)
		return false, nil
	}

	// The policy is re-read rather than trusted: it is local and costs nothing, it keeps a denied
	// route denied even if the file the agent loaded was older than this binary, and it is the
	// slot order needed to map the grant back to access IDs.
	accessInfos, _ := accessHelper.GetAccesosByRoute(funcPath)
	decision := resolveRouteAccess(args.Method, funcPath, args.User.ID, accessInfos)
	if decision.denyMessage != "" {
		denied := args.MakeErrCode(decision.denyMessage, decision.denyCode)
		return true, &denied
	}

	var accessGrant *core.AccessGrant
	if len(decision.requiredAccess) > 0 {
		grantBytes, err := hex.DecodeString(strings.TrimSpace(request.Header.Get(edgeGrantHeader)))
		if err == nil {
			accessGrant, err = fareward.DecodeAccessGrant(grantBytes)
		}
		if err != nil {
			core.Log("fareward edge: unreadable grant::", funcPath, " err::", err)
			failed := args.MakeErrCode("No se pudo leer la autorización del edge.", 503)
			return true, &failed
		}
	}
	applyGrantedSubAccesos(args, accessGrant, decision.accessInfos, accessInfos)
	return true, nil
}

// isLoopbackPeer is true for a TCP peer on loopback and for a Unix socket peer, whose RemoteAddr
// is empty or "@": both can only be opened from this host.
func isLoopbackPeer(remoteAddr string) bool {
	if remoteAddr == "" || remoteAddr == "@" {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	parsed := net.ParseIP(host)
	return parsed != nil && parsed.IsLoopback()
}
