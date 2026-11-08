package security

import (
	"app/core"
	coreTypes "app/core/types"
	"app/db"
)

func GetUsuariosList(companyID int32, userIDs []int32) ([]coreTypes.User, error) {
	ids := core.MakeSliceInclude(userIDs)

	if len(userIDs) == 0 {
		return []coreTypes.User{}, nil
	}

	usuarios := []coreTypes.User{}
	query := db.Query(&usuarios)
	query.Select().
		CompanyID.Equals(companyID).
		ID.In(ids.Values...)

	if err := query.Exec(); err != nil {
		return nil, err
	}
	return usuarios, nil
}
