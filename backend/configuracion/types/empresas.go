package types

import "app/db"

type TAGS struct{}

type Empresa struct {
	db.TableStruct[EmpresaTable, Empresa]
	ID                 int32        `db:"id,pk" col:",sk"`
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
	Updated            int32        `json:"upd" db:"updated" col:",index"`
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
	Updated            db.Col[EmpresaTable, int32]
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
