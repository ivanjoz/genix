# PLAN — Registro público de empresas (3 pasos)

Estado: **propuesta, pendiente de aprobación**. No se ha escrito código todavía.

Objetivo: al pulsar "Regístrese" en `/welcome` se abre un modal grande de 3 pasos —
**Email Verification → Company Information → Initial Data**— con una columna de imagen a
la izquierda (la misma foto del hero) y el contenido del paso a la derecha.

---

## 1. Evaluación del diseño propuesto

Lo que propusiste **funciona y encaja con el proyecto**. Resumen de la revisión:

| Tu propuesta | Veredicto |
| --- | --- |
| Tabla `sign_up_requests` con ID, email, código y `updated` | Correcto. |
| 6 dígitos aleatorios al final del ID para que no se pueda adivinar por ID | Correcto, y el ORM ya lo hace nativo: `Autoincrement(6)` genera `secuencia * 10^6 + 6 dígitos aleatorios` (`scylla/insert-update.go:359`). |
| No hay tenant, así que la partición es la semana `[year][week]` | Correcto. Ya existe el helper: `core.MakeSemanaFromFechaUnix(0, true).Code` devuelve un `int16` = `year*100 + week - 200000` (2026-W32 → `2632`), en `backend/core/time-helpers.go:32`. |
| Revisar siempre la última solicitud del email; si expiró se avisa, si sigue viva se obliga a usarla | Correcto, y además es el único freno anti-spam que vamos a tener (ver §5). |
| Link tipo `/welcome?req=…&code=…` | Correcto. |

### Cambios que sí propongo

1. **El ID debe llevar la semana dentro.** Scylla exige la partición en el `WHERE`, así que
   con solo `req=12` no se puede encontrar la fila. Propongo construir el ID como
   `WeekCode * 10^12 + secuencia * 10^6 + aleatorio6`, y que el handler derive
   `WeekCode = ID / 10^12`. Un solo número en la URL y una sola columna que resolver.
   *(Descarté `KeyIntPacking` del ORM: reparte 19 dígitos y produce IDs de 19 cifras —feos en
   una URL— y de todos modos habría que decodificar la semana a mano para el `WHERE`.)*

2. **Añadir `Status` al registro (máquina de estados), no solo `updated`.**
   `1` = email enviado · `2` = email verificado · `3` = empresa creada · `0` = anulado.
   Sin esto, el paso 2 no puede saber si el email ya se verificó, y quien reabra el link no
   puede retomar donde lo dejó.

3. **Añadir `CompanyID` y `UserID`.** Se rellenan en el paso 2. Sirven para reanudar
   (si el navegador se cierra entre el paso 2 y el 3) y para auditoría.

4. **Añadir `Attempts`.** Contador de intentos fallidos del código; a los 5 el registro pasa a
   `Status = 0`. El código de 8 dígitos ya es difícil de adivinar, pero el endpoint es público
   y sin rate limiter (§5), así que el contador es la red de seguridad barata.

5. **Auto-login al terminar el paso 2.** En cuanto se crean empresa + usuario admin, el
   endpoint devuelve exactamente el mismo payload que `POST p-user-login`
   (`security.MakeUsuarioResponse`). El frontend hace `security.parseLogin(...)` y el **paso 3
   ya corre autenticado**: reutiliza el `POST initial-data` que ya existe, sin endpoint público
   nuevo y sin duplicar la lógica de bootstrap.

---

## 2. Backend — tabla nueva

Archivo nuevo: `backend/security/types/signup_requests.go`

```go
type SignUpRequest struct {
    db.TableStruct[SignUpRequestTable, SignUpRequest]
    // Partición: [year][week] - 200000, el mismo código que devuelve core.MakeSemanaFromFechaUnix.
    // Mantiene acotada una tabla que no tiene tenant y hace trivial purgar semanas viejas.
    WeekCode int32 `json:",omitempty"`
    // WeekCode*1e12 + secuencia*1e6 + 6 dígitos aleatorios. Los aleatorios evitan que se
    // enumeren solicitudes ajenas; la semana embebida es lo que permite resolver la partición
    // teniendo únicamente el "req" de la URL.
    ID        int64  `json:",omitempty"`
    Email     string `json:",omitempty"`
    Code      string `json:",omitempty"` // 8 dígitos
    Attempts  int8   `json:",omitempty"`
    CompanyID int32  `json:",omitempty"`
    UserID    int32  `json:",omitempty"`
    Created   int32  `json:",omitempty"`
    Updated   int32  `json:"upd,omitempty"`
    // 1 enviado · 2 verificado · 3 empresa creada · 0 anulado/agotado
    Status int8 `json:"ss,omitempty"`
}
```

`GetSchema()`:

```go
db.TableSchema{
    ID:        43,            // 1..42 ya están tomados; 43 es el siguiente libre
    Name:      "sign_up_requests",
    Partition: e.WeekCode,
    Keys:      db.Cols(e.ID),
    // Búsqueda "última solicitud de este email" dentro de la partición de la semana.
    Indexes: []db.Index{
        {Type: db.TypeLocalIndex, Keys: db.Cols(e.Email)},
    },
}
```

Sin `UseSequences`: la secuencia se pide a mano con `db.GetAutoincrementID("signup_<weekCode>", 1)`
para poder componer el ID con la semana delante.

Fechas en `core.SUnixTime()` (int32, **1 unidad = 2 segundos**) como manda AGENTS.md.
La expiración se compara en esas unidades, no en segundos.

Al terminar, correr `static-project-validation` (`cd scripts && go run . check_tables`).

---

## 3. Backend — endpoints

Los tres son públicos (prefijo `p-`, sin token) y viven en `backend/security/signup.go`,
registrados en `backend/security/main.go`.

### `POST p-signup-request`
Body `{ Email }`.
1. Valida el formato del email.
2. Busca la última solicitud del email en la semana actual **y en la anterior** (dos particiones;
   cubre el corte de semana).
3. Si hay una con `Status` 1 o 2 y no expirada → **no** envía otro correo; responde
   `{ RequestID, Pending: true, CreatedAt }` y el frontend muestra
   *"Ya se envió una solicitud el [fecha]. Use ese correo o espere a que expire."*
4. Si no hay ninguna viva → genera código de 8 dígitos, inserta la fila y envía el correo.
5. Responde `{ RequestID }`.

### `POST p-signup-verify`
Body `{ RequestID, Code }`.
Resuelve la partición desde `RequestID / 1e12`, valida expiración, compara el código
(incrementa `Attempts` y anula a los 5 fallos), pone `Status = 2` y responde `{ Email }`.

### `POST p-signup-company`
Body `{ RequestID, Code, CompanyName, Address, RUC, AdminUser, AdminPassword, CipherKey }`.
1. Revalida `(RequestID, Code)` y exige `Status = 2`.
2. Valida: `CompanyName` ≥ 5, `AdminUser` ≥ 4, `AdminPassword` ≥ 6. `Address` y `RUC` opcionales.
3. Inserta `types.Company` (ID autoincremental global) con `Email` = el del registro y
   `EmailVerified = 1`.
4. Inserta `coretypes.User` en esa empresa. La secuencia del ORM está particionada por
   `CompanyID` (`insert-update.go:286`), así que el primer usuario de una empresa nueva
   **obtiene ID = 1**, que es lo que `MakeUsuarioResponse` reconoce como admin con todos los
   accesos. Hash de password igual que `PostUsuarios`:
   `core.FnvHashString64(core.Env.SECRET_PHRASE + password, -1, 20)`.
5. Marca la solicitud `Status = 3` con `CompanyID`/`UserID`.
6. Responde el payload de `security.MakeUsuarioResponse(user, CipherKey)` → auto-login.

### Paso 3
Sin endpoint nuevo: el frontend, ya autenticado, llama al `POST initial-data` existente.

---

## 4. Backend — envío de correo

`backend/core/mailer.go` nuevo: `SendEmail(to, subject, htmlBody string) error`, que saca de
`exec/demo.go:427` la plomería de `go-simple-mail` (ya es dependencia) y lee la config SMTP
de `core.Env.SMTP_*` (ya cargada desde `config.toml`). `demo.go:Test24` pasa a usarlo.

El correo lleva el código de 8 dígitos **y** el link
`<app_url>/welcome?req=<ID>&code=<code>`.

`app_url` es el único dato que falta: hay que añadirlo a `[frontend]` en `config.toml` y a
`core.Env`. Ver pregunta abierta B.

---

## 5. Nota de seguridad (importante)

`backend/main-handlers.go:194` y `:213` saltan el rate limiter cuando la ruta es pública, y
`:126` salta la validación de usuario. Es decir: **estos tres endpoints no tienen throttle de
plataforma**. Los frenos quedan dentro de los handlers:

- una sola solicitud viva por email (bloquea el spam de correos);
- máximo 5 intentos de código por solicitud;
- crear empresa exige un código verificado, así que no se puede crear empresas en masa sin
  controlar un buzón.

Es suficiente para pre-alpha. Si más adelante quieres un límite por IP, el sitio natural es
`main-handlers.go` con una excepción para rutas `p-signup-*`.

---

## 6. Frontend

### `frontend/routes/initial-data/InitialDataForm.svelte` (nuevo)
Extrae tal cual el formulario del `+page.svelte` actual (servicios, `$effect` de sedes,
validaciones, `saveInitialData`). Props: `onSaved?: () => void`.
`routes/initial-data/+page.svelte` queda como envoltorio: la tarjeta y el logo que ya tiene,
más `<InitialDataForm onSaved={() => Env.navigate('/')} />`.

### `frontend/routes/welcome/RegistrationModal.svelte` (reescrito)
`<Modal size={9}>` (1080px, el máximo disponible) con `bodyCss="!p-0"` y dentro una rejilla
de dos columnas: a la izquierda `welcome-hero-v3.webp` a sangre con un degradado y el
indicador de pasos encima; a la derecha el paso activo. En móvil la imagen se oculta.

Pasos:
1. **Email Verification** — email → `p-signup-request`; luego campo de código de 8 dígitos →
   `p-signup-verify`. Si vuelve `Pending: true`, se muestra el aviso y se salta directo al
   campo del código.
2. **Company Information** — `CompanyName` (req.), `Address` (opc.), `RUC` (opc.),
   `AdminUser` (req.), `AdminPassword` (req.), `AdminPasswordRepeat` (req., debe coincidir) →
   `p-signup-company` → `security.parseLogin(...)`.
3. **Initial Data** — `<InitialDataForm onSaved={() => Env.navigate('/')} />`.

### `frontend/services/signup.ts` (nuevo)
Las tres llamadas `POST` con sus interfaces.

### `frontend/routes/welcome/+page.svelte`
En `onMount`, si la URL trae `?req=&code=`, abre el modal en el paso 1 con el código puesto y
lanza la verificación automáticamente.

---

## 7. Archivos tocados

**Nuevos**
- `backend/security/types/signup_requests.go`
- `backend/security/signup.go`
- `backend/core/mailer.go`
- `frontend/services/signup.ts`
- `frontend/routes/initial-data/InitialDataForm.svelte`

**Modificados**
- `backend/security/main.go` — registrar las 3 rutas
- `backend/core/security.go` — `Env.APP_URL` + lectura de `[frontend] app_url`
- `backend/exec/demo.go` — `Test24` pasa a usar `core.SendEmail`
- `config.toml` (+ el ejemplo versionado) — `app_url`
- `frontend/routes/welcome/RegistrationModal.svelte` — reescrito
- `frontend/routes/welcome/+page.svelte` — leer `?req=&code=`
- `frontend/routes/initial-data/+page.svelte` — usar el componente extraído

---

## 8. Decisiones tomadas

**A. Nombre del usuario admin: libre, mínimo 4 caracteres.** Consecuencia: hay que quitar el
`if body.ID == 1 { body.User = "admin" }` de `backend/security/usuarios.go:170`. Si se deja,
editar el usuario 1 desde la página de usuarios le renombraría a `"admin"` y dejaría al dueño
sin poder entrar con el nombre que eligió al registrarse.

**B. URL base del correo: `app_url` en `config.toml`.** Nueva clave en `[frontend]`, expuesta
como `core.Env.APP_URL`. El dominio lo decide el servidor, no el cliente.

**C. Expiración: 2 horas.** En unidades `SUnixTime` (1 unidad = 2 s) son `3600`.

**D. Un solo registro por email.** Requiere poder buscar empresas por email. La tabla
`companies` **no tiene partición**, y `registerSchemaLocalIndex` llama a `GetPartKey().GetName()`
(`scylla/index_config.go:46`), que reventaría con nil. Hay que usar
`{Type: db.TypeGlobalIndex, Keys: db.Cols(e.Email)}`, añadido **al final** de `Indexes` para no
mover el slot `ix1` que ya usa el espejo cloud.
