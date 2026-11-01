package cloud

// WebpageRenderPage es una página a renderizar: su PageID (el que identifica el contenido
// en el snapshot del CDN) y la ruta pública bajo la que se publica el HTML.
type WebpageRenderPage struct {
	ID   int16  `json:"id" cb:"id,1"`
	Path string `json:"path" cb:"path,2"`
}

type WebpageRenderRequest struct {
	CompanyID int32               `json:"companyID" cb:"companyID,1"`
	Hostname  string              `json:"hostname" cb:"hostname,2"`
	Pages     []WebpageRenderPage `json:"pages" cb:"pages,3"`
	// ForceAssets vuelve a subir los js/css aunque la company ya tenga publicada esa
	// versión del renderer (útil si se borraron del CDN a mano).
	ForceAssets bool `json:"forceAssets" cb:"forceAssets,4"`
}

type WebpageRenderResult struct {
	BuildID string `json:"buildId" cb:"buildId,1"`
	Pages   int    `json:"pages" cb:"pages,2"`
	Assets  int    `json:"assets" cb:"assets,3"`
	Bytes   int    `json:"bytes" cb:"bytes,4"`
}
