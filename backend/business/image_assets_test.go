package business

import (
	"encoding/base64"
	"testing"
)

func TestEncodeImageAssetBigramsPreservesSignedBytes(t *testing.T) {
	encoded := encodeImageAssetBigrams([]int8{0, 1, 127, -128, -1})
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode image asset bigrams: %v", err)
	}

	expected := []byte{0, 1, 127, 128, 255}
	if string(decoded) != string(expected) {
		t.Fatalf("decoded bigrams = %v, want %v", decoded, expected)
	}
}
