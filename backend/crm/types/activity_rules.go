package types

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"app/core"
)

// MaxAttachmentBytes caps one attached file. It travels base64 in the upload request and a Lambda
// payload caps at 6 MB, so 4 MB (5.4 MB encoded) is what fits with the request around it.
const MaxAttachmentBytes = 4 * 1024 * 1024

const maxAttachmentsPerActivity = 30
const maxActivityTextLength = 4000

// ActivityAttachmentFolder is the opportunity's folder in the CDN bucket. The company is in it:
// opportunity IDs are per company, so the folder alone must not let one company name another's file.
func ActivityAttachmentFolder(companyID int32, opportunityID int32) string {
	return fmt.Sprintf("crm/c%d/o%d", companyID, opportunityID)
}

var opportunityFileKeyPattern = regexp.MustCompile(`^crm/c([0-9]+)/o([0-9]+)/[0-9a-z]{16}/[0-9A-Za-z._-]{1,100}$`)

// isOpportunityFileKey: an activity only takes the files uploaded to its own opportunity's folder,
// so no entry can list another opportunity's (or company's) file.
func isOpportunityFileKey(companyID int32, opportunityID int32, objectKey string) bool {
	keyParts := opportunityFileKeyPattern.FindStringSubmatch(objectKey)
	return len(keyParts) == 3 && keyParts[1] == strconv.Itoa(int(companyID)) && keyParts[2] == strconv.Itoa(int(opportunityID))
}

func IsActivityIcon(icon string) bool {
	for _, option := range ActivityIconOptions {
		if option.Icon == icon {
			return true
		}
	}
	return false
}

func isSystemActivityType(typeID int16) bool {
	for _, option := range SystemActivityTypes {
		if option.ID == typeID {
			return true
		}
	}
	return false
}

// NormalizeActivityType validates a company activity type before it is saved.
func NormalizeActivityType(activityType *CrmActivityType) error {
	activityType.Name = strings.TrimSpace(activityType.Name)
	if activityType.Name == "" || len(activityType.Name) > 60 {
		return errors.New("El tipo de actividad necesita un nombre de hasta 60 caracteres.")
	}
	if !IsActivityIcon(activityType.Icon) {
		return fmt.Errorf("El ícono \"%s\" no es válido.", activityType.Icon)
	}
	if activityType.ID != 0 && activityType.ID <= ActivityTypeSystemMaxID {
		return errors.New("Los tipos de actividad del sistema no se pueden editar.")
	}
	return nil
}

// PlanActivity builds the entry to insert: a known type, some text or a file, and only files of
// this opportunity's folder. companyTypeIsActive answers for a company type (ID above the reserved
// range); system types are checked against the catalog here.
// pipelineID is the opportunity's current pipeline, never the client's.
func PlanActivity(payload CrmActivity, companyID int32, pipelineID int16, companyTypeIsActive bool, userID int32, now int32) (CrmActivity, error) {
	typeIsKnown := isSystemActivityType(payload.TypeID) || (payload.TypeID > ActivityTypeSystemMaxID && companyTypeIsActive)
	if !typeIsKnown {
		return CrmActivity{}, fmt.Errorf("El tipo de actividad %d no existe.", payload.TypeID)
	}
	payload.Text = strings.TrimSpace(payload.Text)
	if payload.Text == "" && len(payload.Attachments) == 0 {
		return CrmActivity{}, errors.New("La actividad necesita un texto o un archivo.")
	}
	if len(payload.Text) > maxActivityTextLength {
		return CrmActivity{}, errors.New("El texto puede tener hasta 4000 caracteres.")
	}
	if len(payload.Attachments) > maxAttachmentsPerActivity {
		return CrmActivity{}, errors.New("Se pueden adjuntar hasta 30 archivos.")
	}
	for index := range payload.Attachments {
		attachment := &payload.Attachments[index]
		attachment.Name = strings.TrimSpace(attachment.Name)
		if !isOpportunityFileKey(companyID, payload.OpportunityID, attachment.Key) || attachment.Name == "" || len(attachment.Name) > 200 ||
			attachment.Size <= 0 || attachment.Size > MaxAttachmentBytes || len(attachment.ContentType) > 100 {
			return CrmActivity{}, errors.New("Un archivo adjunto no es válido.")
		}
		attachment.UploadedBy, attachment.UploadedAt = userID, now
	}
	payload.CompanyID, payload.ID, payload.AuthorID, payload.Created = companyID, 0, userID, now
	payload.PipelineID, payload.Period = pipelineID, core.PeriodOfDay(core.SUnixTimeToUnixDay(now))
	return payload, nil
}
