package main

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"app/core"
)

// browserEnvelope is `marshal({ID:7,Nombre:'Ana',Items:[...]}).standalone` from the colbin npm
// package, sent as the body of a POST with Content-Type application/vnd.colbin.
const browserEnvelope = "d469030007000249440401064e6f6d6272650b020650726563696f0a030641637469766f00040454616773100507446574616c6c65150106054974656d7316020002000843616e74696461640401044e6f74610b0003000249440401064e6f6d6272650b020443616e7404040716416e61254029304c030102035c04040414786d310a0c07000421436587a91c1e026e30026e31026e32026e33026e34026e35026e36026e37026e38026e392c050002e4e404"

// The request goes through mainHandler like any other: routed, the colbin body decoded after the
// admission checks, and a handler that still reads *req.Body as JSON gets the same record.
func TestMainHandlerAcceptsAColbinRequestBody(t *testing.T) {
	previous := core.Env
	core.Env = &core.EnvStruct{IS_SERVERLESS: true}
	t.Cleanup(func() { core.Env = previous })

	raw, err := hex.DecodeString(browserEnvelope)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)

	var seen struct {
		ID     int32  `json:"ID"`
		Nombre string `json:"Nombre"`
	}
	appHandlers["POST.p-colbin-echo"] = func(req *core.HandlerArgs) core.HandlerResponse {
		if err := jsonUnmarshalBody(req, &seen); err != nil {
			return core.HandlerResponse{Error: err.Error()}
		}
		return core.MakeResponse(req, &seen)
	}
	defer delete(appHandlers, "POST.p-colbin-echo")

	args := &core.HandlerArgs{
		Body:   &body,
		Method: "POST",
		Route:  "p-colbin-echo",
		Headers: map[string]string{
			"Content-Type": core.ColbinMediaType,
			"Accept":       core.ColbinMediaType,
		},
	}
	response := mainHandler(args)
	if response == nil || response.Error != "" || seen.ID != 7 || seen.Nombre != "Ana" {
		t.Fatalf("handler saw %+v, response %+v", seen, response)
	}
}

// What a legacy handler does: read the body as JSON text, unaware of the transport.
func jsonUnmarshalBody(req *core.HandlerArgs, target any) error {
	return json.Unmarshal([]byte(*req.Body), target)
}
