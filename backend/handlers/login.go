package handlers

import (
	"app/aws"
	"app/core"
	"app/types"
	"encoding/json"
	"fmt"
	"time"
)

func PostLogin(req *core.HandlerArgs) core.HandlerResponse {

	type Login struct {
		EmpresaID int32
		Usuario   string
		Password  string
		CipherKey string
	}

	body := Login{}
	err := json.Unmarshal([]byte(*req.Body), &body)
	if err != nil {
		return req.MakeErr("Error al deserilizar el body: " + err.Error())
	}

	if len(body.Usuario) < 4 || len(body.Password) < 5 || body.EmpresaID == 0 {
		return req.MakeErr("El usuario/password enviado no posee el formato correcto.")
	}
	if len(body.CipherKey) == 0 {
		return req.MakeErr("El CipherKey es necesario.")
	}

	usuarioTable := MakeUsuarioTable(body.EmpresaID)
	usuarios, err := usuarioTable.QueryBatch([]aws.DynamoQueryParam{
		{Index: "ix1", Equals: body.Usuario},
	})

	if err != nil {
		return req.MakeErr("Error al consultar el usuario.", err.Error())
	}

	core.Log("usuarios encontrados:: ", len(usuarios))

	if len(usuarios) == 0 {
		return req.MakeErr("No se encontró el usuario o el password es incorrecto.")
	}

	usuario := usuarios[0]
	core.Print(usuario)

	passwordHash := core.Env.SECRET_PHRASE + body.Password
	passwordHash = core.FnvHashString64(passwordHash, -1, 20)

	core.Log(passwordHash, usuario.PasswordHash)

	if passwordHash != usuario.PasswordHash {
		return req.MakeErr("No se encontró el usuario o el password es incorrecto.")
	}

	response, err := MakeUsuarioResponse(usuario, body.CipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(response)
}

func MakeUsuarioResponse(usuario types.Usuario, cipherKey string) (map[string]any, error) {

	usuarioToken := core.IUsuario{
		EmpresaID:   usuario.EmpresaID,
		ID:          usuario.ID,
		Expired:     time.Now().Unix() + (60 * 60 * 5),
		Usuario:     usuario.Usuario,
		RolesIDs:    usuario.RolesIDs,
		PerfilesIDs: usuario.PerfilesIDs,
		AccesosIDs:  []int32{},
	}

	if usuario.ID == 1 {
		usuarioToken.PerfilesIDs = []int32{-1}
		for i := 1; i < 200; i++ {
			usuarioToken.AccesosIDs = append(usuarioToken.AccesosIDs, int32(i)*10+7)
		}
		for i := 1; i < 20; i++ {
			usuarioToken.RolesIDs = append(usuarioToken.RolesIDs, int32(i))
		}
		// Obtiene los acceso en base a los perfiles
	} else if len(usuario.PerfilesIDs) > 0 {
		dynamoTable := MakePerfilTable(usuario.EmpresaID)
		querys := []aws.DynamoQueryParam{}
		for _, perfilID := range usuario.PerfilesIDs {
			querys = append(querys, aws.DynamoQueryParam{
				Index: "sk", Equals: fmt.Sprintf("%v", perfilID),
			})
		}
		perfiles, err := dynamoTable.QueryBatch(querys)
		if err != nil {
			return nil, core.Err("Error al obtener perfiles.", err)
		}
		for _, perfil := range perfiles {
			usuarioToken.AccesosIDs = append(usuarioToken.AccesosIDs, perfil.Accesos...)
		}
		usuarioToken.AccesosIDs = core.MakeUnique(usuarioToken.AccesosIDs)
	}

	usuarioInfoJsonBytes, _ := json.Marshal(usuarioToken)
	usuarioInfoString := string(usuarioInfoJsonBytes)
	core.Log("usuarioInfoString", usuarioInfoString)

	// Crea el Token del usuario encriptado
	usuarioTokenCompressed := core.CompressZstd(&usuarioInfoString)
	usuarioTokenEncrypted, err := core.Encrypt(usuarioTokenCompressed)

	core.Log("Accesos Len:", len(usuarioInfoString), "|", len(usuarioTokenCompressed), "|", len(usuarioTokenEncrypted))

	if err != nil {
		return nil, core.Err("Error al generar el Token de usuario.", err)
	}

	// Crea la informacion del usuario encriptada
	usuarioInfoEncrypted, _ := core.Encrypt([]byte(usuarioInfoString), cipherKey)

	response := map[string]any{
		"UserID":       usuario.ID,
		"UserNames":    usuario.Nombres + "|" + usuario.Apellidos,
		"UserEmail":    usuario.Email,
		"UserToken":    core.BytesToBase64(usuarioTokenEncrypted, true),
		"TokenExpTime": usuarioToken.Expired,
		"UserInfo":     core.BytesToBase64(usuarioInfoEncrypted),
		"EmpresaID":    usuario.EmpresaID,
	}

	return response, nil
}

func ReloadLogin(req *core.HandlerArgs) core.HandlerResponse {

	cipherKey := req.GetQuery("cipher-key")
	usuarioTable := MakeUsuarioTable(req.Usuario.EmpresaID)

	usuarios, err := usuarioTable.QueryBatch([]aws.DynamoQueryParam{
		{Index: "ix1", Equals: req.Usuario.Usuario},
	})

	if err != nil {
		return req.MakeErr("Error al obtener el usuario.", err)
	}

	usuario := usuarios[0]
	response, err := MakeUsuarioResponse(usuario, cipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	return req.MakeResponse(response)
}
