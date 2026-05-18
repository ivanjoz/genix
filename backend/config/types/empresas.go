package types

import "app/db"

type TAGS struct{}

type Company struct {
	db.TableStruct[CompanyTable, Company]
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
	CulqiConfig       CulqiConfig `json:",omitempty" db:"culqui_config" col:""`
	Updated            int32        `json:"upd" db:"updated" col:",index"`
	Status             int8         `json:"ss" db:"status" col:""`
}

type CompanyTable struct {
	db.TableStruct[CompanyTable, Company]
	ID                 db.Col[CompanyTable, int32]
	Nombre             db.Col[CompanyTable, string]
	RazonSocial        db.Col[CompanyTable, string]
	RUC                db.Col[CompanyTable, string]
	Email              db.Col[CompanyTable, string]
	NotificacionEmail  db.Col[CompanyTable, string]
	Telefono           db.Col[CompanyTable, string]
	Representante      db.Col[CompanyTable, string]
	Direccion          db.Col[CompanyTable, string]
	Ciudad             db.Col[CompanyTable, string]
	FormApiKey         db.Col[CompanyTable, string]
	EmailVerificado    db.Col[CompanyTable, int8]
	TelefonoVerificado db.Col[CompanyTable, int8]
	SmtpConfig         db.Col[CompanyTable, SmtpConfig]
	CulqiConfig       db.Col[CompanyTable, CulqiConfig]
	Updated            db.Col[CompanyTable, int32]
	Status             db.Col[CompanyTable, int8]
}

func (e CompanyTable) GetSchema() db.TableSchema {
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

type CulqiConfig struct {
	RsaKey       string `json:",omitempty"`
	RsaKeyID     string `json:",omitempty"`
	LlaveLive    string `json:",omitempty"`
	LlavePubLive string `json:",omitempty"`
	LlaveDev     string `json:",omitempty"`
	LlavePubDev  string `json:",omitempty"`
}

type CompanyPub struct {
	ID            int32  `json:"id"`
	Nombre        string `json:",omitempty"`
	CulqiRsaKey   string `json:",omitempty"`
	CulqiRsaKeyID string `json:",omitempty"`
	CulqiLlave    string `json:",omitempty"`
}
