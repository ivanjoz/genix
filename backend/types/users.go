package types

import "app/db"

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
	TAGS         `table:"usuarios"`
	ID           int32   `json:"id" db:"id,pk"`
	EmpresaID    int32   `json:"empresaID,omitempty" db:"empresa_id,pk"`
	Usuario      string  `json:"usuario" db:"usuario"`
	Apellidos    string  `json:"apellidos,omitempty" db:"apellidos"`
	Nombres      string  `json:"nombres,omitempty" db:"nombres"`
	PerfilesIDs  []int32 `json:"perfilesIDs,omitempty" db:"perfiles_ids"`
	RolesIDs     []int32 `json:"rolesIDs,omitempty" db:"roles_ids"`
	Email        string  `json:"email,omitempty" db:"email"`
	PasswordHash string  `json:"passwordHash,omitempty"`
	Password     string  `json:"password1,omitempty"`
	Created      int64   `json:"created,omitempty" db:"created"`
	CreatedBy    int32   `json:"createdBy,omitempty" db:"created_by"`
	Updated      int64   `json:"upd,omitempty" db:"updated"`
	UpdatedBy    int32   `json:"updatedBy,omitempty" db:"updated_by"`
	Status       int8    `json:"ss,omitempty" db:"status"`
}

func (e Usuario) ID_() db.CoI32          { return db.CoI32{"id"} }
func (e Usuario) EmpresaID_() db.CoI32   { return db.CoI32{"empresa_id"} }
func (e Usuario) Usuario_() db.CoStr     { return db.CoStr{"usuario"} }
func (e Usuario) Apellidos_() db.CoStr   { return db.CoStr{"apellidos"} }
func (e Usuario) Nombres_() db.CoStr     { return db.CoStr{"nombres"} }
func (e Usuario) PerfilesIDs_() db.CsI32 { return db.CsI32{"perfiles_ids"} }
func (e Usuario) RolesIDs_() db.CsI32    { return db.CsI32{"roles_ids"} }
func (e Usuario) Email_() db.CoStr       { return db.CoStr{"email"} }
func (e Usuario) Status_() db.CoI8       { return db.CoI8{"status"} }
func (e Usuario) Updated_() db.CoI64     { return db.CoI64{"updated"} }
func (e Usuario) UpdatedBy_() db.CoI32   { return db.CoI32{"updated_by"} }
func (e Usuario) Created_() db.CoI64     { return db.CoI64{"created"} }
func (e Usuario) CreatedBy_() db.CoI32   { return db.CoI32{"created_by"} }

func (e Usuario) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:      "usuarios",
		Partition: e.EmpresaID_(),
		Keys:      []db.Coln{e.ID_()},
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
