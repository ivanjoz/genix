//go:build !avif

package exec

import "app/core"

// CompressImage is registered in ExecHandlers unconditionally, so it must exist in every build.
// Without the `avif` tag the encoder is not linked (it embeds a 4.17 MB binary), so the handler
// reports that rather than pretending to convert. See cloud/image_convert_disabled.go.
func CompressImage(args *core.ExecArgs) core.FuncResponse {
	return args.MakeErr("la conversión de imágenes no está compilada en este binario " +
		"(build tag `avif` ausente)")
}
