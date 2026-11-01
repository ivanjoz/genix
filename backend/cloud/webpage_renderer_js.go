//go:build js

package cloud

import "errors"

// Publishing a webpage runs the Node renderer (a Lambda or a local process) and uploads to the CDN;
// neither exists in a browser tab.
func InvokeWebpageRenderer(request WebpageRenderRequest) (WebpageRenderResult, error) {
	return WebpageRenderResult{}, errors.New("webpage publishing is not available in the browser build")
}
