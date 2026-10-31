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
	_, release, response, isDenied := openProjectWrite(req, document.ProjectID, types.ProjectRoleMember)
	if isDenied {
		return response
	}
	defer release()
	companyID := req.User.CompanyID

	if document.Status == types.StatusDeleted {
		storedDocuments := []types.Document{}
		query := db.Query(&storedDocuments)
		query.CompanyID.Equals(companyID).ID.Equals(document.ID)
		if err := query.Exec(); err != nil {
			return req.MakeErr("Error al obtener el documento:", err)
		}
		if len(storedDocuments) == 0 || storedDocuments[0].ProjectID != document.ProjectID || storedDocuments[0].Status != types.StatusActive {
			return req.MakeErr("El documento no existe.")
		}
		deletedDocument := storedDocuments[0]
		if deletedDocument.StoryID != 0 {
			return req.MakeErr("Los archivos de una historia se editan en la historia.")
		}
		deletedDocument.Status, deletedDocument.UpdatedBy = types.StatusDeleted, req.User.ID
		if err := saveDocumentChanges(companyID, []types.Document{deletedDocument}); err != nil {
			return req.MakeErr("Error al eliminar el documento:", err)
		}
		deleteProjectFiles([]string{deletedDocument.Key})
		return req.MakeResponse(deletedDocument)
	}

	// The same validation and stamping as a story's attachment.
	attachments, _, err := mergeAttachments(companyID, document.ProjectID, nil, []types.Attachment{{
		Key: document.Key, Name: document.Name, ContentType: document.ContentType, Size: document.Size,
	}}, req.User.ID, core.SUnixTime())
	if err != nil {
		return req.MakeErr(err)
	}
	newDocuments := []types.Document{documentOfAttachment(document.ProjectID, 0, 0, attachments[0], req.User.ID)}
	if err := saveDocumentChanges(companyID, newDocuments); err != nil {
		return req.MakeErr("Error al guardar el documento:", err)
	}
	return req.MakeResponse(newDocuments[0])
}
