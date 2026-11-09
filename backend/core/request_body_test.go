package core

import (
	"encoding/hex"
	"encoding/json"
	"testing"
)

// browserBody is what the colbin npm package (0.10, the `doc` fast path) writes for
// marshal(body).standalone — a self-describing message whose schema the browser inferred:
//
//	{ ID: 7, Nombre: 'Ana', Precio: 12.5, Activo: true, Tags: [1,2,3],
//	  Detalle: { Cantidad: 4, Nota: 'x' }, Items: 10 × { ID, Nombre, Cant } }
const browserBody = "d469030007000249440401064e6f6d6272650b020650726563696f0a030641637469766f00040454616773100507446574616c6c65150106054974656d7316020002000843616e74696461640401044e6f74610b0003000249440401064e6f6d6272650b020443616e7404040716416e61254029304c030102035c04040414786d310a0c07000421436587a91c1e026e30026e31026e32026e33026e34026e35026e36026e37026e38026e392c050002e4e404"

type browserDetail struct {
	Cantidad int    `json:"Cantidad"`
	Nota     string `json:"Nota"`
}

type browserItem struct {
	ID     int32  `json:"ID"`
	Nombre string `json:"Nombre"`
	Cant   int32  `json:"Cant"`
}

type browserRequest struct {
	ID      int32         `json:"ID"`
	Nombre  string        `json:"Nombre"`
	Precio  float64       `json:"Precio"`
	Activo  bool          `json:"Activo"`
	Tags    []int         `json:"Tags"`
	Detalle browserDetail `json:"Detalle"`
	Items   []browserItem `json:"Items"`
}

func colbinRequest(t *testing.T) *HandlerArgs {
	t.Helper()
	raw, err := hex.DecodeString(browserBody)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	return &HandlerArgs{Body: &body, ContentType: ColbinMediaType}
}

func checkBrowserRequest(t *testing.T, got browserRequest) {
	t.Helper()
	if got.ID != 7 || got.Nombre != "Ana" || got.Precio != 12.5 || !got.Activo ||
		len(got.Tags) != 3 || got.Tags[2] != 3 || got.Detalle.Cantidad != 4 || got.Detalle.Nota != "x" ||
		len(got.Items) != 10 || got.Items[9].Nombre != "n9" || got.Items[9].Cant != 1 {
		t.Fatalf("decoded wrong: %+v", got)
	}
}

// The direct path: no JSON text in between.
func TestUnmarshalBodyReadsABrowserColbinMessage(t *testing.T) {
	req := colbinRequest(t)
	var got browserRequest
	if err := req.UnmarshalBody(&got); err != nil {
		t.Fatal(err)
	}
	checkBrowserRequest(t, got)
}

// The compatibility path: handlers that json.Unmarshal(*req.Body) never learn it was colbin.
func TestNormalizeBodyLetsJSONHandlersReadAColbinBody(t *testing.T) {
	req := colbinRequest(t)
	if !req.BodyIsColbin() {
		t.Fatal("not detected as colbin")
	}
	if err := req.NormalizeBody(); err != nil {
		t.Fatal(err)
	}
	if req.BodyIsColbin() {
		t.Fatal("still marked colbin after normalizing")
	}
	var got browserRequest
	if err := json.Unmarshal([]byte(*req.Body), &got); err != nil {
		t.Fatalf("%v: %s", err, *req.Body)
	}
	checkBrowserRequest(t, got)
}

func TestJSONBodiesAreLeftAlone(t *testing.T) {
	body := `{"ID":7,"Nombre":"Ana"}`
	req := &HandlerArgs{Body: &body, ContentType: "application/json"}
	if err := req.NormalizeBody(); err != nil || *req.Body != body {
		t.Fatalf("a JSON body changed: %v %s", err, *req.Body)
	}
	var got browserRequest
	if err := req.UnmarshalBody(&got); err != nil || got.ID != 7 {
		t.Fatalf("%v %+v", err, got)
	}
}

// Hostile bytes are an error, never a panic: the schema section comes from the client.
func TestAHostileColbinBodyIsRefused(t *testing.T) {
	good := colbinRequest(t)
	for cut := 0; cut < len(*good.Body); cut += 3 {
		body := (*good.Body)[:cut]
		req := &HandlerArgs{Body: &body, ContentType: ColbinMediaType}
		_ = req.NormalizeBody()
		var got browserRequest
		_ = (&HandlerArgs{Body: &body, ContentType: ColbinMediaType}).UnmarshalBody(&got)
	}
	for at := 0; at < len(*good.Body); at++ {
		raw := []byte(*good.Body)
		raw[at] ^= 0xff
		body := string(raw)
		req := &HandlerArgs{Body: &body, ContentType: ColbinMediaType}
		_ = req.NormalizeBody()
	}
}
