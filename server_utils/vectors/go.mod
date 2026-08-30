// A module of its own, so the vector generator can depend on colbin without
// joining the backend's module graph.
module github.com/ivanjoz/genix/server_utils/vectors

go 1.27

require github.com/ivanjoz/colbin v0.1.0

require (
	github.com/viant/xreflect v0.0.0-20230303201326-f50afb0feb0d // indirect
	github.com/viant/xunsafe v0.10.3 // indirect
)
