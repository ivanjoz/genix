//go:build !js

package main

import (
	"app/core"
	"net/http/httptest"
	"strconv"
	"testing"
)

// edgeRoutes picks, from the real route table, one private GET the agent asks nothing about and
// one the agent asks slots for: the two shapes of verdict the headers can carry.
func edgeRoutes(t *testing.T) (open, gated farewardRoute) {
	t.Helper()
	routes, err := resolveFarewardRoutes(*makeAppHandlers())
	if err != nil {
		t.Fatal(err)
	}
	var foundOpen, foundGated bool
	for _, route := range routes {
		if route.public || route.saasOnly || route.deny != 0 || route.method != "GET" {
			continue
		}
		if len(route.slots) == 0 && !foundOpen {
			open, foundOpen = route, true
		}
		if len(route.slots) > 0 && !foundGated {
			gated, foundGated = route, true
		}
	}
	if !foundOpen || !foundGated {
		t.Skip("the route table has no open or no gated private GET")
	}
	return open, gated
}

func edgeArgs(route farewardRoute, remoteAddr string, headers map[string]string) *core.HandlerArgs {
	request := httptest.NewRequest(route.method, "/api/"+route.path, nil)
	request.RemoteAddr = remoteAddr
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	return &core.HandlerArgs{
		Method:     route.method,
		Route:      route.path,
		ReqContext: request,
		RouteID:    route.routeID,
		User:       &core.UsuarioToken{CompanyID: 7, ID: 42},
	}
}

func admittedHeaders(route farewardRoute, grant string) map[string]string {
	headers := map[string]string{
		edgeAdmittedHeader: "1",
		edgeCompanyHeader:  "7",
		edgeUserHeader:     "42",
		edgeRouteHeader:    strconv.Itoa(int(route.routeID)),
	}
	if grant != "" {
		headers[edgeGrantHeader] = grant
	}
	return headers
}

func withTrustedEdge(t *testing.T) {
	t.Helper()
	previous := core.Env
	core.Env = &core.EnvStruct{FAREWARD_TRUST_EDGE: true}
	t.Cleanup(func() { core.Env = previous })
}

func TestEdgeAdmissionTakesTheAgentsVerdict(t *testing.T) {
	withTrustedEdge(t)
	open, gated := edgeRoutes(t)

	args := edgeArgs(open, "127.0.0.1:50000", admittedHeaders(open, ""))
	if handled, response := edgeAdmission(args, open.method+"."+open.path); !handled || response != nil {
		t.Fatalf("open route: handled=%v response=%v", handled, response)
	}

	// Slot 0 granted, no sub-accesses: the access is in the map with a zero mask.
	args = edgeArgs(gated, "[::1]:50000", admittedHeaders(gated, "010000"))
	if handled, response := edgeAdmission(args, gated.method+"."+gated.path); !handled || response != nil {
		t.Fatalf("gated route: handled=%v response=%v", handled, response)
	}
	if len(args.User.SubAccesos) != 1 {
		t.Fatalf("gated route: sub-accesses %v, want the one granted access", args.User.SubAccesos)
	}

	// A Unix socket peer is as local as loopback.
	args = edgeArgs(open, "@", admittedHeaders(open, ""))
	if handled, _ := edgeAdmission(args, open.method+"."+open.path); !handled {
		t.Fatal("a Unix socket peer was not believed")
	}
}

func TestEdgeAdmissionFallsBackToTheGate(t *testing.T) {
	withTrustedEdge(t)
	open, _ := edgeRoutes(t)
	funcPath := open.method + "." + open.path

	otherUser := admittedHeaders(open, "")
	otherUser[edgeUserHeader] = "43"
	otherRoute := admittedHeaders(open, "")
	otherRoute[edgeRouteHeader] = strconv.Itoa(int(open.routeID) + 1)
	cases := map[string]*core.HandlerArgs{
		"not admitted":      edgeArgs(open, "127.0.0.1:50000", nil),
		"remote peer":       edgeArgs(open, "203.0.113.9:50000", admittedHeaders(open, "")),
		"another user":      edgeArgs(open, "127.0.0.1:50000", otherUser),
		"another route":     edgeArgs(open, "127.0.0.1:50000", otherRoute),
		"no request at all": {User: &core.UsuarioToken{CompanyID: 7, ID: 42}, RouteID: open.routeID},
	}
	for name, args := range cases {
		if handled, _ := edgeAdmission(args, funcPath); handled {
			t.Errorf("%s: the headers were believed", name)
		}
	}

	core.Env.FAREWARD_TRUST_EDGE = false
	if handled, _ := edgeAdmission(edgeArgs(open, "127.0.0.1:50000", admittedHeaders(open, "")), funcPath); handled {
		t.Error("believed with trust_edge_headers off")
	}
}

// The edge already charged, so a grant that does not read cannot fall back to a second charge: it
// fails closed.
func TestEdgeAdmissionRefusesAnUnreadableGrant(t *testing.T) {
	withTrustedEdge(t)
	_, gated := edgeRoutes(t)
	for _, grant := range []string{"", "zz", "000000", "01000105"} {
		args := edgeArgs(gated, "127.0.0.1:50000", admittedHeaders(gated, grant))
		handled, response := edgeAdmission(args, gated.method+"."+gated.path)
		if !handled || response == nil || response.StatusCode != 503 {
			t.Errorf("grant %q: handled=%v response=%+v", grant, handled, response)
		}
	}
}
