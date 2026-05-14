package types

import (
	"app/db"
)

// AgentMessage stores chat history for the in-app agent (the widget that lets
// users say "navigate to products" etc.). Schema details and rationale live
// in backend/agent/AGENTIC_LOOP_DESIGN.md.
//
// Partition is the synthetic CompanyUserID = CompanyID*1_000_000 + UserID, so
// every user's chat history lives in its own partition — "load last N
// messages for this user in this session" is a single intra-partition slice.
// Clustering on (SessionID, Timestamp) orders messages naturally inside the
// partition. AttachedContent is reserved for future use (page snapshot or
// screenshot ref) and is written empty for now.
type AgentMessage struct {
	db.TableStruct[AgentMessageTable, AgentMessage]
	CompanyUserID   int64  `db:"company_user_id,pk" col:"company_user_id,pk"`
	SessionID       int64  `db:"session_id,pk" col:"session_id,pk"`
	Timestamp       int64  `db:"timestamp,pk" col:"timestamp,pk,sk"`
	CompanyID       int32  `json:",omitempty" db:"company_id" col:"company_id"`
	UserID          int32  `json:",omitempty" db:"user_id" col:"user_id"`
	Role            int8   `json:",omitempty" db:"role" col:"role"`
	Message         string `json:",omitempty" db:"message" col:"message"`
	AttachedContent string `json:",omitempty" db:"attached_content" col:"attached_content"`
	Summary         string `json:",omitempty" db:"summary" col:"summary"`
	TokensUsed      int32  `json:",omitempty" db:"tokens_used" col:"tokens_used"`
	Status          int8   `json:"ss" db:"status" col:"status"`
	Updated         int32  `json:"upd" db:"updated" col:"updated"`
}

// PrepareCloudSync derives the synthetic partition key so callers fill only
// CompanyID + UserID — the partition value never needs to be computed at the
// call site.
func (e *AgentMessage) PrepareCloudSync() {
	e.CompanyUserID = int64(e.CompanyID)*1_000_000 + int64(e.UserID)
}

type AgentMessageTable struct {
	db.TableStruct[AgentMessageTable, AgentMessage]
	CompanyUserID   db.Col[AgentMessageTable, int64]
	SessionID       db.Col[AgentMessageTable, int64]
	Timestamp       db.Col[AgentMessageTable, int64]
	CompanyID       db.Col[AgentMessageTable, int32]
	UserID          db.Col[AgentMessageTable, int32]
	Role            db.Col[AgentMessageTable, int8]
	Message         db.Col[AgentMessageTable, string]
	AttachedContent db.Col[AgentMessageTable, string]
	Summary         db.Col[AgentMessageTable, string]
	TokensUsed      db.Col[AgentMessageTable, int32]
	Status          db.Col[AgentMessageTable, int8]
	Updated         db.Col[AgentMessageTable, int32]
}

func (e AgentMessageTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "agent_messages",
		Partition: e.CompanyUserID,
		Keys:      []db.Coln{e.SessionID, e.Timestamp},
	}
}
