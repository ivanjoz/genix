package agent

import (
	"strings"
	"time"

	"app/agent/routing"
	"app/agent/types"
	"app/core"
	"app/db"
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
func saveMessage(s *AgentSession, role int8, message, summary, attachedContent string, tokensUsed int32) (int64, error) {
	ts := s.nextMessageTimestamp()
	row := types.AgentMessage{
		CompanyID:       s.CompanyID,
		UserID:          s.UserID,
		SessionID:       s.SessionID,
		Timestamp:       ts,
		Role:            role,
		Message:         message,
		Summary:         summary,
		AttachedContent: attachedContent,
		TokensUsed:      tokensUsed,
		Status:          1,
		Updated:         core.SUnixTime(),
	}
	row.PrepareCloudSync()
	rows := []types.AgentMessage{row}
	if err := db.Insert(&rows); err != nil {
		return 0, err
	}
	return ts, nil
}

func (s *AgentSession) nextMessageTimestamp() int64 {
	for {
		previous := s.lastMessageTimestamp.Load()
		next := time.Now().UnixMilli()
		if next <= previous {
			next = previous + 1
		}
		if s.lastMessageTimestamp.CompareAndSwap(previous, next) {
			return next
		}
	}
}

// saveUserMessage records the route visible when the turn started. It is compact
// classifier context, not a page snapshot, and lets follow-ups retain navigation meaning.
func saveUserMessage(s *AgentSession, message, route string) (int64, error) {
	return saveMessage(s, RoleUser, message, "", strings.TrimSpace(route), 0)
}

func saveAgentMessage(s *AgentSession, message, summary string, tokensUsed int32) (int64, error) {
	return saveMessage(s, RoleAgent, message, summary, "", tokensUsed)
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

// loadCompletedTurns returns only complete user/assistant pairs. Extra rows are
// fetched because an interrupted request may leave an unmatched user message.
func loadCompletedTurns(s *AgentSession, maximumTurns int) ([]routing.CompletedTurn, error) {
	if maximumTurns <= 0 {
		return nil, nil
	}
	rows, err := loadLastN(s, maximumTurns*4)
	if err != nil {
		return nil, err
	}
	return assembleCompletedTurns(rows, maximumTurns), nil
}

func assembleCompletedTurns(rows []types.AgentMessage, maximumTurns int) []routing.CompletedTurn {
	if maximumTurns <= 0 {
		return nil
	}
	completed := make([]routing.CompletedTurn, 0, maximumTurns)
	var pendingUser *types.AgentMessage
	for index := range rows {
		row := rows[index]
		switch row.Role {
		case RoleUser:
			// A newer user row supersedes an interrupted turn that never received a reply.
			pendingUser = &row
		case RoleAgent:
			if pendingUser == nil {
				continue
			}
			completed = append(completed, routing.CompletedTurn{
				UserMessage:      strings.TrimSpace(pendingUser.Message),
				AssistantMessage: strings.TrimSpace(row.Message),
				ActionSummary:    strings.TrimSpace(row.Summary),
				Route:            strings.TrimSpace(pendingUser.AttachedContent),
			})
			pendingUser = nil
		}
	}
	if len(completed) > maximumTurns {
		completed = completed[len(completed)-maximumTurns:]
	}
	for index := range completed {
		completed[index].Offset = index - len(completed)
	}
	return completed
}
