package types

import "github.com/ivanjoz/genix-orm/scylla"

type TAGS struct{}

type Company struct {
	scylla.TableStruct[CompanyTable, Company]
	ID                int32       `db:"id,pk" col:",sk"`
	Name              string      `json:",omitempty" col:""`
	LegalName         string      `json:",omitempty" col:""`
	RUC               string      `json:",omitempty" db:"ruc" col:",index"`
	Email             string      `json:",omitempty" db:"email" col:",index"`
	NotificationEmail string      `json:",omitempty" col:""`
	Phone             string      `json:",omitempty" col:""`
	Representative    string      `json:",omitempty" col:""`
	Address           string      `json:",omitempty" col:""`
	City              string      `json:",omitempty" col:""`
	FormApiKey        string      `json:",omitempty" db:"form_api_key" col:""`
	EmailVerified     int8        `json:",omitempty" col:""`
	PhoneVerified     int8        `json:",omitempty" col:""`
	SmtpConfig        SmtpConfig  `json:",omitempty" db:"smtp_config" col:""`
	CulqiConfig       CulqiConfig `json:",omitempty" db:"culqui_config" col:""`
	Updated           int32       `json:"upd" db:"updated" col:",index"`
	Status            int8        `json:"ss" db:"status" col:""`
}

type CompanyTable struct {
	scylla.TableStruct[CompanyTable, Company]
	ID                scylla.Col[CompanyTable, int32]
	Name              scylla.Col[CompanyTable, string]
	LegalName         scylla.Col[CompanyTable, string]
	RUC               scylla.Col[CompanyTable, string]
	Email             scylla.Col[CompanyTable, string]
	NotificationEmail scylla.Col[CompanyTable, string]
	Phone             scylla.Col[CompanyTable, string]
	Representative    scylla.Col[CompanyTable, string]
	Address           scylla.Col[CompanyTable, string]
	City              scylla.Col[CompanyTable, string]
	FormApiKey        scylla.Col[CompanyTable, string]
	EmailVerified     scylla.Col[CompanyTable, int8]
	PhoneVerified     scylla.Col[CompanyTable, int8]
	SmtpConfig        scylla.Col[CompanyTable, SmtpConfig]
	CulqiConfig       scylla.Col[CompanyTable, CulqiConfig]
	Updated           scylla.Col[CompanyTable, int32]
	Status            scylla.Col[CompanyTable, int8]
}

func (e CompanyTable) GetSchema() scylla.TableSchema {
	return scylla.TableSchema{
		Name:         "companies",
		UseSequences: true,
		Keys:         scylla.Cols(e.ID.Autoincrement(0)),
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
