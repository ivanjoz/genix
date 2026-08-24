package cloud

// ImageConvertInput and Image mirror the structs of the same name in
// github.com/ivanjoz/avif-webp-encoder/imageconv field-for-field — same names, same types, no
// tags — because they cross the conversion-Lambda boundary as JSON encoded with Go's default
// field-name marshalling. Renaming or reordering a field here silently breaks that wire format.
//
// They are declared locally instead of imported because importing imageconv at all links a 4.17 MB
// embedded encoder binary into the process: imageconv unconditionally imports its `binaries`
// package, which is a single //go:embed of a prebuilt Rust executable. The default build must not
// carry that, so nothing outside a file tagged `avif` may import imageconv. See RATIONALE.md.

type ImageConvertInput struct {
	Image              []byte   // Image as binary
	ImagePath          string   // Path or name of the image
	Resolutions        []uint16 // Slice of resolutions. Example 800 mean 800x800px density
	UseWebp            bool     // Not necesary if WebpQuality or WebpMethod is configured
	WebpQuality        uint8    // From 1 - 100
	WebpMethod         uint8    // From 1 - 6; default 6 (best quality, slowest)
	UseAvif            bool     // Not necesary if AvifSpeed or AvifQuality is configured
	AvifSpeed          uint8    // From 1 - 11; default 2 (more speed, less quality)
	AvifQuality        uint8    // From 1 - 100
	StdoutBufferSize   int      // Default 1024*1000
	OutputDirectory    string   // If want to save the images to directory
	UseDebugLogs       bool     // Default false
	TempDirOfExecution string
}

type Image struct {
	Content    []byte
	Name       string
	Resolution int32
	Format     string
}

// imageConversionDisabledMessage is what every entry point reports when the binary was built
// without the `avif` tag. It names the tag so the reader knows the fix.
const imageConversionDisabledMessage = "la conversión de imágenes en el backend no está compilada en este binario " +
	"(build tag `avif` ausente). El frontend debe enviar las resoluciones ya convertidas."
