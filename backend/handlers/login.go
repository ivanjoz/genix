package handlers

import (
	"app/aws"
	"app/core"
	"encoding/json"
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

	usuarioToken := core.IUsuario{
		EmpresaID:   body.EmpresaID,
		ID:          usuario.ID,
		Created:     time.Now().Unix(),
		Usuario:     usuario.Usuario,
		RolesIDs:    usuario.RolesIDs,
		PerfilesIDs: []int32{-1},
		AccesosIDs:  []int32{},
	}

	if usuario.ID == 1 {
		for i := 1; i < 200; i++ {
			usuarioToken.AccesosIDs = append(usuarioToken.AccesosIDs, int32(i)*10+7)
		}
		for i := 1; i < 20; i++ {
			usuarioToken.RolesIDs = append(usuarioToken.RolesIDs, int32(i))
		}
	}

	usuarioInfoJsonBytes, _ := json.Marshal(usuarioToken)
	usuarioInfoString := string(usuarioInfoJsonBytes)

	// Crea el Token del usuario encriptado
	usuarioTokenCompressed := core.CompressZstd(&usuarioInfoString)
	usuarioTokenEncrypted, err := core.Encrypt(usuarioTokenCompressed)

	if err != nil {
		return req.MakeErr("Error al generar el Token de usuario.", err.Error())
	}

	// Crea la informacion del usuario encriptada
	usuarioInfoEncrypted, _ := core.Encrypt([]byte(usuarioInfoString), body.CipherKey)

	response := map[string]any{
		"UserID":       usuario.ID,
		"UserNames":    usuario.Nombres + "|" + usuario.Apellidos,
		"UserEmail":    usuario.Email,
		"UserToken":    core.BytesToBase64(usuarioTokenEncrypted, true),
		"TokenExpTime": time.Now().Unix() + (60 * 60 * 4),
		"UserInfo":     core.BytesToBase64(usuarioInfoEncrypted),
		"EmpresaID":    usuario.EmpresaID,
	}

	return req.MakeResponse(response)
}
