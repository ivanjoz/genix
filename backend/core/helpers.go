package core

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/kr/pretty"
	"github.com/martinlindhe/base36"
	"github.com/mashingan/smapping"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/rodaine/table"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/exp/constraints"
)

func Print(Struct any) {
	if Env != nil && !Env.IS_LOCAL {
		return
	}
	pretty.Println(Struct)
}

func Filter[T any](slice []T, f func(T) bool) []T {
	var n []T
	for _, e := range slice {
		if f(e) {
			n = append(n, e)
		}
	}
	return n
}

func Find[T any](slice []T, f func(T) bool) *T {
	for _, e := range slice {
		if f(e) {
			return &e
		}
	}
	return nil
}

func BTreeSearch[T any, E Number1](valueToFind E, records []T, f func(T) E) int {
	i := sort.Search(len(records), func(i int) bool {
		return f(records[i]) >= valueToFind
	})
	if i < len(records) || i >= len(records) {
		return -1
	}
	return i
}

func BTreeSearchX[T any, E Number1](valueToFind E, records []T, f func(T) E) int {
	i := sort.Search(len(records), func(i int) bool {
		return f(records[i]) >= valueToFind
	})
	return i
}

func BTreeBetweenSearch[T any, E Number1](vStart E, vEnd E, records []T, f func(T) E) []T {

	ln := len(records)
	idx_1 := BTreeSearchX(vStart, records, f)
	idx_2 := BTreeSearchX(vEnd, records, f)
	// Log("Indexes:: ", idx_1, " : ", idx_2, " | Len=", len(records))

	base := []T{}
	if idx_2 < ln && f(records[idx_2]) == vEnd {
		base = append(base, records[idx_2])
	}

	if idx_1 > ln || idx_2 < 0 {
		return []T{}
	} else if idx_1 <= 0 && idx_2 >= ln {
		return records
	} else if idx_2 >= ln {
		// Log("estamos aqui.. 1")
		return records[idx_1:]
	} else if idx_1 <= 0 {
		// Log("estamos aqui.. 2")
		return append(records[0:idx_2], base...)
	} else {
		// Log("estamos aqui.. 3")
		return append(records[idx_1:idx_2], base...)
	}
}

type NumberStr interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64 | string
}
type NumberStr2 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64 | float32 | float64 | string
}
type Number1 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64
}
type Number2 interface {
	int | int32 | int8 | uint8 | int16 | uint16 | int64 | float32 | float64
}
type Int16 interface {
	int | int16 | uint16
}
type NumFloat interface {
	float32 | float64
}

// Merge un slice con otro si el perimero no posee un valor del otro
func MergeSliceUnique[T NumberStr](slice1 *[]T, slice2 *[]T) {

	for _, value := range *slice2 {
		isNotIncluded := true
		for _, value2 := range *slice1 {
			if value == value2 {
				isNotIncluded = false
				break
			}
		}
		if isNotIncluded {
			*slice1 = append(*slice1, value)
		}
	}
}

func MakePgIntArray[T Number2](slice1 []T) string {
	sliceOfStrings := []string{}

	for _, value := range slice1 {
		sliceOfStrings = append(sliceOfStrings, fmt.Sprintf("%v", value))
	}
	return "{" + strings.Join(sliceOfStrings, ",") + "}"
}

func MakeUniqueInts[T any, N Number1](slice []T, f func(T) N) []N {
	keys := map[N]bool{}
	list := []N{}

	for _, value := range slice {
		valueInt := f(value)
		if valueInt == 0 {
			continue
		}
		if _, isHere := keys[valueInt]; !isHere {
			keys[valueInt] = true
			list = append(list, valueInt)
		}
	}

	return list
}

func MakeUniqueIntsInclude[T any, N Number1](slice []T, f func(T) N) SliceInclude[N] {
	values := MakeUniqueInts(slice, f)

	reg := SliceInclude[N]{
		Values:    values,
		valuesMap: map[N]bool{},
	}
	for _, value := range values {
		reg.valuesMap[value] = true
	}
	return reg
}

func MakeSliceInclude[N NumberStr](values []N) SliceInclude[N] {
	reg := SliceInclude[N]{
		Values:    []N{},
		valuesMap: map[N]bool{},
	}
	for _, value := range values {
		if _, ok := reg.valuesMap[value]; !ok {
			reg.valuesMap[value] = true
			reg.Values = append(reg.Values, value)
		}
	}
	return reg
}

type SliceInclude[T NumberStr] struct {
	Values    []T
	valuesMap map[T]bool
}

func (e *SliceInclude[T]) Add(value T) {
	if e.valuesMap == nil {
		e.valuesMap = map[T]bool{}
	}
	if _, ok := e.valuesMap[value]; !ok {
		e.Values = append(e.Values, value)
		e.valuesMap[value] = true
	}
}

func (e *SliceInclude[T]) AddIf(value T) {
	if e.valuesMap == nil {
		e.valuesMap = map[T]bool{}
	}

	isEmpty := false

	switch any(value).(type) {
	case string:
		isEmpty = any(value).(string) == ""
	case int32:
		isEmpty = any(value).(int32) == 0
	case int:
		isEmpty = any(value).(int) == 0
	case int16:
		isEmpty = any(value).(int16) == 0
	case int64:
		isEmpty = any(value).(int64) == 0
	}

	if isEmpty {
		return
	}

	if _, ok := e.valuesMap[value]; !ok {
		e.Values = append(e.Values, value)
		e.valuesMap[value] = true
	}
}

func (e *SliceInclude[T]) Include(id T) bool {
	if _, ok := e.valuesMap[id]; ok {
		return true
	} else {
		return false
	}
}

// Si el slice no contiene valores devuelve TRUE
func (e *SliceInclude[T]) IncludeN(id T) bool {
	if len(e.Values) == 0 {
		return true
	}
	return e.Include(id)
}

func SliceExtract[T any, N Number2](slice []T, f func(T) N) []N {
	arr := []N{}
	for _, value := range slice {
		arr = append(arr, f(value))
	}
	return arr
}

func MakeUnique[T NumberStr](slice []T) []T {
	keys := map[T]bool{}
	list := []T{}
	for _, valueInt := range slice {
		if _, isHere := keys[valueInt]; !isHere {
			keys[valueInt] = true
			list = append(list, valueInt)
		}
	}
	return list
}

func Unref[T any](value *T) T {
	if value == nil {
		return *(new(T))
	}
	return *value
}

func Floor[T NumFloat](value T) int32 {
	return int32(math.Floor(float64(value)))
}
func Ceil[T NumFloat](value T) int32 {
	return int32(math.Ceil(float64(value)))
}

func SliceUnref[T any](sourceSlice []*T) []T {
	newSlice := []T{}
	for _, e := range sourceSlice {
		newSlice = append(newSlice, *e)
	}
	return newSlice
}

func AppendIfUnique[T NumberStr](slice []T, e T) []T {
	for _, valueInt := range slice {
		if valueInt == e {
			return slice
		}
	}
	return append(slice, e)
}

func Base64ToBytes(content string) []byte {
	contentBytes, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		fmt.Println("Error en desencriptar Base64: ", err)
		return []byte{}
	}
	return contentBytes
}

func BytesToBase64(source []byte, useUrlEncoded ...bool) string {
	contentString := base64.StdEncoding.EncodeToString(source)
	if len(useUrlEncoded) == 1 && useUrlEncoded[0] {
		contentString = MakeB64UrlEncode(contentString)
	}
	return contentString
}

func Base64ToStruct[T any](base64Str *string, target *T) error {
	contentBytes := Base64ToBytes(*base64Str)
	err := json.Unmarshal(contentBytes, target)
	if err != nil {
		return err
	}
	return nil
}

func MsgPEncode(msg any) ([]byte, error) {
	var buffer bytes.Buffer
	msgEncoder := msgpack.NewEncoder(&buffer)
	msgEncoder.SetCustomStructTag("ms")
	msgEncoder.SetOmitEmpty(true)
	msgEncoder.UseCompactInts(true)
	msgEncoder.UseCompactFloats(true)
	// msgEncoder.UseArrayEncodedStructs(true)
	// msgEncoder.UseInternedStrings(true)
	err := msgEncoder.Encode(msg)

	return buffer.Bytes(), err
}

func MsgPDecode(msgBytes []byte, msg any) error {
	buffer := bytes.NewBuffer(msgBytes)
	msgDecoder := msgpack.NewDecoder(buffer)
	msgDecoder.SetCustomStructTag("ms")

	err := msgDecoder.Decode(msg)
	if err != nil {
		Log("Error decoding MsgPack: ", err)
		return err
	}
	return nil
}

func Err(content ...any) error {
	errMessage := Concatx(" ", content)
	Log(errMessage)
	return errors.New(errMessage)
}

func MapToSlice[T any, N NumberStr](map1 map[N]*T) []T {
	slice1 := []T{}
	for _, value := range map1 {
		slice1 = append(slice1, *value)
	}
	return slice1
}

func MapToSliceT[T any, N NumberStr](map1 map[N]T) []T {
	slice1 := []T{}
	for _, value := range map1 {
		slice1 = append(slice1, value)
	}
	return slice1
}

func ToSrt(d float64) string {
	return strconv.FormatFloat(d, 'f', -1, 64)
}

func ToSrtN(content any, lenght int, prefix string) string {
	srt := fmt.Sprintf("%v", content)
	for len(srt) < lenght {
		srt = prefix + srt
	}
	return srt
}

func Sort[T constraints.Ordered](s []T) {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}

func SortSlice[T any, N Number2](
	s []T, mode int8, GetIndex func(e T) N,
) {
	if mode == 1 { // Ascendente: 1,2,3,4,5
		sort.Slice(s, func(i, j int) bool {
			return GetIndex(s[i]) < GetIndex(s[j])
		})
	} else if mode == 2 { // Descendente 5,4,3,2,1
		sort.Slice(s, func(i, j int) bool {
			return GetIndex(s[i]) > GetIndex(s[j])
		})
	}
}

func CompareSlice[T NumberStr](slice1 []T, slice2 []T) bool {
	Sort(slice1)
	Sort(slice2)
	return Concatn(slice1) == Concatn(slice2)
}

func Concat(sep string, slice1 ...any) string {
	sliceOfStrings := []string{}
	for _, value := range slice1 {
		sliceOfStrings = append(sliceOfStrings, fmt.Sprintf("%v", value))
	}
	return strings.Join(sliceOfStrings, sep)
}

func Concats(slice1 ...any) string {
	return Concat(" ", slice1...)
}

func Concatx[T any](sep string, slice1 []T) string {
	sliceOfStrings := []string{}
	for _, value := range slice1 {
		sliceOfStrings = append(sliceOfStrings, fmt.Sprintf("%v", value))
	}
	return strings.Join(sliceOfStrings, sep)
}

func Concatn(slice1 ...any) string {
	return Concat("_", slice1...)
}

func IntToPointer[T Number1](num T) *T {
	if num == 0 {
		return nil
	}
	return &num
}

func SliceToMap[T any, N NumberStr](slice []T, makeKey func(T) N) map[N][]*T {
	sliceMap := map[N][]*T{}

	for idx := range slice {
		key := makeKey(slice[idx])

		if slice1, ok := sliceMap[key]; ok {
			sliceMap[key] = append(slice1, &slice[idx])
		} else {
			sliceMap[key] = []*T{&slice[idx]}
		}
	}
	return sliceMap
}

func SliceToMapP[T any, N NumberStr](slice []T, makeKey func(T) N) map[N][]T {
	sliceMap := map[N][]T{}

	for idx := range slice {
		key := makeKey(slice[idx])

		if slice1, ok := sliceMap[key]; ok {
			sliceMap[key] = append(slice1, slice[idx])
		} else {
			sliceMap[key] = []T{slice[idx]}
		}
	}
	return sliceMap
}

func SliceToMapK[T any, N NumberStr](slice []T, makeKey func(T) N) map[N]*T {
	sliceMap := map[N]*T{}
	for idx := range slice {
		key := makeKey(slice[idx])
		sliceMap[key] = &slice[idx]
	}
	return sliceMap
}

func SliceToMapE[T any, N NumberStr](slice []T, makeKey func(T) N) map[N]T {
	sliceMap := map[N]T{}
	for idx := range slice {
		key := makeKey(slice[idx])
		sliceMap[key] = slice[idx]
	}
	return sliceMap
}

func Reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func Contains[T NumberStr](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func Containx[T NumberStr](e T, s ...T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func RemoveSpaces(input string) string {
	noSpaces := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, input)

	return noSpaces
}

func GetFechaHoraGrupo(fechaHora int64) int32 {
	return int32(math.Floor(float64(fechaHora) / float64(1800)))
}

func GetFechaHoraUsuarioGrupo(fechaHora int64, usuarioID int32) int32 {
	fechaHoraGrupo := int32(math.Floor(float64(fechaHora) / float64(1800)))
	return fechaHoraGrupo*1000 + usuarioID
}

func GetBase36Time(idx int64) string {
	now := time.Now().UnixMilli() + idx
	return base36.Encode(uint64(now))
}

// Compresión con Zstd
func CompressZstd(content *string) []byte {
	encoder, _ := zstd.NewWriter(nil)
	src := []byte(*content)
	compressed := encoder.EncodeAll(src, make([]byte, 0, len(src)))
	return compressed
}

func DecompressZstd(content *[]byte) string {
	decoder, _ := zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	decompressed, err := decoder.DecodeAll(*content, nil)
	if err != nil {
		Log("Error al descomprimir: " + err.Error())
		return ""
	}
	return string(decompressed)
}

// Compresión con Brotli
func CompressBrotli(content *string, quality int) ([]byte, error) {
	contentBytes := []byte(*content)
	contentCompressed := bytes.Buffer{}

	writer := brotli.NewWriterOptions(&contentCompressed, brotli.WriterOptions{Quality: quality})

	reader := bytes.NewReader(contentBytes)
	bodySize, err := io.Copy(writer, reader)
	if err != nil {
		return nil, errors.New("Error al descomprimir: " + err.Error())
	}

	if int(bodySize) != len(contentBytes) {
		return nil, errors.New("error al descomprimir: size mismatch")
	}
	if err := writer.Close(); err != nil {
		return nil, errors.New("error al descomprimir: no se pudo cerrar el writer")
	}

	return contentCompressed.Bytes(), nil
}

func CompressBrotli64(content *string, quality int) (string, error) {
	compressedBytes, err := CompressBrotli(content, quality)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(compressedBytes), nil
}

func DecompressBrotli(content *[]byte) string {
	brotliReader := brotli.NewReader(bytes.NewBuffer(*content))
	contentUncompressed, err := io.ReadAll(brotliReader)

	if err != nil {
		Log("Error al descomprimir: " + err.Error())
		return ""
	}
	return string(contentUncompressed)
}

func DecompressGzip(content *[]byte) string {
	// Create a reader from the compressed data
	reader := bytes.NewReader(*content)

	// Create a GZIP reader
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		fmt.Println("Error al crear reader: " + err.Error())
		return ""
	}
	defer gzipReader.Close()

	// Read the decompressed data
	decompressedData, err := io.ReadAll(gzipReader)
	if err != nil {
		fmt.Println("Error al descomprimir: " + err.Error())
		return ""
	}

	return string(decompressedData)
}

func DecompressBrotli64(content *string) string {
	compressedBytes, err := base64.StdEncoding.DecodeString(*content)
	if err != nil {
		panic("Error convertir []bytes a base64: " + err.Error())
	}
	return DecompressBrotli(&compressedBytes)
}

func DecompressGzip64(content *string) string {
	compressedBytes, err := base64.StdEncoding.DecodeString(*content)
	if err != nil {
		panic("Error convertir []bytes a base64: " + err.Error())
	}
	return DecompressGzip(&compressedBytes)
}

func NormaliceStringT(content string) string {
	return NormaliceString(&content)
}

func NormaliceString(content *string) string {

	content2 := strings.ToLower(*content)
	finalRunes := []rune{}

	runesReplaceMap := map[rune]rune{
		225: 97, 233: 101, 237: 105, 243: 111, 250: 117,
	}

	for _, rune := range content2 {
		// numeros y letras en minúscula
		if (rune > 47 && rune < 58) || (rune > 96 && rune < 123) || rune == 95 {
			finalRunes = append(finalRunes, rune)
		} else if rune == 32 || rune == 160 || rune == 45 || rune == 95 { // space
			if len(finalRunes) > 0 && finalRunes[len(finalRunes)-1] != 95 {
				finalRunes = append(finalRunes, 95) // _ (low line)
			}
		} else {
			if newRune, ok := runesReplaceMap[rune]; ok {
				finalRunes = append(finalRunes, newRune)
			}
		}
	}
	// Log("string normalizado:: ", *content, " - ", string(finalRunes))
	return string(finalRunes)
}

func ToBase36(num int64) string {
	if num == 0 {
		num = time.Now().UnixMilli()
	}
	strValue := strings.ToLower(big.NewInt(num).Text(36))
	return strValue
}

func ToBase32cf(num int32) string {
	strValue := strings.ToUpper(big.NewInt(int64(num)).Text(36))
	strValue = strings.ReplaceAll(strValue, "o", "x")
	strValue = strings.ReplaceAll(strValue, "1", "y")
	strValue = strings.ReplaceAll(strValue, "0", "z")
	return strValue
}

func Base32cfToInt(strValue string) int32 {
	strValue = strings.TrimRight(strValue, "=")
	strValue = strings.ReplaceAll(strValue, "x", "o")
	strValue = strings.ReplaceAll(strValue, "y", "1")
	strValue = strings.ReplaceAll(strValue, "z", "0")

	decodedBytes, err := base32.StdEncoding.DecodeString(strValue)
	if err != nil {
		return 0
	}
	num := *new(big.Int).SetBytes(decodedBytes)
	return int32(num.Int64())
}

func ToBase36s(num int32) string {
	strValue := strings.ToLower(big.NewInt(int64(num)).Text(36))
	return strValue
}

func Base36ToInt(str string) int64 {
	str = strings.ToUpper(str)
	value, err := strconv.ParseInt(str, 36, 64)
	if err != nil {
		Log("No se pudo convertir de base36 a int: ", str)
		return 0
	}
	return value
}

func BasicHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
func BasicHashInt(s string) int32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int32(h.Sum32())
}

func Base64MD5Hash(content string, len int32) string {
	hash := md5.Sum([]byte(content))
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	encoded = strings.ReplaceAll(encoded, "/", "-")[2 : len+2]
	return encoded
}

func IntToBase64(number int64, maxLen int8) string {
	byteData := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteData, uint64(number))
	base64Str := base64.StdEncoding.EncodeToString(byteData)

	if maxLen > 0 {
		base64Str = strings.ReplaceAll(base64Str, "=", "")
		base64Str = strings.ReplaceAll(base64Str, "/", "-")
		base64Str = base64Str[:(int(maxLen))]
	}
	return base64Str
}

func FnvHashStringBase(input string, isBase64 bool, intSize int, strSize ...int) string {
	_strSize := 0
	if len(strSize) > 0 {
		_strSize = strSize[0]
	}

	var hashBytes []byte
	hashString := ""
	if intSize == 32 {
		hash := fnv.New32a()
		hash.Write([]byte(input))
		hashBytes = hash.Sum(nil)
	} else if intSize == 64 {
		hash := fnv.New64a()
		hash.Write([]byte(input))
		hashBytes = hash.Sum(nil)
	} else {
		hash := fnv.New128a()
		hash.Write([]byte(input))
		hashBytes = hash.Sum(nil)
	}

	if isBase64 {
		hashString = base64.StdEncoding.EncodeToString(hashBytes)
		hashString = MakeB64UrlEncode(hashString)
	} else {
		hashString = base32.StdEncoding.EncodeToString(hashBytes)
	}
	if _strSize > 0 && len(hashString) > int(_strSize) {
		hashString = hashString[:_strSize]
	}

	return hashString
}

func FnvHashString32(input string, intSize int, strSize ...int) string {
	hash := FnvHashStringBase(input, false, intSize, strSize...)
	return hash
}

func FnvHashString64(input string, intSize int, strSize ...int) string {
	hash := FnvHashStringBase(input, true, intSize, strSize...)
	return hash
}

func Utf8ToBase32(content string) string {
	encoded := base32.StdEncoding.EncodeToString([]byte(content))
	return encoded
}

func CompleteChar(content, char string, minLen int) string {
	for len(content) < minLen {
		content = char + content
	}
	return content
}

func Base32ToUtf8(content string) string {
	content = strings.ToUpper(content)
	content = strings.ReplaceAll(content, "-", "=")

	decoded, err := base32.StdEncoding.DecodeString(content)
	if err != nil {
		Log("Error al decodificar string:: ", content)
		return ""
	}
	return string(decoded)
}

func Round(h float32) int32 {
	if h == 0 {
		return 0
	}
	return int32(math.Round(float64(h)))
}

func HashSlice[T any](structSlice []T) string {
	options := hashstructure.HashOptions{
		ZeroNil:         true,
		IgnoreZeroValue: true,
		UseStringer:     true,
	}

	hashSum := uint64(0)
	for _, e := range structSlice {
		hash, err := hashstructure.Hash(e, hashstructure.FormatV2, &options)
		if err != nil {
			fmt.Println("Error al calcular el hash.")
			panic(err)
		}
		hashSum += hash
	}

	// Convert the uint64 to a byte slice
	byteValue := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		byteValue[i] = byte(hashSum)
		hashSum >>= 8
	}

	base32String := base32.StdEncoding.EncodeToString(byteValue)

	return strings.ToLower(strings.ReplaceAll(base32String, "=", ""))
}

func TextToIntSlice(text, separator string) []int32 {
	sliceOfTexts := strings.Split(text, separator)
	sliceOfInts := []int32{}
	for _, e := range sliceOfTexts {
		value, err := strconv.Atoi(e)
		if err != nil {
			sliceOfInts = append(sliceOfInts, 0)
		} else {
			sliceOfInts = append(sliceOfInts, int32(value))
		}
	}
	return sliceOfInts
}

func SanitizeUrl(queryParams string) string {
	queryParams = strings.ReplaceAll(queryParams, "/", "%2F")
	queryParams = strings.ReplaceAll(queryParams, " ", "%20")
	queryParams = strings.ReplaceAll(queryParams, ":", "%3A")
	return queryParams
}

func SendHttpRequest[T any](req *http.Request, output *T) error {
	return SendHttpRequestBase(req, output, true)
}

func SendHttpRequestNoTLS[T any](req *http.Request, output *T) error {
	return SendHttpRequestBase(req, output, false)
}

func SendHttpRequestBase[T any](req *http.Request, output *T, useTls bool) error {

	Log("Realizando HTTP request: ", req.Host, req.URL.RawPath)

	bodyBytes, err := SendHttpRequestT(req, useTls)
	if err != nil {
		return err
	}

	Log("Consulta recibida. Leyendo Body....", " | ", len(bodyBytes)/1000, " kb", "|", req.URL)

	err = json.Unmarshal(bodyBytes, output)
	if err != nil {
		bodyString := string(bodyBytes)
		bodylen := If(len(bodyString) > 400, 400, len(bodyString))

		Log("Body recibido:: ")
		Log(bodyString[0:bodylen])
		//Print(bodyString)
		return errors.New("Error al deserializar el body: " + err.Error())
	}

	return nil
}

func SendHttpRequestT(req *http.Request, useTls bool) ([]byte, error) {

	client := &http.Client{
		Timeout: time.Second * 60,
	}

	if useTls {
		client.Transport = &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
			},
		}
	}

	response, err := client.Do(req)
	if err != nil {
		return []byte{}, errors.New("Error en el request: " + err.Error())
	}
	defer response.Body.Close()

	bodyReader := response.Body
	buf := new(strings.Builder)
	_, err = io.Copy(buf, bodyReader)

	if err != nil {
		return []byte{}, errors.New("Error al obtener el body: " + err.Error())
	}
	bodyString := buf.String()
	bodyString = strings.Trim(bodyString, " \n\t")
	bodyBytes := []byte(bodyString)
	bodyBytes = bytes.TrimPrefix(bodyBytes, []byte("\xef\xbb\xbf"))

	if response.StatusCode != 200 {
		Log(bodyString)
		return bodyBytes, Err("Error en respuesta. Status: ", response.StatusCode)
	}

	return bodyBytes, nil
}

func TruncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func SrtToInt32(srt string) int32 {
	return int32(SrtToInt(srt))
}

func SrtToInt(srt string) int32 {
	if srt == "" {
		return 0
	}

	value, err := strconv.Atoi(srt)
	if err != nil {
		Log("Error en convertir a int: ", srt, " | ", err)
		return 0
	}
	return int32(value)
}

func AnyToInt(anyValue any) int32 {
	if srt, ok := anyValue.(string); ok {
		return SrtToInt(srt)
	}
	if floatValue, ok := anyValue.(float64); ok {
		return int32(floatValue)
	}
	if floatValue, ok := anyValue.(float32); ok {
		return int32(floatValue)
	}

	if intValue, ok := anyValue.(int); ok {
		return int32(intValue)
	}
	if intValue, ok := anyValue.(int32); ok {
		return intValue
	}

	Err("No se pudo convertir a int32", anyValue)
	return 0
}

func MapGetKeys(map1 map[string]string, keys ...string) string {
	for _, key := range keys {
		if value2, ok := map1[key]; ok {
			if value2 != "" {
				return value2
			}
		}
	}
	return ""
}

func HoraStringToInt(horaStr string) int32 {

	horaStrSlice := strings.Split(strings.TrimSpace(horaStr), ":")

	if len(horaStrSlice) != 3 {
		return 0
	}

	horas := SrtToInt32(horaStrSlice[0])
	minutos := SrtToInt32(horaStrSlice[1])
	segundos := SrtToInt32(horaStrSlice[2])

	return (horas * 60 * 60) + (minutos * 60) + segundos
}

func SrtToFloat(srt string) float32 {
	if srt == "" {
		return 0
	}
	value, err := strconv.ParseFloat(srt, 32)
	if err != nil {
		Log("Error en convertir a int: ", srt, " | ", err)
		return 0
	}
	return float32(value)
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func ArrayToString[T Number1](a []T, delim string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
}

func Coalesce[T Number2](num1, num2 T) T {
	if num1 == 0 {
		return num2
	}
	return num1
}

func IfNull[T any](num1 *T, num2 T) T {
	if num1 == nil {
		return num2
	} else {
		return *num1
	}
}

func SrtNull(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func If[T any](ok bool, A T, B T) T {
	if ok {
		return A
	} else {
		return B
	}
}

func IF(ok bool, exec func()) {
	if ok {
		exec()
	}
}

func SafeNull[T Number2](num *T) T {
	if num == nil {
		return 0
	} else {
		return *num
	}
}

func PrintTable[T any](records []T, maxLenSlice, maxLenContent int, columns ...string) {
	if maxLenSlice > 0 && len(records) > maxLenSlice {
		records = records[0:maxLenSlice]
	}

	// Log("registros mapeados:: ", len(records))
	// Print(records)
	recordsMapped := []map[string]any{}
	avoidKeys := map[string]bool{}
	includedColumns := map[string]bool{}
	for _, e := range columns {
		includedColumns[e] = true
	}

	for _, e := range records {
		rec := smapping.MapFields(e)
		for key, value := range rec {
			if _, ok := avoidKeys[key]; ok {
				delete(rec, key)
				continue
			}
			if len(columns) > 0 {
				if _, ok := includedColumns[key]; !ok {
					delete(rec, key)
					continue
				}
			}
			valueString := fmt.Sprintf("%v", value)
			if maxLenContent > 0 && len(valueString) > maxLenContent {
				valueString = valueString[0:maxLenContent]
				rec[key] = valueString
			}
			includedColumns[key] = true
		}
		recordsMapped = append(recordsMapped, rec)
	}

	columnsAll := []any{}
	columnsAllString := []string{}
	for e := range includedColumns {
		columnsAll = append(columnsAll, e)
		columnsAllString = append(columnsAllString, e)
	}

	newTable := table.New(columnsAll...)
	for _, e := range recordsMapped {
		row := []any{}
		for _, col := range columnsAllString {
			row = append(row, e[col])
		}
		// Log("agregando: ", row)
		newTable.AddRow(row...)
	}
	newTable.Print()
}

func GetHoursMinutes() string {
	currentTime := time.Now()
	for i := -1; i < 2; i++ {
		currTime := currentTime.Add(time.Minute * time.Duration(i))

		hour := currTime.Hour()
		minutes := currTime.Minute()
		minutes10 := int(math.Round(float64(minutes)/10)) * 10
		// Log("minutes 10:: ", minutes10)
		if minutes != minutes10 {
			continue
		}
		hourSrt := strconv.Itoa(hour)
		if len(hourSrt) == 1 {
			hourSrt = "0" + hourSrt
		}
		minutesSrt := strconv.Itoa(minutes)
		if len(minutesSrt) == 1 {
			minutesSrt = "0" + minutesSrt
		}
		return hourSrt + ":" + minutesSrt
	}
	return ""
}

func MakeB64UrlEncode(contentString string) string {
	contentString = strings.ReplaceAll(contentString, "/", "_")
	contentString = strings.ReplaceAll(contentString, "+", "-")
	contentString = strings.ReplaceAll(contentString, "=", "~")
	return contentString
}

func MakeB64UrlDecode(contentString string) string {
	contentString = strings.ReplaceAll(contentString, "_", "/")
	contentString = strings.ReplaceAll(contentString, "-", "+")
	contentString = strings.ReplaceAll(contentString, "~", "=")
	return contentString
}

func MakeCipherKey() string {
	key := ""
	for len(key) < 32 {
		key += Env.SECRET_PHRASE
	}
	return key[:32]
}

func Encrypt(data []byte, cypherKey_ ...string) ([]byte, error) {
	cypherKey := MakeCipherKey()
	if len(cypherKey_) == 1 {
		cypherKey = cypherKey_[0][:32]
	}

	block, err := aes.NewCipher([]byte(cypherKey))
	if err != nil {
		return nil, err
	}

	// Generate a random nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Encrypt the data
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)

	// Combine the nonce and ciphertext for storage
	encriptedBytes := append(nonce, ciphertext...)
	return encriptedBytes, nil
}

func Decrypt(encryptedData []byte, cypherKey_ ...string) ([]byte, error) {
	cypherKey := MakeCipherKey()
	if len(cypherKey_) == 1 {
		cypherKey = cypherKey[:32]
	}

	block, err := aes.NewCipher([]byte(cypherKey))
	if err != nil {
		return nil, err
	}

	// Extract the nonce from the first 12 bytes
	nonce := encryptedData[:12]
	ciphertext := encryptedData[12:]

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Decrypt the data
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
