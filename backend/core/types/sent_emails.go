package types

import "app/db"

// SentEmail is one email core.SendEmail handled, written after the SMTP attempt. With
// [smtp].store_only nothing goes out over SMTP and this row IS the delivery: the System →
// Development → Email Inbox page (frontend/routes/system/email-inbox) is the virtual mailbox
// developers and agents read to test flows that send email without mailing anybody.
type SentEmail struct {
	db.TableStruct[SentEmailTable, SentEmail]
	// CompanyID is the tenant the email belongs to; platform emails (sign-up, contact form) go to
	// the SaaS company 1.
	CompanyID int32    `cb:"1"`
	ID        int32    `cb:"2"`
	To        []string `cb:"3"`
	Cc        []string `cb:"4"`
	Bcc       []string `cb:"5"`
	Subject   string   `cb:"6"`
	// HTMLBody is excluded from the inbox list: the preview fetches it by ID.
	HTMLBody string `cb:"7"`
	// AttachmentNames keeps only the names: the content would bloat a row nobody downloads.
	AttachmentNames []string `cb:"8"`
	Sender          string   `cb:"9"`
	// SendError is the SMTP failure of a SentEmailStatusFailed row.
	SendError string `cb:"10"`
	// See the SentEmailStatus* constants.
	Status  int8  `json:"ss" cb:"ss,11"`
	Updated int32 `json:"upd" cb:"upd,12"`
	// ReplyTo is where answers go (a tenant's seller): the Sender address is the platform's.
	ReplyTo string `cb:"13"`
}

const (
	// SentEmailStatusStored: [smtp].store_only, the email was only written to this table.
	SentEmailStatusStored = int8(1)
	// SentEmailStatusSent: the SMTP server accepted the email.
	SentEmailStatusSent = int8(2)
	// SentEmailStatusFailed: the SMTP send failed, SendError says why.
	SentEmailStatusFailed = int8(3)
)

type SentEmailTable struct {
	db.TableStruct[SentEmailTable, SentEmail]
	CompanyID       db.Col[*SentEmailTable, int32]
	ID              db.Col[*SentEmailTable, int32]
	To              db.ColSlice[*SentEmailTable, string]
	Cc              db.ColSlice[*SentEmailTable, string]
	Bcc             db.ColSlice[*SentEmailTable, string]
	Subject         db.Col[*SentEmailTable, string]
	HTMLBody        db.Col[*SentEmailTable, string]
	AttachmentNames db.ColSlice[*SentEmailTable, string]
	Sender          db.Col[*SentEmailTable, string]
	SendError       db.Col[*SentEmailTable, string]
	Status          db.Col[*SentEmailTable, int8]
	Updated         db.Col[*SentEmailTable, int32]
	ReplyTo         db.Col[*SentEmailTable, string]
}

// The autoincrement ID is the send order within the company: the inbox reads it descending, newest
// first, and pages back with ID < the oldest one shown. No index: that range is the primary key.
func (e SentEmailTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        88,
		Name:      "sent_emails",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
	}
}
