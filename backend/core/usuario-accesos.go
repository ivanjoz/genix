package core

import (
	coretypes "app/core/types"
	"app/db"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
)

type UsuarioInfo struct {
	EmpresaID    int32   `json:"c"`
	ID           int32   `json:"d"`
	PerfilesIDs  []int32 `json:"p"`
	ModulesIDs   []int32 `json:"m"`
	BaseModuleID int32   `json:"b"`
	RolesIDs     []int32 `json:"r"`
	Device       string  `json:"v,omitempty"`
	IP           string  `json:"i,omitempty"`
	Usuario      string  `json:"u,omitempty"`
	AccesosIDs   []int32 `json:"a"`
	Expired      int64   `json:"e"`
	ZonaHoraria  int16   `json:"h,omitempty"`
	DeviceID     int     `json:"q,omitempty"`
	Error        string  `json:"-"`
}

var Usuario UsuarioInfo

type UsuarioAccesos struct {
	updated              int32 // SUnixTime()
	accesosNivelComputed []uint16
}

var companyUsuarioAccesos = map[uint64]UsuarioAccesos{}
var companyUsuarioAccesosMu sync.RWMutex

const usuarioAccesosCacheTTL int32 = 150 // 5 minutes with SUnixTime 2-second ticks.

func makeCompanyUsuarioAccesosKey(empresaID, usuarioID int32) uint64 {
	return uint64(uint32(empresaID))<<32 | uint64(uint32(usuarioID))
}

func loadUsuarioAccesosComputed(empresaID, usuarioID int32) ([]uint16, error) {
	nowTime := SUnixTime()
	cacheKey := makeCompanyUsuarioAccesosKey(empresaID, usuarioID)

	companyUsuarioAccesosMu.RLock()
	cachedUsuarioAccesos, cacheFound := companyUsuarioAccesos[cacheKey]
	companyUsuarioAccesosMu.RUnlock()

	cacheIsFresh := nowTime >= cachedUsuarioAccesos.updated && (nowTime-cachedUsuarioAccesos.updated) <= usuarioAccesosCacheTTL

	if !cacheFound {
		Log("CheckUser:: cache miss", "empresaID", empresaID, "usuarioID", usuarioID)
	} else if !cacheIsFresh {
		Log("CheckUser:: cache stale", "empresaID", empresaID, "usuarioID", usuarioID, "cacheAge", nowTime-cachedUsuarioAccesos.updated)
	}

	if !cacheIsFresh {
		usuarios := []coretypes.Usuario{}
		usuarioQuery := db.Query(&usuarios)
		usuarioQuery.Select(usuarioQuery.EmpresaID, usuarioQuery.ID, usuarioQuery.AccesosComputed).
			EmpresaID.Equals(empresaID).ID.Equals(usuarioID).Limit(1)

		Log("CheckUser:: querying usuario accesos", "empresaID", empresaID, "usuarioID", usuarioID)
		if err := usuarioQuery.Exec(); err != nil {
			return nil, Err("Error al obtener los accesos computados del usuario en ScyllaDB:", err)
		}
		if len(usuarios) == 0 {
			return nil, Err(fmt.Sprintf("No se encontró el usuario %d de la empresa %d en ScyllaDB.", usuarioID, empresaID))
		}
		slices.Sort(usuarios[0].AccesosComputed)

		companyUsuarioAccesosMu.Lock()
		companyUsuarioAccesos[cacheKey] = UsuarioAccesos{
			updated:              nowTime,
			accesosNivelComputed: usuarios[0].AccesosComputed,
		}
		cachedUsuarioAccesos = companyUsuarioAccesos[cacheKey]
		companyUsuarioAccesosMu.Unlock()

		Log("CheckUser:: cache updated", "empresaID", empresaID, "usuarioID", usuarioID, "accesosComputed", len(cachedUsuarioAccesos.accesosNivelComputed), "updated", nowTime)
	} else {
		Log("CheckUser:: cache hit", "empresaID", empresaID, "usuarioID", usuarioID, "accesosComputed", len(cachedUsuarioAccesos.accesosNivelComputed))
	}

	return cachedUsuarioAccesos.accesosNivelComputed, nil
}

func CheckUser(req *HandlerArgs, access int) *UsuarioInfo {
	userToken := req.Headers["authorization"]
	if len(userToken) < 8 {
		userToken = req.Headers["Authorization"]
	}

	usuario := UsuarioInfo{}

	if len(userToken) < 8 || !strings.Contains(userToken, "Bearer ") {
		usuario.Error = "No se suministró un Token de usuario"
		return &usuario
	}

	encryptedInfo := strings.Split(userToken, " ")[1]
	if len(encryptedInfo) < 8 {
		usuario.Error = "No se encontró la informaación encriptada del usuario"
		return &usuario
	}

	encryptedInfoBytes := Base64ToBytes(MakeB64UrlDecode(encryptedInfo))
	if len(encryptedInfoBytes) < 16 {
		usuario.Error = "El tocken de inicio de sesión es inválido"
		return &usuario
	}
	decriptedBytes, err := Decrypt(encryptedInfoBytes)

	if err != nil {
		Log("Error desencriptar:", err)
		usuario.Error = "Hubo un error al desencriptar el Token."
		return &usuario
	}

	decriptedBytesJson := DecompressZstd(&decriptedBytes)

	if err := json.Unmarshal([]byte(decriptedBytesJson), &usuario); err != nil {
		Log("Error recuperar info:", err)
		usuario.Error = "Error al recuperar la información del usuario."
	}

	var accesosErr error

	if usuario.Error == "" && usuario.EmpresaID > 0 && usuario.ID > 0 {
		req.accesosNivel, accesosErr = loadUsuarioAccesosComputed(usuario.EmpresaID, usuario.ID)
		if accesosErr != nil {
			Log("CheckUser:: error cargar accesos", accesosErr)
			usuario.Error = accesosErr.Error()
		} else {
			Log("CheckUser:: accesos computados cargados", "empresaID", usuario.EmpresaID, "usuarioID", usuario.ID, "accesosComputed", len(req.accesosNivel), "requiredAccess", access)
		}
	}

	// NOTE: In local/VPS HTTP mode requests are concurrent; avoid mutating global user state.
	if Env.IS_SERVERLESS {
		Usuario = usuario
		Env.USUARIO_ID = usuario.ID
	}

	return &usuario
}
