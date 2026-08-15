package security

import (
	"app/cloud"
	"app/core"
	coretypes "app/core/types"
	"app/db"
	"net"
	"strings"
)

// isLoopbackClient reports whether the caller reached this process over the loopback interface.
//
// Second half of the DevLogin guard. The first half (is_local) is a config value, and a config
// value that is wrong turns a password-less session mint into a full authentication bypass, so
// the address is checked independently: an is_local=true deploy that is publicly reachable still
// refuses everyone who is not on the machine itself.
func isLoopbackClient(clientIP string) bool {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		// Absent address (Lambda invocation shapes that carry no source IP) is never local.
		return false
	}
	// Trim a "host:port" form; ParseIP rejects it otherwise.
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}
	parsed := net.ParseIP(clientIP)
	return parsed != nil && parsed.IsLoopback()
}

// DevLogin mints a session for an arbitrary (companyID, userID) without a password, so the
// headless dev browser (scripts/agent_browser) can attach to the app as a real user. It is the
// only way an automated tool can reach an authenticated page: every other entry point needs a
// password nobody stores for test users.
//
// Guarded twice — the process must be configured local AND the caller must be on loopback. Both
// must hold; see isLoopbackClient for why is_local alone is not enough.
//
// The response is deliberately identical in shape to PostLogin's: it is built by the same
// MakeUsuarioResponse, so the frontend consumes it with the unmodified security.parseLogin and no
// second session-hydration path exists to drift.
func DevLogin(req *core.HandlerArgs) core.HandlerResponse {
	if !core.Env.IS_LOCAL || !isLoopbackClient(req.ClientIP) {
		core.Log("DevLogin:: rechazado. is_local::", core.Env.IS_LOCAL, " clientIP::", req.ClientIP)
		return req.MakeErr401("La ruta de desarrollo p-dev-login no está habilitada.")
	}

	// The cipher key travels from the caller because MakeUsuarioResponse encrypts UserInfo with
	// it, exactly as the real login does — the browser generates it and decrypts with the same key.
	cipherKey := req.GetQuery("cipher-key")
	if len(cipherKey) < 16 {
		return req.MakeErr("El cipher-key es necesario y debe tener al menos 16 caracteres.")
	}

	// Company 1 / user 1 is the bootstrap admin, which is what a dev session wants by default.
	companyID := req.GetQueryInt("company")
	if companyID == 0 {
		companyID = 1
	}
	userID := req.GetQueryInt("user")
	if userID == 0 {
		userID = 1
	}

	var user *coretypes.User
	var err error
	if cloud.IsDataMirrorEnabled() {
		user, err = cloud.GetByID(coretypes.User{CompanyID: companyID, ID: userID})
	} else {
		users := []coretypes.User{}
		userQuery := db.Query(&users)
		userQuery.CompanyID.Equals(companyID).ID.Equals(userID).Limit(1)
		err = userQuery.Exec()
		if err == nil && len(users) > 0 {
			user = &users[0]
		}
	}
	if err != nil {
		return req.MakeErr("Error al obtener el user.", err)
	}
	if user == nil {
		return req.MakeErr("No se encontró el user", userID, "de la empresa", companyID)
	}

	response, err := MakeUsuarioResponse(*user, cipherKey)
	if err != nil {
		return req.MakeErr(err)
	}

	core.Log("DevLogin:: sesión emitida para company::", companyID, " user::", userID)
	return req.MakeResponse(response)
}
