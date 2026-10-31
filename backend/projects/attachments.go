package projects

import (
	"encoding/base64"
	"encoding/json"
	"path"
	"strings"

	"app/cloud"
	"app/core"
	"app/projects/types"
)

type attachmentUploadRequest struct {
	ProjectID   int32
	Name        string
	ContentType string
	// Content is the file, base64.
	Content string
}

// PostAttachmentUpload stores one file in the project's folder of the CDN bucket and answers the
// attachment. The file is not a story's yet: it becomes one when the story or comment listing it is
// saved (mergeAttachments checks it is a file of this project's folder).
func PostAttachmentUpload(req *core.HandlerArgs) core.HandlerResponse {
	request := attachmentUploadRequest{}
	if err := json.Unmarshal([]byte(*req.Body), &request); err != nil {
		return req.MakeErr("Error al deserializar el body:", err)
	}
	if _, response, isDenied := authorizeProjectWrite(req, request.ProjectID, types.ProjectRoleMember); isDenied {
		return response
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len(request.Name) > 200 {
		return req.MakeErr("El archivo necesita un nombre de hasta 200 caracteres.")
	}
	fileContent, err := base64.StdEncoding.DecodeString(request.Content)
	if err != nil {
		return req.MakeErr("El contenido del archivo no es base64 válido.")
	}
	if len(fileContent) == 0 || len(fileContent) > MaxAttachmentBytes {
		return req.MakeErr("Un archivo puede pesar hasta 4 MB.")
	}
	// An unknown or overlong type is stored as plain bytes: the browser then downloads the file.
	if request.ContentType == "" || len(request.ContentType) > 100 {
		request.ContentType = "application/octet-stream"
	}

	fileToken := core.MakeSecureRandomBase36String(attachmentFileTokenLength)
	objectKey := attachmentObjectKey(req.User.CompanyID, request.ProjectID, fileToken, request.Name)
	err = cloud.SaveFile(cloud.SaveFileArgs{
		Path: path.Dir(objectKey), Name: path.Base(objectKey), FileContent: fileContent, ContentType: request.ContentType,
	})
	if err != nil {
		return req.MakeErr("Error al guardar el archivo:", err)
	}
	return req.MakeResponse(types.Attachment{
		Key: objectKey, Name: request.Name, ContentType: request.ContentType, Size: int32(len(fileContent)),
		UploadedBy: req.User.ID, UploadedAt: core.SUnixTime(),
	})
}
