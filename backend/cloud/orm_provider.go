//go:build !js

package cloud

import (
	"app/core"
	"app/db"
	"errors"
)

// getProviderORM initializes the cloud data mirror selected by BACKEND_PROVIDER. The
// table type parameters are inferred from the record type, exactly as in db.Query, which
// is what lets the mirror read the record's schema without any call site naming it.
func getProviderORM[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]]() (ORM[RecordT], error) {
	if core.Env == nil {
		core.PopulateVariables()
	}

	provider := core.Env.BACKEND_PROVIDER
	if provider != "aws" && provider != "cloudflare" {
		if provider == "none" {
			return nil, errors.New("cloud data mirror is disabled because providers.backend is 'none'")
		}
		return nil, errors.New("providers.backend en config.toml no está definido o es inválido (debe ser 'aws', 'cloudflare' o 'none')")
	}

	tableMeta, err := buildTableMeta[RecordT, TableT, D]()
	if err != nil {
		return nil, err
	}

	if provider == "aws" {
		return NewDynamoORM[RecordT](tableMeta), nil
	}

	if core.Env.CLOUDFLARE_ACCOUNT == "" || core.Env.CLOUDFLARE_TOKEN == "" || core.Env.CLOUDFLARE_DATABASE_ID == "" {
		panic("cloudflare.account, cloudflare.token, and cloudflare.database_id must be set in config.toml when providers.backend is 'cloudflare'")
	}
	return NewSqliteORM[RecordT](tableMeta), nil
}

// IsDataMirrorEnabled reports whether auth and tenant data must also use a cloud database.
func IsDataMirrorEnabled() bool {
	if core.Env == nil {
		core.PopulateVariables()
	}
	return core.Env.BACKEND_PROVIDER != "none"
}
