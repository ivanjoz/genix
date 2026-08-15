package types

import "app/db"

// ContactMessage is one message left on the public contact form of /welcome. Like SignUpRequest it
// belongs to no tenant — whoever writes it does not have one — so the partition is the ISO week it
// arrived in. That keeps a table nobody reads in bulk from growing one unbounded partition, and
// makes purging old messages a matter of dropping whole weeks.
type ContactMessage struct {
	db.TableStruct[ContactMessageTable, ContactMessage]
	// WeekCode is core.MakeSemanaFromFechaUnix(...).Code: year*100 + isoWeek - 200000.
	// 2026-W32 => 2632.
	WeekCode int32 `json:",omitempty"`
	// IP is core.ClientIPKey of the sender: the IPv4 value, or the IPv6 /63 prefix for addresses
	// where a single customer owns a whole block. It leads the clustering key rather than sitting
	// in an index because the one query this table serves on the hot path is "how many messages
	// has this IP sent inside the window", and as a key that is a partition slice instead of an
	// index lookup that would load every message the address ever sent this week.
	IP int64 `json:",omitempty"`
	// Created is SUnixTime (1 unit = 2 seconds) and is what the rate-limit window ranges over.
	Created int32 `json:",omitempty"`
	// ID is a global sequence, last in the key so two messages the same IP sends inside one
	// 2-second tick stay distinct rows. It is never returned to the caller nor carried in a link,
	// so unlike a sign-up request it needs no random tail to stop the table being walked.
	ID      int64  `json:",omitempty"`
	Name    string `json:",omitempty"`
	Email   string `json:",omitempty"`
	Company string `json:",omitempty"`
	Message string `json:",omitempty"`
	Updated int32  `json:"upd,omitempty"`
	// 1 = stored and notified · 2 = stored but the notification email failed. The row is written
	// before the mail is sent, so a broken SMTP costs the notification and never the message.
	Status int8 `json:"ss,omitempty"`
}

type ContactMessageTable struct {
	db.TableStruct[ContactMessageTable, ContactMessage]
	WeekCode db.Col[ContactMessageTable, int32]
	IP       db.Col[ContactMessageTable, int64]
	Created  db.Col[ContactMessageTable, int32]
	ID       db.Col[ContactMessageTable, int64]
	Name     db.Col[ContactMessageTable, string]
	Email    db.Col[ContactMessageTable, string]
	Company  db.Col[ContactMessageTable, string]
	Message  db.Col[ContactMessageTable, string]
	Updated  db.Col[ContactMessageTable, int32]
	Status   db.Col[ContactMessageTable, int8]
}

func (e ContactMessageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        46,
		Name:      "contact_messages",
		Partition: e.WeekCode,
		// (WeekCode = ?, IP = ?, Created BETWEEN ? AND ?) is a key prefix with a range on the next
		// clustering column, which is exactly the capability the rate-limit count needs. No index
		// is declared because no other lookup exists: nothing reads this table by hand yet.
		Keys: db.Cols(e.IP, e.Created, e.ID),
	}
}
