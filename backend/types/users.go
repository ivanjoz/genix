package types

type TAGS struct{}

type Empresa struct { // DynamoDB
	TAGS               `table:"empresas"`
	ID                 int32  `json:"id"`
	Nombre             string `json:"nombre"`
	RazonSocial        string `json:"razonSocial"`
	RUC                string `json:"ruc"`
	Email              string `json:"email"`
	Telefono           string `json:"telefono"`
	EmailVerificado    int8   `json:"emailVeridicado,omitempty"`
	TelefonoVerificado int8   `json:"telefonoVerificado,omitempty"`
	Updated            int64  `json:"upd"`
	Status             int8   `json:"ss"`
}

type Usuario struct { // DynamoDB
	TAGS         `table:"usuarios"`
	ID           int32   `json:"id" db:"usuario_id,pk"`
	EmpresaID    int32   `json:"empresaID" db:"empresa_id,pk"`
	Usuario      string  `json:"usuario" db:"usuario"`
	Apellidos    string  `json:"apellidos" db:"apellidos"`
	Nombres      string  `json:"nombres" db:"nombres"`
	Created      int64   `json:"created" db:"created"`
	CreatedBy    int32   `json:"createdBy" db:"created_by"`
	UpdatedBy    int32   `json:"updatedBy" db:"updated_by"`
	PerfilesIDs  []int32 `json:"perfilesIDs" db:"perfiles_ids"`
	RolesIDs     []int32 `json:"rolesIDs" db:"roles_ids"`
	Email        string  `json:"email" db:"email"`
	PasswordHash string  `json:"passwordHash" db:"password_hash"`
	Status       int8    `json:"ss" db:"status"`
	Updated      int64   `json:"upd" db:"updated"`
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
