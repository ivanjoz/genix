package agent

import (
	"time"

	"app/core"
	"app/db"
	"app/agent/types"
)

// Persistence for the in-app agent chat. Backed by the agent_messages table
// (see backend/agent/types/agent_messages.go). The partition key (CompanyUserID)
// is filled by PrepareCloudSync so callers only set CompanyID/UserID.

// Role values for AgentMessage.Role.
const (
	RoleUser  int8 = 1
	RoleAgent int8 = 2
)

// saveMessage inserts one row. Timestamp is unix milliseconds — also serves
// as the message id. Caller passes the cumulative token count for the turn
// (0 for user rows). Returns the timestamp used so the caller can include it
// in the wire reply.
func saveMessage(s *AgentSession, role int8, message, summary string, tokensUsed int32) (int64, error) {
	ts := time.Now().UnixMilli()
	row := types.AgentMessage{
		CompanyID:  s.CompanyID,
		UserID:     s.UserID,
		SessionID:  s.SessionID,
		Timestamp:  ts,
		Role:       role,
		Message:    message,
		Summary:    summary,
		TokensUsed: tokensUsed,
		Status:     1,
		Updated:    core.SUnixTime(),
	}
	row.PrepareCloudSync()
	rows := []types.AgentMessage{row}
	if err := db.Insert(&rows); err != nil {
		return 0, err
	}
	return ts, nil
}

// loadLastN returns the n most recent messages for this session, ordered
// oldest→newest (so the LLM sees the conversation in chronological order
// even though we fetch DESC). n <= 0 returns an empty slice.
func loadLastN(s *AgentSession, n int) ([]types.AgentMessage, error) {
	if n <= 0 {
		return nil, nil
	}
	rows := []types.AgentMessage{}
	q := db.Query(&rows).Limit(int32(n)).OrderDesc()
	q.CompanyUserID.Equals(int64(s.CompanyID)*1_000_000 + int64(s.UserID)).
		SessionID.Equals(s.SessionID)
	if err := q.Exec(); err != nil {
		return nil, err
	}
	// Reverse in place: ScyllaDB returned DESC; the loop wants ASC.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}
