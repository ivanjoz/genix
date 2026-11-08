package security

import (
	"context"
	"fmt"
	"time"

	config "app/config/types"
	"app/core"
	coreTypes "app/core/types"
	"app/db"
)

// Fareward no longer reads the users table: it answers the route gate from the grants this
// process pushes to it (fareward plan D7). Only what the gate needs travels — the status and the
// two grant blobs — and the push itself drops fareward's cached copy of each user.

// readUserGrants reads the users of a company, or only userIDs when given, with just the columns
// fareward needs.
func readUserGrants(companyID int32, userIDs ...int32) ([]core.UserGrants, error) {
	users := []coreTypes.User{}
	query := db.Query(&users)
	query.Select(query.CompanyID, query.ID, query.Status, query.AccesosComputed, query.AccesosSubComputed).
		CompanyID.Equals(companyID)
	if len(userIDs) > 0 {
		query.ID.In(userIDs...)
	}
	if err := query.Exec(); err != nil {
		return nil, err
	}
	grants := make([]core.UserGrants, 0, len(users))
	for _, user := range users {
		grants = append(grants, core.UserGrants{
			CompanyID: user.CompanyID, UserID: user.ID, Status: user.Status,
			Grants: user.AccesosComputed, SubGrants: user.AccesosSubComputed,
		})
	}
	return grants, nil
}

// pushUserGrants sends one user's grants to fareward, or every user's of the company when userID is
// core.InvalidateAllCompanyUsers. A user that no longer exists is sent as deleted, so fareward
// denies it instead of serving its last grants.
//
// Best effort, as the invalidation it replaces was: a save is never reverted because the push did
// not land. What changes is the backstop: fareward keeps the last grants it was sent, so a lost push
// is repaired by the next save of that user or by SyncAllFarewardGrants, not by a TTL.
func pushUserGrants(req *core.HandlerArgs, companyID, userID int32) {
	// This process's own grant cache first: it is what the request gate reads on a VPS.
	core.DropCachedAccessGrants(companyID, userID)
	if core.Env.FAREWARD_BYPASS {
		return
	}
	requestContext := context.Background()
	if req != nil && req.ReqContext != nil {
		requestContext = req.ReqContext.Context()
	}

	var grants []core.UserGrants
	var err error
	if userID == core.InvalidateAllCompanyUsers {
		grants, err = readUserGrants(companyID)
	} else {
		grants, err = readUserGrants(companyID, userID)
		if err == nil && len(grants) == 0 {
			grants = []core.UserGrants{{CompanyID: companyID, UserID: userID, Deleted: true}}
		}
	}
	if err == nil {
		err = core.PutUserGrants(requestContext, grants)
	}
	if err != nil {
		core.Log("no se pudieron enviar los accesos a fareward::", "company", companyID,
			"user", userID, "err", err)
	}
}

// SyncAllFarewardGrants pushes every user of every company: the first start of a fareward store,
// and the repair for any push that was lost. Re-running it sends the same bytes.
func SyncAllFarewardGrants(ctx context.Context) (companies int, users int, err error) {
	companyRows := []config.Company{}
	companyQuery := db.Query(&companyRows)
	companyQuery.Select(companyQuery.ID)
	if err := companyQuery.Exec(); err != nil {
		return 0, 0, fmt.Errorf("no se pudieron obtener las empresas: %w", err)
	}
	for _, company := range companyRows {
		grants, err := readUserGrants(company.ID)
		if err != nil {
			return companies, users, fmt.Errorf("company %d: %w", company.ID, err)
		}
		if err := core.PutUserGrants(ctx, grants); err != nil {
			return companies, users, fmt.Errorf("company %d: %w", company.ID, err)
		}
		companies++
		users += len(grants)
	}
	return companies, users, nil
}

// SyncFarewardGrants is SyncAllFarewardGrants as an exec function (fn-sync-fareward-grants).
func SyncFarewardGrants(args *core.ExecArgs) core.FuncResponse {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	companies, users, err := SyncAllFarewardGrants(ctx)
	if err != nil {
		return args.MakeErr("sincronización de accesos con fareward incompleta:", err)
	}
	message := fmt.Sprintf("Accesos enviados a fareward: %d empresa(s), %d user(s).", companies, users)
	core.Log(message)
	return core.FuncResponse{Message: message}
}
