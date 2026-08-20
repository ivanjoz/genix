package main

import (
	"app/core"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The charge frame carries a fixed number of packed-grant slots, so a backend route mapped to more
// accesses than fit cannot be authorized. The gate refuses such a route with a 500 at request time,
// which is the safe answer but a terrible way to find out.
//
// This is where it is meant to be found instead: adding a fifth access to one route's backend_apis
// fails the build rather than one endpoint in production. If this test ever fails legitimately,
// widening MaxRequiredAccess means widening CHARGE_PAYLOAD_SIZE on both sides — the two byte-layout
// tests will say so.
func TestEveryRouteFitsTheRequiredAccessSlots(t *testing.T) {
	parsed := core.AccessListYaml{}
	if err := yaml.Unmarshal(accessListYamlContent, &parsed); err != nil {
		t.Fatalf("access_list.yml did not parse: %v", err)
	}

	accessesByRoute := map[string][]string{}
	for _, entry := range parsed.AccessList {
		for _, route := range strings.Split(entry.BackendAPIs, ",") {
			route = strings.TrimSpace(route)
			if route == "" {
				continue
			}
			accessesByRoute[route] = append(accessesByRoute[route], entry.Name)
		}
	}
	if len(accessesByRoute) == 0 {
		t.Fatal("no backend routes were mapped; the embedded catalogue is not being read")
	}

	for route, accessNames := range accessesByRoute {
		if len(accessNames) > core.MaxRequiredAccess {
			t.Errorf("route %q maps to %d accesses (%s); the charge frame holds %d",
				route, len(accessNames), strings.Join(accessNames, ", "), core.MaxRequiredAccess)
		}
	}
}

// The packed form is the one thing both processes must agree on byte for byte. Its Rust twin is
// `packed` in server_utils/src/limiter/access.rs; the level occupies the low two bits and nothing
// else, which is what makes `required | 0b11` the bucket ceiling over there.
func TestAccesoNivelPacking(t *testing.T) {
	checks := []struct {
		accesoID int32
		nivel    uint8
		want     uint16
	}{
		{1, 1, 0b100}, {1, 4, 0b111}, {3, 2, 13}, {34, 4, 139},
		// An out-of-range level clamps down to 1 rather than up: a malformed level must never
		// widen what is being asked for.
		{7, 0, 28}, {7, 5, 28},
	}
	for _, check := range checks {
		if got := core.MakeAccesoNivelPacked(check.accesoID, check.nivel); got != check.want {
			t.Errorf("MakeAccesoNivelPacked(%d, %d) = %d; want %d",
				check.accesoID, check.nivel, got, check.want)
		}
	}
}

// The gate decides authorization for the whole backend, and returning an empty required-access list
// means "do not ask the daemon". Every path that returns one has to be deliberate, so all of them
// are pinned here — including the deny-by-default that a mapped-route lookup miss must produce.
func TestResolveRouteAccess(t *testing.T) {
	twoAccesses := []core.AccessInfo{{ID: 3, Name: "Comercial"}, {ID: 9, Name: "Logística"}}

	t.Run("an unmapped GET is free to any session", func(t *testing.T) {
		decision := resolveRouteAccess("GET", "GET.something-new", 42, nil)
		if len(decision.requiredAccess) != 0 || decision.denyMessage != "" {
			t.Fatalf("an unmapped GET was gated: %+v", decision)
		}
	})

	t.Run("an unmapped POST is refused without asking the daemon", func(t *testing.T) {
		decision := resolveRouteAccess("POST", "POST.something-new", 42, nil)
		if decision.denyMessage == "" || decision.denyCode != 403 {
			t.Fatalf("an unmapped POST was not refused: %+v", decision)
		}
		if len(decision.requiredAccess) != 0 {
			t.Fatal("a refusal must not also send a frame")
		}
	})

	t.Run("a self-service route needs a session and no access", func(t *testing.T) {
		for route := range selfServiceRoutes {
			decision := resolveRouteAccess("POST", route, 42, nil)
			if len(decision.requiredAccess) != 0 || decision.denyMessage != "" {
				t.Fatalf("%s was gated: %+v", route, decision)
			}
		}
	})

	// If this ever starts failing, the daemon is being asked to authorize a user whose grants are
	// synthesized in the login response and never persisted. It would deny, and user 1 would be
	// locked out of the whole application.
	t.Run("user 1 is never sent to the daemon", func(t *testing.T) {
		decision := resolveRouteAccess("POST", "POST.users", 1, twoAccesses)
		if len(decision.requiredAccess) != 0 || decision.denyMessage != "" {
			t.Fatalf("user 1 was gated: %+v", decision)
		}
	})

	t.Run("levels follow the method", func(t *testing.T) {
		// GET wants nivel 1, POST and PUT want 2.
		readDecision := resolveRouteAccess("GET", "GET.mapped", 42, twoAccesses)
		if want := []uint16{core.MakeAccesoNivelPacked(3, 1), core.MakeAccesoNivelPacked(9, 1)}; !slices.Equal(readDecision.requiredAccess, want) {
			t.Fatalf("GET required %v; want %v", readDecision.requiredAccess, want)
		}
		for _, method := range []string{"POST", "PUT"} {
			writeDecision := resolveRouteAccess(method, "POST.mapped", 42, twoAccesses)
			want := []uint16{core.MakeAccesoNivelPacked(3, 2), core.MakeAccesoNivelPacked(9, 2)}
			if !slices.Equal(writeDecision.requiredAccess, want) {
				t.Fatalf("%s required %v; want %v", method, writeDecision.requiredAccess, want)
			}
			if len(writeDecision.accessNames) != 2 || writeDecision.accessNames[0] != "Comercial" {
				t.Fatalf("%s lost the access names: %v", method, writeDecision.accessNames)
			}
		}
	})

	t.Run("more accesses than the frame holds is refused", func(t *testing.T) {
		tooMany := make([]core.AccessInfo, core.MaxRequiredAccess+1)
		for index := range tooMany {
			tooMany[index] = core.AccessInfo{ID: int32(index + 1), Name: "x"}
		}
		decision := resolveRouteAccess("POST", "POST.overmapped", 42, tooMany)
		if decision.denyMessage == "" || decision.denyCode != 500 {
			t.Fatalf("an over-mapped route was accepted: %+v", decision)
		}
	})

	// The credit-exempt routes skip the charge, never the frame. Three of them are access-mapped and
	// two are SaaS-only, so a gate that waved them through would open them to any session.
	t.Run("credit-exempt routes are still gated", func(t *testing.T) {
		for route := range creditControlRoutes {
			accessInfos, _ := accessHelper.GetAccesosByRoute(route)
			if len(accessInfos) == 0 {
				continue
			}
			method := "GET"
			if strings.HasPrefix(route, "POST.") {
				method = "POST"
			}
			decision := resolveRouteAccess(method, route, 42, accessInfos)
			if len(decision.requiredAccess) == 0 {
				t.Fatalf("%s is access-mapped but the gate asks for nothing", route)
			}
		}
	})
}

// PUT has no credit tariff — APICPUCredits knows only GET and POST — and it was never charged before
// this change either. Charging it would have made every PUT a 503, since the formula returns an error
// for an unknown method and the router fails closed on one.
func TestChargedMethodFor(t *testing.T) {
	checks := []struct {
		method   string
		funcPath string
		want     string
	}{
		{"GET", "GET.productos", "GET"},
		{"POST", "POST.productos", "POST"},
		// Mapped, authorized, and deliberately free: see chargedMethodFor.
		{"PUT", "PUT.purchase-orders", ""},
		{"DELETE", "DELETE.whatever", ""},
		// The credit panel reading itself.
		{"GET", "GET.credit-usage", ""},
		{"POST", "POST.company-credit-budget", ""},
	}
	for _, check := range checks {
		if got := chargedMethodFor(check.method, check.funcPath); got != check.want {
			t.Errorf("chargedMethodFor(%q, %q) = %q; want %q",
				check.method, check.funcPath, got, check.want)
		}
	}

	// Every credit-exempt route must be exempt whatever method reaches it.
	for route := range creditControlRoutes {
		for _, method := range []string{"GET", "POST"} {
			if got := chargedMethodFor(method, route); got != "" {
				t.Errorf("%s %s was charged; the credit panel must stay reachable", method, route)
			}
		}
	}
}
