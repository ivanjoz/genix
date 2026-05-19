package types

import "app/db"

type TAGS struct{}

type Company struct {
	db.TableStruct[CompanyTable, Company]
	ID                 int32        `db:"id,pk" col:",sk"`
	Name               string       `json:",omitempty" col:""`
	LegalName          string       `json:",omitempty" col:""`
	RUC                string       `json:",omitempty" db:"ruc" col:",index"`
	Email              string       `json:",omitempty" db:"email" col:",index"`
	NotificationEmail  string       `json:",omitempty" col:""`
	Phone              string       `json:",omitempty" col:""`
	Representative     string       `json:",omitempty" col:""`
	Address            string       `json:",omitempty" col:""`
	City               string       `json:",omitempty" col:""`
	FormApiKey         string       `json:",omitempty" db:"form_api_key" col:""`
	EmailVerified      int8         `json:",omitempty" col:""`
	PhoneVerified      int8         `json:",omitempty" col:""`
	SmtpConfig         SmtpConfig   `json:",omitempty" db:"smtp_config" col:""`
	CulqiConfig        CulqiConfig  `json:",omitempty" db:"culqui_config" col:""`
	Updated            int32        `json:"upd" db:"updated" col:",index"`
	Status             int8         `json:"ss" db:"status" col:""`
}

type CompanyTable struct {
	db.TableStruct[CompanyTable, Company]
	ID                db.Col[CompanyTable, int32]
	Name              db.Col[CompanyTable, string]
	LegalName         db.Col[CompanyTable, string]
	RUC               db.Col[CompanyTable, string]
	Email             db.Col[CompanyTable, string]
	NotificationEmail db.Col[CompanyTable, string]
	Phone             db.Col[CompanyTable, string]
	Representative    db.Col[CompanyTable, string]
	Address           db.Col[CompanyTable, string]
	City              db.Col[CompanyTable, string]
	FormApiKey        db.Col[CompanyTable, string]
	EmailVerified     db.Col[CompanyTable, int8]
	PhoneVerified     db.Col[CompanyTable, int8]
	SmtpConfig        db.Col[CompanyTable, SmtpConfig]
	CulqiConfig       db.Col[CompanyTable, CulqiConfig]
	Updated           db.Col[CompanyTable, int32]
	Status            db.Col[CompanyTable, int8]
}

func (e CompanyTable) GetSchema() db.TableSchema {
	return db.TableSchema{
		Name:         "companies",
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
	RsaKey    string `json:",omitempty"`
	RsaKeyID  string `json:",omitempty"`
	KeyLive   string `json:",omitempty"`
	PubKeyLive string `json:",omitempty"`
	KeyDev    string `json:",omitempty"`
	PubKeyDev string `json:",omitempty"`
}

type CompanyPub struct {
	ID            int32  `json:"id"`
	Name          string `json:",omitempty"`
	CulqiRsaKey   string `json:",omitempty"`
	CulqiRsaKeyID string `json:",omitempty"`
	CulqiLlave    string `json:",omitempty"`
}
