package types

type TAGS struct{}

type Empresa struct { // DynamoDB
	TAGS               `table:"empresas"`
	ID                 int32      `json:"id"`
	Nombre             string     `json:"nombre"`
	RazonSocial        string     `json:"razonSocial"`
	RUC                string     `json:"ruc"`
	Email              string     `json:"email"`
	NotificacionEmail  string     `json:"notifEmail"`
	Telefono           string     `json:"telefono"`
	Representante      string     `json:"representante"`
	Direccion          string     `json:"direccion"`
	Ciudad             string     `json:"ciudad"`
	FormApiKey         string     `json:"formApiKey"`
	EmailVerificado    int8       `json:"emailVeridicado,omitempty"`
	TelefonoVerificado int8       `json:"telefonoVerificado,omitempty"`
	SmtpConfig         SmtpConfig `json:"smtp"`
	Updated            int64      `json:"upd"`
	Status             int8       `json:"ss"`
}

type SmtpConfig struct {
	Email    string `json:"email,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"pwd,omitempty"`
	Post     int32  `json:"port,omitempty"`
	Host     string `json:"host,omitempty"`
}

type Usuario struct { // DynamoDB + ScyllaDB
	TAGS         `table:"usuarios"`
	ID           int32   `json:"id" db:"id,pk"`
	EmpresaID    int32   `json:"empresaID,omitempty" db:"empresa_id,pk"`
	Usuario      string  `json:"usuario" db:"usuario"`
	Apellidos    string  `json:"apellidos,omitempty" db:"apellidos"`
	Nombres      string  `json:"nombres,omitempty" db:"nombres"`
	Created      int64   `json:"created,omitempty" db:"created"`
	CreatedBy    int32   `json:"createdBy,omitempty" db:"created_by"`
	UpdatedBy    int32   `json:"updatedBy,omitempty" db:"updated_by"`
	PerfilesIDs  []int32 `json:"perfilesIDs,omitempty" db:"perfiles_ids"`
	RolesIDs     []int32 `json:"rolesIDs,omitempty" db:"roles_ids"`
	Email        string  `json:"email,omitempty" db:"email"`
	PasswordHash string  `json:"passwordHash,omitempty"`
	Status       int8    `json:"ss,omitempty" db:"status"`
	Updated      int64   `json:"upd,omitempty" db:"updated"`
	Password     string  `json:"password1,omitempty"`
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
