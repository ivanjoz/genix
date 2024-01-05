package exec

import (
	"app/aws"
	"app/core"
	"app/facturacion"
	"app/types"
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"os"
	"time"
)

func TestScyllaDBConnection(args *core.ExecArgs) core.FuncResponse {
	usuarios := []types.Usuario{}
	if err := core.DBSelect(&usuarios); err != nil {
		panic(err)
	}

	core.Log("usuarios obtenidos:: ", len(usuarios))
	core.Print(usuarios)

	return core.FuncResponse{}
}

func TestScyllaDBInsert(args *core.ExecArgs) core.FuncResponse {
	counter, err := core.GetCounter("usuarios_1", 1)
	if err != nil {
		panic(err)
	}

	usuarios := []types.Usuario{
		{
			ID:          int32(counter),
			Nombres:     "Hola 2",
			Apellidos:   "Mundo 2",
			PerfilesIDs: []int32{2, 3, 4},
			Updated:     time.Now().Unix(),
			Created:     time.Now().Unix(),
		},
	}

	core.DBInsert(&usuarios)

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

	usuarios := []types.Usuario{}
	core.DBSelect(&usuarios)

	core.Print(usuarios)

	return core.FuncResponse{}
}

type DemoStruct1 struct {
	Hola  string  `json:"hola"`
	Value float32 `json:"value"`
}

type DemoStruct2 struct {
	Demo int32 `json:"demo"`
}

type DemoStruct struct {
	types.TAGS `table:"demo_structs"`
	CompanyID  int32       `json:"companyID" db:"company_id,pk"`
	ID         int32       `json:"id" db:"id,pk"`
	Edad       int32       `json:"edad" db:"edad,zx1,zx2"`
	Nombre     string      `json:"nombre" db:"nombre,zx1"`
	Palabras   []string    `json:"palabras" db:"palabras"`
	Peso       float32     `json:"peso" db:"peso"`
	Peso64     float64     `json:"peso64" db:"peso_64"`
	Rangos     []int32     `json:"rangos" db:"rangos"`
	Smallint   int16       `json:"small_int" db:"small_int,zx2"`
	Struct1    DemoStruct1 `json:"struct_1" db:"struct_1"`
	Struct2    DemoStruct2 `json:"struct_2" db:"struct_2"`
}

type DemoStruct4 struct {
	types.TAGS `table:"demo_structs"`
	CompanyID  int32    `json:"companyID" db:"company_id,pk"`
	ID         int32    `json:"id" db:"id,pk"`
	Edad       int32    `json:"edad" db:"edad,zx1,zx2"`
	Nombre     string   `json:"nombre" db:"nombre,zx1"`
	Palabras   []string `json:"palabras" db:"palabras"`
	Rangos     []int32  `json:"rangos" db:"rangos"`
	Smallint   int16    `json:"small_int" db:"small_int,zx2"`
	Peso       float32  `json:"peso" db:"peso"`
	Peso64     float64  `json:"peso64" db:"peso_64"`
}

func Test18(args *core.ExecArgs) core.FuncResponse {

	registros := []DemoStruct{}
	err := core.DBSelect(&registros).
		Where("company_id").Equals(1).Where("edad").GreatThan(20).Exec()

	if err != nil {
		panic(err)
	}

	core.Print(registros)

	return core.FuncResponse{}
}

func Test19(args *core.ExecArgs) core.FuncResponse {

	st1 := DemoStruct1{Hola: "hola", Value: 3.123}
	st2 := DemoStruct2{Demo: 555}

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
	}

	registros := []DemoStruct{demo}
	err = core.DBInsert(&registros)
	if err != nil {
		panic(err)
	}

	return core.FuncResponse{}
}

func Test20(args *core.ExecArgs) core.FuncResponse {

	words := []string{"dasd", "dasdasdad", "12312312", "dasdasd123123", "dqw986nwqd9a"}

	for _, w := range words {
		fmt.Println(core.BasicHashInt(w))
	}

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

	fmt.Println("Encoded Struct ", b.Bytes())

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

	for _, controller := range MakeScyllaControllers() {
		controller.InitTable(2)
	}

	return core.FuncResponse{}
}
