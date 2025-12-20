package core

import (
	"app/serialize"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DmitriyVTitov/size"
	"github.com/andybalholm/brotli"
	"github.com/aws/aws-lambda-go/events"
	"github.com/bytedance/sonic"
	"github.com/klauspost/compress/zstd"
)

type HandlerArgs struct {
	Body           *string
	ResponseWriter *http.ResponseWriter
	ReqContext     *http.Request
	Headers        map[string]string
	Query          map[string]string
	QueryString    string
	Method         string
	Route          string
	Authorization  string
	MergedID       int32
	ResponseBody   *[]byte
	ResponseError  string
	ReqParams      string
	Encoding       string
	Usuario        *IUsuario
	StartTime      int64
}

func PrintMemUsage() {
	/*
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		// memory := m.TotalAlloc / 1024 / 1024
		// Log("Memory TotalAlloc = %v MiB\n", memory)
		msg := fmt.Sprintf("nAlloc = %v MiB | TotalAlloc = %v | Sys = %v | NumGC = %v\n", m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)
		Log(msg)
	*/
}

func CompressBrotliOnFile(filePath string) []byte {

	fileInput, err := os.Open(filePath)
	if err != nil {
		panic("Error al abrir output file json. " + err.Error())
	}
	defer fileInput.Close()

	fileOutputPath := Env.TMP_DIR + "output.brotli"
	fmt.Println("output path:: ", fileOutputPath)

	fileOutput, err := os.Create(fileOutputPath)
	if err != nil {
		panic("Error al abrir output file brotli. " + err.Error())
	}
	defer fileOutput.Close()

	writer := brotli.NewWriterV2(fileOutput, 4)

	_, err = io.Copy(writer, fileInput)
	if err != nil {
		panic("Error al comprimir output file brotli. " + err.Error())
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}

	fileBytes, _ := os.ReadFile(fileOutputPath)
	if err := writer.Close(); err != nil {
		panic("Error al momento de comprimir la respuesta: al cerrar Buffer")
	}

	return fileBytes
}

func CompressGzipOnFile(filePath string) []byte {

	fileInput, err := os.Open(filePath)
	if err != nil {
		panic("Error al abrir output file gzip. " + err.Error())
	}
	defer fileInput.Close()

	fileOutputPath := Env.TMP_DIR + "output.gzip"
	fileOutput, err := os.Create(fileOutputPath)
	if err != nil {
		panic("Error al abrir output file json. " + err.Error())
	}
	defer fileOutput.Close()

	gzipWriter := gzip.NewWriter(fileOutput)
	defer gzipWriter.Close()

	// Copy the contents from the input file to the gzip writer
	_, err = io.Copy(gzipWriter, fileInput)
	if err != nil {
		panic("Error al comprimir output file gzip. " + err.Error())
	}

	err = gzipWriter.Flush()
	if err != nil {
		panic(err)
	}

	fileBytes, _ := os.ReadFile(fileOutputPath)

	return fileBytes
}

func DecompressBase64Gzip[T any](base64String *string, output *T) error {
	// Decode the base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(*base64String)
	if err != nil {
		return errors.New("Error al decodificar: " + err.Error())
	}

	// Create a reader from the decoded bytes
	reader := strings.NewReader(string(decodedBytes))

	// Create a GZIP reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return errors.New("Error al crear reader: " + err.Error())
	}
	defer gzipReader.Close()

	// Specify the output file path
	outputFilePath := Env.TMP_DIR + "request.json"

	// Create the output file
	uncompressedFile, err := os.Create(outputFilePath)
	if err != nil {
		return errors.New("Error al crear output file: " + err.Error())
	}
	defer uncompressedFile.Close()

	// Copy the contents of the gzip reader to the new file
	_, err = io.Copy(uncompressedFile, gzipReader)
	if err != nil {
		return errors.New("Error al descomprimir: " + err.Error())
	}

	// Open the JSON file
	file, err := os.Open(outputFilePath)
	if err != nil {
		return errors.New("Error abrir el descomprimido: " + err.Error())
	}
	defer file.Close()

	// Decode the JSON into the struct
	decoder := sonic.ConfigDefault.NewDecoder(file)
	err = decoder.Decode(&output)
	if err != nil {
		return errors.New("Error deserializar JSON: " + err.Error())
	}
	return nil
}

func DecompressBase64GzipM(base64String *string, isUrl ...bool) (string, error) {

	if len(isUrl) == 1 && isUrl[0] {
		str := *base64String
		str = strings.ReplaceAll(str, ".", "/")
		str = strings.ReplaceAll(str, "_", "=")
		str = strings.ReplaceAll(str, "-", "+")
		base64String = &str
	}

	// Decode the base64 string
	decodedBytes, err := base64.StdEncoding.DecodeString(*base64String)
	if err != nil {
		return "", errors.New("Error al decodificar: " + err.Error())
	}

	reader := bytes.NewReader(decodedBytes)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", errors.New("Error creating gzip reader: " + err.Error())
	}

	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		return "", errors.New("Error reading decompressed data: " + err.Error())
	}

	return string(decompressed), nil
}

func MakeResponseFinal(handlerResponse *HandlerResponse) *events.APIGatewayV2HTTPResponse {
	response := &events.APIGatewayV2HTTPResponse{}
	response.StatusCode = http.StatusOK
	response.Headers = handlerResponse.Headers
	if _, ok := response.Headers["Content-Type"]; !ok {
		response.Headers["Content-Type"] = "application/json; charset=utf-8"
	}

	PrintMemUsage()
	// Si es una respuesta que viene desde disco
	if len(handlerResponse.BodyOnDisk) > 0 {
		if strings.Contains(handlerResponse.Encoding, "br") {
			bodyBytes := CompressBrotliOnFile(handlerResponse.BodyOnDisk)
			response.Body = base64.StdEncoding.EncodeToString(bodyBytes)
			response.Headers["Content-Encoding"] = "br"
		} else {
			bodyBytes := CompressGzipOnFile(handlerResponse.BodyOnDisk)
			response.Body = base64.StdEncoding.EncodeToString(bodyBytes)
			response.Headers["Content-Encoding"] = "gzip"
		}
		response.IsBase64Encoded = true
		return response
	}

	body := *handlerResponse.Body
	handlerResponse.Body = nil

	isMaxLen := len(body) > 5*1000*1000
	Log("Len del body:: ", len(body))

	// revisa si la respuesta puede ser comprimida con brotli
	if strings.Contains(handlerResponse.Encoding, "br") {
		Log("Enviando respuesta comprimida con brotli")
		// Log(body)
		bodyBytes := body
		bodyCompressed := bytes.Buffer{}
		writer := brotli.NewWriterV2(&bodyCompressed, 4)
		in := bytes.NewReader(bodyBytes)
		n, err := io.Copy(writer, in)
		if err != nil {
			panic(err)
		}
		if int(n) != len(bodyBytes) {
			panic("Error al momento de comprimir la respuesta: size mismatch")
		}
		if err := writer.Close(); err != nil {
			panic("Error al momento de comprimir la respuesta: al cerrar Buffer")
		}
		response.Body = base64.StdEncoding.EncodeToString(bodyCompressed.Bytes())
		// Log(response.Body)
		response.Headers["Content-Encoding"] = "br"
		response.IsBase64Encoded = true
	} else if isMaxLen || strings.Contains(handlerResponse.Encoding, "gzip") {
		Log("Enviando respuesta comprimida con gzip")

		var bodyCompressed bytes.Buffer
		gz := gzip.NewWriter(&bodyCompressed)
		if _, err := gz.Write(body); err != nil {
			log.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			log.Fatal(err)
		}
		response.Body = base64.StdEncoding.EncodeToString(bodyCompressed.Bytes())
		fmt.Println("Len del body comprimido:: ", len(body))

		// Log(response.Body)
		response.Headers["Content-Encoding"] = "gzip"
		response.IsBase64Encoded = true
	} else {
		response.Body = string(body)
	}
	return response
}

type ErrorMsg struct {
	Error string `json:"error"`
}

func MakeErrRespFinal(statusCode int32, body string) *events.APIGatewayV2HTTPResponse {
	response := &events.APIGatewayV2HTTPResponse{}

	if statusCode == 500 {
		response.StatusCode = http.StatusInternalServerError
	} else {
		response.StatusCode = http.StatusBadRequest
	}
	response.Headers = make(map[string]string)
	// responseErr.Headers["Content-Type"] = "plain/text"
	response.Headers["Content-Type"] = "application/json; charset=utf-8"
	errorMsg := ErrorMsg{Error: body}
	responseJSON, _ := sonic.Marshal(errorMsg)
	response.Body = string(responseJSON)
	// Log("Error a enviar::", body)
	return response
}

func (e HandlerArgs) HasRol(rolesIDs ...int32) bool {
	if e.Usuario.ID == 0 {
		return false
	}
	rolesIDsInclude := MakeSliceInclude(rolesIDs)
	for _, rolID := range e.Usuario.RolesIDs {
		if rolesIDsInclude.Include(rolID) {
			return true
		}
	}
	return false
}

func (e HandlerArgs) HasAcceso(accesosIDs ...int32) bool {
	if e.Usuario.ID == 0 {
		return false
	}
	accesosIDsInclude := MakeSliceInclude(accesosIDs)
	for _, accesoID := range e.Usuario.AccesosIDs {
		if accesosIDsInclude.Include(accesoID) {
			return true
		}
	}
	return false
}

func (e HandlerArgs) IsUser(usuarioIDs ...int32) bool {
	usuariosIDsInclude := MakeSliceInclude(usuarioIDs)
	return usuariosIDsInclude.Include(e.Usuario.ID)
}

func (e HandlerArgs) GetQueryInt64(key string) int64 {
	if strVar, ok := e.Query[key]; ok {
		value, err := strconv.ParseInt(strVar, 10, 64)
		if err == nil {
			return value
		}
	}
	return 0
}

func (e HandlerArgs) GetQueryInt16(key string) int16 {
	if strVar, ok := e.Query[key]; ok {
		value, err := strconv.ParseInt(strVar, 10, 16)
		if err == nil {
			return int16(value)
		}
	}
	return 0
}

func (e HandlerArgs) GetQueryInt(key string) int32 {
	if strVar, ok := e.Query[key]; ok {
		value, err := strconv.Atoi(strVar)
		if err == nil {
			return int32(value)
		}
	}
	return 0
}

// Obtiene un parámetro como un slice de enteros desde un string separado por comas
func (e HandlerArgs) GetQueryIntSliceBase(key string, sep string) []int32 {
	intSlice := []int32{}
	if strVar, ok := e.Query[key]; ok && !(strVar == "" || strVar == "0") {
		if strings.Contains(key, "aplicacion") {
			Log(key, strVar[:4], strVar)
		}

		if len(strVar) > 4 && strVar[:4] == "gz--" {
			strVar = strVar[4:]
			newStrVar, err := DecompressBase64GzipM(&strVar, true)
			if err == nil {
				strVar = newStrVar
			} else {
				Log("No se pudo descomprimir:: ", strVar)
			}
		}

		values := strings.Split(strVar, sep)
		for _, value := range values {
			if len(value) == 0 {
				continue
			}
			valueInt, err := strconv.Atoi(value)
			if err == nil {
				intSlice = append(intSlice, int32(valueInt))
			} else {
				Log("No es un número:: ", value)
			}
		}
	}
	return intSlice
}

func (e HandlerArgs) GetQueryIntSlice(key string) []int32 {
	return e.GetQueryIntSliceBase(key, ",")
}

func (e HandlerArgs) GetQueryIntSliceB64(key string) []int32 {
	content := e.GetQuery(key)
	if len(content) == 0 {
		return []int32{}
	}
	content = strings.ReplaceAll(content, "-", "+")
	content = strings.ReplaceAll(content, "_", "=")
	content = strings.ReplaceAll(content, ".", "/")
	decompressed, err := DecompressBase64GzipM(&content)
	if err != nil {
		Log(err)
		return []int32{}
	}

	intSlice := []int32{}
	for _, value := range strings.Split(decompressed, ",") {
		if len(value) == 0 || value == "0" {
			continue
		}
		valueInt, err := strconv.Atoi(value)
		if err == nil {
			intSlice = append(intSlice, int32(valueInt))
		}
	}
	return intSlice
}

func (e HandlerArgs) GetQueryIntPairsBase(key string, separator string) [][2]int32 {

	intsPairs := [][2]int32{}

	if strVar, ok := e.Query[key]; ok {
		values := strings.Split(strVar, ",")
		for _, value := range values {
			if !strings.Contains(value, separator) {
				Log("El string: " + value + " no contiene un ',' para poder separarlo")
				return intsPairs
			}
			valueInts := strings.Split(value, separator)

			intsPairs = append(intsPairs, [2]int32{SrtToInt32(valueInts[0]), SrtToInt32(valueInts[1])})
		}
	}
	return intsPairs
}

func (e HandlerArgs) GetQueryIntPairs(key string) [][2]int32 {
	return e.GetQueryIntPairsBase(key, ".")
}

func (e HandlerArgs) GetQueryInt64PairsBase(key string, separator string) [][2]int64 {

	intsPairs := [][2]int64{}

	if strVar, ok := e.Query[key]; ok {
		values := strings.Split(strVar, ",")
		for _, value := range values {
			if !strings.Contains(value, separator) {
				Log("El string: " + value + " no contiene un ',' para poder separarlo")
				return intsPairs
			}
			valueInts := strings.Split(value, separator)
			value1, err1 := strconv.ParseInt(valueInts[0], 10, 64)
			value2, err2 := strconv.ParseInt(valueInts[1], 10, 64)

			if err1 != nil || err2 != nil {
				Log("El valor " + valueInts[0] + " o el " + valueInts[1] + " no se pudieron convertir a int")
				return intsPairs
			}
			intsPairs = append(intsPairs, [2]int64{value1, value2})
		}
	}
	return intsPairs
}

func (e HandlerArgs) GetQueryInt64Pairs(key string) [][2]int64 {
	return e.GetQueryInt64PairsBase(key, ".")
}

// Obtiene un parámetro del query como un slice de 3 int32 (trio de enteros)
func (e HandlerArgs) GetQueryIntThreeBase(key string, separator string) [][3]int32 {
	intsPairs := [][3]int32{}

	if strVar, ok := e.Query[key]; ok {
		values := strings.Split(strVar, ",")
		for _, value := range values {
			if !strings.Contains(value, separator) {
				Log("El string: " + value + " no contiene un ',' para poder separarlo")
				return intsPairs
			}
			valueInts := strings.Split(value, separator)

			intsPairs = append(intsPairs, [3]int32{SrtToInt32(valueInts[0]), SrtToInt32(valueInts[1]), SrtToInt32(valueInts[2])})
		}
	}
	return intsPairs
}

func (e HandlerArgs) GetQueryIntThree(key string) [][3]int32 {
	return e.GetQueryIntThreeBase(key, ".")
}

// Obtiene un parámetro del query como un string
func (e HandlerArgs) GetQuery(key string) string {
	if strVar, ok := e.Query[key]; ok {
		return strVar
	}
	return ""
}

// Obtiene un parámetro del query como un slice de strings
func (e HandlerArgs) GetQuerySlice(key string) []string {
	if strVar, ok := e.Query[key]; ok {
		srtVarSlice := strings.Split(strVar, ",")
		srtVarSliceFiltered := []string{}
		for _, elm := range srtVarSlice {
			if len(elm) > 0 {
				srtVarSliceFiltered = append(srtVarSliceFiltered, elm)
			}
		}
		return srtVarSliceFiltered
	}
	return []string{}
}

type HandlerResponse struct {
	Body       *[]byte
	BodyOnDisk string
	StatusCode int
	Error      string
	Encoding   string
	Headers    map[string]string
	Route      string
	MergeID    int32
}

type MainResponse struct {
	LambdaResponse *events.APIGatewayV2HTTPResponse
	Error          error
}

func (req *HandlerArgs) MakeErrCode(message string, code int32) HandlerResponse {
	response := HandlerResponse{Headers: makeHeaders()}

	if req.MergedID > 0 {
		req.ResponseError = message
	}

	response.Error = message
	response.Route = req.Route
	response.MergeID = req.MergedID
	Log("Req Error:: ", message)

	if code == 400 {
		response.StatusCode = http.StatusBadRequest
	} else if code == 401 {
		response.StatusCode = http.StatusUnauthorized
	} else if code == 500 {
		response.StatusCode = http.StatusInternalServerError
	}
	return response
}

func (req *HandlerArgs) MakeErr(message ...any) HandlerResponse {
	return req.MakeErrCode(Concat(" ", message...), 400)
}

func (req *HandlerArgs) MakeErr401(message ...any) HandlerResponse {
	return req.MakeErrCode(Concat(" ", message...), 401)
}

func (req *HandlerArgs) MakeErr500(message ...any) HandlerResponse {
	return req.MakeErrCode(Concat(" ", message...), 500)
}

func makeHeaders() map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
	return headers
}

// Crea una respuesta serializando un struct
func (req *HandlerArgs) MakeResponse(respStruct any) HandlerResponse {
	return MakeResponse(req, &respStruct)
}

func MakeResponse[T any](req *HandlerArgs, respStruct *T) HandlerResponse {

	response := HandlerResponse{
		StatusCode: http.StatusOK,
		Headers:    makeHeaders(),
		Route:      req.Route,
		MergeID:    req.MergedID,
	}

	structLen := size.Of(*respStruct)
	// Si es menor a 100kb entonces lo serializa aquí
	if fmt.Sprintf("%T", *new(T)) == "string" {
		body := []byte(fmt.Sprintf("%v", *respStruct))
		response.Body = &body
	} else if structLen < 102400 || Env.IS_LOCAL {
		/*
			marshall1, _ := serialize.Marshal(respStruct)
			fmt.Println(string(marshall1))
		*/
		bodyBytes, err := serialize.Marshal(respStruct)
		fmt.Println(string(bodyBytes))
		// fmt.Println("Json Size:", len(bodyBytes), "| vs:", len(marshall1))

		if err != nil {
			return req.MakeErr("No se pudo serializar respuesta:", err)
		}
		if bytes.Equal(bodyBytes, []byte("null")) {
			bodyBytes = []byte("[]")
		}
		response.Body = &bodyBytes
	} else {
		fileName := fmt.Sprintf("output-%v", req.MergedID)
		response.BodyOnDisk = EncodeJsonToFileX(respStruct, fileName)
	}

	return response
}

func CombineResponses(responses []*HandlerResponse) HandlerResponse {
	// Crea un archivo
	flags := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	outputJsonPath := Env.TMP_DIR + "output.json"
	outputJson, err := os.OpenFile(outputJsonPath, flags, os.ModePerm)
	if err != nil {
		panic("Error opening output.json: " + err.Error())
	}
	defer outputJson.Close()

	_, err = outputJson.WriteString("[")
	if err != nil {
		panic("Error appending line to output.json: " + err.Error())
	}

	for i, res := range responses {
		outputJson.WriteString("\n")
		header := `{"id":%v, "route": "%v", "statusCode": %v, "message": "%v", "body": `
		header = fmt.Sprintf(header, res.MergeID, res.Route, res.StatusCode, res.Error)

		if len(res.BodyOnDisk) > 0 {
			outputJson.WriteString(header)
			file, err := os.Open(res.BodyOnDisk)
			if err != nil {
				panic("Error opening body (response.json) " + err.Error())
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 5*1024*1024)

			size := 0
			linesCount := 0
			for scanner.Scan() {
				line := scanner.Text()
				linesCount++
				size += len(line)
				_, err := outputJson.WriteString(line)
				if err != nil {
					panic("Error writing to output.json: " + err.Error())
				}
			}
			fileName := strings.ReplaceAll(res.BodyOnDisk, Env.TMP_DIR, "")
			Log("Agregando a Body:: ", fileName, " | Lines:", linesCount, " | Size:", size)
			outputJson.WriteString("}\n")
		} else if res.Body != nil {
			outputJson.WriteString(header)
			outputJson.Write(*res.Body)
			outputJson.WriteString("}\n")
		} else if len(res.Error) > 0 {
			header += "null }\n"
			outputJson.WriteString(header)
		} else {
			Print(*res)
			panic("response invalid")
		}
		if i < (len(responses) - 1) {
			outputJson.WriteString(",")
		}
	}

	outputJson.WriteString("]")

	response := HandlerResponse{
		BodyOnDisk: outputJsonPath,
		Headers: map[string]string{
			"Content-Type": "application/json; charset=utf-8",
		},
		StatusCode: http.StatusOK,
	}
	return response
}

func EncodeJsonToFileX[T any](respStruct *T, name ...string) string {

	fileName := "output"
	if len(name) == 1 && len(name[0]) > 0 {
		fileName = name[0]
	}

	outputPath := Env.TMP_DIR + fileName + ".json"
	file, err := os.Create(outputPath)
	if err != nil {
		panic("Error al crear el output.json" + err.Error())
	}
	defer file.Close()

	encoder := sonic.ConfigDefault.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	PrintMemUsage()

	err = encoder.Encode(respStruct)
	if err != nil {
		panic("Error al generar el JSON en el output.json" + err.Error())
	}

	PrintMemUsage()
	return outputPath
}

func (req *HandlerArgs) MakeResponseDisk(respStruct any) HandlerResponse {

	if req.MergedID > 0 {
		panic("No se puede pre-almacenar la respuesta en una merged-API")
	}

	outputPath := Env.TMP_DIR + "output.json"
	file, err := os.Create(outputPath)
	if err != nil {
		panic("Error al crear el output.json" + err.Error())
	}
	defer file.Close()
	// TODO: aqui hay un salto de memoria, revisar
	encoder := sonic.ConfigDefault.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	PrintMemUsage()

	err = encoder.Encode(respStruct)
	if err != nil {
		panic("Error al generar el JSON en el output.json" + err.Error())
	}

	PrintMemUsage()

	fi, err := os.Stat(outputPath)
	if err == nil {
		size := fi.Size()
		fmt.Println("Respuesta en:: ", outputPath, " | Size (kb): ", float64(size)/1000)
		fmt.Println("File exists")
	} else if os.IsNotExist(err) {
		panic("File does not exist")
	} else {
		panic("Error occurred while checking file:" + err.Error())
	}

	response := HandlerResponse{
		BodyOnDisk: outputPath,
		StatusCode: http.StatusOK,
		Headers:    makeHeaders(),
	}

	return response
}

func (req *HandlerArgs) MakeResponseDiskT(outputPath string) HandlerResponse {

	if req.MergedID > 0 {
		panic("No se puede pre-almacenar la respuesta en una merged-API")
	}

	PrintMemUsage()

	_, err := os.Stat(outputPath)
	if err == nil {
		fmt.Println("File exists")
	} else if os.IsNotExist(err) {
		panic("File does not exist")
	} else {
		panic("Error occurred while checking file:" + err.Error())
	}

	response := HandlerResponse{
		BodyOnDisk: outputPath,
		StatusCode: http.StatusOK,
		Headers:    makeHeaders(),
	}

	return response
}

func (req *HandlerArgs) MakeResponsePlain(body *[]byte) HandlerResponse {
	response := HandlerResponse{
		Body:       body,
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/plain; charset=utf-8",
		},
	}

	if req.MergedID > 0 {
		req.ResponseBody = body
	}

	return response
}

func SendLocalResponse(args HandlerArgs, response HandlerResponse) {
	respWriter := *args.ResponseWriter
	respWriter.Header().Set("Access-Control-Allow-Origin", "*")

	PrintMemUsage()
	// Setea los headers de la respuesta
	if response.Headers != nil {
		for key, value := range response.Headers {
			respWriter.Header().Set(key, value)
		}
	}
	if len(respWriter.Header().Get("Content-Type")) == 0 {
		respWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	var bodyBytes []byte

	// Revisa si hay que enviar error
	if len(response.Error) > 0 {
		respWriter.WriteHeader(http.StatusBadRequest)
		errorMap := map[string]string{
			"error": response.Error,
		}
		errorJson, _ := sonic.Marshal(errorMap)

		Log("enviando error:: ", response.Error)
		_, err := respWriter.Write(errorJson)
		if err != nil {
			Log("Hubo un error al enviar la respuesta::", err)
		}
		return
	}

	// Envía respuesta ok
	if response.Body == nil {
		Log("El body es nil!")
	} else if strings.Contains(args.Encoding, "zstd") {
		Log("Comprimiendo body con: zstd")
		encoder, _ := zstd.NewWriter(nil)
		bb := *response.Body
		bodyBytes = encoder.EncodeAll(bb, make([]byte, 0, len(bb)))
		respWriter.Header().Set("Content-Encoding", "zstd")
	} else {
		Log("Comprimiendo body con: gzip")
		var bodyCompressed bytes.Buffer
		gz := gzip.NewWriter(&bodyCompressed)
		if _, err := gz.Write(*response.Body); err != nil {
			log.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			log.Fatal(err)
		}

		bodyBytes = bodyCompressed.Bytes()
		respWriter.Header().Set("Content-Encoding", "gzip")
	}

	elapsed := time.Now().UnixMilli() - args.StartTime
	serverInfo := fmt.Sprintf("Genix-v1.0:%v", elapsed)
	respWriter.Header().Set("Server", serverInfo)

	/*
		bodyLen := 240
		if len(bodyBytes) < bodyLen {
			bodyLen = len(bodyBytes) - 1
		}
		Log("Body:", response.Route)
		Log(string(bodyBytes[0:bodyLen]))
	*/
	respWriter.Write(bodyBytes)
}

// TODO: deprecar después
type MergedRoute struct {
	Id       int32             `json:"id"`
	FuncPath string            `json:"funcPath"`
	Route    string            `json:"route"`
	Query    map[string]string `json:"query"`
}

type MergedResponse struct {
	Route      string `json:"route"`
	Id         int32  `json:"id"`
	Body       string `json:"body"`
	StatusCode int32  `json:"statusCode"`
	Message    string `json:"message"`
}

func ParseMergedUri(query map[string]string) []MergedRoute {
	mapOfRoutes := map[int]MergedRoute{}
	mapOfRoutesValues := map[string]string{}

	for key, value := range query {
		// si es una ruta
		if key[0:3] == "i--" {
			routeID, err := strconv.Atoi(key[3:])
			if err != nil {
				Log("Hubo un error al convertir ", key[3:], " en int")
			}
			mapOfRoutes[routeID] = MergedRoute{
				Id:       int32(routeID),
				Route:    value,
				FuncPath: "GET." + value,
				Query:    map[string]string{},
			}
		} else if strings.Contains(key, "--") {
			mapOfRoutesValues[key] = value
		}
	}

	for key, value := range mapOfRoutesValues {
		routeIDParamName := strings.Split(key, "--")
		routeID, err := strconv.Atoi(routeIDParamName[0])
		if err != nil {
			Log("Hubo un error al convertir ", key[3:], " en int")
		}
		mergedRoute := mapOfRoutes[routeID]
		paramName := routeIDParamName[1]
		mergedRoute.Query[paramName] = value
	}

	mergedRoutes := []MergedRoute{}
	for _, value := range mapOfRoutes {
		mergedRoutes = append(mergedRoutes, value)
	}

	return mergedRoutes
}
