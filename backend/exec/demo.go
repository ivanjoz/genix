package exec

import (
	"app/aws"
	"app/core"
	"app/db"
	"app/facturacion"
	s "app/types"
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/fxamacker/cbor/v2"
	mail "github.com/xhit/go-simple-mail/v2"
	"golang.org/x/sync/errgroup"
)

/*
func TestScyllaDBConnection(args *core.ExecArgs) core.FuncResponse {
	usuarios := []types.Usuario{}
	if err := core.DBSelect(&usuarios); err != nil {
		panic(err)
	}

	core.Log("usuarios obtenidos:: ", len(usuarios))
	core.Print(usuarios)

	return core.FuncResponse{}
}
*/

func TestScyllaDBInsert(args *core.ExecArgs) core.FuncResponse {
	counter, err := core.GetCounter("usuarios_1", 1)
	if err != nil {
		panic(err)
	}

	usuarios := []s.Usuario{
		{
			ID:          int32(counter),
			Nombres:     "Hola 2",
			Apellidos:   "Mundo 2",
			PerfilesIDs: []int32{2, 3, 4},
			Updated:     time.Now().Unix(),
			Created:     time.Now().Unix(),
		},
	}
	core.Log(usuarios)

	return core.FuncResponse{}
}

const DEMO_TEXT = `Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s`

func TestZstdCompression(args *core.ExecArgs) core.FuncResponse {
	text := DEMO_TEXT
	core.Log(text)
	core.Log("Longitud inicial:", len(text))
	textCompressed := core.CompressZstd(&text)
	core.Log("Longitud comprimida:", len(textCompressed))
	textUncompressed := core.DecompressZstd(&textCompressed)
	core.Log("Texto Descomprimido:")
	core.Log(textUncompressed)

	return core.FuncResponse{}
}

func TestDynamoCounter(args *core.ExecArgs) core.FuncResponse {

	counter, err := aws.GetDynamoCounter("demo")

	if err != nil {
		panic(err)
	}

	fmt.Println("Counter actual:: ", counter)

	return core.FuncResponse{}
}

func Test14(args *core.ExecArgs) core.FuncResponse {
	text := DEMO_TEXT

	encryptedBytes, err := core.Encrypt([]byte(text))

	if err != nil {
		panic("Error al encriptar el string:: " + err.Error())
	}

	core.Log("Longitud Inicial:", len(text), "| Final:", len(encryptedBytes))

	decriptedBytes, err := core.Decrypt(encryptedBytes)

	if err != nil {
		panic("Error al desencriptar el string:: " + err.Error())
	}

	core.Log("String Desencriptado:: ", string(decriptedBytes))

	return core.FuncResponse{}
}

func Test15(args *core.ExecArgs) core.FuncResponse {
	type Address struct {
		City, State string
	}
	type Person struct {
		XMLName   xml.Name `xml:"person"`
		Id        int      `xml:"id,attr"`
		FirstName string   `xml:"name>first"`
		LastName  string   `xml:"name>last"`
		Age       int      `xml:"age"`
		Height    float32  `xml:"height,omitempty"`
		Married   bool
		Address
		Comment string `xml:",comment"`
	}

	v := &Person{Id: 13, FirstName: "John", LastName: "Doe", Age: 42}
	// v.Comment = " Need more details. "
	v.Address = Address{"Hanga Roa", "Easter Island"}

	var buffer bytes.Buffer
	enc := xml.NewEncoder(&buffer)
	enc.Indent("  ", "    ")
	if err := enc.Encode(v); err != nil {
		fmt.Printf("error: %v\n", err)
	}

	core.Log(buffer.String())

	invoice := facturacion.NewInvoice()

	var buffer2 bytes.Buffer
	enc = xml.NewEncoder(&buffer2)
	enc.Indent("  ", "    ")
	if err := enc.Encode(invoice); err != nil {
		fmt.Printf("error: %v\n", err)
	}

	core.Log(buffer2.String())

	return core.FuncResponse{}
}

func Test16(args *core.ExecArgs) core.FuncResponse {

	signatureArgs := facturacion.MakeSignatureArgs{
		SignatureID:     "signatureKG",
		DigestValue:     "ld6X+TvM42Fe+F1KM/OB jiKpnko=",
		SignatureValue:  "W6DbMHJEFmU7GuiU0O+HRUqVzQZZW3QndYtUyeL0VxXuTafHu2vBC+OXvnnali43VXRGQ+/E0tPlZAssqI/PEPfzIU79Wufq6saxYGHKvzdnBi6hnaMuCSG5THHNFppx4aT1KNg7p/koBB3U8PT9C6m6		UnkJJNUquHkFc9BCqI8=",
		X509SubjectName: "1.2.840.113549.1.9.1=#161a4253554c434140534f55544845524e504552552e434f4d2e5045,CN=CarlosVega,OU=10200545523,O=Vega Poblete Carlos Enrique,L=CHICLAYO,ST=LAMBAYEQUE,C=PE",
		X509Certificate: "MIIESTCCAz		GgAwIBAgIKWOCRzgAAAAAAIjANBgkqhkiG9w0BAQUFADAnMRUwEwYKCZImiZPyLGQB		GRYFU1VOQVQxDjAMBgNVBAMTBVNVTkFUMB4XDTEwMTIyODE5NTExMFoXDTExMTIyODIwMDExMFowgZUxCzAJBgNVBAYTAlBFMQ0wCwYDVQQIEwRMSU1BMQ0wCwYDVQQHEwRMSU1BMREwDwYDVQQKEwhT",
	}

	signature := facturacion.MakeSignature(signatureArgs)

	var buffer bytes.Buffer
	enc := xml.NewEncoder(&buffer)
	enc.Indent("  ", "    ")
	if err := enc.Encode(signature); err != nil {
		fmt.Printf("error: %v\n", err)
	}

	core.Log(buffer.String())

	return core.FuncResponse{}
}

func Test17(args *core.ExecArgs) core.FuncResponse {

	usuarios := []s.Usuario{}
	// core.DBSelect(&usuarios)

	core.Print(usuarios)

	return core.FuncResponse{}
}

type DemoStruct1 struct {
	Hola  string  `json:"a,omitempty" ms:"h"`
	Value float32 `json:"b,omitempty" ms:"v"`
	Hola2 int     `json:"c,omitempty" ms:"c"`
}

type DemoStruct2 struct {
	Hola   string  `json:"a" ms:"h"`
	Value  float32 `json:"b" ms:"v"`
	Value2 float32 `json:"c" ms:"v2"`
}

type DemoStruct3 struct {
	Nombre string `json:"nombre1" msgpack:"1,omitempty"`
	Demo   int32  `json:"demo" msgpack:"2,omitempty"`
}

type DemoStruct struct {
	s.TAGS    `table:"demo_structs"`
	CompanyID int32         `cbor:"1,keyasint,omitempty" json:"companyID,omitempty" db:"company_id,pk"`
	ID        int32         `cbor:"2,keyasint,omitempty" json:"id,omitempty" db:"id,pk"`
	Edad      int32         `cbor:"3,keyasint,omitempty" json:"edad,omitempty" db:"edad,zx1,zx2"`
	Nombre    string        `cbor:"4,keyasint,omitempty" json:"nombre,omitempty" db:"nombre,zx1"`
	Palabras  []string      `cbor:"5,keyasint,omitempty" json:"palabras,omitempty" db:"palabras"`
	Peso      float32       `cbor:"6,keyasint,omitempty" json:"peso,omitempty" db:"peso"`
	Peso64    float64       `cbor:"7,keyasint,omitempty" json:"peso64,omitempty" db:"peso_64"`
	Rangos    []int32       `cbor:"8,keyasint,omitempty" json:"rangos,omitempty" db:"rangos"`
	Smallint  int16         `cbor:"9,keyasint,omitempty" db:"small_int,zx2"`
	Struct1   DemoStruct1   `cbor:"10,keyasint,omitempty" json:"struct_1,omitempty" db:"struct_1"`
	Struct2   DemoStruct3   `cbor:"11,keyasint,omitempty" json:"struct_2,omitempty" db:"struct_2"`
	Struct3   []DemoStruct1 `cbor:"12,keyasint,omitempty" json:"struct_3,omitempty" db:"struct_3"`
}

type DemoStruct4 struct {
	s.TAGS    `table:"demo_structs"`
	CompanyID int32    `cbor:"1,keyasint,omitempty" json:"companyID,omitempty" db:"company_id,pk"`
	ID        int32    `cbor:"2,keyasint,omitempty" json:"id,omitempty" db:"id,pk"`
	Edad      int32    `cbor:"3,keyasint,omitempty" json:"edad,omitempty" db:"edad,zx1,zx2"`
	Nombre    string   `cbor:"4,keyasint,omitempty" json:"nombre,omitempty" db:"nombre,zx1"`
	Palabras  []string `cbor:"5,keyasint,omitempty" json:"palabras,omitempty" db:"palabras"`
	Rangos    []int32  `cbor:"6,keyasint,omitempty" json:"rangos,omitempty" db:"rangos"`
	Smallint  int16    `cbor:"7,keyasint,omitempty" json:"small_int,omitempty" db:"small_int,zx2"`
	Peso      float32  `cbor:"8,keyasint,omitempty" json:"peso,omitempty" db:"peso"`
	Peso64    float64  `cbor:"9,keyasint,omitempty" json:"peso64,omitempty" db:"peso_64"`
}

func Test18(args *core.ExecArgs) core.FuncResponse {
	/*
		registros := []DemoStruct{}
		err := core.DBSelect(&registros).
			Where("company_id").Equals(1).Where("id").GreatThan(2).Exec()

		if err != nil {
			panic(err)
		}

		core.Print(registros)
	*/
	return core.FuncResponse{}
}

func Test19(args *core.ExecArgs) core.FuncResponse {

	st1 := DemoStruct1{Hola: "hola", Value: 3.123}
	st2 := DemoStruct3{Nombre: "lalademo1234", Demo: 555}
	st3 := []DemoStruct1{
		{Hola: "hola", Value: 3.123},
		{Hola: "gato", Value: 123.12300109863281},
		{Hola: "perro", Value: 222.12300109863282},
		{Hola: "dada", Value: 222.1423},
		{Hola: "demo", Value: 123.123}}

	id, err := core.GetCounter("demo_structs_1", 1)
	if err != nil {
		panic(err)
	}

	demo := DemoStruct{
		CompanyID: 1,
		ID:        int32(id),
		Edad:      43,
		Nombre:    "11prueba",
		Palabras:  []string{"hola", "mundo", "lala"},
		Peso:      4.1234,
		Peso64:    8.123,
		Rangos:    []int32{12, 13},
		Smallint:  44,
		Struct1:   st1,
		Struct2:   st2,
		Struct3:   st3,
	}

	registros := []DemoStruct{demo}
	core.Log(registros)

	return core.FuncResponse{}
}

func Test20(args *core.ExecArgs) core.FuncResponse {
	/*
		words := []string{"dasd", "dasdasdad", "12312312", "dasdasd123123", "dqw986nwqd9a"}

		for _, w := range words {
			fmt.Println(core.BasicHashInt(w))
		}
	*/

	demo1 := []DemoStruct1{
		{Hola: "hola", Value: 3},
		{Hola: "gato", Value: 123},
		{Hola: "perro", Value: 222.12300109863281},
		/*
			{Hola: "perro", Value: 222},
			{Hola: "perro", Value: 222.1423},
			{Hola: "perro", Value: 222.1423},
			{Hola: "perro", Value: 222.1423},
			{Hola: "dada", Value: 222.1423},
			{Hola: "gato", Value: 123.3},
			{Hola: "perro", Value: 222.1423},
			{Hola: "perro", Value: 222.1423},
			{Hola: "perro", Value: 222.1423},
			{Hola: "perro", Value: 222.1423},
			{Hola: "perro", Value: 222.1423},
			{Hola: "dada", Value: 222.1423},
		*/
		{Hola: "demo", Value: 123.12300109863282}}

	var buffer1 bytes.Buffer
	enc := gob.NewEncoder(&buffer1)

	enc.Encode(demo1)
	core.Log("Bytes encoding .gob:", len(buffer1.Bytes()), " | ", buffer1.String())

	encoded1, _ := core.MsgPEncode(&demo1)
	core.Log("Bytes encoding .msgpack:", len(encoded1), " | ", string(encoded1))

	demo2 := []DemoStruct2{}
	err := core.MsgPDecode(encoded1, &demo2)
	if err != nil {
		panic(err)
	}
	core.Print(demo2)
	/*
		encoded, _ := binary.Marshal(&demo1)
		core.Log("Bytes encoding .binary:", len(encoded), " | ", string(encoded))

		var demo3 []DemoStruct1
		err = binary.Unmarshal(encoded, &demo3)
		if err != nil {
			panic(err)
		}
		core.Print(demo3)
	*/
	jsonBytes, _ := json.Marshal(&demo1)
	core.Log("Bytes encoding .json:", len(string(jsonBytes)), " | ", string(jsonBytes))

	return core.FuncResponse{}
}

func Test21(args *core.ExecArgs) core.FuncResponse {

	demo := DemoStruct{
		CompanyID: 1,
		ID:        int32(1),
		Edad:      43,
		Nombre:    "11prueba",
		Palabras:  []string{"hola", "mundo", "lala"},
		Peso:      4.1234,
		Peso64:    8.123,
		Rangos:    []int32{12, 13},
		Smallint:  44,
	}

	var b bytes.Buffer
	e := gob.NewEncoder(&b)
	if err := e.Encode(demo); err != nil {
		panic(err)
	}

	fmt.Println("Encoded Struct ", b.String())

	var demoDecode DemoStruct4
	d := gob.NewDecoder(&b)
	if err := d.Decode(&demoDecode); err != nil {
		panic(err)
	}

	core.Print(demoDecode)

	return core.FuncResponse{}
}

func Test22(args *core.ExecArgs) core.FuncResponse {

	filePath := "/home/ivanjoz/projects/genix/backend/tmp/output-2.json"
	file, err := os.Open(filePath)
	if err != nil {
		panic("Error opening body (response.json) " + err.Error())
	}
	defer file.Close()

	scanner2 := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner2.Buffer(buf, 5*1024*1024)

	size := 0
	lines := 0
	for scanner2.Scan() {
		line := scanner2.Text()
		core.Log("Line: ", line)
		lines++
		size += len(line)
	}

	if err := scanner2.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	core.Log("Agregando a Body:: ", filePath, " | Lines:", lines, " | Size:", size)

	return core.FuncResponse{}
}

func Test23(args *core.ExecArgs) core.FuncResponse {

	wd, _ := os.Getwd()
	tmpDir := wd + "/tmp"
	core.Log("tmpDir:", tmpDir)

	if runtime.GOOS == "windows" {
		fmt.Println("Can't Execute this on a windows machine")
	} else {
		inputFile := tmpDir + "/demo.png"
		outputFile := tmpDir + "/demo.avif"

		cmd := fmt.Sprintf("%v/libs/cavif_linux_x64", wd)
		args := []string{"-i", inputFile, "-o", outputFile, "--resize-mode", "fixed", "--resize-denominator_", "16"}

		cmdToExec := exec.Command(cmd, args...)
		core.Log("Ejecutando: ", cmdToExec.String())
		out, err := cmdToExec.Output()
		core.Log(string(out))
		if err != nil {
			panic(err)
		}
	}

	return core.FuncResponse{}
}

func Test24(args *core.ExecArgs) core.FuncResponse {
	if core.Env.SMTP_PORT == 0 || len(core.Env.SMTP_EMAIL) == 0 ||
		len(core.Env.SMTP_PASSWORD) == 0 || len(core.Env.SMTP_HOST) == 0 {
		panic("faltan datos para enviar el SMTP")
	}

	const htmlBody = `<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<title>Hello Gophers!</title>
	</head>
	<body>
		<p>This is the <b>Go gopher</b>.</p>
	</body>
</html>`

	server := mail.NewSMTPClient()

	// SMTP Server
	server.Host = core.Env.SMTP_HOST
	server.Port = int(core.Env.SMTP_PORT)
	server.Username = core.Env.SMTP_USER
	server.Password = core.Env.SMTP_PASSWORD
	server.Encryption = mail.EncryptionSTARTTLS

	smtpClient, err := server.Connect()

	if err != nil {
		log.Fatal(err)
	}

	// New email simple html with inline and CC
	email := mail.NewMSG()
	email.SetFrom(fmt.Sprintf("From Ivan <%v>", core.Env.SMTP_EMAIL)).
		AddTo("anguloivan3@gmail.com").
		SetSubject("Email test from Go")

	email.SetBody(mail.TextHTML, htmlBody)
	/*
		core.Log(ExecutableLinux)
		core.Log(ExecutableWindows)
	*/

	// always check error after send
	if email.Error != nil {
		log.Fatal(email.Error)
	}

	// Call Send and pass the client
	err = email.Send(smtpClient)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Email Sent")
	}

	return core.FuncResponse{}
}

type Hola1 struct {
	Hola   string
	Nombre string
	Edad   int
}

func (e *Hola1) Saludar(num int32) {
	core.Log("numero:", num)
}

type SaludarInterface interface {
	Saludar(num int32)
}

func TestGeneric[T any](obj T) {
	if r, ok := any(obj).(SaludarInterface); ok {
		r.Saludar(1234)
		core.Log("implementa la interfaz::", r)
	} else {
		core.Log("NO implementa la interfaz::")
	}
}

func TestInterface(obj SaludarInterface) {
	obj.Saludar(1234)
}

func Test25(args *core.ExecArgs) core.FuncResponse {

	obj := Hola1{Hola: "dasd", Edad: 22}
	TestGeneric(&obj)
	TestInterface(&obj)

	num := int64(1111)
	base62encode := core.EncodeToBase62(num)
	fmt.Println("Base62 encoded:", base62encode)
	fmt.Println("Base62 decoded:", core.DecodeFromBase62(base62encode))

	key := core.Concat62(123987, 12387, 23, "dasd11")
	fmt.Println(key)

	return core.FuncResponse{}
}

func Test26(args *core.ExecArgs) core.FuncResponse {

	return core.FuncResponse{}
}

func Test27(args *core.ExecArgs) core.FuncResponse {
	/*
		dest := bytes.Buffer{}
		encoder := cbor.new(&dest)
	*/
	/*
		opts := cbor.CoreDetEncOptions()
		tags := cbor.NewTagSet()
		tags.Add(
			cbor.TagOptions{EncTag: cbor.EncTagRequired, DecTag: cbor.DecTagRequired},
			reflect.TypeOf(signedCWT{}),
			18)
		opts.EncModeWithTags()
	*/

	array1 := []DemoStruct4{
		{Nombre: "ho1",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
		{Nombre: "ho1",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
	}

	array2 := []DemoStruct4{
		{Nombre: "demo1",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
		{Nombre: "dasdasd",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
		{Nombre: "dasda 123",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
		{Nombre: "dasd",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
		{Nombre: "11",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
		{Nombre: "222",
			Edad:      1,
			CompanyID: 1,
			Smallint:  1,
		},
	}

	bytes1, err := cbor.Marshal(array1)
	if err != nil {
		panic(err)
	}

	bytes2, err := cbor.Marshal(array2)
	if err != nil {
		panic(err)
	}

	core.Log("comparing len::", len(bytes1), " | ", len(bytes2))
	core.Log(string(bytes2))

	return core.FuncResponse{}
}

func Test28(args *core.ExecArgs) core.FuncResponse {

	fmt.Printf("Tunix Time: %v\n", core.SUnixTime())
	fmt.Printf("Tunix Time Milli: %v\n", core.SUnixTimeMilli())
	fmt.Print("Probando UUID")
	fmt.Printf("Tunix UUID: %v\n", core.SUnixTimeUUID())
	fmt.Printf("Tunix ID + UUID: %v\n", core.SUnixTimeUUIDConcatID(120))

	return core.FuncResponse{}
}

func Test29(args *core.ExecArgs) core.FuncResponse {

	db.RecalcVirtualColumns[s.ListaCompartidaRegistro]()

	return core.FuncResponse{}
}

func Test30(args *core.ExecArgs) core.FuncResponse {

	listasIDs := []int32{1, 2}
	updated := int64(789456123)
	errGroup := errgroup.Group{}

	listasRegistrosMap := map[int32]*[]s.ListaCompartidaRegistro{}
	for _, listaID := range listasIDs {
		listasRegistrosMap[listaID] = &[]s.ListaCompartidaRegistro{}
	}

	type r = s.ListaCompartidaRegistro
	errGroup.Go(func() error {
		result := db.Select(func(q *db.Query[r], col r) {
			q.Where(col.EmpresaID_().Equals(1))
			q.Where(col.ListaID_().In(listasIDs...))
			if updated > 0 {
				q.Where(col.Updated_().GreaterThan(updated))
			} else {
				q.Where(col.Status_().Equals(1))
			}
		})

		core.Log("resultado obtenidos::", len(result.Records))

		return result.Err
	})

	err := errGroup.Wait()
	if err != nil {
		panic(err)
	}

	listasRegistros := []s.ListaCompartidaRegistro{}
	for _, registros := range listasRegistrosMap {
		listasRegistros = append(listasRegistros, *registros...)
	}

	core.Log("nro registros::", len(listasRegistros))

	return core.FuncResponse{}
}

func Test32(args *core.ExecArgs) core.FuncResponse {
	/*
		err1 := db.QueryExec(`DROP MATERIALIZED VIEW IF EXISTS genix.lista_compartida_registros__lista_id_view`)
		if err1 != nil {
			fmt.Println("error:", err1)
		}
	*/
	db.DeployScylla(0, s.ListaCompartidaRegistro{})
	return core.FuncResponse{}
}

type DemoType1[T DemoStruct1] struct {
	Table DemoStruct1
	ID    int32
}

func Test33(args *core.ExecArgs) core.FuncResponse {

	id := int32(1)

	uuid := core.SUnixTimeUUIDConcatID(id)

	core.Log(uuid)

	return core.FuncResponse{}
}
