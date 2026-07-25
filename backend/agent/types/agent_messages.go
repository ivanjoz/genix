package types

import (
	"github.com/ivanjoz/genix-orm/scylla"
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
	scylla.TableStruct[AgentMessageTable, AgentMessage]
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
	scylla.TableStruct[AgentMessageTable, AgentMessage]
	CompanyUserID   scylla.Col[AgentMessageTable, int64]
	SessionID       scylla.Col[AgentMessageTable, int64]
	Timestamp       scylla.Col[AgentMessageTable, int64]
	CompanyID       scylla.Col[AgentMessageTable, int32]
	UserID          scylla.Col[AgentMessageTable, int32]
	Role            scylla.Col[AgentMessageTable, int8]
	Message         scylla.Col[AgentMessageTable, string]
	AttachedContent scylla.Col[AgentMessageTable, string]
	Summary         scylla.Col[AgentMessageTable, string]
	TokensUsed      scylla.Col[AgentMessageTable, int32]
	Status          scylla.Col[AgentMessageTable, int8]
	Updated         scylla.Col[AgentMessageTable, int32]
}

func (e AgentMessageTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:      "agent_messages",
		Partition: e.CompanyUserID,
		Keys:      scylla.Cols(e.SessionID, e.Timestamp),
	}
}
