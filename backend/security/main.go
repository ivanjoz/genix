package security

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.p-user-login":   PostLogin,
	"GET.reload-login":    ReloadLogin,
	"GET.usuarios":        GetUsuarios,
	"GET.usuarios-ids":    GetUsuariosByIDs,
	"POST.usuarios":       PostUsuarios,
	"POST.user-propio": PostUsuarios,
	"POST.perfiles":       PostPerfiles,
	"GET.perfiles":        GetPerfiles,
}
