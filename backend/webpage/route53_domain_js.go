//go:build js

package webpage

import "errors"

// El build de navegador no enlaza el SDK de AWS (wasm_deps_test.go): publicar tiendas exige el
// backend de servidor.
var errStorefrontDNSUnavailable = errors.New("los dominios de tienda no están disponibles en el modo local")

func provisionStorefrontDomain(hostname string, allowExistingStorefrontDomain bool) error {
	return errStorefrontDNSUnavailable
}

func removeStorefrontDomain(hostname string) error {
	return errStorefrontDNSUnavailable
}
