package types

import "app/db2"

type TAGS struct{}

type Empresa struct { // DynamoDB
	TAGS               `table:"empresas"`
	ID                 int32        `json:"id"`
	Nombre             string       `json:",omitempty"`
	RazonSocial        string       `json:",omitempty"`
	RUC                string       `json:",omitempty"`
	Email              string       `json:",omitempty"`
	NotificacionEmail  string       `json:",omitempty"`
	Telefono           string       `json:",omitempty"`
	Representante      string       `json:",omitempty"`
	Direccion          string       `json:",omitempty"`
	Ciudad             string       `json:",omitempty"`
	FormApiKey         string       `json:",omitempty"`
	EmailVerificado    int8         `json:",omitempty"`
	TelefonoVerificado int8         `json:",omitempty"`
	SmtpConfig         SmtpConfig   `json:",omitempty"`
	CulquiConfig       CulquiConfig `json:",omitempty"`
	Updated            int64        `json:"upd"`
	Status             int8         `json:"ss"`
}

type SmtpConfig struct {
	Email    string `json:",omitempty"`
	User     string `json:",omitempty"`
	Password string `json:",omitempty"`
	Post     int32  `json:",omitempty"`
	Host     string `json:",omitempty"`
}

type CulquiConfig struct {
	RsaKey       string `json:",omitempty"`
	RsaKeyID     string `json:",omitempty"`
	LlaveLive    string `json:",omitempty"`
	LlavePubLive string `json:",omitempty"`
	LlaveDev     string `json:",omitempty"`
	LlavePubDev  string `json:",omitempty"`
}

type EmpresaPub struct {
	ID            int32  `json:"id"`
	Nombre        string `json:",omitempty"`
	CulqiRsaKey   string `json:",omitempty"`
	CulqiRsaKeyID string `json:",omitempty"`
	CulqiLlave    string `json:",omitempty"`
}

type Usuario struct { // DynamoDB + ScyllaDB
	db2.TableStruct[UsuarioTable, Usuario]
	ID           int32   `json:"id" db:"id,pk"`
	EmpresaID    int32   `json:"empresaID,omitempty" db:"empresa_id,pk"`
	Usuario      string  `json:"usuario" db:"usuario"`
	Apellidos    string  `json:"apellidos,omitempty" db:"apellidos"`
	Nombres      string  `json:"nombres,omitempty" db:"nombres"`
	PerfilesIDs  []int32 `json:"perfilesIDs,omitempty" db:"perfiles_ids"`
	RolesIDs     []int32 `json:"rolesIDs,omitempty" db:"roles_ids"`
	Email        string  `json:"email,omitempty" db:"email"`
	Cargo        string  `json:"cargo,omitempty" db:"cargo"`
	DocumentoNro string  `json:"documentoNro,omitempty" db:"documento_nro"`
	PasswordHash string  `json:"passwordHash,omitempty"`
	Password     string  `json:"password1,omitempty"`
	Created      int64   `json:"created,omitempty" db:"created"`
	CreatedBy    int32   `json:"createdBy,omitempty" db:"created_by"`
	Updated      int64   `json:"upd,omitempty" db:"updated"`
	UpdatedBy    int32   `json:"updatedBy,omitempty" db:"updated_by"`
	Status       int8    `json:"ss,omitempty" db:"status"`
}

type UsuarioTable struct {
	db2.TableStruct[UsuarioTable, Usuario]
	ID           db2.Col[UsuarioTable, int32]
	EmpresaID    db2.Col[UsuarioTable, int32]
	Usuario      db2.Col[UsuarioTable, string]
	Apellidos    db2.Col[UsuarioTable, string]
	Nombres      db2.Col[UsuarioTable, string]
	PerfilesIDs  db2.ColSlice[UsuarioTable, int32] `db:"perfiles_ids"`
	RolesIDs     db2.ColSlice[UsuarioTable, int32] `db:"roles_ids"`
	Email        db2.Col[UsuarioTable, string]
	Cargo        db2.Col[UsuarioTable, string]
	DocumentoNro db2.Col[UsuarioTable, string]
	Created      db2.Col[UsuarioTable, int64]
	CreatedBy    db2.Col[UsuarioTable, int32]
	Updated      db2.Col[UsuarioTable, int64]
	UpdatedBy    db2.Col[UsuarioTable, int32]
	Status       db2.Col[UsuarioTable, int8]
}

func (e UsuarioTable) GetSchema() db2.TableSchema {
	return db2.TableSchema{
		Name:      "usuarios",
		Partition: e.EmpresaID,
		Keys:      []db2.Coln{e.ID},
	}
}

type SeguridadAcceso struct { // DynamoDB
	TAGS        `table:"seguridad_acceso"`
	ID          int32   `json:"id"`
	Nombre      string  `json:"nombre" db:"nombre"`
	Descripcion string  `json:"descripcion" db:"descripcion"`
	Grupo       int16   `json:"grupo" db:"grupo"`
	Orden       int16   `json:"orden" db:"orden"`
	Modulos     []int16 `json:"modulosIDs" db:"modulos_ids"`
	Acciones    []int16 `json:"acciones" db:"acciones"`
	Status      int8    `json:"ss" db:"status"`
	Updated     int64   `json:"upd" db:"updated"`
}

type Perfil struct { // DynamoDB
	TAGS        `table:"seguridad_perfiles"`
	ID          int32   `json:"id"`
	EmpresaID   int32   `json:"empresaID" db:"company_id,pk"`
	Nombre      string  `json:"nombre" db:"nombre"`
	Descripcion string  `json:"descripcion" db:"descripcion"`
	Modulos     []int16 `json:"modulosIDs" db:"modulos_ids"`
	Accesos     []int32 `json:"accesos" db:"accesos"`
	Status      int8    `json:"ss" db:"status"`
	Updated     int64   `json:"upd" db:"updated"`
}
