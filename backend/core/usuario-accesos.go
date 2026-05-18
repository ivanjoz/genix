package core

import (
	"app/libs/cbor"
	coretypes "app/core/types"
	"app/db"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type UsuarioToken struct {
	CompanyID int32  `json:"c" cbor:"1,keyasint"`
	ID        int32  `json:"i" cbor:"2,keyasint"`
	Created   int32  `json:"e" cbor:"3,keyasint"`
	Hash      uint64 `json:"h" cbor:"4,keyasint"`
	User   string `json:"u" cbor:"5,keyasint"`
	Error     string `json:"-" cbor:"-"`
}

var User UsuarioToken

type UsuarioAccesos struct {
	updated              int32 // SUnixTime()
	accesosNivelComputed []uint16
}

type AccessInfo struct {
	ID   int32
	Name string
}

type AccessListYaml struct {
	AccessList []struct {
		ID             int32  `yaml:"id"`
		Name           string `yaml:"name"`
		Group          int32  `yaml:"group"`
		Levels         int32  `yaml:"levels"`
		FrontendRoutes string `yaml:"frontend_routes"`
		BackendAPIs    string `yaml:"backend_apis"`
	} `yaml:"access_list"`
}

type AccessHelper struct {
	loadOnce              sync.Once
	routeAccessMap        map[string][]AccessInfo
	allAccessIDs          []int32
	embeddedAccessListYml []byte
	loadErr               error
}

var companyUsuarioAccesos = map[uint64]UsuarioAccesos{}
var companyUsuarioAccesosMu sync.RWMutex
var embeddedAccessHelper = &AccessHelper{}

const usuarioAccesosCacheTTL int32 = 150 // 5 minutes with SUnixTime 2-second ticks.

func (accessHelper *AccessHelper) Load(accessListYamlContent []byte) {
	// Build the access map eagerly during startup so request-time checks only perform lookups.
	accessHelper.embeddedAccessListYml = accessListYamlContent
	accessHelper.loadOnce.Do(func() {
		accessHelper.routeAccessMap = make(map[string][]AccessInfo)

		if len(accessHelper.embeddedAccessListYml) == 0 {
			accessHelper.loadErr = Err("No se encontró el contenido embebido de access_list.yml.")
			Log("AccessHelper:: missing embedded access list content")
			return
		}

		parsedAccessList := AccessListYaml{}
		if err := yaml.Unmarshal(accessHelper.embeddedAccessListYml, &parsedAccessList); err != nil {
			accessHelper.loadErr = Err("No se pudo interpretar access_list.yml embebido:", err)
			Log("AccessHelper:: yaml unmarshal error", err)
			return
		}

		// Precompute the backend route map once so each request only performs constant-time lookups.
		for _, accessListEntry := range parsedAccessList.AccessList {
			accessHelper.allAccessIDs = append(accessHelper.allAccessIDs, accessListEntry.ID)
			for _, backendRoute := range strings.Split(accessListEntry.BackendAPIs, ",") {
				trimmedBackendRoute := strings.TrimSpace(backendRoute)
				if trimmedBackendRoute == "" {
					continue
				}

				accessHelper.routeAccessMap[trimmedBackendRoute] = append(accessHelper.routeAccessMap[trimmedBackendRoute], AccessInfo{
					ID:   accessListEntry.ID,
					Name: accessListEntry.Name,
				})
			}
		}

		Log("AccessHelper:: access list loaded", "routes", len(accessHelper.routeAccessMap))
	})
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

func LoadEmbeddedAccessList(accessListYamlContent []byte) *AccessHelper {
	embeddedAccessHelper.Load(accessListYamlContent)
	return embeddedAccessHelper
}

func GetEmbeddedAccessHelper() *AccessHelper {
	return embeddedAccessHelper
}

func GetAllEmbeddedAccesosIDs() ([]int32, error) {
	return embeddedAccessHelper.GetAllAccesosIDs()
}

func ComputeUsuarioTokenHash(usuarioToken UsuarioToken) uint64 {
	// Bind the token payload to the server secret so the uint64 hash can be recomputed during auth.
	hashMac := hmac.New(sha256.New, []byte(Env.SECRET_PHRASE))
	tokenPayloadBuffer := make([]byte, 12)
	binary.BigEndian.PutUint32(tokenPayloadBuffer[0:4], uint32(usuarioToken.CompanyID))
	binary.BigEndian.PutUint32(tokenPayloadBuffer[4:8], uint32(usuarioToken.ID))
	binary.BigEndian.PutUint32(tokenPayloadBuffer[8:12], uint32(usuarioToken.Created))
	hashMac.Write([]byte("usrToken:v1"))
	hashMac.Write(tokenPayloadBuffer)
	hashMac.Write([]byte(usuarioToken.User))
	return binary.BigEndian.Uint64(hashMac.Sum(nil)[:8])
}

func makeCompanyUsuarioAccesosKey(companyID, userID int32) uint64 {
	return uint64(uint32(companyID))<<32 | uint64(uint32(userID))
}

func formatAccesosNivelForLog(accesosNivel []uint16) string {
	// Decode packed acceso+nivel values into a readable list for auth debugging.
	if len(accesosNivel) == 0 {
		return ""
	}

	formattedAccesos := make([]string, 0, len(accesosNivel))
	for _, packedAccesoNivel := range accesosNivel {
		accesoID := int32(packedAccesoNivel >> 2)
		nivel := uint8(packedAccesoNivel&0b11) + 1
		formattedAccesos = append(formattedAccesos, fmt.Sprintf("%d:%d", accesoID, nivel))
	}

	return strings.Join(formattedAccesos, ",")
}

func loadUsuarioAccesosComputed(companyID, userID int32) ([]uint16, error) {
	nowTime := SUnixTime()
	cacheKey := makeCompanyUsuarioAccesosKey(companyID, userID)

	companyUsuarioAccesosMu.RLock()
	cachedUsuarioAccesos, cacheFound := companyUsuarioAccesos[cacheKey]
	companyUsuarioAccesosMu.RUnlock()

	cacheIsFresh := nowTime >= cachedUsuarioAccesos.updated && (nowTime-cachedUsuarioAccesos.updated) <= usuarioAccesosCacheTTL

	if !cacheFound {
		Log("CheckUser:: cache miss", "companyID", companyID, "userID", userID)
	} else if !cacheIsFresh {
		Log("CheckUser:: cache stale", "companyID", companyID, "userID", userID, "cacheAge", nowTime-cachedUsuarioAccesos.updated)
	}

	if !cacheIsFresh {
		usuarios := []coretypes.User{}
		usuarioQuery := db.Query(&usuarios)
		usuarioQuery.Select(usuarioQuery.CompanyID, usuarioQuery.ID, usuarioQuery.AccesosComputed).
			CompanyID.Equals(companyID).ID.Equals(userID).Limit(1)

		Log("CheckUser:: querying user accesos", "companyID", companyID, "userID", userID)
		if err := usuarioQuery.Exec(); err != nil {
			return nil, Err("Error al obtener los accesos computados del user en ScyllaDB:", err)
		}
		if len(usuarios) == 0 {
			return nil, Err(fmt.Sprintf("No se encontró el user %d de la company %d en ScyllaDB.", userID, companyID))
		}
		slices.Sort(usuarios[0].AccesosComputed)

		companyUsuarioAccesosMu.Lock()
		companyUsuarioAccesos[cacheKey] = UsuarioAccesos{
			updated:              nowTime,
			accesosNivelComputed: usuarios[0].AccesosComputed,
		}
		cachedUsuarioAccesos = companyUsuarioAccesos[cacheKey]
		companyUsuarioAccesosMu.Unlock()

		Log("CheckUser:: cache updated", "companyID", companyID, "userID", userID, "accesosComputed", len(cachedUsuarioAccesos.accesosNivelComputed), "accesosDetalle", formatAccesosNivelForLog(cachedUsuarioAccesos.accesosNivelComputed), "updated", nowTime)
	} else {
		Log("CheckUser:: cache hit", "companyID", companyID, "userID", userID, "accesosComputed", len(cachedUsuarioAccesos.accesosNivelComputed), "accesosDetalle", formatAccesosNivelForLog(cachedUsuarioAccesos.accesosNivelComputed))
	}

	return cachedUsuarioAccesos.accesosNivelComputed, nil
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

	if err := cbor.Unmarshal(tokenBytes, &user); err != nil {
		Log("CheckUser:: error decodificando CBOR", err)
		user.Error = "Error al recuperar la información del user."
	}

	if user.Error == "" {
		expectedTokenHash := ComputeUsuarioTokenHash(user)
		if user.Hash != expectedTokenHash {
			Log("CheckUser:: hash inválido", "companyID", user.CompanyID, "userID", user.ID, "receivedHash", user.Hash, "expectedHash", expectedTokenHash)
			user.Error = "El token de inicio de sesión no es válido."
		}
	}

	var accesosErr error

	if user.Error == "" && user.CompanyID > 0 && user.ID > 0 {
		req.accesosNivel, accesosErr = loadUsuarioAccesosComputed(user.CompanyID, user.ID)
		if accesosErr != nil {
			Log("CheckUser:: error cargar accesos", accesosErr)
			user.Error = accesosErr.Error()
		} else {
			Log("CheckUser:: accesos computados cargados", "companyID", user.CompanyID, "userID", user.ID, "accesosComputed", len(req.accesosNivel), "requiredAccess", access)
		}
	}

	// NOTE: In local/VPS HTTP mode requests are concurrent; avoid mutating global user state.
	if Env.IS_SERVERLESS {
		User = user
		Env.USUARIO_ID = user.ID
	}

	return &user
}
