package types

import "app/db"

// SignUpRequest is one public sign-up attempt: an email that asked to register a company, the
// verification code sent to it, and how far the registration got. It has no tenant — it exists
// precisely to create one — so the partition is the ISO week the request was created in. That
// keeps a table nobody ever reads in bulk from growing a single unbounded partition, and makes
// purging old requests a matter of dropping whole weeks.
type SignUpRequest struct {
	db.TableStruct[SignUpRequestTable, SignUpRequest]
	// WeekCode is core.MakeSemanaFromFechaUnix(...).Code: year*100 + isoWeek - 200000.
	// 2026-W32 => 2632.
	WeekCode int32 `json:",omitempty"`
	// ID is a global sequence*10^6 + 6 random digits. The random tail stops anyone from walking the
	// table by guessing consecutive IDs. It carries no partition: a request lives 2 hours, so a
	// lookup by bare ID just searches the current and previous week.
	ID    int64  `json:",omitempty"`
	Email string `json:",omitempty"`
	// Code is the 8-digit verification code delivered in the email body and in the link.
	Code string `json:",omitempty"`
	// Attempts counts failed code checks. These endpoints are public and therefore skip the
	// platform rate limiter, so this counter is what stops a brute-force sweep of the code.
	Attempts int8 `json:",omitempty"`
	// CompanyID and UserID are filled once step 2 creates them, so a registration interrupted
	// before "Initial Data" can be resumed instead of started over.
	CompanyID int32 `json:",omitempty"`
	UserID    int32 `json:",omitempty"`
	// Created anchors the 2-hour expiry; LastSent anchors the short resend cooldown. They are
	// separate columns because Updated also moves on failed code attempts, which must not extend
	// the window before the user is allowed to ask for the email again.
	Created  int32 `json:",omitempty"`
	LastSent int32 `json:",omitempty"`
	Updated  int32 `json:"upd,omitempty"`
	// 1 = email sent · 2 = email verified · 3 = company created · 0 = cancelled or attempts spent
	Status int8 `json:"ss,omitempty"`
	// IP is core.ClientIPKey of the caller: the IPv4 value, or the IPv6 /63 prefix for addresses
	// where a single customer owns a whole block. It is the subject of the "N distinct emails per
	// window" limit, which is the only brake that survives an attacker owning many mailboxes.
	IP int64 `json:",omitempty"`
}

type SignUpRequestTable struct {
	db.TableStruct[SignUpRequestTable, SignUpRequest]
	WeekCode  db.Col[SignUpRequestTable, int32]
	ID        db.Col[SignUpRequestTable, int64]
	Email     db.Col[SignUpRequestTable, string]
	Code      db.Col[SignUpRequestTable, string]
	Attempts  db.Col[SignUpRequestTable, int8]
	CompanyID db.Col[SignUpRequestTable, int32]
	UserID    db.Col[SignUpRequestTable, int32]
	Created   db.Col[SignUpRequestTable, int32]
	LastSent  db.Col[SignUpRequestTable, int32]
	Updated   db.Col[SignUpRequestTable, int32]
	Status    db.Col[SignUpRequestTable, int8]
	IP        db.Col[SignUpRequestTable, int64]
}

func (e SignUpRequestTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        43,
		Name:      "sign_up_requests",
		Partition: e.WeekCode,
		Keys:      db.Cols(e.ID),
		// Two lookups besides "by ID", both run against the current week and the previous one:
		// the latest request from an email, and every request from an IP so the per-IP window
		// can be counted. Local rather than global: both are scoped to a week partition.
		Indexes: []db.Index{
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.Email)},
			{Type: db.TypeLocalIndex, Keys: db.Cols(e.IP)},
		},
	}
}
