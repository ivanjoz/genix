package projects

import (
	"encoding/json"

	"app/core"
	"app/db"
	"app/projects/types"
)

// GetDocuments is the delta of a project's documents: its own and its stories' files.
func GetDocuments(req *core.HandlerArgs) core.HandlerResponse {
	documents := []types.Document{}
	query := db.Query(&documents)
	query.CompanyID.Equals(req.User.CompanyID).ProjectID.Equals(req.GetQueryInt("project-id")).
		Delta(req.GetDeltaCursor(), types.StatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los documentos:", err)
	}
	return core.MakeResponse(req, &documents)
}

// PostDocument adds a document to the project (a file uploaded by POST.projects-attachment-upload),
// or deletes one (ss 0) with its file. Team members. Only the project's own documents: a story's
// files are added and removed by saving the story.
func PostDocument(req *core.HandlerArgs) core.HandlerResponse {
	document := types.Document{}
	if err := json.Unmarshal([]byte(*req.Body), &document); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if document.StoryID != 0 {
		return req.MakeErr("Los archivos de una historia se editan en la historia.")
	}
	savedDocument := types.Document{}
	err := db.RunTransaction(func(transaction *db.Transaction) error {
		var err error
		savedDocument, err = saveProjectDocumentIn(transaction, req, document)
		return err
	})
	if err != nil {
		return makeProjectErr(req, err)
	}
	// The file goes once the record no longer names it: a failed write keeps both.
	if savedDocument.Status == types.StatusDeleted {
		deleteProjectFiles([]string{savedDocument.Key})
	}
	return req.MakeResponse(savedDocument)
}

// saveProjectDocumentIn adds or deletes a project's own document under the project's row lock.
func saveProjectDocumentIn(transaction *db.Transaction, req *core.HandlerArgs, document types.Document) (types.Document, error) {
	if _, err := lockProjectWriteIn(transaction, req, document.ProjectID, types.ProjectRoleMember); err != nil {
		return types.Document{}, err
	}
	companyID := req.User.CompanyID

	if document.Status == types.StatusDeleted {
		storedDocuments := []types.Document{{CompanyID: companyID, ID: document.ID}}
		if err := db.LockByKeys(transaction, &storedDocuments); err != nil {
			return types.Document{}, core.Err("Error al obtener el documento:", err)
		}
		if len(storedDocuments) == 0 || storedDocuments[0].ProjectID != document.ProjectID || storedDocuments[0].Status != types.StatusActive {
			return types.Document{}, core.Err("El documento no existe.")
		}
		deletedDocument := storedDocuments[0]
		if deletedDocument.StoryID != 0 {
			return types.Document{}, core.Err("Los archivos de una historia se editan en la historia.")
		}
		deletedDocument.Status, deletedDocument.UpdatedBy = types.StatusDeleted, req.User.ID
		if err := saveDocumentChangesIn(transaction, companyID, []types.Document{deletedDocument}); err != nil {
			return types.Document{}, core.Err("Error al eliminar el documento:", err)
		}
		return deletedDocument, nil
	}

	// The same validation and stamping as a story's attachment.
	attachments, _, err := mergeAttachments(companyID, document.ProjectID, nil, []types.Attachment{{
		Key: document.Key, Name: document.Name, ContentType: document.ContentType, Size: document.Size,
	}}, req.User.ID, core.SUnixTime())
	if err != nil {
		return types.Document{}, err
	}
	newDocuments := []types.Document{documentOfAttachment(document.ProjectID, 0, 0, attachments[0], req.User.ID)}
	if err := saveDocumentChangesIn(transaction, companyID, newDocuments); err != nil {
		return types.Document{}, core.Err("Error al guardar el documento:", err)
	}
	return newDocuments[0], nil
}
