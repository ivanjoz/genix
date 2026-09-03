package core

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"strings"
	"sync"

	"github.com/ivanjoz/colbin"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/crypto/blake2s"
)

type UsuarioToken struct {
	CompanyID int32  `json:"c"`
	ID        int32  `json:"i"`
	Created   int32  `json:"e"`
	Hash      []byte `json:"h"`
	User      string `json:"u"`
	Error     string `json:"-" cb:"-"` // transient; never serialized into the token
	// SubAccesos is what the gate learned from fareward about THIS request: the sub-access mask of
	// each access that authorized the route being served, keyed by access id.
	//
	// `cb:"-"` like Error, and for a sharper reason: the token is the session credential and lives
	// in the browser, so anything serialized into it is client-supplied on the way back. These
	// grants are re-answered by the daemon on every request precisely so they cannot be.
	//
	// Only the route's own accesses are here. A handler cannot ask about an access that did not
	// gate it — the frame never asked, so the daemon never answered.
	SubAccesos map[int32]uint16 `json:"-" cb:"-"`
}

// HasSubAcceso reports whether this request's caller holds one sub-access of one of the accesses
// that authorized the route. Handlers call this; nothing else should read SubAccesos directly.
//
// False when the access did not gate this route, which is the safe reading: a handler asking about
// an access it is not mapped to is a mapping bug, and answering true would grant on the strength of
// a question nobody verified.
func (usuarioToken *UsuarioToken) HasSubAcceso(accesoID int32, subAccesoID int32) bool {
	subMask, accesoAuthorizedThisRoute := usuarioToken.SubAccesos[accesoID]
	return accesoAuthorizedThisRoute && HasSubAcceso(subMask, subAccesoID)
}

var User UsuarioToken

type AccessInfo struct {
	ID   int32
	Name string
	// SubAccesosMask are the sub-accesses this access declares, as the same bitmask the grant
	// blob stores: bit (subID-1). Carried here so a validator can reject a profile naming a sub
	// id the access never declared, without re-parsing the catalog.
	SubAccesosMask uint16
}

// MaxSubAccesoID is the highest sub-access id an access may declare. The blob spends 7 bits per
// sub byte and allows a second byte, so 14 fit; 13 is the declared ceiling, leaving one spare.
// Id 1 is reserved for "Todos" and is never declared in the catalog — SubAccesoTodosID below.
const MaxSubAccesoID = 13

// SubAccesoTodosID is the implicit "every sub-access of this access" grant. It is never declared
// in access.toml and never offered as a checkbox of its own beyond the synthetic one: holding it
// satisfies every sub-access check on that access. Expanded in Go and TS only — the daemon
// returns the raw mask and knows nothing about what bit 0 means.
const SubAccesoTodosID = 1

type AccessCatalog struct {
	Access []struct {
		ID             int32    `toml:"id"`
		Name           string   `toml:"name"`
		Group          int32    `toml:"group"`
		Levels         int32    `toml:"levels"`
		FrontendRoutes string   `toml:"frontend_routes"`
		BackendAPIs    string   `toml:"backend_apis"`
		SubAccesosIDs  []int32  `toml:"sub_accesses_ids"`
		SubAccesosName []string `toml:"sub_accesses_names"`
	} `toml:"access"`
}

type AccessHelper struct {
	loadOnce              sync.Once
	routeAccessMap        map[string][]AccessInfo
	accessInfoByID        map[int32]AccessInfo
	allAccessIDs          []int32
	embeddedAccessCatalog []byte
	loadErr               error
}

var embeddedAccessHelper = &AccessHelper{}

func (accessHelper *AccessHelper) Load(accessCatalogContent []byte) {
	// Build the access map eagerly during startup so request-time checks only perform lookups.
	accessHelper.embeddedAccessCatalog = accessCatalogContent
	accessHelper.loadOnce.Do(func() {
		accessHelper.routeAccessMap = make(map[string][]AccessInfo)

		if len(accessHelper.embeddedAccessCatalog) == 0 {
			accessHelper.loadErr = Err("No se encontró el contenido embebido de access.toml.")
			Log("AccessHelper:: missing embedded access catalog content")
			return
		}

		parsedAccessList := AccessCatalog{}
		if err := toml.Unmarshal(accessHelper.embeddedAccessCatalog, &parsedAccessList); err != nil {
			accessHelper.loadErr = Err("No se pudo interpretar access.toml embebido:", err)
			Log("AccessHelper:: toml unmarshal error", err)
			return
		}

		// Precompute the backend route map once so each request only performs constant-time lookups.
		accessHelper.accessInfoByID = make(map[int32]AccessInfo, len(parsedAccessList.Access))
		for _, accessListEntry := range parsedAccessList.Access {
			subAccesosMask, subErr := makeSubAccesosMask(
				accessListEntry.ID, accessListEntry.SubAccesosIDs, accessListEntry.SubAccesosName)
			if subErr != nil {
				accessHelper.loadErr = subErr
				Log("AccessHelper:: sub-access declaration rejected", subErr)
				return
			}

			accessInfo := AccessInfo{
				ID:             accessListEntry.ID,
				Name:           accessListEntry.Name,
				SubAccesosMask: subAccesosMask,
			}
			accessHelper.allAccessIDs = append(accessHelper.allAccessIDs, accessListEntry.ID)
			accessHelper.accessInfoByID[accessListEntry.ID] = accessInfo

			for _, backendRoute := range strings.Split(accessListEntry.BackendAPIs, ",") {
				trimmedBackendRoute := strings.TrimSpace(backendRoute)
				if trimmedBackendRoute == "" {
					continue
				}

				accessHelper.routeAccessMap[trimmedBackendRoute] = append(
					accessHelper.routeAccessMap[trimmedBackendRoute], accessInfo)
			}
		}

		Log("AccessHelper:: access list loaded", "routes", len(accessHelper.routeAccessMap))
	})
}

// makeSubAccesosMask folds an access's parallel sub-access arrays into the bitmask the grant blob
// stores. It refuses rather than repairs: the catalog is embedded at build time, so a malformed
// declaration is a build-time mistake and must surface as one, not as an access that silently
// offers fewer sub-accesses than its author wrote.
//
// Parallel arrays are the format's one hazard — two lists that drift out of step mis-name every
// sub-access after the gap — so the length check comes first and names the access id.
func makeSubAccesosMask(accesoID int32, subAccesoIDs []int32, subAccesoNames []string) (uint16, error) {
	if len(subAccesoIDs) != len(subAccesoNames) {
		return 0, Err("access.toml: el acceso", accesoID, "declara", len(subAccesoIDs),
			"sub_accesses_ids y", len(subAccesoNames), "sub_accesses_names; deben ser iguales.")
	}

	subAccesosMask := uint16(0)
	for index, subAccesoID := range subAccesoIDs {
		if subAccesoID <= SubAccesoTodosID || subAccesoID > MaxSubAccesoID {
			return 0, Err("access.toml: el acceso", accesoID, "declara el sub-acceso", subAccesoID,
				"fuera del rango permitido (2 a", MaxSubAccesoID, "; el 1 está reservado para \"Todos\").")
		}
		subAccesoBit := uint16(1) << uint16(subAccesoID-1)
		if subAccesosMask&subAccesoBit != 0 {
			return 0, Err("access.toml: el acceso", accesoID, "declara el sub-acceso",
				subAccesoID, "más de una vez.")
		}
		if strings.TrimSpace(subAccesoNames[index]) == "" {
			return 0, Err("access.toml: el acceso", accesoID, "declara el sub-acceso",
				subAccesoID, "sin nombre.")
		}
		subAccesosMask |= subAccesoBit
	}

	return subAccesosMask, nil
}

// GetAccesoInfo resolves one access by id. Used by the profile validator, which has to know which
// sub-accesses an access actually declares before it will store a grant for one.
func (accessHelper *AccessHelper) GetAccesoInfo(accesoID int32) (AccessInfo, bool) {
	if accessHelper.loadErr != nil {
		Log("AccessHelper:: lookup skipped due to load error", accessHelper.loadErr)
		return AccessInfo{}, false
	}

	accessInfo, accessFound := accessHelper.accessInfoByID[accesoID]
	return accessInfo, accessFound
}

func (accessHelper *AccessHelper) GetAccesosByRoute(route string) ([]AccessInfo, bool) {
	if accessHelper.routeAccessMap == nil {
		panic("AccessHelper:: Load must be called before GetAccesosByRoute")
	}

	if accessHelper.loadErr != nil {
		Log("AccessHelper:: lookup skipped due to load error", accessHelper.loadErr)
		return nil, false
	}

	accessInfos, accessFound := accessHelper.routeAccessMap[route]
	return accessInfos, accessFound
}

func (accessHelper *AccessHelper) GetAllAccesosIDs() ([]int32, error) {
	if accessHelper.routeAccessMap == nil && accessHelper.loadErr == nil {
		return nil, Err("AccessHelper:: Load debe ejecutarse antes de GetAllAccesosIDs.")
	}

	if accessHelper.loadErr != nil {
		return nil, accessHelper.loadErr
	}

	// Return a copy so callers cannot mutate the cached source of truth.
	return append([]int32{}, accessHelper.allAccessIDs...), nil
}

func LoadEmbeddedAccessList(accessCatalogContent []byte) *AccessHelper {
	embeddedAccessHelper.Load(accessCatalogContent)
	return embeddedAccessHelper
}

func GetEmbeddedAccessHelper() *AccessHelper {
	return embeddedAccessHelper
}

func GetAllEmbeddedAccesosIDs() ([]int32, error) {
	return embeddedAccessHelper.GetAllAccesosIDs()
}

// usuarioTokenDomain separates this tag from every other keyed hash the project computes with
// SECRET_PHRASE. `:v3` is keyed BLAKE2s-128; `:v1` was truncated HMAC-SHA256 and `:v2` SipHash-2-4,
// both 64-bit. A bump invalidates every token issued under the old one, so sessions log in again.
const usuarioTokenDomain = "usrToken:v3"

// ComputeUsuarioTokenHash keys BLAKE2s-128 with SECRET_PHRASE and returns the token's 16-byte tag.
//
// 128 bits, not the 64 a truncated hash used to leave here: the token carries no random component
// — company, user, created and username are all guessable — so this tag is not integrity
// protection over a secret, it *is* the credential, and its width is the session's whole strength.
//
// Keyed BLAKE2s-128 rather than HMAC: RFC 7693 defines the keyed mode as a MAC, x/crypto refuses
// to construct New128 without a key precisely because a 128-bit digest is only safe as one, and it
// is a purpose-built 128-bit tag rather than a 256-bit digest truncated down to fit.
//
// The key is SHA-256 of the phrase, so a configuration string of any length reaches the key in
// full and lands on exactly the 32 bytes BLAKE2s takes at most. Mirrored byte for byte by
// compute_user_token_hash in fareward/src/bridge/auth.rs, which is how the SSE bridge authenticates
// a browser without asking this backend: a change on either side rejects every client.
func ComputeUsuarioTokenHash(usuarioToken UsuarioToken) []byte {
	tokenKey := sha256.Sum256([]byte(Env.SECRET_PHRASE))
	hasher, err := blake2s.New128(tokenKey[:])
	if err != nil {
		// Unreachable: the key is a SHA-256 digest, always 32 bytes, always within BLAKE2s' limit.
		Log("ComputeUsuarioTokenHash:: blake2s rechazó la clave derivada", err)
		return nil
	}

	tokenPayloadBuffer := make([]byte, 12)
	binary.BigEndian.PutUint32(tokenPayloadBuffer[0:4], uint32(usuarioToken.CompanyID))
	binary.BigEndian.PutUint32(tokenPayloadBuffer[4:8], uint32(usuarioToken.ID))
	binary.BigEndian.PutUint32(tokenPayloadBuffer[8:12], uint32(usuarioToken.Created))

	hasher.Write([]byte(usuarioTokenDomain))
	hasher.Write(tokenPayloadBuffer)
	hasher.Write([]byte(usuarioToken.User))
	return hasher.Sum(nil)
}

func CheckUser(req *HandlerArgs, access int) *UsuarioToken {
	userToken := req.Headers["authorization"]
	if len(userToken) < 8 {
		userToken = req.Headers["Authorization"]
	}

	user := UsuarioToken{}

	if len(userToken) < 8 {
		user.Error = "No se suministró un Token de user"
		return &user
	}

	tokenBase64 := strings.TrimSpace(strings.TrimPrefix(userToken, "Bearer "))
	if len(tokenBase64) < 8 {
		user.Error = "No se encontró la información del user."
		return &user
	}

	tokenBytes := Base64ToBytes(MakeB64UrlDecode(tokenBase64))
	if len(tokenBytes) < 8 {
		user.Error = "El token de inicio de sesión es inválido."
		return &user
	}

	if err := colbin.Unmarshal(tokenBytes, &user); err != nil {
		Log("CheckUser:: error decodificando token", err)
		user.Error = "Error al recuperar la información del user."
	}

	if user.Error == "" {
		expectedTokenHash := ComputeUsuarioTokenHash(user)
		// Constant-time: a byte-by-byte early exit would leak the expected tag to a caller able to
		// time many attempts, which is the whole credential.
		if subtle.ConstantTimeCompare(user.Hash, expectedTokenHash) != 1 {
			Log("CheckUser:: hash inválido", "companyID", user.CompanyID, "userID", user.ID)
			user.Error = "El token de inicio de sesión no es válido."
		}
	}

	// Los accesos ya no se cargan aquí. El gate los pide a fareward dentro del mismo frame que
	// cobra la request, que es lo que le quita a Lambda una lectura a ScyllaDB en el camino de
	// autorización: un entorno de ejecución nuevo empieza con la caché vacía y pagaba ese viaje
	// antes de que el handler hiciera nada.
	// NOTE: In local/VPS HTTP mode requests are concurrent; avoid mutating global user state.
	if Env.IS_SERVERLESS {
		User = user
		Env.USUARIO_ID = user.ID
	}

	return &user
}
