//go:build js

package cloud

import "errors"

// The browser build has no S3 or R2. The local object store lives in the browser and comes with the
// IndexedDB driver (docs/WASM_LOCAL_MODE_PLAN.md, phase 2); until then every call fails loudly
// instead of pretending to store the file.
var errNoObjectStore = errors.New("the browser build has no object store yet")

func SaveFile(args SaveFileArgs) error { return errNoObjectStore }

func GetFile(args SaveFileArgs) ([]byte, error) { return nil, errNoObjectStore }

func ListFiles(args SaveFileArgs) ([]FileInfo, error) { return nil, errNoObjectStore }

func DeleteFile(args SaveFileArgs) error { return errNoObjectStore }
