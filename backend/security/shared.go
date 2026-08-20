package security

import (
	"app/core"
	coretypes "app/core/types"
	"app/db"
	"context"
)

// invalidateCachedUserAccess le dice a server_utils que vuelva a leer los accesos de un user, o de
// toda una company cuando userID es core.InvalidateAllCompanyUsers.
//
// El daemon cachea users.accesos_computed para responder el gate de rutas sin ir a ScyllaDB, así que
// cualquier escritura de esa columna tiene que avisar. Best-effort a propósito: el frame no se
// contesta y el TTL del daemon es el respaldo, de modo que un guardado correcto nunca se revierte
// porque una pista de caché no llegó.
func invalidateCachedUserAccess(req *core.HandlerArgs, companyID, userID int32) {
	requestContext := context.Background()
	if req != nil && req.ReqContext != nil {
		requestContext = req.ReqContext.Context()
	}
	if err := core.InvalidateUserAccess(requestContext, companyID, userID); err != nil {
		core.Log("no se pudo invalidar el caché de accesos::", "company", companyID,
			"user", userID, "err", err)
	}
}

func GetUsuariosList(companyID int32, userIDs []int32) ([]coretypes.User, error) {
	ids := core.MakeSliceInclude(userIDs)

	if len(userIDs) == 0 {
		return []coretypes.User{}, nil
	}

	usuarios := []coretypes.User{}
	query := db.Query(&usuarios)
	query.Select().
		CompanyID.Equals(companyID).
		ID.In(ids.Values...)

	if err := query.Exec(); err != nil {
		return nil, err
	}
	return usuarios, nil
}
