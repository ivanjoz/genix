package projects

import (
	"testing"

	"app/projects/types"
)

func TestStoryDocumentChanges(t *testing.T) {
	kept := types.Attachment{Key: "projects/p3/AAAAAAAAAAAA/kept.pdf", Name: "kept.pdf", DocumentID: 40}
	removed := types.Attachment{Key: "projects/p3/BBBBBBBBBBBB/removed.pdf", Name: "removed.pdf", DocumentID: 41}
	added := types.Attachment{Key: "projects/p3/CCCCCCCCCCCC/added.pdf", Name: "added.pdf"}

	changes := storyDocumentChanges(3, 7, []types.Attachment{kept, removed}, []types.Attachment{kept, added}, 9)
	if len(changes) != 2 {
		t.Fatalf("changes = %+v, want the added and the removed file only", changes)
	}
	statusByKey, idByKey := map[string]int8{}, map[string]int64{}
	for _, change := range changes {
		if change.ProjectID != 3 || change.StoryID != 7 || change.ActivityID != 0 || change.UpdatedBy != 9 {
			t.Errorf("change %+v is not a row of story 7 of project 3 written by user 9", change)
		}
		statusByKey[change.Key], idByKey[change.Key] = change.Status, change.ID
	}
	if statusByKey[added.Key] != types.StatusActive || statusByKey[removed.Key] != types.StatusDeleted {
		t.Errorf("statuses = %v, want the added file active and the removed one deleted", statusByKey)
	}
	// The removed file's row is soft-deleted by its ID; the added one takes the next from the sequence.
	if idByKey[removed.Key] != 41 || idByKey[added.Key] != 0 {
		t.Errorf("IDs = %v, want 41 for the removed file and 0 for the added one", idByKey)
	}
	if len(storyDocumentChanges(3, 7, []types.Attachment{kept}, []types.Attachment{kept}, 9)) != 0 {
		t.Error("an unchanged story must rewrite no document")
	}
}

func TestLinkAttachmentDocuments(t *testing.T) {
	attachments := []types.Attachment{{Key: "a", DocumentID: 40}, {Key: "b"}}
	linkAttachmentDocuments(attachments, []types.Document{{Key: "b", ID: 52}})
	if attachments[0].DocumentID != 40 || attachments[1].DocumentID != 52 {
		t.Errorf("attachments = %+v, want a kept at 40 and b linked to 52", attachments)
	}
}

func TestCommentDocuments(t *testing.T) {
	comment := types.StoryActivity{ProjectID: 3, StoryID: 7, ID: 12, Attachments: []types.Attachment{{Key: "projects/p3/AAAAAAAAAAAA/a.png", Size: 10}}}
	documents := commentDocuments(comment, 9)
	if len(documents) != 1 || documents[0].ActivityID != 12 || documents[0].StoryID != 7 || documents[0].Size != 10 || documents[0].Status != types.StatusActive {
		t.Errorf("documents = %+v, want one active row of comment 12", documents)
	}
}
