package types

import "app/db"

// CompanyCreditAggregateUserID is not stored anywhere: the split into two tables is what removed it
// from the schema. It survives only as the discriminator the credit daemon uses in memory to charge
// a request against its user and its company in one pass.
const CompanyCreditAggregateUserID = int32(-1)

// CreditUsageCompany is one company's absolute daily total, written by the Rust limiter and never
// by this backend. UsedCredits also carries the per-route split, which is why the API breakdown is
// answerable from this table alone and the per-user table stays narrow.
//
// TimeFrame doubles as the delta watermark for the SaaS panel: the row is rewritten in place all
// day under a fixed key, so a `>=` bound refreshes today and appends any day that has since started.
type CreditUsageCompany struct {
	db.TableStruct[CreditUsageCompanyTable, CreditUsageCompany]
	CompanyID   int32
	TimeFrame   int32
	UsedCredits []byte
}

type CreditUsageCompanyTable struct {
	db.TableStruct[CreditUsageCompanyTable, CreditUsageCompany]
	CompanyID   db.Col[CreditUsageCompanyTable, int32]
	TimeFrame   db.Col[CreditUsageCompanyTable, int32]
	UsedCredits db.Col[CreditUsageCompanyTable, []byte]
}

func (e CreditUsageCompanyTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		// Externally aggregated absolute snapshots need no ORM-managed created/updated columns.
		ID:                    50,
		Name:                  "credit_usage_company",
		Partition:             e.CompanyID,
		Keys:                  db.Cols(e.TimeFrame),
		DisableDefaultColumns: true,
		Indexes: []db.Index{
			// ((time_frame), company_id) — one partition per day holds every company's row, so
			// ranking the platform costs days rather than tenants. The view keys on a single real
			// column: a multi-column key would be an ORM-computed zz_ hash column, and the Rust
			// limiter that writes this table never runs that computation.
			{Type: db.TypeView, Keys: db.Cols(e.CompanyID), Partition: e.TimeFrame},
		},
	}
}

// CreditUsageUser is one user's absolute daily total. It carries no route split: thirty days of
// per-route breakdown for every user of a company is a payload nothing renders, and the company
// table already answers "which endpoint cost this tenant the most".
//
// The key order is the limiter's, not the panel's: the daemon reads one user's frame range while
// enforcing quota, which needs UserID pinned ahead of the range.
type CreditUsageUser struct {
	db.TableStruct[CreditUsageUserTable, CreditUsageUser]
	CompanyID   int32
	UserID      int32
	TimeFrame   int32
	UsedCredits []byte
}

type CreditUsageUserTable struct {
	db.TableStruct[CreditUsageUserTable, CreditUsageUser]
	CompanyID   db.Col[CreditUsageUserTable, int32]
	UserID      db.Col[CreditUsageUserTable, int32]
	TimeFrame   db.Col[CreditUsageUserTable, int32]
	UsedCredits db.Col[CreditUsageUserTable, []byte]
}

func (e CreditUsageUserTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:                    51,
		Name:                  "credit_usage_user",
		Partition:             e.CompanyID,
		Keys:                  db.Cols(e.UserID, e.TimeFrame),
		DisableDefaultColumns: true,
		Indexes: []db.Index{
			// ((company_id), time_frame, user_id) — the panel reads every user of one company over
			// a day range, which the base key cannot express with UserID leading.
			{Type: db.TypeView, Keys: db.Cols(e.TimeFrame)},
		},
	}
}
