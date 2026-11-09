package types

import "app/db"

// An activity is something a rep did about an opportunity (a call, a meeting, a note), written on
// its "Activity" timeline next to the stage history. Its type is one ID space shared by two sources:
//
//   - 1..ActivityTypeSystemMaxID: the common types, fixed here (SystemActivityTypes) for every company.
//   - above ActivityTypeSystemMaxID: the company's own types in crm_activity_types, whose
//     autoincrement starts after the reserved range.
const ActivityTypeSystemMaxID int16 = 40

const (
	ActivityTypeNote     int16 = 1
	ActivityTypeCall     int16 = 2
	ActivityTypeMeeting  int16 = 3
	ActivityTypeEmail    int16 = 4
	ActivityTypeWhatsApp int16 = 5
	ActivityTypeVisit    int16 = 6
	ActivityTypeProposal int16 = 7
	ActivityTypeDemo     int16 = 8
	ActivityTypeFollowUp int16 = 9
)

// ActivityTypeOption is one fixed activity type. Name is bilingual ("EN|ES") because the frontend
// copy feeds the timeline buttons directly.
type ActivityTypeOption struct {
	ID   int16  `cb:"1"`
	Name string `cb:"2"`
	Icon string `cb:"3"`
}

// SystemActivityTypes is what the save path accepts below ActivityTypeSystemMaxID and — via the
// //CATALOG: marker in frontend/services/crm/crm-activity.svelte.ts — the buttons the timeline offers.
// The IDs are persisted; never renumber them.
var SystemActivityTypes = []ActivityTypeOption{
	{ID: ActivityTypeNote, Name: "Note|Nota", Icon: "icon-[fa--sticky-note]"},
	{ID: ActivityTypeCall, Name: "Call|Llamada", Icon: "icon-[fa--phone]"},
	{ID: ActivityTypeMeeting, Name: "Meeting|Reunión", Icon: "icon-[fa--users]"},
	{ID: ActivityTypeEmail, Name: "Email|Email", Icon: "icon-[fa--envelope]"},
	{ID: ActivityTypeWhatsApp, Name: "WhatsApp|WhatsApp", Icon: "icon-[fa--whatsapp]"},
	{ID: ActivityTypeVisit, Name: "Visit|Visita", Icon: "icon-[fa--map-marker]"},
	{ID: ActivityTypeProposal, Name: "Proposal sent|Propuesta enviada", Icon: "icon-[fa--file-text]"},
	{ID: ActivityTypeDemo, Name: "Demo|Demostración", Icon: "icon-[fa--desktop]"},
	{ID: ActivityTypeFollowUp, Name: "Follow-up|Seguimiento", Icon: "icon-[fa--refresh]"},
}

// ActivityIconOption is one icon a company type may use.
type ActivityIconOption struct {
	Icon string `cb:"1"`
}

// ActivityIconOptions is the closed set of icons a company activity type picks from: the CSS only
// ships icons named literally in the source, so a free-typed one would not render. Copied to the
// frontend through the //CATALOG: marker in crm-activity.svelte.ts.
var ActivityIconOptions = []ActivityIconOption{
	{Icon: "icon-[fa--sticky-note]"},
	{Icon: "icon-[fa--phone]"},
	{Icon: "icon-[fa--users]"},
	{Icon: "icon-[fa--envelope]"},
	{Icon: "icon-[fa--whatsapp]"},
	{Icon: "icon-[fa--map-marker]"},
	{Icon: "icon-[fa--file-text]"},
	{Icon: "icon-[fa--desktop]"},
	{Icon: "icon-[fa--refresh]"},
	{Icon: "icon-[fa--calendar]"},
	{Icon: "icon-[fa--handshake-o]"},
	{Icon: "icon-[fa--truck]"},
	{Icon: "icon-[fa--money]"},
	{Icon: "icon-[fa--video-camera]"},
	{Icon: "icon-[fa--comments]"},
	{Icon: "icon-[fa--flag]"},
}

// CrmActivityType is a company's own activity type, edited in /crm/dictionaries.
type CrmActivityType struct {
	db.TableStruct[CrmActivityTypeTable, CrmActivityType]
	CompanyID int32  `cb:"1"`
	ID        int16  `cb:"2"`
	Name      string `cb:"3"`
	Icon      string `cb:"4"`
	Status    int8   `json:"ss" cb:"ss,5"`
	Updated   int32  `json:"upd" cb:"upd,6"`
	Version   int16  `json:"ver" cb:"ver,7"`
	UpdatedBy int32  `cb:"8"`
	// Metadata is set only on the metadata record a Delta() read appends.
	Metadata *string `json:"mdt" cb:"mdt,9"`
}

type CrmActivityTypeTable struct {
	db.TableStruct[CrmActivityTypeTable, CrmActivityType]
	CompanyID db.Col[*CrmActivityTypeTable, int32]
	ID        db.Col[*CrmActivityTypeTable, int16]
	Name      db.Col[*CrmActivityTypeTable, string]
	Icon      db.Col[*CrmActivityTypeTable, string]
	Status    db.Col[*CrmActivityTypeTable, int8]
	Updated   db.Col[*CrmActivityTypeTable, int32]
	Version   db.Col[*CrmActivityTypeTable, int16]
	UpdatedBy db.Col[*CrmActivityTypeTable, int32]
}

func (e CrmActivityTypeTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          86,
		Name:        "crm_activity_types",
		Partition:   e.CompanyID,
		Keys:        db.Cols(e.ID.AutoincrementFrom(int64(ActivityTypeSystemMaxID) + 1)),
		FixedValues: []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.Status)}},
	}
}

// CrmAttachment is a file of an opportunity in the CDN bucket, public behind an unguessable key
// (ActivityAttachmentKey).
type CrmAttachment struct {
	Key         string `cb:"1"`
	Name        string `cb:"2"`
	ContentType string `cb:"3"`
	Size        int32  `cb:"4"`
	UploadedBy  int32  `cb:"5"`
	// UploadedAt is a SUnixTime.
	UploadedAt int32 `cb:"6"`
}

// CrmActivity is one entry of an opportunity's timeline. Entries are append-only, like the stage log.
type CrmActivity struct {
	db.TableStruct[CrmActivityTable, CrmActivity]
	CompanyID     int32           `cb:"1"`
	ID            int32           `cb:"2"`
	OpportunityID int32           `cb:"3"`
	TypeID        int16           `cb:"4"` // system (1..ActivityTypeSystemMaxID) or company type
	Text          string          `cb:"5"`
	Attachments   []CrmAttachment `cb:"6"`
	AuthorID      int32           `cb:"7"`
	Created       int32           `cb:"8"` // SUnixTime
}

type CrmActivityTable struct {
	db.TableStruct[CrmActivityTable, CrmActivity]
	CompanyID     db.Col[*CrmActivityTable, int32]
	ID            db.Col[*CrmActivityTable, int32]
	OpportunityID db.Col[*CrmActivityTable, int32]
	TypeID        db.Col[*CrmActivityTable, int16]
	Text          db.Col[*CrmActivityTable, string]
	Attachments   db.Col[*CrmActivityTable, []CrmAttachment]
	AuthorID      db.Col[*CrmActivityTable, int32]
	Created       db.Col[*CrmActivityTable, int32]
}

func (e CrmActivityTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        87,
		Name:      "crm_activity",
		Partition: e.CompanyID,
		Keys:      db.Cols(e.ID.Autoincrement(0)),
		// The timeline of the opportunity panel reads every entry of one opportunity.
		Indexes: []db.Index{{Type: db.TypeLocalIndex, Keys: db.Cols(e.OpportunityID)}},
	}
}
