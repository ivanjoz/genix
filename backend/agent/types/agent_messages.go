package types

import (
	"app/db"
)

// AgentMessage stores chat history for the in-app agent (the widget that lets
// users say "navigate to products" etc.).
//
// Partition is the synthetic CompanyUserID = CompanyID*1_000_000 + UserID, so
// every user's chat history lives in its own partition — "load last N
// messages for this user in this session" is a single intra-partition slice.
// Clustering on (SessionID, Timestamp) orders messages naturally inside the
// partition. AttachedContent stores the compact route on user rows; it never
// stores page HTML, builder state, retrieved documentation, or screenshots.
type AgentMessage struct {
	db.TableStruct[AgentMessageTable, AgentMessage]
	CompanyUserID   int64  `db:"company_user_id,pk" cb:"CompanyUserID,1"`
	SessionID       int64  `db:"session_id,pk" cb:"SessionID,2"`
	Timestamp       int64  `db:"timestamp,pk" cb:"Timestamp,3"`
	CompanyID       int32  `db:"company_id" cb:"CompanyID,4"`
	UserID          int32  `db:"user_id" cb:"UserID,5"`
	Role            int8   `db:"role" cb:"Role,6"`
	Message         string `db:"message" cb:"Message,7"`
	AttachedContent string `db:"attached_content" cb:"AttachedContent,8"`
	Summary         string `db:"summary" cb:"Summary,9"`
	TokensUsed      int32  `db:"tokens_used" cb:"TokensUsed,10"`
	Status          int8   `json:"ss" db:"status" cb:"ss,11"`
	Updated         int32  `json:"upd" db:"updated" cb:"upd,12"`
}

// PrepareCloudSync derives the synthetic partition key so callers fill only
// CompanyID + UserID — the partition value never needs to be computed at the
// call site.
func (e *AgentMessage) PrepareCloudSync() {
	e.CompanyUserID = int64(e.CompanyID)*1_000_000 + int64(e.UserID)
}

type AgentMessageTable struct {
	db.TableStruct[AgentMessageTable, AgentMessage]
	CompanyUserID   db.Col[*AgentMessageTable, int64]
	SessionID       db.Col[*AgentMessageTable, int64]
	Timestamp       db.Col[*AgentMessageTable, int64]
	CompanyID       db.Col[*AgentMessageTable, int32]
	UserID          db.Col[*AgentMessageTable, int32]
	Role            db.Col[*AgentMessageTable, int8]
	Message         db.Col[*AgentMessageTable, string]
	AttachedContent db.Col[*AgentMessageTable, string]
	Summary         db.Col[*AgentMessageTable, string]
	TokensUsed      db.Col[*AgentMessageTable, int32]
	Status          db.Col[*AgentMessageTable, int8]
	Updated         db.Col[*AgentMessageTable, int32]
}

func (e AgentMessageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		ID:        2,
		Name:      "agent_messages",
		Partition: e.CompanyUserID,
		Keys:      db.Cols(e.SessionID, e.Timestamp),
	}
}
