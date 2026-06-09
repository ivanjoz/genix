package ecommerce

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"GET.ecommerce-page-content":  GetPageContent,
	"POST.ecommerce-page-content": PostPageContent,
}
