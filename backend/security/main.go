package security

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.p-user-login": PostLogin,
	// Registro público de empresas: correo → código → empresa + usuario admin (auto-login).
	"POST.p-signup-request": PostSignUpRequest,
	"POST.p-signup-verify":  PostSignUpVerify,
	"POST.p-signup-company": PostSignUpCompany,
	"GET.reload-login":      ReloadLogin,
	"GET.usuarios":          GetUsuarios,
	"GET.usuarios-ids":      GetUsuariosByIDs,
	"POST.usuarios":         PostUsuarios,
	"POST.user-propio":      PostUsuarios,
	"POST.perfiles":         PostPerfiles,
	"GET.perfiles":          GetPerfiles,
}
