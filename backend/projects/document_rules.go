package projects

import "app/projects/types"

// documentOfAttachment is the Documents page's copy of an attached file. A new file has DocumentID
// 0, so the write takes the next ID from the sequence.
func documentOfAttachment(projectID int32, storyID int64, activityID int64, attachment types.Attachment, userID int32) types.Document {
	return types.Document{
		ProjectID: projectID, ID: attachment.DocumentID, StoryID: storyID, Key: attachment.Key, ActivityID: activityID,
		Name: attachment.Name, ContentType: attachment.ContentType, Size: attachment.Size,
		UploadedBy: attachment.UploadedBy, UploadedAt: attachment.UploadedAt,
		Status: types.StatusActive, UpdatedBy: userID,
	}
}

// storyDocumentChanges is what a story save changes in the project's documents: a row for each file
// added to the story, and a soft delete for each one removed. Unchanged files are not rewritten.
func storyDocumentChanges(projectID int32, storyID int64, storedAttachments []types.Attachment, savedAttachments []types.Attachment, userID int32) []types.Document {
	storedKeys := map[string]bool{}
	for _, storedAttachment := range storedAttachments {
		storedKeys[storedAttachment.Key] = true
	}
	savedKeys := map[string]bool{}
	documentChanges := []types.Document{}
	for _, savedAttachment := range savedAttachments {
		savedKeys[savedAttachment.Key] = true
		if !storedKeys[savedAttachment.Key] {
			documentChanges = append(documentChanges, documentOfAttachment(projectID, storyID, 0, savedAttachment, userID))
		}
	}
	for _, storedAttachment := range storedAttachments {
		if !savedKeys[storedAttachment.Key] {
			removedDocument := documentOfAttachment(projectID, storyID, 0, storedAttachment, userID)
			removedDocument.Status = types.StatusDeleted
			documentChanges = append(documentChanges, removedDocument)
		}
	}
	return documentChanges
}

// linkAttachmentDocuments sets each attachment's DocumentID from the written document rows, matched
// by the file's Key: the IDs the write just assigned to the new files.
func linkAttachmentDocuments(attachments []types.Attachment, writtenDocuments []types.Document) {
	documentIDByKey := map[string]int64{}
	for _, document := range writtenDocuments {
		documentIDByKey[document.Key] = document.ID
	}
	for attachmentIndex, attachment := range attachments {
		if documentID, isWritten := documentIDByKey[attachment.Key]; isWritten {
			attachments[attachmentIndex].DocumentID = documentID
		}
	}
}

// commentDocuments are the rows of the files attached to a comment, once it has its ID.
func commentDocuments(comment types.StoryActivity, userID int32) []types.Document {
	documents := []types.Document{}
	for _, attachment := range comment.Attachments {
		documents = append(documents, documentOfAttachment(comment.ProjectID, comment.StoryID, comment.ID, attachment, userID))
	}
	return documents
}
