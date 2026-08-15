package security

import "app/core"

var ModuleHandlers = core.AppRouterType{
	"POST.p-user-login": PostLogin,
	// Registro público de empresas: correo → código → empresa + usuario admin (auto-login).
	"POST.p-signup-request": PostSignUpRequest,
	"POST.p-signup-verify":  PostSignUpVerify,
	"POST.p-signup-company": PostSignUpCompany,
	// Formulario de contacto de /welcome: se guarda el mensaje y se notifica a contact.email.
	"POST.p-contact-message": PostContactMessage,
	"GET.reload-login":       ReloadLogin,
	// Sesión sin password para el navegador headless de desarrollo. Sólo responde con
	// is_local y desde loopback; ver dev_login.go.
	"GET.p-dev-login": DevLogin,
	"GET.users":             GetUsuarios,
	"GET.users-ids":         GetUsuariosByIDs,
	"POST.users":            PostUsuarios,
	"POST.user-self":        PostUsuarios,
	"POST.perfiles":         PostPerfiles,
	"GET.perfiles":          GetPerfiles,
}
