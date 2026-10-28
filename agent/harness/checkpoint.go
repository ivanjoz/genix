package harness

import (
	"context"
	"errors"
	"fmt"

	"app/core"
	"app/db"

	"github.com/ivanjoz/genix-orm/utils"
)

// CheckPointStore keeps a paused turn's Eino graph state in the ORM's file store (the S3 Express
// bucket of [file_store]), so a resume that lands in another Lambda container still finds it.
//
// The key carries the company and the user, so a checkpoint ID echoed back by a browser can only
// ever reach its own user's checkpoints: the ID is not trusted, the session is. Expiry is meant
// to be a 1-day lifecycle rule on the agent-checkpoints/ prefix (not configured yet): an unanswered
// question is abandoned, not resumed days later.
type CheckPointStore struct {
	CompanyID int32
	UserID    int32
}

func (store CheckPointStore) key(checkPointID string) string {
	return fmt.Sprintf("agent-checkpoints/%d/%d/%s.bin", store.CompanyID, store.UserID, checkPointID)
}

// specKey holds what the paused turn's agent was built from (model, instruction, tool names): a
// resume must rebuild the same agent, and none of that is in Eino's checkpoint.
func (store CheckPointStore) specKey(checkPointID string) string {
	return fmt.Sprintf("agent-checkpoints/%d/%d/%s.turn.json", store.CompanyID, store.UserID, checkPointID)
}

// SetTurnSpec stores the turn's build spec next to its checkpoint.
func (store CheckPointStore) SetTurnSpec(ctx context.Context, checkPointID string, spec []byte) error {
	return store.overwrite(ctx, store.specKey(checkPointID), spec)
}

// GetTurnSpec reads it back; a missing spec is an error, because the resume cannot rebuild the turn.
func (store CheckPointStore) GetTurnSpec(checkPointID string) ([]byte, error) {
	content, _, err := db.FileStore().GetFile(store.specKey(checkPointID))
	return content, err
}

// Get tells "no checkpoint" from "the read failed": Eino starts over on the first, and the second
// must surface, or a resume that failed on permissions would silently restart the turn.
func (store CheckPointStore) Get(_ context.Context, checkPointID string) ([]byte, bool, error) {
	content, _, err := db.FileStore().GetFile(store.key(checkPointID))
	if errors.Is(err, utils.ErrFileNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	core.Log("agent.checkpoint read::", checkPointID, " bytes::", len(content))
	return content, true, nil
}

// Set overwrites: a resumed turn that pauses again writes under the same ID. The store only offers
// a conditional write, so the current ETag is read first ("" when the file does not exist yet).
func (store CheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	if err := store.overwrite(ctx, store.key(checkPointID), checkPoint); err != nil {
		return fmt.Errorf("write checkpoint %s: %w", checkPointID, err)
	}
	core.Log("agent.checkpoint written::", checkPointID, " bytes::", len(checkPoint))
	return nil
}

func (store CheckPointStore) overwrite(ctx context.Context, fileKey string, content []byte) error {
	_, currentETag, err := db.FileStore().GetFile(fileKey)
	if err != nil && !errors.Is(err, utils.ErrFileNotFound) {
		return err
	}
	_, err = db.FileStore().PutFileIfMatch(ctx, fileKey, content, currentETag)
	return err
}
