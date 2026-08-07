package types

import "app/db"

type TAGS struct{}

type Company struct {
	db.TableStruct[CompanyTable, Company]
	ID                int32       `db:"id,pk"`
	Name              string      `json:",omitempty"`
	LegalName         string      `json:",omitempty"`
	RUC               string      `json:",omitempty" db:"ruc"`
	Email             string      `json:",omitempty" db:"email"`
	NotificationEmail string      `json:",omitempty"`
	Phone             string      `json:",omitempty"`
	Representative    string      `json:",omitempty"`
	Address           string      `json:",omitempty"`
	City              string      `json:",omitempty"`
	FormApiKey        string      `json:",omitempty" db:"form_api_key"`
	EmailVerified     int8        `json:",omitempty"`
	PhoneVerified     int8        `json:",omitempty"`
	SmtpConfig        SmtpConfig  `json:",omitempty" db:"smtp_config"`
	CulqiConfig       CulqiConfig `json:",omitempty" db:"culqui_config"`
	Updated           int32       `json:"upd" db:"updated"`
	Status            int8        `json:"ss" db:"status"`
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
		ID:           9,
		Name:         "companies",
		UseSequences: true,
		Keys:         db.Cols(e.ID.Autoincrement(0)),
		// Companies are global, so this index carries no tenant prefix. The delta read is
		// the only lookup any caller makes; RUC and Email are resolved by ID, never scanned.
		Indexes: []db.Index{
			{Type: db.TypeView, Keys: db.Cols(e.Updated)},
		},
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
	RsaKey     string `json:",omitempty"`
	RsaKeyID   string `json:",omitempty"`
	KeyLive    string `json:",omitempty"`
	PubKeyLive string `json:",omitempty"`
	KeyDev     string `json:",omitempty"`
	PubKeyDev  string `json:",omitempty"`
}

type CompanyPub struct {
	ID            int32  `json:"id"`
	Name          string `json:",omitempty"`
	CulqiRsaKey   string `json:",omitempty"`
	CulqiRsaKeyID string `json:",omitempty"`
	CulqiLlave    string `json:",omitempty"`
}
