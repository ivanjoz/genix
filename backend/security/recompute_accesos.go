package security

import (
	"bytes"
	"fmt"

	config "app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
)

// RecomputeUserAccesos rebuilds users.accesos_computed and users.accesos_sub_computed for every
// user of every company, from the profiles and direct grants they hold.
//
// It exists because the grant word changed container and endianness in the same release: the
// stored bytes were little-endian []uint16 and are now big-endian []byte. The word survived, its
// byte order did not, so under the new reader every stored blob decodes to accesses nobody granted
// and every user is denied. Pre-alpha means no compatibility shim — the data is rebuilt instead.
//
// Deliberately not idempotency-guarded: it recomputes from the profiles, which are the source of
// truth, so re-running it writes the same bytes. That also makes it the repair tool for any user
// whose blobs drifted, not only a one-off migration.
func RecomputeUserAccesos(args *core.ExecArgs) core.FuncResponse {
	companies := []config.Company{}
	companyQuery := db.Query(&companies)
	companyQuery.Status.GreaterEqual(1).AllowFilter()
	if err := companyQuery.Exec(); err != nil {
		return args.MakeErr("no se pudieron obtener las empresas:", err)
	}
	if len(companies) == 0 {
		return args.MakeErr("no hay empresas activas; nada que recomputar.")
	}

	usersRead, usersWritten := 0, 0
	for _, company := range companies {
		companyUsersRead, companyUsersWritten, err := recomputeCompanyUserAccesos(company.ID)
		if err != nil {
			return args.MakeErr(fmt.Sprintf("company %d:", company.ID), err)
		}
		usersRead += companyUsersRead
		usersWritten += companyUsersWritten

		// One wildcard per company rather than one frame per user: on the migration run every
		// user changed, so naming them individually would be the same invalidation in N frames.
		if companyUsersWritten > 0 {
			invalidateCachedUserAccess(nil, company.ID, core.InvalidateAllCompanyUsers)
		}
		args.AddMessage(fmt.Sprintf("company %d: %d user(s) leídos, %d reescritos",
			company.ID, companyUsersRead, companyUsersWritten))
	}

	message := fmt.Sprintf("Recompute de accesos completado: %d empresa(s), %d user(s) leídos, %d reescritos.",
		len(companies), usersRead, usersWritten)
	core.Log(message)
	return core.FuncResponse{Message: message}
}

// recomputeCompanyUserAccesos rebuilds one company's users. Every user is read, including inactive
// ones: their blobs are what a reactivation would authorize with, and nothing else recomputes them.
func recomputeCompanyUserAccesos(companyID int32) (int, int, error) {
	users := []coreTypes.User{}
	userQuery := db.Query(&users)
	userQuery.CompanyID.Equals(companyID)
	if err := userQuery.Exec(); err != nil {
		return 0, 0, core.Err("no se pudieron obtener los users:", err)
	}
	if len(users) == 0 {
		return 0, 0, nil
	}

	allProfileIDs := make([]int32, 0, len(users)*2)
	for _, user := range users {
		allProfileIDs = append(allProfileIDs, user.ProfileIDs...)
	}
	perfilesByID, err := getPerfilesMapByIDs(companyID, allProfileIDs)
	if err != nil {
		return 0, 0, core.Err("no se pudieron obtener los perfiles:", err)
	}

	// User 1's grants are synthesized at login and never stored, so its row legitimately keeps
	// empty blobs and comes out of this loop unchanged. buildBootstrapAdminAccesos and
	// resolveRouteAccess's user-1 bypass are what make that correct; see login.go.
	usersWithChangedAccesos := make([]coreTypes.User, 0, len(users))
	for userIndex := range users {
		user := &users[userIndex]

		grantsByAccesoID, buildErr := buildAccesosComputedFromPerfiles(perfilesByID, user.ProfileIDs)
		if buildErr != nil {
			return 0, 0, core.Err("user", user.ID, "::", buildErr)
		}
		// The same second source PostUsuarios merges. Leaving it out would silently strip every
		// access granted to a user directly rather than through a profile.
		for _, accesoNivelID := range user.AccessLevelIDs {
			addAccesoNivelToGrants(grantsByAccesoID, accesoNivelID)
		}

		accesosBlob, accesosSubBlob, encodeErr := encodeMergedAccesoGrants(grantsByAccesoID)
		if encodeErr != nil {
			return 0, 0, core.Err("user", user.ID, "::", encodeErr)
		}
		if bytes.Equal(user.AccesosComputed, accesosBlob) &&
			bytes.Equal(user.AccesosSubComputed, accesosSubBlob) {
			continue
		}

		user.AccesosComputed = accesosBlob
		user.AccesosSubComputed = accesosSubBlob
		core.Log("RecomputeUserAccesos:: user", user.ID, "accesos bytes", len(accesosBlob),
			"sub bytes", len(accesosSubBlob))
		usersWithChangedAccesos = append(usersWithChangedAccesos, *user)
	}

	if len(usersWithChangedAccesos) == 0 {
		return len(users), 0, nil
	}

	// Only the grant columns are named: everything else on the row is whatever the read
	// returned, and rewriting it would let a concurrent edit of an unrelated field be lost.
	// Status is the exception and is not optional -- the table declares a composite view on
	// (status, updated) and the ORM assigns updated on every write, so leaving status out fails
	// with "requires the columns status, updated be updated together". Its value is the one the
	// read returned, so naming it changes nothing.
	usuarioQuery := db.Query(&usersWithChangedAccesos)
	if err := db.Update(&usersWithChangedAccesos,
		usuarioQuery.AccesosComputed, usuarioQuery.AccesosSubComputed,
		usuarioQuery.Status); err != nil {
		return 0, 0, core.Err("no se pudieron actualizar los users:", err)
	}

	return len(users), len(usersWithChangedAccesos), nil
}
