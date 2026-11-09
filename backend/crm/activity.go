package crm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"app/cloud"
	"app/core"
	"app/crm/types"
	"app/db"
)

// GetCrmActivityTypes syncs the company's own activity types; the system ones are a frontend
// catalog (types.SystemActivityTypes).
func GetCrmActivityTypes(req *core.HandlerArgs) core.HandlerResponse {
	activityTypes := []types.CrmActivityType{}
	query := db.Query(&activityTypes)
	query.Select().CompanyID.Equals(req.User.CompanyID).Delta(req.GetDeltaCursor(), types.CrmStatusActive)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener los tipos de actividad:", err)
	}
	return req.MakeResponse(activityTypes)
}

// PostCrmActivityType creates or edits a company activity type; Status 0 deletes it (soft, past
// activities keep pointing at it).
func PostCrmActivityType(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmActivityType{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if err := types.NormalizeActivityType(&payload); err != nil {
		return req.MakeErr(err)
	}
	payload.CompanyID, payload.UpdatedBy = req.User.CompanyID, req.User.ID
	// Status 0 deletes an existing type; a new one is always active.
	if payload.ID == 0 || payload.Status != types.CrmStatusDeleted {
		payload.Status = types.CrmStatusActive
	}

	records := []types.CrmActivityType{payload}
	var saveError error
	if payload.ID == 0 {
		saveError = db.Insert(&records)
	} else {
		if _, err := loadActivityType(payload.CompanyID, payload.ID); err != nil {
			return req.MakeErr(err)
		}
		table := db.TableOf[types.CrmActivityType]()
		saveError = db.Update(&records, table.Name, table.Icon, table.Status, table.UpdatedBy)
	}
	if saveError != nil {
		return req.MakeErr("Error al guardar el tipo de actividad:", saveError)
	}
	return req.MakeResponse(records[0])
}

// GetCrmOpportunityActivity returns the activity timeline of one opportunity (?opportunity-id=),
// for users who can see its pipeline.
func GetCrmOpportunityActivity(req *core.HandlerArgs) core.HandlerResponse {
	opportunity, err := authorizeOpportunityView(req, int32(req.GetQueryInt("opportunity-id")))
	if err != nil {
		return req.MakeErr(err)
	}
	activities := []types.CrmActivity{}
	query := db.Query(&activities)
	query.Select().CompanyID.Equals(req.User.CompanyID).OpportunityID.Equals(opportunity.ID)
	if err := query.Exec(); err != nil {
		return req.MakeErr("Error al obtener la actividad:", err)
	}
	return req.MakeResponse(activities)
}

// PostCrmOpportunityActivity appends one entry to an opportunity's timeline. Anyone who can see
// the pipeline may write it, even without the right to edit the opportunity.
func PostCrmOpportunityActivity(req *core.HandlerArgs) core.HandlerResponse {
	payload := types.CrmActivity{}
	if err := json.Unmarshal([]byte(*req.Body), &payload); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if _, err := authorizeOpportunityView(req, payload.OpportunityID); err != nil {
		return req.MakeErr(err)
	}
	companyTypeIsActive := false
	if payload.TypeID > types.ActivityTypeSystemMaxID {
		activityType, err := loadActivityType(req.User.CompanyID, payload.TypeID)
		if err != nil {
			return req.MakeErr(err)
		}
		companyTypeIsActive = activityType.Status == types.CrmStatusActive
	}
	activity, err := types.PlanActivity(payload, req.User.CompanyID, companyTypeIsActive, req.User.ID, core.SUnixTime())
	if err != nil {
		return req.MakeErr(err)
	}
	records := []types.CrmActivity{activity}
	if err := db.Insert(&records); err != nil {
		return req.MakeErr("Error al guardar la actividad:", err)
	}
	return req.MakeResponse(records[0])
}

type activityAttachmentUploadRequest struct {
	OpportunityID int32
	Name          string
	ContentType   string
	// Content is the file, base64.
	Content string
}

// PostCrmAttachmentUpload stores one file in the opportunity's folder of the CDN bucket and answers
// the attachment. It becomes an activity's file when the activity listing it is saved.
func PostCrmAttachmentUpload(req *core.HandlerArgs) core.HandlerResponse {
	request := activityAttachmentUploadRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if _, err := authorizeOpportunityView(req, request.OpportunityID); err != nil {
		return req.MakeErr(err)
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len(request.Name) > 200 {
		return req.MakeErr("El archivo necesita un nombre de hasta 200 caracteres.")
	}
	fileContent, err := base64.StdEncoding.DecodeString(request.Content)
	if err != nil {
		return req.MakeErr("El contenido del archivo no es base64 válido.")
	}
	if len(fileContent) == 0 || len(fileContent) > types.MaxAttachmentBytes {
		return req.MakeErr("Un archivo puede pesar hasta 4 MB.")
	}
	// An unknown or overlong type is stored as plain bytes: the browser then downloads the file.
	if request.ContentType == "" || len(request.ContentType) > 100 {
		request.ContentType = "application/octet-stream"
	}

	// The random token makes the public link unguessable; the file name keeps it readable.
	folder := types.ActivityAttachmentFolder(req.User.CompanyID, request.OpportunityID) + "/" + core.MakeSecureRandomBase36String(16)
	fileName := core.SafeFileName(request.Name)
	err = cloud.SaveFile(cloud.SaveFileArgs{Path: folder, Name: fileName, FileContent: fileContent, ContentType: request.ContentType})
	if err != nil {
		return req.MakeErr("Error al guardar el archivo:", err)
	}
	return req.MakeResponse(types.CrmAttachment{
		Key: path.Join(folder, fileName), Name: request.Name, ContentType: request.ContentType, Size: int32(len(fileContent)),
		UploadedBy: req.User.ID, UploadedAt: core.SUnixTime(),
	})
}

// authorizeOpportunityView loads the opportunity and refuses a user who is not a member of its pipeline.
func authorizeOpportunityView(req *core.HandlerArgs, opportunityID int32) (types.CrmOpportunity, error) {
	opportunity, pipeline, err := loadOpportunityWithPipeline(req.User.CompanyID, opportunityID)
	if err != nil {
		return types.CrmOpportunity{}, err
	}
	if types.MemberLevel(pipeline, req.User.ID) < types.PipelineLevelView {
		return types.CrmOpportunity{}, fmt.Errorf("No tiene acceso a este pipeline.")
	}
	return opportunity, nil
}

func loadActivityType(companyID int32, activityTypeID int16) (types.CrmActivityType, error) {
	activityTypes := []types.CrmActivityType{}
	query := db.Query(&activityTypes)
	query.CompanyID.Equals(companyID).ID.Equals(activityTypeID)
	if err := query.Exec(); err != nil {
		return types.CrmActivityType{}, fmt.Errorf("error al leer el tipo de actividad: %v", err)
	}
	if len(activityTypes) == 0 {
		return types.CrmActivityType{}, fmt.Errorf("El tipo de actividad %d no existe.", activityTypeID)
	}
	return activityTypes[0], nil
}
