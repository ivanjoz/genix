//go:build !js

package main

import (
	"app/core"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	fareward "github.com/ivanjoz/fareward/go"
)

// The route policy table the fareward SPOE agent enforces at the HAProxy edge (fareward plan D1),
// written from this process's own gate so the edge and the backend cannot disagree: every field
// below comes from the same functions enforceAccessAndCredits calls, not from a second copy of the
// rules. Written at startup, so a deploy of the backend updates the edge too; the daemon reloads
// the file when it changes.

// farewardRoute is one [[route]] entry, as the gate resolves it for an ordinary user.
type farewardRoute struct {
	method, path string
	routeID      int16
	slots        []uint16
	baseCredits  uint16
	stepCredits  uint16
	public       bool
	saasOnly     bool
	deny         int32
}

// resolveFarewardRoutes resolves every served route the way the gate does. Sorted, so the same
// catalogue always produces the same bytes and the same version.
func resolveFarewardRoutes(handlers core.AppRouterType) ([]farewardRoute, error) {
	funcPaths := make([]string, 0, len(handlers))
	for funcPath := range handlers {
		funcPaths = append(funcPaths, funcPath)
	}
	sort.Strings(funcPaths)

	routes := make([]farewardRoute, 0, len(funcPaths))
	for _, funcPath := range funcPaths {
		method, path, found := strings.Cut(funcPath, ".")
		if !found || method == "" || path == "" {
			return nil, fmt.Errorf("handler %q is not METHOD.route", funcPath)
		}
		route := farewardRoute{method: method, path: path, routeID: core.APIRouteID(funcPath)}

		// The same test mainHandler applies: a "p-" route needs no session at all.
		if len(path) > 2 && path[0:2] == "p-" {
			route.public = true
			routes = append(routes, route)
			continue
		}

		route.saasOnly = saasOnlyRoutes[funcPath]
		accessInfos, _ := accessHelper.GetAccesosByRoute(funcPath)
		// An ordinary user: user 1's exemption depends on the identity, so the agent applies it.
		// A denied route still gets its tariff below: user 1 passes the denial and is charged.
		decision := resolveRouteAccess(method, funcPath, 0, accessInfos)
		if decision.denyMessage != "" {
			route.deny = decision.denyCode
		} else {
			route.slots = decision.requiredAccess
		}

		if chargedMethod := chargedMethodFor(method, funcPath); chargedMethod != "" {
			baseCredits, err := core.APICPUBaseCredits(chargedMethod)
			if err != nil {
				return nil, fmt.Errorf("tariff for %s: %w", funcPath, err)
			}
			// APICPUCredits charges one credit per started block past the base.
			route.baseCredits, route.stepCredits = baseCredits, 1
		}
		routes = append(routes, route)
	}
	return routes, nil
}

// renderFarewardRouteFile renders the TOML the daemon parses (fareward src/routes/table.rs). The
// version is the hash of everything else, so it changes exactly when the policy does.
func renderFarewardRouteFile(routes []farewardRoute) []byte {
	var body strings.Builder
	quote := strconv.Quote

	body.WriteString("[policy]\n")
	prefixes := make([]string, len(apiNames))
	for index, name := range apiNames {
		prefixes[index] = quote(name)
	}
	fmt.Fprintf(&body, "api_prefixes = [%s]\n", strings.Join(prefixes, ", "))
	fmt.Fprintf(&body, "saas_company_id = %d\n", saasCompanyID)
	// resolveRouteAccess's own exemption: login synthesizes this user's grants and never stores them.
	body.WriteString("access_exempt_user_id = 1\n")
	// TolerateCreditRefusal's company: charged, but never locked out by its own budget.
	fmt.Fprintf(&body, "operator_company_id = %d\n", fareward.OperatorCompanyID)

	// The size half of the tariff, global per method, exactly as APICPUCredits applies it: the
	// blocks come from [rate_limit] through core.ConfigureTariff.
	for _, tariff := range core.Tariff() {
		fmt.Fprintf(&body, "\n[tariff.%s]\nbase_kib = %d\nstep_kib = %d\n",
			tariff.Method, tariff.BaseKiB, tariff.StepKiB)
	}

	for _, route := range routes {
		fmt.Fprintf(&body, "\n[[route]]\nmethod = %s\npath = %s\nroute_id = %d\n",
			quote(route.method), quote(route.path), route.routeID)
		if len(route.slots) > 0 {
			slots := make([]string, len(route.slots))
			for index, slot := range route.slots {
				slots[index] = strconv.Itoa(int(slot))
			}
			fmt.Fprintf(&body, "slots = [%s]\n", strings.Join(slots, ", "))
		}
		if route.baseCredits > 0 {
			fmt.Fprintf(&body, "base_credits = %d\nstep_credits = %d\n", route.baseCredits, route.stepCredits)
		}
		if route.public {
			body.WriteString("public = true\n")
		}
		if route.saasOnly {
			body.WriteString("saas_only = true\n")
		}
		if route.deny != 0 {
			fmt.Fprintf(&body, "deny = %d\n", route.deny)
		}
	}

	digest := sha256.Sum256([]byte(body.String()))
	return []byte("version = " + quote(hex.EncodeToString(digest[:16])) + "\n\n" + body.String())
}

// writeFarewardRouteFile writes the table where [fareward].routes_file points, as a temporary file
// renamed over the old one: the daemon polls the file and must never read half of it.
func writeFarewardRouteFile(path string, handlers core.AppRouterType) error {
	routes, err := resolveFarewardRoutes(handlers)
	if err != nil {
		return err
	}
	content := renderFarewardRouteFile(routes)

	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	// Readable by the daemon's user; the file holds policy, not secrets.
	if err := os.Chmod(temporary.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
