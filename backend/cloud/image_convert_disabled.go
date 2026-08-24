//go:build !avif

package cloud

import "app/core"

// SaveConvertImage fails loudly in the default build rather than silently storing nothing.
//
// Backend image conversion is legacy: the frontend converts and uploads every resolution itself,
// so the only caller (business.SaveProductImage) reaches this just when a client sent a single
// unconverted image. Compiling the encoder in for that fallback cost 4.17 MB of embedded Rust
// binary — 7.7% of the Lambda — so the default build drops it and reports the problem instead.
//
// Build with `-tags avif` to restore the real implementation in image_convert_avif.go.
func SaveConvertImage(args ImageArgs) ([]Image, error) {
	return nil, core.Err(imageConversionDisabledMessage)
}
