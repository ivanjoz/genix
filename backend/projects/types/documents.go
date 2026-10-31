package types

import "app/db"

// Document is one file of a project as the Documents page lists it: the project's own documents
// (StoryID 0, added on that page) and a copy of every file attached to a story or to one of its
// comments. StoryDetail and StoryActivity keep their Attachments; their writers keep these rows in
// step (storyDocumentChanges, commentDocuments), through Attachment.DocumentID.
type Document struct {
	db.TableStruct[DocumentTable, Document]
	CompanyID int32 `cb:"1"`
	ID        int64 `cb:"2"`
	ProjectID int32 `cb:"3"`
	// StoryID 0 = a document of the project itself.
	StoryID int64 `cb:"4"`
	// Key is the file in the CDN bucket (Attachment.Key).
	Key string `cb:"5"`
	// ActivityID is the comment the file was attached to, 0 = the story's own files.
	ActivityID  int64  `cb:"6"`
	Name        string `cb:"7"`
	ContentType string `cb:"8"`
	Size        int32  `cb:"9"`
	UploadedBy  int32  `cb:"10"`
	// UploadedAt is a SUnixTime.
	UploadedAt int32   `cb:"11"`
	UpdatedBy  int32   `cb:"12"`
	Status     int8    `json:"ss" cb:"ss,13"`
	Updated    int32   `json:"upd" cb:"upd,14"`
	Version    int16   `json:"ver" cb:"ver,15"`
	Metadata   *string `json:"mdt" cb:"mdt,16"`
}

type DocumentTable struct {
	db.TableStruct[DocumentTable, Document]
	CompanyID   db.Col[*DocumentTable, int32]
	ID          db.Col[*DocumentTable, int64]
	ProjectID   db.Col[*DocumentTable, int32]
	StoryID     db.Col[*DocumentTable, int64]
	Key         db.Col[*DocumentTable, string]
	ActivityID  db.Col[*DocumentTable, int64]
	Name        db.Col[*DocumentTable, string]
	ContentType db.Col[*DocumentTable, string]
	Size        db.Col[*DocumentTable, int32]
	UploadedBy  db.Col[*DocumentTable, int32]
	UploadedAt  db.Col[*DocumentTable, int32]
	UpdatedBy   db.Col[*DocumentTable, int32]
	Status      db.Col[*DocumentTable, int8]
	Updated     db.Col[*DocumentTable, int32]
	Version     db.Col[*DocumentTable, int16]
}

// The delta index pinned on ProjectID syncs the browser cache per project (GET.projects-documents).
func (e DocumentTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:          64,
		Name:        "projects_document",
		Partition:   e.CompanyID,
		Keys:              db.Cols(e.ID),
		KeyIntPacking:     db.Cols(e.ProjectID.Size(63-projectSequenceBits), e.Autoincrement(0)),
		AutoincrementPart: e.ProjectID,
		FixedValues: []db.FixedValues{{Col: e.Status, Values: statusValues}},
		Indexes:     []db.Index{{Type: db.TypeDelta, Keys: db.Cols(e.ProjectID, e.Status)}},
	}
}
