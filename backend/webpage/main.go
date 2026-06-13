package webpage

import (
	"app/core"
)

var ModuleHandlers = core.AppRouterType{
	"GET.ecommerce-page-content":  GetPageContent,
	"POST.ecommerce-page-content": PostPageContent,
	"GET.webpage-pages":           GetWebpages,
	"POST.webpage-page":           PostWebpage,
	"POST.webpage-showcase-image": PostWebpageShowcaseImage,
	"GET.website-config":          GetWebsiteConfig,
	"POST.website-seo":            PostWebsiteSeo,
	"POST.website-domain":         PostWebsiteDomain,
	// Public (no-auth, "p-" prefix) read for the prerender build + deployed
	// storefronts: one call returns a page's SEO config + content, scoped by the
	// company-id query param. GET.p-webpage?company-id=<id>&id=<pageID>.
	"GET.p-webpage": GetWebpagePublic,
}
