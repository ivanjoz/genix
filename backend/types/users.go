package types

import (
	"app/db"
	"fmt"
)

type TAGS struct{}

type Empresa struct {
	db.TableStruct[EmpresaTable, Empresa]
	ID                 int32        `json:"id" db:"id,pk" col:",sk"`
	Nombre             string       `json:",omitempty" db:"nombre" col:""`
	RazonSocial        string       `json:",omitempty" db:"razon_social" col:""`
	RUC                string       `json:",omitempty" db:"ruc" col:",index"`
	Email              string       `json:",omitempty" db:"email" col:",index"`
	NotificacionEmail  string       `json:",omitempty" db:"notificacion_email" col:""`
	Telefono           string       `json:",omitempty" db:"telefono" col:""`
	Representante      string       `json:",omitempty" db:"representante" col:""`
	Direccion          string       `json:",omitempty" db:"direccion" col:""`
	Ciudad             string       `json:",omitempty" db:"ciudad" col:""`
	FormApiKey         string       `json:",omitempty" db:"form_api_key" col:""`
	EmailVerificado    int8         `json:",omitempty" db:"email_verificado" col:""`
	TelefonoVerificado int8         `json:",omitempty" db:"telefono_verificado" col:""`
	SmtpConfig         SmtpConfig   `json:",omitempty" db:"smtp_config" col:""`
	CulquiConfig       CulquiConfig `json:",omitempty" db:"culqui_config" col:""`
	Updated            int64        `json:"upd" db:"updated" col:",index"`
	Status             int8         `json:"ss" db:"status" col:""`
}

type EmpresaTable struct {
	db.TableStruct[EmpresaTable, Empresa]
	ID                 db.Col[EmpresaTable, int32]
	Nombre             db.Col[EmpresaTable, string]
	RazonSocial        db.Col[EmpresaTable, string]
	RUC                db.Col[EmpresaTable, string]
	Email              db.Col[EmpresaTable, string]
	NotificacionEmail  db.Col[EmpresaTable, string]
	Telefono           db.Col[EmpresaTable, string]
	Representante      db.Col[EmpresaTable, string]
	Direccion          db.Col[EmpresaTable, string]
	Ciudad             db.Col[EmpresaTable, string]
	FormApiKey         db.Col[EmpresaTable, string]
	EmailVerificado    db.Col[EmpresaTable, int8]
	TelefonoVerificado db.Col[EmpresaTable, int8]
	SmtpConfig         db.Col[EmpresaTable, SmtpConfig]
	CulquiConfig       db.Col[EmpresaTable, CulquiConfig]
	Updated            db.Col[EmpresaTable, int64]
	Status             db.Col[EmpresaTable, int8]
}

func (e EmpresaTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "empresas",
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
	}
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
	db.TableStruct[UsuarioTable, Usuario]
	EmpresaID               int32   `json:",omitempty" db:"empresa_id,pk" col:"empresa_id,pk"`
	ID                      int32   `json:",omitempty" db:"id,pk" col:"id,pk,sk"`
	Usuario                 string  `json:",omitempty" db:"usuario" col:"usuario,index"`
	Apellidos               string  `json:",omitempty" db:"apellidos" col:"apellidos"`
	Nombres                 string  `json:",omitempty" db:"nombres" col:"nombres"`
	PerfilesIDs             []int32 `json:",omitempty" db:"perfiles_ids" col:"perfiles_ids"`
	RolesIDs                []int32 `json:",omitempty" db:"roles_ids" col:"roles_ids"`
	Email                   string  `json:",omitempty" db:"email" col:"email,index"`
	Cargo                   string  `json:",omitempty" db:"cargo" col:"cargo"`
	DocumentoNro            string  `json:",omitempty" db:"documento_nro" col:"documento_nro"`
	PasswordHash            string  `json:",omitempty" col:"password_hash"`
	Password                string  `json:",omitempty" col:"-"`
	Created                 int64   `json:",omitempty" db:"created" col:"created"`
	CreatedBy               int32   `json:",omitempty" db:"created_by" col:"created_by"`
	Updated                 int64   `json:",omitempty" db:"updated" col:"updated"`
	UpdatedBy               int32   `json:",omitempty" db:"updated_by" col:"updated_by"`
	Status                  int8    `json:",omitempty" db:"status" col:"status"`
	CompanyUserIndex        string  `json:"-" col:"company_usuario,index"`
	CompanyStatusIndex      string  `json:"-" col:"company_status_updated,index"`
	// CacheVersion is returned in delta-by-IDs endpoints to let clients track per-record cache freshness.
	CacheVersion uint8 `json:",omitempty" col:"-"`
}

func (e *Usuario) PrepareCloudSync() {
	// Company + status + padded updated keeps delta queries lexicographically sortable across providers.
	e.CompanyUserIndex = fmt.Sprintf("%d_%s", e.EmpresaID, e.Usuario)
	e.CompanyStatusIndex = fmt.Sprintf("%d_%d_%020d", e.EmpresaID, e.Status, e.Updated)
}

type UsuarioTable struct {
	db.TableStruct[UsuarioTable, Usuario]
	ID           db.Col[UsuarioTable, int32]
	EmpresaID    db.Col[UsuarioTable, int32]
	Usuario      db.Col[UsuarioTable, string]
	Apellidos    db.Col[UsuarioTable, string]
	Nombres      db.Col[UsuarioTable, string]
	PerfilesIDs  db.ColSlice[UsuarioTable, int32] `db:"perfiles_ids"`
	RolesIDs     db.ColSlice[UsuarioTable, int32] `db:"roles_ids"`
	Email        db.Col[UsuarioTable, string]
	Cargo        db.Col[UsuarioTable, string]
	DocumentoNro db.Col[UsuarioTable, string]
	Created      db.Col[UsuarioTable, int64]
	CreatedBy    db.Col[UsuarioTable, int32]
	Updated      db.Col[UsuarioTable, int64]
	UpdatedBy    db.Col[UsuarioTable, int32]
	Status       db.Col[UsuarioTable, int8]
}

func (e UsuarioTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:             "usuarios",
		Partition:        e.EmpresaID,
		UseSequences:     true,
		SaveCacheVersion: true,
		Keys:             []db.Coln{e.ID.Autoincrement(0)},
	}
}

type SeguridadAcceso struct { // DynamoDB
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
	db.TableStruct[PerfilTable, Perfil]
	EmpresaID           int32   `json:"empresaID" db:"empresa_id,pk" col:"empresa_id,pk"`
	ID                  int32   `json:"id" db:"id,pk" col:"id,pk,sk"`
	Nombre             string  `json:"nombre" db:"nombre" col:"nombre"`
	Descripcion        string  `json:"descripcion" db:"descripcion" col:"descripcion"`
	Modulos            []int16 `json:"modulosIDs" db:"modulos_ids" col:"modulos_ids"`
	Accesos            []int32 `json:"accesos" db:"accesos" col:"accesos"`
	Status             int8    `json:"ss" db:"status" col:"status"`
	Updated            int64   `json:"upd" db:"updated" col:"updated"`
	CompanyUpdatedIndex string  `json:"-" col:"company_updated,index"`
	CompanyStatusIndex  string  `json:"-" col:"company_status_updated,index"`
}

func (e *Perfil) PrepareCloudSync() {
	// Synthetic keys keep profile lookups and delta queries scoped by company across providers.
	e.CompanyUpdatedIndex = fmt.Sprintf("%d_%020d", e.EmpresaID, e.Updated)
	e.CompanyStatusIndex = fmt.Sprintf("%d_%d_%020d", e.EmpresaID, e.Status, e.Updated)
}

type PerfilTable struct {
	db.TableStruct[PerfilTable, Perfil]
	ID          db.Col[PerfilTable, int32]
	EmpresaID   db.Col[PerfilTable, int32]
	Nombre      db.Col[PerfilTable, string]
	Descripcion db.Col[PerfilTable, string]
	Modulos     db.ColSlice[PerfilTable, int16] `db:"modulos_ids"`
	Accesos     db.ColSlice[PerfilTable, int32] `db:"accesos"`
	Status      db.Col[PerfilTable, int8]
	Updated     db.Col[PerfilTable, int64]
}

func (e PerfilTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "seguridad_perfiles",
		Partition:    e.EmpresaID,
		UseSequences: true,
		Keys:         []db.Coln{e.ID.Autoincrement(0)},
	}
}
